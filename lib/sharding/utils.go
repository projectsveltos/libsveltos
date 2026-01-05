/*
Copyright 2023. projectsveltos.io. All rights reserved.

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

package sharding

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

// Sharding can be used to to horizontal scale sveltos.
// When sharding is used, there will be:
// - One deployment with shard-key argument not set.
// - Zero or more deployments with shard-key argument set.
// A cluster with shardAnnotation will only be managed by
// the corresponding deployment with matching shard-key.
// A cluster with no shardAnnotation will only be managed by
// the deployment started with no shard-key arg.

// IsShardAMatch returns true if a cluster is a shard match.
func IsShardAMatch(shardKey string, cluster client.Object) bool {
	annotations := cluster.GetAnnotations()
	if len(annotations) == 0 {
		// A cluster with no ShardAnnotation is only managed by
		// deployment started with no shard-key arg.
		return shardKey == ""
	}

	v, ok := annotations[libsveltosv1beta1.ShardAnnotation]
	if !ok {
		// A cluster with no ShardAnnotation is only managed by
		// deployment started with no shard-key arg.
		return shardKey == ""
	}

	return v == shardKey
}

// When sharding is used, each cluster at any point of time
// it has none or a shard key.
// It is sometimes necessary to keep track of cluster shard
// being modified. For instance, with respect to add-on and application
// deployment, each addon-controller keeps in memory information
// of which clusterSummary instance is managing an helm chart in a given
// cluster.
// If cluster shard is changed, the new addon-controller will have to detect
// it and rebuild the in-memory information.
// A ConfigMap is used for that. Following methods need to be used by
// any sveltos controller that wants to track this information

// RegisterClusterShard register a cluster,shard pair.
// feature is a string representing the entity requesting the tracking.
// component indicates the sveltos component requesting it.
// returns a bool indicating whether the cluster:shard pair has changed and an error
// if any occurred
func RegisterClusterShard(ctx context.Context, c client.Client, component libsveltosv1beta1.Component,
	feature, shard, clusterNamespace, clusterName string, clusterType libsveltosv1beta1.ClusterType) (bool, error) {

	cm, err := getConfigMap(ctx, c, component, feature)
	if err != nil {
		return false, err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	clusterId := fmt.Sprintf("%s-%s-%s", clusterType, clusterNamespace, clusterName)
	currentShard, ok := cm.Data[clusterId]
	if !ok {
		// First time we track cluster:shard pair

		cm.Data[clusterId] = shard
		return false, updateConfigMap(ctx, c, cm)
	}

	if currentShard != shard {
		// Cluster shard has changed
		cm.Data[clusterId] = shard
		return true, updateConfigMap(ctx, c, cm)
	}

	// Cluster shard has not changed
	return false, nil
}

const (
	configMapName      = "clustersharding"
	configMapNamespace = "projectsveltos"
)

func getConfigMapName(component libsveltosv1beta1.Component, feature string) string {
	return fmt.Sprintf("%s-%s-%s", configMapName,
		strings.ToLower(string(component)),
		strings.ToLower(feature))
}

func getConfigMap(ctx context.Context, c client.Client, component libsveltosv1beta1.Component,
	feature string) (*corev1.ConfigMap, error) {

	cm := &corev1.ConfigMap{}
	name := getConfigMapName(component, feature)

	err := c.Get(ctx, types.NamespacedName{Namespace: configMapNamespace, Name: name}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return createConfigMap(ctx, c, component, feature)
		}
	}

	return cm, nil
}

func createConfigMap(ctx context.Context, c client.Client, component libsveltosv1beta1.Component,
	feature string) (*corev1.ConfigMap, error) {

	name := getConfigMapName(component, feature)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: configMapNamespace,
			Name:      name,
		},
	}

	err := c.Create(ctx, cm)
	return cm, err
}

func updateConfigMap(ctx context.Context, c client.Client, cm *corev1.ConfigMap) error {
	return c.Update(ctx, cm)
}
