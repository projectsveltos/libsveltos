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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/logsettings"
)

const (
	clusterNameLabelKey      = "pullmode.projectsveltos.io/clustername"
	requestorKindLabelKey    = "pullmode.projectsveltos.io/requestorkind"
	requestorFeatureLabelKey = "pullmode.projectsveltos.io/requestorfeature"

	requestorNameAnnotationKey = "pullmode.projectsveltos.io/requestorname"

	indexAnnotationKey = "pullmode.projectsveltos.io/index"
	stagedLabelKey     = "pullmode.projectsveltos.io/staged"
	stagedLabelValue   = "staged"
)

type bundleData struct {
	Name string
	Hash []byte
}

// In Sveltos pull mode, management cluster components define configurations to deploy in managed clusters.
// Agents running in these managed clusters fetch and apply these configurations.
//
// Agents identify relevant ConfigurationGroups using labels (and filtering by namespace):
// - clusterNameLabelKey
//
// Multiple ConfigurationGroups can exist for a given managed cluster and a Sveltos management component.
// To identify the proper ConfigurationGroups following labels/annotations are used (and filtering by namespace):
// - clusterNameLabelKey
// - requestorKindLabelKey (e.g., clustersummary)
// - requestorFeatureLabelKey (e.g., helm, kustomize, policyrefs)
// - requestorNameAnnotationKey (e.g., the ClusterSummary's name) (this is an annotation to avoid limit of 63
// character value a label value can have)
//
// Continuing on the clusterSummary example, a given ClusterSummary might contain multiple
// helm charts. So a ConfigurationBundle is created for each helm chart. To identify the
// proper ConfigurationBundle following labels/annotations are used (and filtering by namespace):
// - clusterNameLabelKey
// - requestorKindLabelKey (e.g., clustersummary)
// - requestorFeatureLabelKey (e.g., helm, kustomize, policyrefs)
// - indexLabelKey (e.g., helm chart 1 .. potentially even more ConfigurationBundle for a given helm chart
// if number of resources is high)
// - requestorNameAnnotationKey (e.g., the ClusterSummary's name) (this is an annotation to avoid limit of 63
// character value a label value can have)
//
// ConfigurationGroup has a finalizer allowing the agent to see and process deleted ConfigurationGroup.
// There is no finalizer on the ConfigurationBundle. Agent needs to find all stale resources.

func reconcileConfigurationBundle(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature, index string,
	resources []unstructured.Unstructured, skipTracking, isStaged bool, logger logr.Logger,
) (*libsveltosv1beta1.ConfigurationBundle, error) {

	labels := getConfigurationBundleLabels(clusterName, requestorKind, requestorFeature)

	name, currentBundle, err := getConfigurationBundleName(ctx, c, clusterNamespace, requestorName, index, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationBundle name: %v", err))
		return nil, err
	}

	// If ConfigurationBundle is still present but marked for deletion, return an error
	// it means agent has not pulled it yet or finished cleaning
	if currentBundle != nil && !currentBundle.GetDeletionTimestamp().IsZero() {
		msgError := "ConfigurationBundle is currently existing but marked for deletion"
		logger.V(logsettings.LogInfo).Info(msgError)
		return nil, errors.New(msgError)
	}

	if currentBundle == nil {
		return createConfigurationBundle(ctx, c, clusterNamespace, name, requestorName, index,
			resources, labels, skipTracking, isStaged, logger)
	}

	return updateConfigurationBundle(ctx, c, clusterNamespace, name, requestorName, index,
		resources, skipTracking, isStaged, logger)
}

