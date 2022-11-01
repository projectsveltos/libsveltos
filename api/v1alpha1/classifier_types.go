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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClassifierFinalizer allows ClassifierReconciler to clean up resources associated with
	// Classifier before removing it from the apiserver.
	ClassifierFinalizer = "classifierfinalizer.projectsveltos.io"

	ClassifierKind = "Classifier"

	FeatureClassifier = "Classifier"
)

// +kubebuilder:validation:Enum:=Provisioning;Provisioned;Failed;Removing;Removed
type ClassifierFeatureStatus string

const (
	// ClassifierStatusProvisioning indicates that classifier is being
	// provisioned in the workload cluster
	ClassifierStatusProvisioning = ClassifierFeatureStatus("Provisioning")

	// ClassifierStatusProvisioned indicates that classifier has been
	// provisioned in the workload cluster
	ClassifierStatusProvisioned = ClassifierFeatureStatus("Provisioned")

	// ClassifierStatusFailed indicates that configuring classifier
	// in the workload cluster failed
	ClassifierStatusFailed = ClassifierFeatureStatus("Failed")

	// ClassifierStatusRemoving indicates that classifier is being
	// removed
	ClassifierStatusRemoving = ClassifierFeatureStatus("Removing")

	// ClassifierStatusRemoved indicates that classifier is removed
	ClassifierStatusRemoved = ClassifierFeatureStatus("Removed")
)

// Operation specifies
// +kubebuilder:validation:Enum:=Equal;Different
type Operation string

const (
	// OperationEqual will verify equality. Corresponds to ==
	OperationEqual = Operation("Equal")

	// OperationDifferent will verify difference. Corresponds to !=
	OperationDifferent = Operation("Different")
)

type ClassifierLabel struct {
	// Key is the label key
	Key string `json:"key"`

	// Value is the label value
	Value string `json:"value"`
}

type LabelFilter struct {
	// Key is the label key
	Key string `json:"key"`

	// Operation is the comparison operation
	Operation Operation `json:"operation"`

	// Value is the label value
	Value string `json:"value"`
}

type DeployedResource struct {
	// Namespace of the resource deployed in the CAPI Cluster.
	// Empty for resources scoped at cluster level.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Group of the resource deployed in the CAPI Cluster.
	Group string `json:"group"`

	// Version of the resource deployed in the CAPI Cluster.
	Version string `json:"version"`

	// Kind of the resource deployed in the CAPI Cluster.
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// LabelFilters allows to filter resources based on current labels.
	LabelFilters []LabelFilter `json:"labelFilters,omitempty"`
}

type KubernetesComparison string

// Define the Action constants.
const (
	ComparisonEqual                KubernetesComparison = "Equal"
	ComparisoNotEqual              KubernetesComparison = "NotEqual"
	ComparisonGreaterThan          KubernetesComparison = "GreaterThan"
	ComparisonLessThan             KubernetesComparison = "LessThan"
	ComparisonGreaterThanOrEqualTo KubernetesComparison = "GreaterThanOrEqualTo"
	ComparisonLessThanOrEqualTo    KubernetesComparison = "LessThanOrEqualTo"
)

type KubernetesVersion struct {
	// Version is the kubernetes version
	Version string `json:"version"`

	// Comparison indicate how to compare cluster kubernetes version with the specified version
	// +kubebuilder:validation:Enum=Equal;NotEqual;GreaterThan;LessThan;GreaterThanOrEqualTo;LessThanOrEqualTo
	Comparison string `json:"comparison"`
}

// ClassifierSpec defines the desired state of Classifier
type ClassifierSpec struct {
	// DeployedResources allows to classify based on current deployed resources
	DeployedResources []DeployedResource `json:"deployedResources,omitempty"`

	// KubernetesVersion allows to classify based on current kubernetes version
	KubernetesVersion KubernetesVersion `json:"kubernetesVersion,omitempty"`

	// ClassifierLabels is set of labels, key,value pair, that will be added to each
	// cluster matching Classifier instance
	ClassifierLabels []ClassifierLabel `json:"classifierLabels"`
}

type ClusterInfo struct {
	// Cluster references the CAPI Cluster
	Cluster corev1.ObjectReference `json:"cluster"`

	// Hash represents the hash of the Classifier currently deployed
	// in the CAPI Cluster
	Hash []byte `json:"hash"`

	// Status represents the state of the feature in the workload cluster
	Status ClassifierFeatureStatus `json:"status"`

	// FailureMessage provides more information about the error.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

type UnManagedLabel struct {
	// Key represents a label Classifier would like to manage
	// but cannot because currently managed by different instance
	Key string `json:"key"`

	// FailureMessage is a human consumable message explaining the
	// misconfiguration
	FailureMessage *string `json:"failureMessage"`
}

type MachingClusterStatus struct {
	// ClusterRef references the matching Cluster
	ClusterRef corev1.ObjectReference `json:"clusterRef"`

	// ManagedLabels indicates the labels being managed on
	// the cluster by this Classifier instance
	// +optional
	ManagedLabels []string `json:"managedLabels,omitempty"`

	// UnManagedLabel indicates the labels this Classifier instance
	// would like to manage but cannot because different instance is
	// already managing it
	// +optional
	UnManagedLabels []UnManagedLabel `json:"unManagedLabels,omitempty"`
}

// ClassifierStatus defines the observed state of Classifier
type ClassifierStatus struct {
	// MatchingClusterRefs reference all the cluster-api Cluster currently matching
	// Classifier
	MachingClusterStatuses []MachingClusterStatus `json:"machingClusterStatuses,omitempty"`

	// ClusterInfo reference all the cluster-api Cluster where Classifier
	// has been/is being deployed
	ClusterInfo []ClusterInfo `json:"clusterInfo,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=classifiers,scope=Cluster
//+kubebuilder:subresource:status

// Classifier is the Schema for the classifiers API
type Classifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClassifierSpec   `json:"spec,omitempty"`
	Status ClassifierStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClassifierList contains a list of Classifier
type ClassifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Classifier `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Classifier{}, &ClassifierList{})
}
