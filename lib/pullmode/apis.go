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
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

// ProcessingMismatchError represents an error when the resource has not been fully processed,
// either due to generation mismatch or requestor hash mismatch, indicating reconciliation is needed.
type ProcessingMismatchError struct {
	Message string
}

func (e *ProcessingMismatchError) Error() string {
	return e.Message
}

// NewProcessingMismatchError creates a new ProcessingMismatchError
func NewProcessingMismatchError(msg string) *ProcessingMismatchError {
	return &ProcessingMismatchError{
		Message: msg,
	}
}

// IsProcessingMismatch checks if an error is a ProcessingMismatchError
func IsProcessingMismatch(err error) bool {
	var genErr *ProcessingMismatchError
	return errors.As(err, &genErr)
}

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
		bundle, err := reconcileConfigurationBundle(ctx, c, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, k, resources[k], false, false, logger)
		if err != nil {
			return err
		}
		hash, err := getHash(resources[k])
		if err != nil {
			logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to evaluate hash: %v", err))
		}

		bundles[i] = bundleData{Name: bundle.Name, Hash: hash}
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
	resources map[string][]unstructured.Unstructured, skipTracking bool, logger logr.Logger,
	setters ...BundleOption) error {

	err := markConfigurationGroupAsPreparing(ctx, c, clusterNamespace, clusterName, requestorKind,
		requestorName, requestorFeature, logger)
	if err != nil {
		return err
	}

	manager := getStagedResourcesManager()

	logger.V(logs.LogDebug).Info(fmt.Sprintf("staging %d resources for deployment",
		len(resources)))

	// Create all ConfigurationBundles. There one configurationBundle per key.
	// If Requestor is ClusterSummary each key represents a different ConfigMap/Secret referenced in
	// policyRef section or a different helm chart in the helmChart section.
	for k := range resources {
		bundle, err := reconcileConfigurationBundle(ctx, c, clusterNamespace, clusterName, requestorKind,
			requestorName, requestorFeature, k, resources[k], skipTracking, true, logger, setters...)
		if err != nil {
			return err
		}

		manager.storeBundle(clusterNamespace, clusterName, requestorName, requestorFeature, bundle)
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

	logger.V(logs.LogDebug).Info("discarding staged resources")

	// Get current referenced configurationBundles. All ConfigurationBundles currently not referenced
	// by ConfigurationGroup will be discarded
	currentBundles, err := getReferencedConfigurationBundles(ctx, c, clusterNamespace, clusterName,
		requestorKind, requestorName, requestorFeature, logger)
	if err != nil {
		return err
	}

	manager := getStagedResourcesManager()
	manager.clearBundles(clusterNamespace, clusterName, requestorName, requestorFeature)

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

	manager := getStagedResourcesManager()
	stagedBundles := manager.getBundles(clusterNamespace, clusterName, requestorName, requestorFeature)

	bundles := make([]bundleData, len(stagedBundles))
	for i := range stagedBundles {
		b := &stagedBundles[i]
		bundles[i] = bundleData{Name: b.Name, Hash: b.Status.Hash}
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

// GetSourceStatus retrieves the SourceStatus of the ConfigurationGroup associated with a specific requestor.
//
// This function identifies a ConfigurationGroup using the provided cluster details (namespace, name),
// requestor details (kind, name, feature), and then extracts its SourceStatus.
//
// Returns:
//
//	*libsveltosv1beta1.SourceStatus: A pointer to the SourceStatus of the found ConfigurationGroup.
//	                                 Returns nil if the ConfigurationGroup is not found or in case of an error.
//	error: An error if there was a problem retrieving the ConfigurationGroup or its status.
//	       Returns nil if the operation was successful and the status was retrieved (or if the CG was not found).
func GetSourceStatus(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) (*libsveltosv1beta1.SourceStatus, error) {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	_, currentConfigurationGroup, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return nil, err
	}

	if currentConfigurationGroup == nil {
		// The resource currently does not exist. Nothing to do
		return nil, nil
	}

	cg := currentConfigurationGroup.(*libsveltosv1beta1.ConfigurationGroup)
	return &cg.Spec.SourceStatus, nil
}

// For SveltosClusters operating in pull mode, this method is invoked by components
// on the management cluster to retrieve the deployment status of managed resources.
func GetDeploymentStatus(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) (*libsveltosv1beta1.ConfigurationGroupStatus, error) {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, _, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return nil, err
	}

	currentCG := &libsveltosv1beta1.ConfigurationGroup{}
	err = c.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name},
		currentCG)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup: %v", err))
		return nil, err
	}

	if currentCG.Spec.Action != libsveltosv1beta1.ActionDeploy {
		return &currentCG.Status, fmt.Errorf("ConfigurationGroup action not set to deploy")
	}

	if currentCG.Spec.UpdatePhase != libsveltosv1beta1.UpdatePhaseReady {
		return &currentCG.Status, fmt.Errorf("ConfigurationGroup updatePhase not set to ready")
	}

	if currentCG.Status.ObservedGeneration != 0 {
		if currentCG.Status.ObservedGeneration != currentCG.Generation {
			msg := fmt.Sprintf("ConfigurationGroup Status.ObservedGeneration (%d) does not match Generation (%d)",
				currentCG.Status.ObservedGeneration, currentCG.Generation)
			logger.V(logs.LogInfo).Info(msg)
			return &currentCG.Status, NewProcessingMismatchError(msg)
		}
	}

	if !reflect.DeepEqual(currentCG.Status.ObservedRequestorHash, currentCG.Spec.RequestorHash) {
		msg := "ConfigurationGroup Status.ObservedRequestorHash does not match Spec.RequestorHash"
		logger.V(logs.LogInfo).Info(msg)
		return &currentCG.Status, NewProcessingMismatchError(msg)
	}

	return &currentCG.Status, nil
}