func prepareConfigurationBundle(namespace, name string, resources []unstructured.Unstructured,
) (*libsveltosv1beta1.ConfigurationBundle, error) {

	confBundle := &libsveltosv1beta1.ConfigurationBundle{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	content := make([]string, len(resources))

	for i := range resources {
		// Reset managed fields
		resources[i].SetManagedFields(nil)
		// Reset resourceVersion
		resources[i].SetResourceVersion("")
		resources[i].SetUID("")
		data, err := yaml.Marshal(resources[i].UnstructuredContent())
		if err != nil {
			return nil, err
		}

		content[i] = string(data)
	}

	confBundle.Spec.Resources = content

	return confBundle, nil
}

func createConfigurationBundle(ctx context.Context, c client.Client, namespace, name, requestorName, index string,
	resources []unstructured.Unstructured, labels client.MatchingLabels, skipTracking, isStaged bool,
	logger logr.Logger) (*libsveltosv1beta1.ConfigurationBundle, error) {

	bundle, err := prepareConfigurationBundle(namespace, name, resources)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to prepare configurationBundle: %v", err))
		return nil, err
	}

	if labels == nil {
		labels = map[string]string{}
	}
	if isStaged {
		labels[stagedLabelKey] = stagedLabelValue
	}

	bundle.Annotations = map[string]string{
		requestorNameAnnotationKey: requestorName,
		indexAnnotationKey:         index,
	}
	bundle.Labels = labels
	bundle.Spec.NotTracked = skipTracking

	err = c.Create(ctx, bundle)
	if err != nil {
		return nil, err
	}

	// For staged ConfigurationBundle also stores current hash
	hash, err := getHash(resources)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to evaluate hash: %v", err))
	}
	bundle.Status.Hash = hash
	err = c.Status().Update(ctx, bundle)
	return bundle, err
}

func updateConfigurationBundle(ctx context.Context, c client.Client, namespace, name, requestorName, index string,
	resources []unstructured.Unstructured, skipTracking, isStaged bool, logger logr.Logger,
) (*libsveltosv1beta1.ConfigurationBundle, error) {

	bundle, err := prepareConfigurationBundle(namespace, name, resources)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to prepare configurationBundle: %v", err))
		return nil, err
	}

	currentBundle := &libsveltosv1beta1.ConfigurationBundle{}
	err = c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, currentBundle)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get configurationBundle: %v", err))
		return nil, err
	}

	if currentBundle.Labels == nil {
		currentBundle.Labels = map[string]string{}
	}
	if isStaged {
		currentBundle.Labels[stagedLabelKey] = stagedLabelValue
	}

	currentBundle.Annotations = map[string]string{
		requestorNameAnnotationKey: requestorName,
		indexAnnotationKey:         index,
	}

	currentBundle.Spec = bundle.Spec
	currentBundle.Spec.NotTracked = skipTracking
	err = c.Update(ctx, currentBundle)
	if err != nil {
		return nil, err
	}

	// For staged ConfigurationBundle also stores current hash
	hash, err := getHash(resources)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to evaluate hash: %v", err))
	}
	currentBundle.Status.Hash = hash
	err = c.Status().Update(ctx, currentBundle)
	return currentBundle, err
}

func getConfigurationBundles(ctx context.Context, c client.Client, namespace, requestorName, index string,
	labels client.MatchingLabels) (*libsveltosv1beta1.ConfigurationBundleList, error) {

	listOptions := []client.ListOption{
		client.InNamespace(namespace),
		labels,
	}

	configurationBundles := &libsveltosv1beta1.ConfigurationBundleList{}
	if err := c.List(ctx, configurationBundles, listOptions...); err != nil {
		return nil, err
	}

	// Filter configurationBundles by annotation
	filtered := []libsveltosv1beta1.ConfigurationBundle{}
	for i := range configurationBundles.Items {
		bundle := &configurationBundles.Items[i]
		if bundle.Annotations != nil {
			annotationValue, exists := bundle.Annotations[requestorNameAnnotationKey]
			if !exists || annotationValue != requestorName {
				continue
			}
			if index != "" {
				value, exists := bundle.Annotations[indexAnnotationKey]
				if !exists || value != index {
					continue
				}
			}
			filtered = append(filtered, *bundle)
		}
	}

	configurationBundles.Items = filtered

	return configurationBundles, nil
}

