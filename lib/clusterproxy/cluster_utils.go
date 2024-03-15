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

package clusterproxy

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/logsettings"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
	"github.com/projectsveltos/libsveltos/lib/roles"
	"github.com/projectsveltos/libsveltos/lib/sharding"
)

const (
	kubernetesAdmin    = "kubernetes-admin"
	nilSelectorMessage = "selector is nil"
)

var (
	checkedCAPIPresence int32
	capiPresent         int32
)

// getSveltosCluster returns SveltosCluster
func getSveltosCluster(ctx context.Context, c client.Client,
	clusterNamespace, clusterName string) (*libsveltosv1alpha1.SveltosCluster, error) {

	clusterNamespacedName := types.NamespacedName{
		Namespace: clusterNamespace,
		Name:      clusterName,
	}

	cluster := &libsveltosv1alpha1.SveltosCluster{}
	if err := c.Get(ctx, clusterNamespacedName, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// getCAPICluster returns CAPI Cluster
func getCAPICluster(ctx context.Context, c client.Client,
	clusterNamespace, clusterName string) (*clusterv1.Cluster, error) {

	clusterNamespacedNamed := types.NamespacedName{
		Namespace: clusterNamespace,
		Name:      clusterName,
	}

	cluster := &clusterv1.Cluster{}
	if err := c.Get(ctx, clusterNamespacedNamed, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// getCluster returns the cluster object
func GetCluster(ctx context.Context, c client.Client,
	clusterNamespace, clusterName string, clusterType libsveltosv1alpha1.ClusterType) (client.Object, error) {

	if clusterType == libsveltosv1alpha1.ClusterTypeSveltos {
		return getSveltosCluster(ctx, c, clusterNamespace, clusterName)
	}
	return getCAPICluster(ctx, c, clusterNamespace, clusterName)
}

// isCAPIClusterPaused returns true if CAPI Cluster is paused
func isCAPIClusterPaused(ctx context.Context, c client.Client,
	clusterNamespace, clusterName string) (bool, error) {

	cluster, err := getCAPICluster(ctx, c, clusterNamespace, clusterName)
	if err != nil {
		return false, err
	}

	return cluster.Spec.Paused, nil
}

// isSveltosClusterPaused returns true if Cluster is paused
func isSveltosClusterPaused(ctx context.Context, c client.Client,
	clusterNamespace, clusterName string) (bool, error) {

	cluster, err := getSveltosCluster(ctx, c, clusterNamespace, clusterName)
	if err != nil {
		return false, err
	}

	return cluster.Spec.Paused, nil
}

// IsClusterPaused returns true if cluster is currently paused
func IsClusterPaused(ctx context.Context, c client.Client,
	clusterNamespace, clusterName string, clusterType libsveltosv1alpha1.ClusterType) (bool, error) {

	if clusterType == libsveltosv1alpha1.ClusterTypeSveltos {
		return isSveltosClusterPaused(ctx, c, clusterNamespace, clusterName)
	}
	return isCAPIClusterPaused(ctx, c, clusterNamespace, clusterName)
}

func getKubernetesRestConfigForAdmin(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, adminNamespace, adminName string,
	clusterType libsveltosv1alpha1.ClusterType, logger logr.Logger) (*rest.Config, error) {

	kubeconfigContent, err := roles.GetKubeconfig(ctx, c, clusterNamespace, clusterName,
		adminNamespace, adminName, clusterType)
	if err != nil {
		return nil, err
	}

	kubeconfig, err := CreateKubeconfig(logger, kubeconfigContent)
	if err != nil {
		return nil, err
	}
	defer os.Remove(kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Error(err, "BuildConfigFromFlags")
		return nil, errors.Wrap(err, "BuildConfigFromFlags")
	}

	return config, nil
}

func getKubernetesClientForAdmin(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, adminNamespace, adminName string,
	clusterType libsveltosv1alpha1.ClusterType, logger logr.Logger) (client.Client, error) {

	config, err := getKubernetesRestConfigForAdmin(ctx, c, clusterNamespace, clusterName,
		adminNamespace, adminName, clusterType, logger)
	if err != nil {
		return nil, err
	}
	logger.V(logs.LogVerbose).Info("return new client")
	return client.New(config, client.Options{Scheme: c.Scheme()})
}

// GetSecretData returns Kubeconfig to access cluster
func GetSecretData(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, adminNamespace, adminName string,
	clusterType libsveltosv1alpha1.ClusterType, logger logr.Logger) ([]byte, error) {

	if adminName != "" && adminName != kubernetesAdmin {
		return roles.GetKubeconfig(ctx, c, clusterNamespace, clusterName,
			adminNamespace, adminName, clusterType)
	}

	if clusterType == libsveltosv1alpha1.ClusterTypeSveltos {
		return GetSveltosSecretData(ctx, logger, c, clusterNamespace, clusterName)
	}
	return GetCAPISecretData(ctx, logger, c, clusterNamespace, clusterName)
}

// GetKubernetesRestConfig returns restConfig for a cluster
func GetKubernetesRestConfig(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, adminNamespace, adminName string,
	clusterType libsveltosv1alpha1.ClusterType, logger logr.Logger) (*rest.Config, error) {

	if adminName != "" && adminName != kubernetesAdmin {
		return getKubernetesRestConfigForAdmin(ctx, c, clusterNamespace, clusterName,
			adminNamespace, adminName, clusterType, logger)
	}

	if clusterType == libsveltosv1alpha1.ClusterTypeSveltos {
		return GetSveltosKubernetesRestConfig(ctx, logger, c, clusterNamespace, clusterName)
	}
	return GetCAPIKubernetesRestConfig(ctx, logger, c, clusterNamespace, clusterName)
}

// GetKubernetesClient returns client to access cluster
func GetKubernetesClient(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, adminNamespace, adminName string,
	clusterType libsveltosv1alpha1.ClusterType, logger logr.Logger) (client.Client, error) {

	if adminName != "" && adminName != kubernetesAdmin {
		return getKubernetesClientForAdmin(ctx, c, clusterNamespace, clusterName,
			adminNamespace, adminName, clusterType, logger)
	}

	if clusterType == libsveltosv1alpha1.ClusterTypeSveltos {
		return GetSveltosKubernetesClient(ctx, logger, c, c.Scheme(), clusterNamespace, clusterName)
	}
	return GetCAPIKubernetesClient(ctx, logger, c, c.Scheme(), clusterNamespace, clusterName)
}

// GetClusterType returns clustertype for a given cluster
func GetClusterType(cluster *corev1.ObjectReference) libsveltosv1alpha1.ClusterType {
	// TODO: remove this
	if cluster.APIVersion != libsveltosv1alpha1.GroupVersion.String() &&
		cluster.APIVersion != clusterv1.GroupVersion.String() {

		panic(1)
	}

	clusterType := libsveltosv1alpha1.ClusterTypeCapi
	if cluster.APIVersion == libsveltosv1alpha1.GroupVersion.String() {
		clusterType = libsveltosv1alpha1.ClusterTypeSveltos
	}
	return clusterType
}

func addTypeInformationToObject(scheme *runtime.Scheme, obj client.Object) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		panic(1)
	}

	for _, gvk := range gvks {
		if gvk.Kind == "" {
			continue
		}
		if gvk.Version == "" || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}
}

// getListOfCAPIClusters returns all CAPI Clusters.
// If shard is set, returns only clusters matching shard.
func getListOfCAPICluster(ctx context.Context, c client.Client, namespace string, shard *string,
	logger logr.Logger) ([]corev1.ObjectReference, error) {

	present, err := isCAPIPresent(ctx, c, logger)
	if err != nil {
		logger.Error(err, "failed to verify if ClusterAPI Cluster CRD is installed")
		return nil, err
	}

	if !present {
		return nil, nil
	}

	listOptions := []client.ListOption{}
	if namespace != "" {
		listOptions = append(listOptions, client.InNamespace(namespace))
	}

	clusterList := &clusterv1.ClusterList{}
	if err := c.List(ctx, clusterList, listOptions...); err != nil {
		logger.Error(err, "failed to list all Cluster")
		return nil, err
	}

	clusters := make([]corev1.ObjectReference, 0)

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]

		if !cluster.DeletionTimestamp.IsZero() {
			// Only existing cluster can match
			continue
		}

		if shard != nil && !sharding.IsShardAMatch(*shard, cluster) {
			continue
		}

		addTypeInformationToObject(c.Scheme(), cluster)

		clusters = append(clusters, corev1.ObjectReference{
			Namespace:  cluster.Namespace,
			Name:       cluster.Name,
			APIVersion: cluster.APIVersion,
			Kind:       cluster.Kind,
		})
	}

	return clusters, nil
}

