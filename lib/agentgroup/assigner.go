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

package agentgroup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

const (
	// defaultResyncPeriod is how long the groups read from the management cluster are trusted
	// without reading them again. A change made by this process always makes them be read again.
	defaultResyncPeriod = 5 * time.Minute
)

// poolKey identifies what an Assigner keeps in memory and serializes: all the groups of a
// component for one cluster namespace (every shard of it).
type poolKey struct {
	component        string
	clusterNamespace string
}

// poolState is what an Assigner knows about the groups of one poolKey.
type poolState struct {
	// mu serializes the decisions (and the writes) about this key: two reconcilers handling
	// the same cluster, or clusters of the same pool, never interleave.
	mu sync.Mutex

	// groups are the groups as last read. They are only valid when loaded is true.
	groups   []Group
	loaded   bool
	loadedAt time.Time
}

// Assigner places clusters in groups and takes them out again, like AssignCluster and
// RemoveCluster, keeping the groups in memory.
//
// The ConfigMaps of the groups remain the source of truth, read by the agents. The Assigner is
// a write-through cache of them:
//   - the groups of a pool are read from the management cluster the first time they are needed,
//     which is also how they are rebuilt when the process restarts;
//   - a cluster already in a group is answered from memory, without calling the API server;
//   - decisions about a pool are taken one at a time, so they never see a stale view of it;
//   - every write still carries the resourceVersion that was read, so a change made by someone
//     else (another process, a person) makes the write fail with a conflict. The groups are
//     then read again and the operation retried;
//   - the groups are read again after any change made here and after resyncPeriod, which also
//     picks up ConfigMaps changed or deleted by someone else.
//
// It assumes one process makes the placement decisions for a given component and cluster
// namespace (as the controller owning the agent does). reader should not be served from a
// cache. An Assigner is safe for concurrent use.
type Assigner struct {
	reader       client.Reader
	writer       client.Client
	namespace    string
	resyncPeriod time.Duration

	mu    sync.Mutex
	pools map[poolKey]*poolState
}

// NewAssigner returns an Assigner for the groups (ConfigMaps) living in namespace, which is
// where the agents run.
func NewAssigner(reader client.Reader, writer client.Client, namespace string) *Assigner {
	return &Assigner{
		reader:       reader,
		writer:       writer,
		namespace:    namespace,
		resyncPeriod: defaultResyncPeriod,
		pools:        map[poolKey]*poolState{},
	}
}

// AssignCluster makes sure member is part of a group of pool and returns that group. See the
// package level AssignCluster for how the group is chosen.
func (a *Assigner) AssignCluster(ctx context.Context, pool Pool, member Member, capacity int,
) (*AssignResult, error) {

	if capacity < 1 {
		return nil, fmt.Errorf("capacity must be at least 1, got %d", capacity)
	}

	state := a.getPoolState(pool.Component, pool.ClusterNamespace)
	state.mu.Lock()
	defer state.mu.Unlock()

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := backoff(ctx, attempt); err != nil {
			return nil, err
		}

		if err := a.load(ctx, state, pool.Component, pool.ClusterNamespace); err != nil {
			return nil, err
		}

		groups := filterShard(state.groups, pool)
		if result := alreadyAssigned(groups, member); result != nil {
			return result, nil
		}

		result, retry, err := assign(ctx, a.writer, a.namespace, pool, groups, member, capacity)

		// Whatever happened, what is in memory may not be what is in the management cluster anymore
		state.loaded = false

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

// RemoveCluster takes the cluster out of every group of the component that lists it, and returns
// the names of the groups that were left without members and so were deleted (the ConfigMap
// only: the caller must delete the agent Deployment with the same name). See the package level
// RemoveCluster.
//
// A cluster that is not in any group (as far as is known) does not make any API call.
func (a *Assigner) RemoveCluster(ctx context.Context, component, clusterNamespace, clusterName string,
	clusterType libsveltosv1beta1.ClusterType) (emptied []string, err error) {

	state := a.getPoolState(component, clusterNamespace)
	state.mu.Lock()
	defer state.mu.Unlock()

	if err := a.load(ctx, state, component, clusterNamespace); err != nil {
		return nil, err
	}

	if findMember(state.groups, Member{Name: clusterName, ClusterType: clusterType}) == nil {
		return nil, nil
	}

	// The groups are read again from the management cluster by RemoveCluster itself
	state.loaded = false
	return RemoveCluster(ctx, a.reader, a.writer, a.namespace, component, clusterNamespace, clusterName, clusterType)
}

// alreadyAssigned returns where member is, if it is already in one of the groups exactly as it
// is asked to be (a change of its watch namespaces has to be written).
func alreadyAssigned(groups []Group, member Member) *AssignResult {
	group := findMember(groups, member)
	if group == nil {
		return nil
	}

	index := memberIndex(group.Members, member.Name, member.ClusterType)
	if !equalMembers(group.Members[index], member) {
		return nil
	}

	return &AssignResult{GroupName: group.Name}
}

func (a *Assigner) getPoolState(component, clusterNamespace string) *poolState {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := poolKey{component: component, clusterNamespace: clusterNamespace}
	state, ok := a.pools[key]
	if !ok {
		state = &poolState{}
		a.pools[key] = state
	}
	return state
}

// load reads the groups from the management cluster, unless the ones in memory can be trusted.
// The caller holds state.mu.
func (a *Assigner) load(ctx context.Context, state *poolState, component, clusterNamespace string) error {
	if state.loaded && time.Since(state.loadedAt) < a.resyncPeriod {
		return nil
	}

	groups, err := ListGroups(ctx, a.reader, a.namespace, component, clusterNamespace)
	if err != nil {
		state.loaded = false
		return err
	}

	state.groups = groups
	state.loaded = true
	state.loadedAt = time.Now()
	return nil
}