// getConfigurationBundleName returns ConfigurationBundle name.
func getConfigurationBundleName(ctx context.Context, c client.Client, namespace, requestorName, index string,
	labels client.MatchingLabels) (name string, currentCG client.Object, err error) {

	configurationBundles, err := getConfigurationBundles(ctx, c, namespace, requestorName, index, labels)
	if err != nil {
		return "", nil, err
	}

	if len(configurationBundles.Items) > 1 {
		// this should never happen. If it ever happens recover by deleting all bundles
		for i := range configurationBundles.Items {
			// Ignore eventual error, since we are returning an error anyway
			_ = c.Delete(ctx, &configurationBundles.Items[i])
		}
		err := fmt.Errorf("more than one configurationBundle found")
		return "", nil, err
	}

	objects := make([]client.Object, len(configurationBundles.Items))
	for i := range configurationBundles.Items {
		objects[i] = &configurationBundles.Items[i]
	}

	return getInstantiatedObjectName(objects)
}

func markConfigurationGroupAsPreparing(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) error {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	_, currentCG, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return err
	}

	if currentCG != nil {
		cg := currentCG.(*libsveltosv1beta1.ConfigurationGroup)
		cg.Spec.UpdatePhase = libsveltosv1beta1.UpdatePhasePreparing
		return c.Update(ctx, cg)
	}

	return nil
}

func reconcileConfigurationGroup(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	bundles []bundleData, logger logr.Logger, setters ...Option) error {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, currentCG, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return err
	}

	// If ConfigurationGroup is still present but marked for deletion, return an error
	// it means agent has not pulled it yet or finished cleaning
	if currentCG != nil && !currentCG.GetDeletionTimestamp().IsZero() {
		msgError := "ConfigurationGroup is currently existing but marked for deletion"
		logger.V(logsettings.LogInfo).Info(msgError)
		return errors.New(msgError)
	}

	l := logger.WithValues("configurationgropup", fmt.Sprintf("%s/%s", clusterNamespace, name))
	action := libsveltosv1beta1.ActionDeploy
	if currentCG == nil {
		l.V(logsettings.LogDebug).Info(fmt.Sprintf("creating configurationGroup for requestor %s (bundles %d)",
			requestorName, len(bundles)))
		return createConfigurationGroup(ctx, c, clusterNamespace, name, requestorName,
			bundles, labels, action, setters...)
	}

	l.V(logsettings.LogDebug).Info(fmt.Sprintf("updating configurationGroup for requestor %s (bundles %d)",
		requestorName, len(bundles)))
	return updateConfigurationGroup(ctx, c, clusterNamespace, name, bundles, action, logger, setters...)
}

func markConfigurationGroupForRemoval(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger, setters ...Option) error {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	name, currentCG, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return err
	}

	action := libsveltosv1beta1.ActionRemove
	if currentCG == nil {
		return createConfigurationGroup(ctx, c, clusterNamespace, name, requestorName,
			nil, labels, action, setters...)
	}

	return updateConfigurationGroup(ctx, c, clusterNamespace, name, nil, action, logger, setters...)
}

func deleteConfigurationGroup(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) error {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	_, currentConfigurationGroup, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return err
	}

	if currentConfigurationGroup == nil {
		// The resource currently does not exist. Nothing to do
		return nil
	}

	l := logger.WithValues("configurationgropup",
		fmt.Sprintf("%s/%s", currentConfigurationGroup.GetNamespace(), currentConfigurationGroup.GetName()))
	l.V(logsettings.LogDebug).Info(fmt.Sprintf("deleting configurationGroup for requestor %s",
		requestorName))
	return c.Delete(ctx, currentConfigurationGroup)
}

func prepareConfigurationGroup(namespace, name string, bundles []bundleData,
	action libsveltosv1beta1.Action, setters ...Option) *libsveltosv1beta1.ConfigurationGroup {

	confGroup := &libsveltosv1beta1.ConfigurationGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	confGroup.Spec = libsveltosv1beta1.ConfigurationGroupSpec{}
	confGroup.Spec.UpdatePhase = libsveltosv1beta1.UpdatePhaseReady
	confGroup.Spec.ConfigurationItems = make([]libsveltosv1beta1.ConfigurationItem, len(bundles))
	for i := range bundles {
		confGroup.Spec.ConfigurationItems[i] = libsveltosv1beta1.ConfigurationItem{
			ContentRef: &corev1.ObjectReference{
				APIVersion: libsveltosv1beta1.GroupVersion.String(),
				Kind:       libsveltosv1beta1.ConfigurationBundleKind,
				Name:       bundles[i].Name,
				Namespace:  namespace,
			},
			Hash: bundles[i].Hash,
		}
	}

	confGroup.Spec.Action = action

	confGroup = applySetters(confGroup, setters...)
	return confGroup
}

