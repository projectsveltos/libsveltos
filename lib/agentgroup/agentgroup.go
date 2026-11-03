/*
Copyright 2026. projectsveltos.io. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package agentgroup lets a single agent instance (drift-detection-manager, sveltos-agent
// running in the management cluster) serve a group of clusters, instead of one instance per
// cluster.
//
// A group is identified by a ConfigMap, created in the namespace where the agents run, whose
// data lists the clusters (members) the group's agent must serve. The agent Deployment of the
// group carries the same name as its ConfigMap. The component creating the Deployment (the
// controller that owns the agent) is the only writer of the ConfigMap: it uses AssignCluster to
// add a cluster to a group and RemoveCluster to take it out again.
//
// A group only contains clusters of one pool: same component, same cluster namespace, same
// shard. A pool may have many groups; clusters are placed in the lowest-numbered group that
// still has room (sticky first-fit). Clusters never move between groups.
//
// AssignCluster and RemoveCluster keep no state in memory. An Assigner wraps them with a
// write-through cache of the groups and serializes the decisions of a pool, which is what a
// controller reconciling several clusters in parallel should use. Whichever is used,
// concurrent writers (several processes, a person editing a ConfigMap) are handled with what
// the API server already guarantees:
//   - group names are deterministic (component, pool hash, index), so two writers creating the
//     same group collide with AlreadyExists;
//   - membership changes are Updates carrying the resourceVersion that was read, so a
//     concurrent change makes the write fail with a conflict and the operation is retried;
//   - a group is deleted with a resourceVersion precondition, so a group that just received a
//     new member is never deleted.
package agentgroup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

const (
	// ComponentLabel identifies the component (for instance drift-detection) a group belongs to.
	ComponentLabel = "agentgroup.projectsveltos.io/component"

	// ClusterNamespaceLabel is the namespace of all the clusters in the group.
	ClusterNamespaceLabel = "agentgroup.projectsveltos.io/cluster-namespace"

	// ShardLabel is set on groups of a sharded pool. Its value is a hash of the shard key
	// (a shard key is free text, so it cannot be used as a label value as is).
	ShardLabel = "agentgroup.projectsveltos.io/shard"

	// membersKey is the key, in the ConfigMap data, holding the JSON encoded list of members.
	membersKey = "members"

	// maxAttempts bounds how many times an operation is retried because of concurrent
	// changes before giving up and returning an error (the caller requeues).
	maxAttempts = 30

	// retryJitter is the base of the random wait before a retry. Writers racing on the same
	// group would otherwise keep conflicting with each other in lockstep.
	retryJitter = 5 * time.Millisecond

	hashLength = 8
)

// Member is a cluster served by a group.
type Member struct {
	// Name is the name of the cluster. The namespace is the one of the group.
	Name string `json:"name"`

	// ClusterType is Capi or Sveltos.
	ClusterType libsveltosv1beta1.ClusterType `json:"clusterType"`

	// WatchNamespaces, when not empty, restricts the namespaced resources the agent watches
	// in this cluster (namespace-scoped watch mode).
	WatchNamespaces []string `json:"watchNamespaces,omitempty"`
}

// Pool identifies the set of groups a cluster can be placed in.
type Pool struct {
	// Component is the agent the groups are for (for instance drift-detection).
	Component string

	// ClusterNamespace is the namespace of the clusters.
	ClusterNamespace string

	// Shard is the shard key of the clusters. Empty when the cluster is not sharded.
	Shard string
}

// Group is a group, as read from the management cluster.
type Group struct {
	// Name is the name of the ConfigMap and of the agent Deployment of the group.
	Name string

	// ClusterNamespace is the namespace of all members.
	ClusterNamespace string

	// Members are the clusters served by the group.
	Members []Member

	configMap *corev1.ConfigMap
}

// AssignResult describes where a cluster was placed.
type AssignResult struct {
	// GroupName is the name of the group (ConfigMap and agent Deployment) serving the cluster.
	GroupName string

	// Created is true when the group did not exist and was created by this call. The caller
	// is expected to create the agent Deployment.
	Created bool
}

// Labels returns the labels identifying the pool. They are set on the ConfigMap of each group
// and the caller is expected to set them on the agent Deployment as well.
func (p Pool) Labels() map[string]string {
	lbls := map[string]string{
		ComponentLabel:        p.Component,
		ClusterNamespaceLabel: p.ClusterNamespace,
	}
	if p.Shard != "" {
		lbls[ShardLabel] = shortHash(p.Shard)
	}
	return lbls
}

// GroupName returns the name of the index-th group of the pool.
func (p Pool) GroupName(index int) string {
	key := p.ClusterNamespace + "/" + p.Shard
	return fmt.Sprintf("%s-%s-%d", p.Component, shortHash(key), index)
}

// ParseMembers returns the members listed in the ConfigMap of a group.
func ParseMembers(configMap *corev1.ConfigMap) ([]Member, error) {
	data, ok := configMap.Data[membersKey]
	if !ok || data == "" {
		return nil, nil
	}

	var members []Member
	if err := json.Unmarshal([]byte(data), &members); err != nil {
		return nil, fmt.Errorf("failed to parse members of group %s/%s: %w",
			configMap.Namespace, configMap.Name, err)
	}

	return members, nil
}

// ListGroups returns all the groups of a component, in the given namespace, for the given
// cluster namespace. An empty clusterNamespace returns the groups for all cluster namespaces.
// Use a reader that does not go through a cache when the result decides what to write.
func ListGroups(ctx context.Context, reader client.Reader, namespace, component, clusterNamespace string,
) ([]Group, error) {

	selector := client.MatchingLabels{ComponentLabel: component}
	if clusterNamespace != "" {
		selector[ClusterNamespaceLabel] = clusterNamespace
	}

	configMaps := &corev1.ConfigMapList{}
	if err := reader.List(ctx, configMaps, client.InNamespace(namespace), selector); err != nil {
		return nil, err
	}

	groups := make([]Group, 0, len(configMaps.Items))
	for i := range configMaps.Items {
		members, err := ParseMembers(&configMaps.Items[i])
		if err != nil {
			return nil, err
		}
		groups = append(groups, Group{
			Name:             configMaps.Items[i].Name,
			ClusterNamespace: configMaps.Items[i].Labels[ClusterNamespaceLabel],
			Members:          members,
			configMap:        &configMaps.Items[i],
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groupIndex(groups[i].Name) < groupIndex(groups[j].Name)
	})

	return groups, nil
}

// AssignCluster makes sure member is part of a group of pool and returns that group.
//
// The member is placed in the lowest-numbered group with fewer than capacity members. When all
// groups are full, a new one is created. A member already in a group stays where it is, whatever
// capacity is: lowering the capacity never moves clusters.
//
// reader must not be served from a cache: a stale list could place the same cluster in two
// groups. namespace is where the agents (and the ConfigMaps of the groups) live.
func AssignCluster(ctx context.Context, reader client.Reader, writer client.Client,
	namespace string, pool Pool, member Member, capacity int) (*AssignResult, error) {

	if capacity < 1 {
		return nil, fmt.Errorf("capacity must be at least 1, got %d", capacity)
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := backoff(ctx, attempt); err != nil {
			return nil, err
		}

		groups, err := ListGroups(ctx, reader, namespace, pool.Component, pool.ClusterNamespace)
		if err != nil {
			return nil, err
		}

		groups = filterShard(groups, pool)

		result, retry, err := assign(ctx, writer, namespace, pool, groups, member, capacity)
		if err != nil {
			return nil, err
		}
		if !retry {
			return result, nil
		}
	}

	return nil, fmt.Errorf("failed to assign cluster %s/%s to a group: too many concurrent changes",
		pool.ClusterNamespace, member.Name)
}

// assign tries once to place member. retry is true when a concurrent change was detected and
// the caller must list the groups again and call assign anew.
func assign(ctx context.Context, writer client.Client, namespace string, pool Pool, groups []Group,
	member Member, capacity int) (result *AssignResult, retry bool, err error) {

	if group := findMember(groups, member); group != nil {
		return updateExistingMember(ctx, writer, group, member)
	}

	for i := range groups {
		if len(groups[i].Members) >= capacity {
			continue
		}

		members := append(append([]Member{}, groups[i].Members...), member)
		return writeMembers(ctx, writer, &groups[i], members)
	}

	return createGroup(ctx, writer, namespace, pool, groups, member)
}

// findMember returns the first group listing member (same name and type), or nil.
func findMember(groups []Group, member Member) *Group {
	for i := range groups {
		if memberIndex(groups[i].Members, member.Name, member.ClusterType) >= 0 {
			return &groups[i]
		}
	}
	return nil
}

func updateExistingMember(ctx context.Context, writer client.Client, group *Group, member Member,
) (*AssignResult, bool, error) {

	index := memberIndex(group.Members, member.Name, member.ClusterType)
	if equalMembers(group.Members[index], member) {
		return &AssignResult{GroupName: group.Name}, false, nil
	}

	members := append([]Member{}, group.Members...)
	members[index] = member
	return writeMembers(ctx, writer, group, members)
}

// writeMembers replaces the members of an existing group. A conflict (or the group having just
// been deleted) means groups changed since they were listed: retry is returned.
func writeMembers(ctx context.Context, writer client.Client, group *Group, members []Member,
) (*AssignResult, bool, error) {

	configMap := group.configMap.DeepCopy()
	if err := setMembers(configMap, members); err != nil {
		return nil, false, err
	}

	if err := writer.Update(ctx, configMap); err != nil {
		if apierrors.IsConflict(err) || apierrors.IsNotFound(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	return &AssignResult{GroupName: group.Name}, false, nil
}

func createGroup(ctx context.Context, writer client.Client, namespace string, pool Pool, groups []Group,
	member Member) (*AssignResult, bool, error) {

	name := pool.GroupName(lowestUnusedIndex(groups))

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    pool.Labels(),
		},
	}
	if err := setMembers(configMap, []Member{member}); err != nil {
		return nil, false, err
	}

	if err := writer.Create(ctx, configMap); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// someone else just created this group: look again.
			return nil, true, nil
		}
		return nil, false, err
	}

	return &AssignResult{GroupName: name, Created: true}, false, nil
}

// RemoveCluster takes the cluster out of every group of the component that lists it. Cluster
// namespace is enough to find the groups: the shard of a cluster being deleted is not reliable.
//
// It returns the names of the groups that were left without members, and so were deleted (the
// ConfigMap only: the caller must delete the agent Deployment with the same name). A group that
// receives a new member while its last one is being removed is not deleted.
//
// It is idempotent: removing a cluster that is not in any group does nothing.
func RemoveCluster(ctx context.Context, reader client.Reader, writer client.Client,
	namespace, component, clusterNamespace, clusterName string, clusterType libsveltosv1beta1.ClusterType,
) (emptied []string, err error) {

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := backoff(ctx, attempt); err != nil {
			return emptied, err
		}

		groups, err := ListGroups(ctx, reader, namespace, component, clusterNamespace)
		if err != nil {
			return emptied, err
		}

		retry := false
		for i := range groups {
			index := memberIndex(groups[i].Members, clusterName, clusterType)
			if index < 0 {
				continue
			}

			wasEmptied, conflict, err := removeMember(ctx, writer, &groups[i], index)
			if err != nil {
				return emptied, err
			}
			if conflict {
				retry = true
				continue
			}
			if wasEmptied {
				emptied = append(emptied, groups[i].Name)
			}
		}

		if !retry {
			return emptied, nil
		}
	}

	return emptied, fmt.Errorf("failed to remove cluster %s/%s from its group: too many concurrent changes",
		clusterNamespace, clusterName)
}

func removeMember(ctx context.Context, writer client.Client, group *Group, index int,
) (emptied, conflict bool, err error) {

	if len(group.Members) == 1 {
		// Last member: delete the group, but only if nobody changed it since it was read.
		resourceVersion := group.configMap.ResourceVersion
		err := writer.Delete(ctx, group.configMap, client.Preconditions{ResourceVersion: &resourceVersion})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, false, nil
			}
			if apierrors.IsConflict(err) {
				return false, true, nil
			}
			return false, false, err
		}
		return true, false, nil
	}

	members := append([]Member{}, group.Members[:index]...)
	members = append(members, group.Members[index+1:]...)

	configMap := group.configMap.DeepCopy()
	if err := setMembers(configMap, members); err != nil {
		return false, false, err
	}
	if err := writer.Update(ctx, configMap); err != nil {
		if apierrors.IsConflict(err) || apierrors.IsNotFound(err) {
			return false, true, nil
		}
		return false, false, err
	}

	return false, false, nil
}

// backoff waits a random time before a retry. The first attempt does not wait. The wait grows
// with the attempt number, up to a bound, so a burst of writers spreads out.
func backoff(ctx context.Context, attempt int) error {
	if attempt == 0 {
		return nil
	}

	const maxFactor = 10
	factor := min(attempt, maxFactor)
	wait := time.Duration(rand.Int64N(int64(retryJitter) * int64(factor))) //nolint:gosec // jitter, not security

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		return nil
	}
}

func setMembers(configMap *corev1.ConfigMap, members []Member) error {
	data, err := json.Marshal(members)
	if err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	configMap.Data[membersKey] = string(data)

	return nil
}

func memberIndex(members []Member, name string, clusterType libsveltosv1beta1.ClusterType) int {
	for i := range members {
		if members[i].Name == name && members[i].ClusterType == clusterType {
			return i
		}
	}
	return -1
}

func equalMembers(a, b Member) bool {
	if a.Name != b.Name || a.ClusterType != b.ClusterType || len(a.WatchNamespaces) != len(b.WatchNamespaces) {
		return false
	}
	for i := range a.WatchNamespaces {
		if a.WatchNamespaces[i] != b.WatchNamespaces[i] {
			return false
		}
	}
	return true
}

// filterShard keeps the groups of the pool's shard. Groups of other shards share the cluster
// namespace but must not be used.
func filterShard(groups []Group, pool Pool) []Group {
	want := ""
	if pool.Shard != "" {
		want = shortHash(pool.Shard)
	}

	result := make([]Group, 0, len(groups))
	for i := range groups {
		if groups[i].configMap.Labels[ShardLabel] == want {
			result = append(result, groups[i])
		}
	}
	return result
}

// lowestUnusedIndex returns the smallest group index not used by groups.
func lowestUnusedIndex(groups []Group) int {
	used := make(map[int]struct{}, len(groups))
	for i := range groups {
		used[groupIndex(groups[i].Name)] = struct{}{}
	}

	for index := 0; ; index++ {
		if _, ok := used[index]; !ok {
			return index
		}
	}
}

// groupIndex returns the index in the name of a group (the part after the last dash).
// Names that do not end with a number sort last.
func groupIndex(name string) int {
	const unknown = int(^uint(0) >> 1)

	position := strings.LastIndex(name, "-")
	if position < 0 {
		return unknown
	}

	index, err := strconv.Atoi(name[position+1:])
	if err != nil || index < 0 {
		return unknown
	}

	return index
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:hashLength]
}