// getListOfSveltosClusters returns all Sveltos Clusters.
// If shard is set, returns only clusters matching shard.
func getListOfSveltosCluster(ctx context.Context, c client.Client, namespace string, shard *string,
	logger logr.Logger) ([]corev1.ObjectReference, error) {

	listOptions := []client.ListOption{}
	if namespace != "" {
		listOptions = append(listOptions, client.InNamespace(namespace))
	}

	clusterList := &libsveltosv1alpha1.SveltosClusterList{}
	if err := c.List(ctx, clusterList, listOptions...); err != nil {
		logger.Error(err, "failed to list all Cluster")
		return nil, err
	}

	clusters := make([]corev1.ObjectReference, 0)

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]

		if !cluster.DeletionTimestamp.IsZero() {
			// Only existing cluster can match
			continue
		}

		if shard != nil && !sharding.IsShardAMatch(*shard, cluster) {
			continue
		}

		addTypeInformationToObject(c.Scheme(), cluster)

		clusters = append(clusters, corev1.ObjectReference{
			Namespace:  cluster.Namespace,
			Name:       cluster.Name,
			APIVersion: cluster.APIVersion,
			Kind:       cluster.Kind,
		})
	}

	return clusters, nil
}

// GetListOfClusters returns all existing Sveltos/CAPI Clusters.
// If namespace is not empty, only existing clusters in that namespace will be returned.
func GetListOfClusters(ctx context.Context, c client.Client, namespace string, logger logr.Logger,
) ([]corev1.ObjectReference, error) {

	clusters, err := getListOfCAPICluster(ctx, c, namespace, nil, logger)
	if err != nil {
		return nil, err
	}

	var tmpClusters []corev1.ObjectReference
	tmpClusters, err = getListOfSveltosCluster(ctx, c, namespace, nil, logger)
	if err != nil {
		return nil, err
	}

	clusters = append(clusters, tmpClusters...)
	return clusters, nil
}