func createConfigurationGroup(ctx context.Context, c client.Client, namespace, name, requestorName string,
	bundles []bundleData, labels client.MatchingLabels, action libsveltosv1beta1.Action, setters ...Option) error {

	group := prepareConfigurationGroup(namespace, name, bundles, action, setters...)

	group.Labels = labels
	group.Annotations = map[string]string{
		requestorNameAnnotationKey: requestorName,
	}

	return c.Create(ctx, group)
}

func updateConfigurationGroup(ctx context.Context, c client.Client, namespace, name string,
	bundles []bundleData, action libsveltosv1beta1.Action, logger logr.Logger, setters ...Option) error {

	group := prepareConfigurationGroup(namespace, name, bundles, action, setters...)

	currentGroup := &libsveltosv1beta1.ConfigurationGroup{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, currentGroup)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get configurationGroup: %v", err))
		return err
	}

	currentGroup.Spec = group.Spec
	return c.Update(ctx, currentGroup)
}

func deleteStaleConfigurationBundles(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	referencedBundles []bundleData, logger logr.Logger) error {

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

	currentBundles, err := getConfigurationBundles(ctx, c, clusterNamespace, requestorName, "", labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to list configurationBundles: %v", err))
		return err
	}

	bundleMap := make(map[string][]byte, 0)
	for i := range referencedBundles {
		bundleMap[referencedBundles[i].Name] = referencedBundles[i].Hash
	}

	for i := range currentBundles.Items {
		currentBundle := &currentBundles.Items[i]
		if _, ok := bundleMap[currentBundle.Name]; !ok {
			if err := c.Delete(ctx, currentBundle); err != nil {
				logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to delete stale configurationBundle: %v", err))
				return err
			}
		}
	}

	return nil
}

// getReferencedConfigurationBundles returns all ConfigurationBundles currently referenced
// by a ConfigurationGroup.
// If the ConfigurationGroup does not exist, empty bundle list is returned
func getReferencedConfigurationBundles(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, requestorKind, requestorName, requestorFeature string,
	logger logr.Logger) ([]bundleData, error) {

	var currentBundles []bundleData

	labels := getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
	_, currentConfigurationGroup, err := getConfigurationGroupName(ctx, c, clusterNamespace, requestorName, labels)
	if err != nil {
		logger.V(logsettings.LogInfo).Info(fmt.Sprintf("failed to get ConfigurationGroup name: %v", err))
		return currentBundles, err
	}

	if currentConfigurationGroup != nil {
		currentCG := currentConfigurationGroup.(*libsveltosv1beta1.ConfigurationGroup)
		currentBundles = make([]bundleData, len(currentCG.Spec.ConfigurationItems))

		for i := range currentCG.Spec.ConfigurationItems {
			currentBundles[i] = bundleData{Name: currentCG.Spec.ConfigurationItems[i].ContentRef.Name}
		}
	}

	return currentBundles, nil
}

func getConfigurationGroups(ctx context.Context, c client.Client, namespace, requestorName string,
	labels client.MatchingLabels) (*libsveltosv1beta1.ConfigurationGroupList, error) {

	listOptions := []client.ListOption{
		client.InNamespace(namespace),
		labels,
	}

	configurationGroups := &libsveltosv1beta1.ConfigurationGroupList{}
	if err := c.List(ctx, configurationGroups, listOptions...); err != nil {
		return nil, err
	}

	// Filter configurationGroups by annotation
	filtered := []libsveltosv1beta1.ConfigurationGroup{}
	for i := range configurationGroups.Items {
		group := &configurationGroups.Items[i]
		if group.Annotations != nil {
			annotationValue, exists := group.Annotations[requestorNameAnnotationKey]
			if exists && annotationValue == requestorName {
				filtered = append(filtered, *group)
			}
		}
	}

	configurationGroups.Items = filtered

	return configurationGroups, nil
}

