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

package deployer

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/logsettings"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
	"github.com/projectsveltos/libsveltos/lib/sveltos_upgrade"
)

const (
	// When this annotation is set, resource will be excluded from configuration
	// drift detection
	driftDetectionIgnoreAnnotation = "projectsveltos.io/driftDetectionIgnore"
)

// HasIgnoreConfigurationDriftAnnotation verifies whether resource has
// `projectsveltos.io/driftDetectionIgnore` annotation. Any resource with such
// annotation set won't be tracked for configuration drift.
func HasIgnoreConfigurationDriftAnnotation(resource *unstructured.Unstructured) bool {
	annotations := resource.GetAnnotations()
	if annotations != nil {
		if _, ok := annotations[driftDetectionIgnoreAnnotation]; ok {
			return true
		}
	}

	return false
}

func DeployResourceSummaryInCluster(ctx context.Context, c client.Client, resourceSummaryName types.NamespacedName,
	clusterNamespace, clusterName, requestor string, clusterType libsveltosv1beta1.ClusterType,
	resources []libsveltosv1beta1.Resource, kustomizeResources []libsveltosv1beta1.Resource,
	helmResources []libsveltosv1beta1.HelmResources, driftExclusions []libsveltosv1beta1.DriftExclusion,
	logger logr.Logger) error {

	logger = logger.WithValues("requestor", requestor)
	logger.V(logs.LogDebug).Info("deploy resourcesummary")

	lbls := map[string]string{
		sveltos_upgrade.ClusterNameLabel: clusterName,
		sveltos_upgrade.ClusterTypeLabel: strings.ToLower(string(clusterType)),
	}

	annotations := map[string]string{
		libsveltosv1beta1.ClusterSummaryNameAnnotation:      requestor,
		libsveltosv1beta1.ClusterSummaryNamespaceAnnotation: clusterNamespace,
	}

	// Deploy ResourceSummary instance
	err := deployResourceSummaryInstance(ctx, c, resources, kustomizeResources, helmResources,
		resourceSummaryName.Namespace, resourceSummaryName.Name, lbls, annotations, driftExclusions, logger)
	if err != nil {
		return err
	}

	logger.V(logs.LogDebug).Info("successuflly deployed resourceSummary CRD and instance")
	return nil
}

func deployResourceSummaryInstance(ctx context.Context, clusterClient client.Client,
	resources []libsveltosv1beta1.Resource, kustomizeResources []libsveltosv1beta1.Resource,
	helmResources []libsveltosv1beta1.HelmResources, namespace, name string,
	lbls, annotations map[string]string, driftExclusions []libsveltosv1beta1.DriftExclusion, logger logr.Logger,
) error {

	logger.V(logs.LogDebug).Info("deploy resourceSummary instance")

	patches := TransformDriftExclusionsToPatches(driftExclusions)

	currentResourceSummary := &libsveltosv1beta1.ResourceSummary{}
	err := clusterClient.Get(ctx,
		types.NamespacedName{Namespace: namespace, Name: name},
		currentResourceSummary)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(logsettings.LogDebug).Info("resourceSummary instance not present. creating it.")
			toDeployResourceSummary := &libsveltosv1beta1.ResourceSummary{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
					Labels:      lbls,
					Annotations: annotations,
				},
			}
			if resources != nil {
				toDeployResourceSummary.Spec.Resources = resources
			}
			if kustomizeResources != nil {
				toDeployResourceSummary.Spec.KustomizeResources = kustomizeResources
			}
			if helmResources != nil {
				toDeployResourceSummary.Spec.ChartResources = helmResources
			}
			toDeployResourceSummary.Spec.Patches = patches

			return clusterClient.Create(ctx, toDeployResourceSummary)
		}
		return err
	}

	if resources != nil {
		currentResourceSummary.Spec.Resources = resources
	}
	if kustomizeResources != nil {
		currentResourceSummary.Spec.KustomizeResources = kustomizeResources
	}
	if helmResources != nil {
		currentResourceSummary.Spec.ChartResources = helmResources
	}
	if currentResourceSummary.Labels == nil {
		currentResourceSummary.Labels = map[string]string{}
	}
	currentResourceSummary.Spec.Patches = patches
	currentResourceSummary.Labels = lbls
	currentResourceSummary.Annotations = annotations

	logger.V(logsettings.LogDebug).Info("resourceSummary instance already present. updating it.")
	return clusterClient.Update(ctx, currentResourceSummary)
}