// GetListOfClustersForShardKey returns all existing Sveltos/CAPI Clusters for a given shard
// If namespace is not empty, clusters will be further filtered by namespace.
func GetListOfClustersForShardKey(ctx context.Context, c client.Client, namespace, shard string,
	logger logr.Logger) ([]corev1.ObjectReference, error) {

	clusters, err := getListOfCAPICluster(ctx, c, namespace, &shard, logger)
	if err != nil {
		return nil, err
	}

	var tmpClusters []corev1.ObjectReference
	tmpClusters, err = getListOfSveltosCluster(ctx, c, namespace, &shard, logger)
	if err != nil {
		return nil, err
	}

	clusters = append(clusters, tmpClusters...)
	return clusters, nil
}

func getMatchingCAPIClusters(ctx context.Context, c client.Client, selector labels.Selector,
	namespace string, logger logr.Logger) ([]corev1.ObjectReference, error) {

	if selector == nil {
		logger.V(logs.LogInfo).Info(nilSelectorMessage)
		return nil, fmt.Errorf("%s", nilSelectorMessage)
	}

	present, err := isCAPIPresent(ctx, c, logger)
	if err != nil {
		logger.Error(err, "failed to verify if ClusterAPI Cluster CRD is installed")
		return nil, err
	}

	if !present {
		return nil, nil
	}

	listOptions := []client.ListOption{}
	if namespace != "" {
		listOptions = append(listOptions, client.InNamespace(namespace))
	}

	clusterList := &clusterv1.ClusterList{}
	if err := c.List(ctx, clusterList, listOptions...); err != nil {
		logger.Error(err, "failed to list all Cluster")
		return nil, err
	}

	matching := make([]corev1.ObjectReference, 0)

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]

		if !cluster.DeletionTimestamp.IsZero() {
			// Only existing cluster can match
			continue
		}

		if !isCAPIControlPlaneReady(cluster) {
			// Only ready cluster can match
			continue
		}

		addTypeInformationToObject(c.Scheme(), cluster)
		if selector.Matches(labels.Set(cluster.Labels)) {
			matching = append(matching, corev1.ObjectReference{
				Kind:       cluster.Kind,
				Namespace:  cluster.Namespace,
				Name:       cluster.Name,
				APIVersion: cluster.APIVersion,
			})
		}
	}

	return matching, nil
}