// getConfigurationGroupName returns ConfigurationGroup name.
// A ConfigurationGroup is created for each requestor (identified by kind, name and feature) and cluster (identified
// by cluster namespace and name) pair. If a ConfigurationGroup already exists, the name is returned. If not a random
// name is generated
func getConfigurationGroupName(ctx context.Context, c client.Client, namespace, requestorName string,
	labels client.MatchingLabels) (name string, currentCG client.Object, err error) {

	configurationGroups, err := getConfigurationGroups(ctx, c, namespace, requestorName, labels)
	if err != nil {
		return "", nil, err
	}

	if len(configurationGroups.Items) > 1 {
		// this should never happen. If it ever happens recover by deleting all groups
		for i := range configurationGroups.Items {
			// Ignore eventual error, since we are returning an error anyway
			_ = c.Delete(ctx, &configurationGroups.Items[i])
		}
		err := fmt.Errorf("more than one configurationGroup found")
		return "", nil, err
	}

	objects := make([]client.Object, len(configurationGroups.Items))
	for i := range configurationGroups.Items {
		objects[i] = &configurationGroups.Items[i]
	}

	return getInstantiatedObjectName(objects)
}

func getInstantiatedObjectName(objects []client.Object) (name string, currentObject client.Object, err error) {
	switch len(objects) {
	case 0:
		// no configurationBundle exist yet. Return random name.
		prefix := "config-"
		const nameLength = 20
		name = prefix + util.RandomString(nameLength)
		currentObject = nil
		err = nil
	case 1:
		name = objects[0].GetName()
		currentObject = objects[0]
		err = nil
	default:
		err = fmt.Errorf("more than one configurationBundle found")
	}
	return name, currentObject, err
}

func getConfigurationBundleLabels(clusterName, requestorKind, requestorFeature string) client.MatchingLabels {
	return client.MatchingLabels{
		clusterNameLabelKey:      clusterName,
		requestorKindLabelKey:    requestorKind,
		requestorFeatureLabelKey: requestorFeature,
	}
}

func getConfigurationGroupLabels(clusterName, requestorKind, requestorFeature string) client.MatchingLabels {
	return client.MatchingLabels{
		clusterNameLabelKey:      clusterName,
		requestorKindLabelKey:    requestorKind,
		requestorFeatureLabelKey: requestorFeature,
	}
}

func getHash(resources []unstructured.Unstructured) ([]byte, error) {
	// Create hasher
	hasher := sha256.New()

	for i, item := range resources {
		// Marshal to JSON
		data, err := json.Marshal(item.Object)
		if err != nil {
			return nil, fmt.Errorf("error marshaling item %d: %w", i, err)
		}

		// Write data to hasher
		hasher.Write(data)
	}

	// Return hex-encoded hash
	return hasher.Sum(nil), nil
}

func removeStagedLabel(ctx context.Context, c client.Client,
	configurationBundle *libsveltosv1beta1.ConfigurationBundle) error {

	// ConfigurationBundles are created and then status is updated with hash.
	// If committing staged configurationBundles happen fast, this might fail
	// the cached version used to remove the label might not be the latest one (after status update)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		currentConfigurationBundle := &libsveltosv1beta1.ConfigurationBundle{}
		configurationBundleName := types.NamespacedName{
			Namespace: configurationBundle.Namespace,
			Name:      configurationBundle.Name,
		}

		err := c.Get(ctx, configurationBundleName, currentConfigurationBundle)
		if err != nil {
			return err
		}

		labels := currentConfigurationBundle.Labels
		delete(labels, stagedLabelKey)
		currentConfigurationBundle.Labels = labels

		return c.Update(ctx, currentConfigurationBundle)
	})
	return err
}
