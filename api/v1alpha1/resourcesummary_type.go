/*
Copyright 2022. projectsveltos.io. All rights reserved.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ResourceSummaryFinalizer is finalizer added to ResourceSummary
	ResourceSummaryFinalizer = "resourcesummaryfinalizer.projectsveltos.io"

	ResourceSummaryKind = "ResourceSummary"

	// ClusterSummaryLabelName is added to all ResourceSummary instances
	ClusterSummaryLabelName = "projectsveltos.io/cluster-summary-name"

	// ClusterSummaryLabelNamespace is added to all ResourceSummary instances
	ClusterSummaryLabelNamespace = "projectsveltos.io/cluster-summary-namespace"
)

type Resource struct {
	// Name of the resource deployed in the Cluster.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace of the resource deployed in the Cluster.
	// Empty for resources scoped at cluster level.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Group of the resource deployed in the Cluster.
	Group string `json:"group"`

	// Kind of the resource deployed in the Cluster.
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// Version of the resource deployed in the Cluster.
	Version string `json:"version"`
}

type HelmResources struct {
	// ChartName is the chart name
	// +kubebuilder:validation:MinLength=1
	ChartName string `json:"chartName"`

	// ReleaseName is the chart release
	// +kubebuilder:validation:MinLength=1
	ReleaseName string `json:"releaseName"`

	// ReleaseNamespace is the namespace release will be installed
	// +kubebuilder:validation:MinLength=1
	ReleaseNamespace string `json:"releaseNamespace"`

	// Resources deployed by ClusterSummary because of helm charts
	// +optional
	Resources []Resource `json:"group,omitempty"`
}

type ResourceHash struct {
	// Resource specifies a resource.
	Resource `json:",inline"`

	// Hash is the hash of a resource's data.
	Hash string `json:"hash,omitempty"`
}

// ResourceSummarySpec defines the desired state of ResourceSummary
type ResourceSummarySpec struct {
	// Resources deployed by ClusterSummary because of referenced ConfigMaps/Secrets
	// +optional
	Resources []Resource `json:"resources,omitempty"`

	// Resources deployed by ClusterSummary because of referenced Helm charts
	// +optional
	ChartResources []HelmResources `json:"chartResources,omitempty"`
}

// ResourceSummaryStatus defines the status of ResourceSummary
type ResourceSummaryStatus struct {
	// Resources changed.
	// +optional
	ResourcesChanged bool `json:"resourcesChanged,omitempty"`

	// Helm Resources changed.
	// +optional
	HelmResourcesChanged bool `json:"helmResourcesChanged,omitempty"`

	// ResourceHashes specifies a list of resource plus hash
	ResourceHashes []ResourceHash `json:"resourceHashes,omitempty"`

	// HelmResourceHashes specifies list of resource plus hash.
	HelmResourceHashes []ResourceHash `json:"helmResourceHashes,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=resourcesummaries,scope=Namespaced
//+kubebuilder:subresource:status

// ResourceSummary is the Schema for the ResourceSummary API
type ResourceSummary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceSummarySpec   `json:"spec,omitempty"`
	Status ResourceSummaryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ResourceSummaryList contains a list of ResourceSummary
type ResourceSummaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceSummary `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceSummary{}, &ResourceSummaryList{})
}
