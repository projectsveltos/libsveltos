/*
Copyright 2025. projectsveltos.io. All rights reserved.

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

package pullmode

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/logsettings"
)

// GetClusterLabels returns a map of labels used to filter ConfigurationGroups for a specific cluster.
// It takes the cluster namespace and cluster name as input.
func GetClusterLabels(clusterNamespace, clusterName string) map[string]string {
	return map[string]string{
		clusterNameLabelKey: clusterName,
	}
}

// GetRequestorFeature returns a string representing the feature that caused the ConfigurationGroup
// to be created
func GetRequestorFeature(configurationGroup *libsveltosv1beta1.ConfigurationGroup) (string, error) {
	if configurationGroup == nil {
		return "", fmt.Errorf("nil ConfigurationGroup")
	}

	if configurationGroup.Labels == nil {
		return "", fmt.Errorf("ConfigurationGroup has no labels")
	}

	v, ok := configurationGroup.Labels[requestorFeatureLabelKey]
	if !ok {
		return "", fmt.Errorf("ConfigurationGroup does not have expected label")
	}

	return v, nil
}

func GetRequestorKind(configurationGroup *libsveltosv1beta1.ConfigurationGroup) (string, error) {
	if configurationGroup == nil {
		return "", fmt.Errorf("nil ConfigurationGroup")
	}

	if configurationGroup.Labels == nil {
		return "", fmt.Errorf("ConfigurationGroup has no labels")
	}

	kind, ok := configurationGroup.Labels[requestorKindLabelKey]
	if !ok {
		return "", fmt.Errorf("ConfigurationGroup does not have expected label (%s missing)",
			requestorKindLabelKey)
	}

	return kind, nil
}

func GetRequestorName(configurationGroup *libsveltosv1beta1.ConfigurationGroup) (string, error) {
	if configurationGroup == nil {
		return "", fmt.Errorf("nil ConfigurationGroup")
	}

	if configurationGroup.Annotations == nil {
		return "", fmt.Errorf("ConfigurationGroup has no annotations")
	}

	name, ok := configurationGroup.Annotations[requestorNameAnnotationKey]
	if !ok {
		return "", fmt.Errorf("ConfigurationGroup reuestor Kind (%s missing)",
			requestorNameAnnotationKey)
	}

	return name, nil
}

// For SveltosClusters in pull mode, this method is called by management cluster components
// to register resources intended for deployment. The agent in the managed cluster will
// subsequently fetch these resources.
// - requestorKind, requestorName, and requestorFeature uniquely identify the component in the
// management cluster invoking this method.
// - resources is the list of resources to deploy.
// - clusterNamespace/clusterName identify the target SveltosCluster.
func RecordResourcesForDeployment(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	resources map[string][]unstructured.Unstructured, logger logr.Logger, setters ...Option) error {

	bundles := make([]bundleData, len(resources))
	// Create all ConfigurationBundles. There one configurationBundle per key.
	// If Requestor is ClusterSummary each key represents a different ConfigMap/Secret referenced in
	// policyRef section or a different helm chart in the helmChart section.
	i := 0
	for k := range resources {
		bundleName, err := reconcileConfigurationBundle(ctx, c, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, k, resources[k], false, false, logger)
		if err != nil {
			return err
		}
		hash, err := getHash(resources[k])
		if err != nil {
			logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to evaluate hash: %v", err))
		}

		bundles[i] = bundleData{Name: bundleName, Hash: hash}
		i++
	}

	// Now that we have created all ConfigurationBundles, creates a single ConfigurationGroup
	// that references all bundles
	err := reconcileConfigurationGroup(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, bundles, logger, setters...)
	if err != nil {
		return err
	}

	// If ConfigurationGroup is updated, we might have stale configurationBundles. Contininuing
	// on the ClusterSummary example, previously ClusterSummary was referencing ConfigMap1 now it
	// references ConfigMap2. So find and delete all stale ConfigurationBundles.
	return deleteStaleConfigurationBundles(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, bundles, logger)
}

// StageResourcesForDeployment is called by management cluster components to register resources
// that are being prepared for deployment to SveltosClusters in pull mode.
// Unlike RecordResourcesForDeployment, this method does not immediately make the resources
// available to the agent in the managed cluster. Resources staged via this method
// will be made available for deployment at a later point (after CommitStagedResourcesForDeployment
// is called).
// - requestorKind, requestorName, and requestorFeature uniquely identify the component in the
// management cluster invoking this method.
// - resources is the list of resources to deploy.
// - clusterNamespace/clusterName identify the target SveltosCluster.
func StageResourcesForDeployment(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	resources map[string][]unstructured.Unstructured, skipTracking bool, logger logr.Logger) error {

	// Create all ConfigurationBundles. There one configurationBundle per key.
	// If Requestor is ClusterSummary each key represents a different ConfigMap/Secret referenced in
	// policyRef section or a different helm chart in the helmChart section.
	for k := range resources {
		_, err := reconcileConfigurationBundle(ctx, c, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, k, resources[k], skipTracking, true, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

// DiscardStagedResourcesForDeployment removes all resources that have been temporarily stored
// via calls to StageResourcesForDeployment for a specific component and target cluster.
// This method is intended to be used when the process of preparing resources fails,
// effectively canceling the staging operation and preventing these resources from being
// committed for deployment.
// - requestorKind, requestorName, and requestorFeature uniquely identify the component in the
// management cluster invoking this method.
// - clusterNamespace/clusterName identify the target SveltosCluster.
func DiscardStagedResourcesForDeployment(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) error {

	// Get current referenced configurationBundles. All ConfigurationBundles currently not referenced
	// by ConfigurationGroup will be discarded
	currentBundles, err := getReferencedConfigurationBundles(ctx, c, clusterNamespace, clusterName,
		requestorKind, requestorName, requestorFeature, logger)
	if err != nil {
		return err
	}

	return deleteStaleConfigurationBundles(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, currentBundles, logger)
}

// CommitStagedResourcesForDeployment marks the resources that have been previously staged
// using StageResourcesForDeployment as ready for deployment to the specified SveltosCluster.
// This method is called by a management cluster component after it has successfully prepared
// all the necessary resources across potentially multiple calls to StageResourcesForDeployment.
// Once committed, the agent in the managed cluster will be able to fetch these resources
// for deployment.
// - requestorKind, requestorName, and requestorFeature uniquely identify the component in the
// management cluster invoking this method.
// - clusterNamespace/clusterName identify the target SveltosCluster.
func CommitStagedResourcesForDeployment(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger, setters ...Option) error {

	bundles, err := getStagedConfigurationBundles(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, logger)
	if err != nil {
		return err
	}

	// Now that we have created all ConfigurationBundles, creates a single ConfigurationGroup
	// that references all bundles
	err = reconcileConfigurationGroup(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, bundles, logger, setters...)
	if err != nil {
		return err
	}

	// If ConfigurationGroup is updated, we might have stale configurationBundles. Contininuing
	// on the ClusterSummary example, previously ClusterSummary was referencing ConfigMap1 now it
	// references ConfigMap2. So find and delete all stale ConfigurationBundles.
	return deleteStaleConfigurationBundles(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, bundles, logger)
}

// RemoveDeployedResources marks resources for deletion in the managed cluster.
// For SveltosClusters in pull mode, this method is called by management cluster components
// to indicate that previously deployed resources should be removed. The agent in the managed
// cluster will detect this change and remove the corresponding resources.
//
//   - requestorKind, requestorName, and requestorFeature uniquely identify the component in the
//     management cluster invoking this method.
//   - clusterNamespace/clusterName identify the target SveltosCluster.
//
// This method marks the ConfigurationGroup Action field to ActionRemove and removes all referenced
// ConfigurationBundles, effectively signaling to the agent that all associated resources
// should be cleaned up from the managed cluster.
func RemoveDeployedResources(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger, setters ...Option) error {

	// Mark ConfigurationGroup for removal
	err := markConfigurationGroupForRemoval(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, logger, setters...)
	if err != nil {
		return err
	}

	// All bundles are now stale and can be deleted
	return deleteStaleConfigurationBundles(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, nil, logger)
}

// TerminateDeploymentTracking permanently removes the internal records associated with a deployment
// on a managed cluster. This action signifies the end of the tracking and management
// of resources for a specific deployment, typically after resources have been successfully
// deployed or removed from the managed cluster or if the deployment attempt is being abandoned.
//
// For SveltosClusters in pull mode, this method is called by management cluster components
// to finalize the lifecycle of a deployment and clean up the corresponding internal
// ConfigurationGroup resource.
//
//   - requestorKind, requestorName, and requestorFeature uniquely identify the component in the
//     management cluster invoking this method.
//   - clusterNamespace/clusterName identify the target SveltosCluster for which deployment
//     tracking is being terminated.
//
// This method locates and deletes the ConfigurationGroup resource associated with the
// specified cluster and requestor.
//
// It returns an error if it fails to locate or delete the ConfigurationGroup.
func TerminateDeploymentTracking(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) error {

	// Order is important. First delete all ConfigurationBundles, then delete the ConfigurationGroup.

	// All bundles are now stale and can be deleted
	err := deleteStaleConfigurationBundles(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, nil, logger)
	if err != nil {
		return err
	}

	return deleteConfigurationGroup(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, logger)
}

// For SveltosClusters operating in pull mode, this method is invoked by components
// on the management cluster to retrieve the deployment status of managed resources.
func GetResourceDeploymentStatus(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) (*libsveltosv1beta1.ConfigurationGroupStatus, error) {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, _, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return nil, err
	}

	currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
	err = c.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name},
		currentConfigurationGroup)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup: %v", err))
		return nil, err
	}

	return &currentConfigurationGroup.Status, nil
}

// For SveltosClusters operating in pull mode, this method is invoked by components
// on the management cluster to retrieve the withdrawal status of managed resources.
func GetResourceRemoveStatus(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) (*libsveltosv1beta1.ConfigurationGroupStatus, error) {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, _, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return nil, err
	}

	currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
	err = c.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name},
		currentConfigurationGroup)
	if err != nil {
		if apierrors.IsNotFound(err) {
			status := libsveltosv1beta1.FeatureStatusRemoved
			return &libsveltosv1beta1.ConfigurationGroupStatus{
				DeploymentStatus: &status,
			}, nil
		}
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup: %v", err))
		return nil, err
	}

	return &currentConfigurationGroup.Status, nil
}