func getMatchingSveltosClusters(ctx context.Context, c client.Client, selector labels.Selector,
	namespace string, logger logr.Logger) ([]corev1.ObjectReference, error) {

	if selector == nil {
		logger.V(logs.LogInfo).Info(nilSelectorMessage)
		return nil, fmt.Errorf("%s", nilSelectorMessage)
	}

	listOptions := []client.ListOption{}
	if namespace != "" {
		listOptions = append(listOptions, client.InNamespace(namespace))
	}

	clusterList := &libsveltosv1alpha1.SveltosClusterList{}
	if err := c.List(ctx, clusterList, listOptions...); err != nil {
		logger.Error(err, "failed to list all Cluster")
		return nil, err
	}

	matching := make([]corev1.ObjectReference, 0)

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]

		if !cluster.DeletionTimestamp.IsZero() {
			// Only existing cluster can match
			continue
		}

		if !isSveltosClusterStatusReady(cluster) {
			// Only ready cluster can match
			continue
		}

		addTypeInformationToObject(c.Scheme(), cluster)
		if selector.Matches(labels.Set(cluster.Labels)) {
			matching = append(matching, corev1.ObjectReference{
				Kind:       cluster.Kind,
				Namespace:  cluster.Namespace,
				Name:       cluster.Name,
				APIVersion: cluster.APIVersion,
			})
		}
	}

	return matching, nil
}

// GetMatchingClusters returns all Sveltos/CAPI Clusters currently matching selector
func GetMatchingClusters(ctx context.Context, c client.Client, selector labels.Selector,
	namespace string, logger logr.Logger) ([]corev1.ObjectReference, error) {

	if selector == nil {
		logger.V(logs.LogInfo).Info(nilSelectorMessage)
		return nil, fmt.Errorf("%s", nilSelectorMessage)
	}

	matching := make([]corev1.ObjectReference, 0)

	tmpMatching, err := getMatchingCAPIClusters(ctx, c, selector, namespace, logger)
	if err != nil {
		return nil, err
	}

	matching = append(matching, tmpMatching...)

	tmpMatching, err = getMatchingSveltosClusters(ctx, c, selector, namespace, logger)
	if err != nil {
		return nil, err
	}

	matching = append(matching, tmpMatching...)

	return matching, nil
}

// isCAPIPresent returns whether clusterAPI CRDs are present.
// This is checked only once. In a situation where clusterAPI is not originally
// present and installed later on, all sveltos components that depend on that
// are restarted.
func isCAPIPresent(ctx context.Context, c client.Client, logger logr.Logger) (bool, error) {
	checked := atomic.LoadInt32(&checkedCAPIPresence)
	if checked == 0 {
		clusterCRD := &apiextensionsv1.CustomResourceDefinition{}
		err := c.Get(ctx,
			types.NamespacedName{Name: "clusters.cluster.x-k8s.io"}, clusterCRD)
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.V(logsettings.LogDebug).Info("clusterCRD CRD not present")
				atomic.StoreInt32(&checkedCAPIPresence, 1)
				atomic.StoreInt32(&capiPresent, 0)
				return false, nil
			}
			return false, err
		}
		atomic.StoreInt32(&checkedCAPIPresence, 1)
		atomic.StoreInt32(&capiPresent, 1)
	}

	present := atomic.LoadInt32(&capiPresent)
	return present != 0, nil
}