// For SveltosClusters operating in pull mode, this method is invoked by components
// on the management cluster to retrieve the withdrawal status of managed resources.
func GetRemoveStatus(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) (*libsveltosv1beta1.ConfigurationGroupStatus, error) {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, _, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return nil, err
	}

	currentCG := &libsveltosv1beta1.ConfigurationGroup{}
	err = c.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name},
		currentCG)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup: %v", err))
		return nil, err
	}

	if currentCG.Spec.Action != libsveltosv1beta1.ActionRemove {
		return &currentCG.Status, fmt.Errorf("ConfigurationGroup action not set to remove")
	}

	return &currentCG.Status, nil
}

// IsBeingProvisioned returns true if content is currently being or has been deployed.
// A ConfigurationGroup is considered "being provisioned" when it's ready for processing
// and the requestor hashes match, regardless of whether deployment is complete.
func IsBeingProvisioned(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) bool {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, _, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return false
	}

	currentCG := &libsveltosv1beta1.ConfigurationGroup{}
	err = c.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name},
		currentCG)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup: %v", err))
		return false
	}

	if currentCG.Spec.UpdatePhase != libsveltosv1beta1.UpdatePhaseReady {
		return false
	}

	if currentCG.Spec.Action != libsveltosv1beta1.ActionDeploy {
		return false
	}

	// Verify this is for the current requestor state
	if currentCG.Status.ObservedRequestorHash != nil &&
		!reflect.DeepEqual(currentCG.Status.ObservedRequestorHash, currentCG.Spec.RequestorHash) {

		logger.V(logs.LogInfo).Info("requestor hash mismatch - latest changes not yet processed")
		return false
	}

	if currentCG.Status.DeploymentStatus != nil &&
		*currentCG.Status.DeploymentStatus == libsveltosv1beta1.FeatureStatusFailed {

		return false
	}

	return true
}

// IsBeingRemoved returns true if the ConfigurationGroup is currently being removed
// or has already been removed. This includes cases where the resource no longer exists
// or when it's ready for removal processing.
func IsBeingRemoved(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) bool {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, _, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return false
	}

	currentCG := &libsveltosv1beta1.ConfigurationGroup{}
	err = c.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name}, currentCG)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// ConfigurationGroup has been removed
			return true
		}
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup: %v", err))
		return false
	}

	if currentCG.Spec.UpdatePhase != libsveltosv1beta1.UpdatePhaseReady {
		return false
	}

	if currentCG.Spec.Action != libsveltosv1beta1.ActionRemove {
		return false
	}

	return true
}

// GetRequestorHash returns the hash of the requestor at last time configuration for sveltos-applier
// was created
func GetRequestorHash(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) ([]byte, error) {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, _, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return nil, err
	}

	currentCG := &libsveltosv1beta1.ConfigurationGroup{}
	err = c.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name},
		currentCG)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup: %v", err))
		return nil, err
	}

	return currentCG.Spec.RequestorHash, nil
}
