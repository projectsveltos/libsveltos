/*
Copyright 2024. projectsveltos.io. All rights reserved.

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

package v1beta1

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// PolicyTemplateAnnotation is the annotation that must be set on a policy when the
	// policy is a template and needs variable substitution.
	PolicyTemplateAnnotation = "projectsveltos.io/template"

	// PolicyLuaAnnotation is the annotation that must be set on a policy when the
	// policy contains Lua script.
	PolicyLuaAnnotation = "projectsveltos.io/lua"
)

const (
	// DeployedBySveltosAnnotation is an annotation Sveltos adds to
	// EventSource/HealthCheck/Classifier/ResourceSummary instances deployed
	// by sveltos in managed clusters. Those resources, once deployed in a
	// managed cluster, are evaluated by sveltos services (sveltos-agent and
	// drift-detection-manager) running in the managed cluster
	DeployedBySveltosAnnotation = "projectsveltos.io/deployed-by-sveltos"
)

const (
	// ServiceAccountNameLabel can be set on various Sveltos resources (ClusterProfile/EventSource/...)
	// to indicate which admin (represented by a ServiceAccount) is creating it (service account name).
	// ServiceAccountNameLabel used along with RoleRequest is Sveltos solution for multi tenancy.
	ServiceAccountNameLabel = "projectsveltos.io/serviceaccount-name"

	// ServiceAccountNamespaceLabel can be set on various Sveltos resources (ClusterProfile/EventSource/...)
	// to indicate which admin (represented by a ServiceAccount) is creating it (service account namespace).
	// ServiceAccountNamespaceLabel used along with RoleRequest is Sveltos solution for multi tenancy.
	ServiceAccountNamespaceLabel = "projectsveltos.io/serviceaccount-namespace"
)

// ReferencedResourceKind is a string representation of allowed kind of resources
// that can be referenced in a ClusterProfile
type ReferencedResourceKind string

// Define the ReferencedResourceKind constants.
const (
	SecretReferencedResourceKind    ReferencedResourceKind = "Secret"
	ConfigMapReferencedResourceKind ReferencedResourceKind = "ConfigMap"
)

const (
	// ClusterProfileSecretType is the only accepted type of secret in resources.
	ClusterProfileSecretType corev1.SecretType = "addons.projectsveltos.io/cluster-profile"
)

var (
	// ErrSecretTypeNotSupported signals that a Secret is not supported.
	ErrSecretTypeNotSupported = errors.New("unsupported secret type")
)

type Selector struct {
	metav1.LabelSelector `json:",inline"`
}

// ToSelector converts ClusterSelector to labels.Selector
func (cs *Selector) ToSelector() (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(&cs.LabelSelector)
}

// +kubebuilder:validation:Enum:=Provisioning;Provisioned;Failed;Removing;Removed
type SveltosFeatureStatus string

const (
	// SveltosStatusProvisioning indicates that sveltos feature is being
	// provisioned in the workload cluster
	SveltosStatusProvisioning = SveltosFeatureStatus("Provisioning")

	// SveltosStatusProvisioned indicates that sveltos has been
	// provisioned in the workload cluster
	SveltosStatusProvisioned = SveltosFeatureStatus("Provisioned")

	// SveltosStatusFailed indicates that configuring sveltos feature
	// in the workload cluster failed
	SveltosStatusFailed = SveltosFeatureStatus("Failed")

	// SveltosStatusRemoving indicates that sveltos feature is being
	// removed
	SveltosStatusRemoving = SveltosFeatureStatus("Removing")

	// SveltosStatusRemoved indicates that sveltos feature is removed
	SveltosStatusRemoved = SveltosFeatureStatus("Removed")
)

type ClusterInfo struct {
	// Cluster references the Cluster
	Cluster corev1.ObjectReference `json:"cluster"`

	// Hash represents the hash of the Classifier currently deployed
	// in the Cluster
	Hash []byte `json:"hash"`

	// Status represents the state of the feature in the workload cluster
	// +optional
	Status SveltosFeatureStatus `json:"status,omitempty"`

	// FailureMessage provides more information about the error.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

// Operation specifies
// +kubebuilder:validation:Enum:=Equal;Different;Has;DoesNotHave
type Operation string

const (
	// OperationEqual will verify equality. Corresponds to ==
	OperationEqual = Operation("Equal")

	// OperationDifferent will verify difference. Corresponds to !=
	OperationDifferent = Operation("Different")

	// OperationHas will verify key is present
	OperationHas = Operation("Has")

	// OperationDoesNotHave will verify key is not present
	OperationDoesNotHave = Operation("DoesNotHave")
)

type LabelFilter struct {
	// Key is the label key
	Key string `json:"key"`

	// Operation is the comparison operation
	Operation Operation `json:"operation"`

	// Value is the label value
	// +optional
	Value string `json:"value,omitempty"`
}

// CELRule defines a named CEL rule used in EvaluateCEL.
type CELRule struct {
	// Name is a human-readable identifier for the rule.
	Name string `json:"name"`

	// Rule is the CEL (Common Expression Language) expression to evaluate.
	// It must return a bool
	Rule string `json:"rule"`
}

// ResourceSelector defines what resources are a match
type ResourceSelector struct {
	// Group of the resource deployed in the Cluster.
	Group string `json:"group"`

	// Version of the resource deployed in the Cluster.
	Version string `json:"version"`

	// Kind of the resource deployed in the Cluster.
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// LabelFilters allows to filter resources based on current labels.
	// +optional
	LabelFilters []LabelFilter `json:"labelFilters,omitempty"`

	// Namespace of the resource deployed in the  Cluster.
	// Empty for resources scoped at cluster level.
	// For namespaced resources, an empty string "" indicates all namespaces.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the resource deployed in the  Cluster.
	// +optional
	Name string `json:"name,omitempty"`

	// Evaluate contains a function "evaluate" in lua language.
	// The function will be passed one of the object selected based on
	// above criteria.
	// Must return struct with field "matching" representing whether
	// object is a match and an optional "message" field.
	// +optional
	Evaluate string `json:"evaluate,omitempty"`

	// EvaluateCEL contains a list of named CEL (Common Expression Language) rules.
	// Each rule will be evaluated in order against each object selected based on
	// the criteria defined above. Each rule's expression must return a boolean value
	// indicating whether the object is a match.
	//
	// Evaluation stops at the first rule that returns true; subsequent
	// rules will not be evaluated.
	// +optional
	EvaluateCEL []CELRule `json:"evaluateCEL,omitempty"`
}

type PatchSelector struct {

	// Version of the API Group to select resources from.
	// Together with Group and Kind it is capable of unambiguously identifying and/or selecting resources.
	// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/api-group.md
	// +optional
	Version string `json:"version,omitempty"`

	// Group is the API group to select resources from.
	// Together with Version and Kind it is capable of unambiguously identifying and/or selecting resources.
	// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/api-group.md
	// +optional
	Group string `json:"group,omitempty"`

	// Kind of the API Group to select resources from.
	// Together with Group and Version it is capable of unambiguously
	// identifying and/or selecting resources.
	// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/api-group.md
	// +optional
	Kind string `json:"kind,omitempty"`

	// Namespace to select resources from.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name to match resources with.
	// +optional
	Name string `json:"name,omitempty"`

	// AnnotationSelector is a string that follows the label selection expression
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#api
	// It matches with the resource annotations.
	// +optional
	AnnotationSelector string `json:"annotationSelector,omitempty"`

	// LabelSelector is a string that follows the label selection expression
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#api
	// It matches with the resource labels.
	// +optional
	LabelSelector string `json:"labelSelector,omitempty"`
}

// Patch contains an inline StrategicMerge or JSON6902 patch, and the target the patch should
// be applied to.
type Patch struct {
	// Patch contains an inline StrategicMerge patch or an inline JSON6902 patch with
	// an array of operation objects.
	// These values can be static or leverage Go templates for dynamic customization.
	// When expressed as templates, the values are filled in using information from
	// resources within the management cluster before deployment (Cluster and TemplateResourceRefs)
	// +required
	Patch string `json:"patch"`

	// Target points to the resources that the patch document should be applied to.
	// +optional
	Target *PatchSelector `json:"target,omitempty"`
}
