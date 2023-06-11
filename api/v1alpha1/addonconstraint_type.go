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

package v1alpha1

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AddonConstraintFinalizer finalizer for AddonConstraint instances
	AddonConstraintFinalizer = "addonconstraintfinalizer.projectsveltos.io"

	AddonConstraintKind = "AddonConstraint"
)

// A label with this key is added to AddonConstraint instances for each matching cluster
func GetClusterLabel(clusterNamespace, clusterName string, clusterType *ClusterType) string {
	return fmt.Sprintf("%s--%s--%s",
		strings.ToLower(string(*clusterType)), clusterNamespace, clusterName)
}

// GetClusterAnnotation returns the annotation added on each cluster that indicates
// addon constraints for this cluster, if any, are ready
func GetClusterAnnotation() string {
	return "addon-constraints-ready"
}

type OpenAPIValidationRef struct {
	// Namespace of the referenced resource.
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// Name of the referenced resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Kind of the resource. Supported kinds are:
	// - flux GitRepository;OCIRepository;Bucket
	// - ConfigMap/Secret
	// +kubebuilder:validation:Enum=GitRepository;OCIRepository;Bucket;ConfigMap;Secret
	Kind string `json:"kind"`

	// Path to the directory containing the openapi validations.
	// Defaults to 'None', which translates to the root path of the SourceRef.
	// Ignored for ConfigMap/Secret.
	// +optional
	Path string `json:"path,omitempty"`
}

// AddonConstraintSpec defines the desired state of AddonConstraint
type AddonConstraintSpec struct {
	// ClusterSelector identifies clusters to associate to.
	// +optional
	ClusterSelector Selector `json:"clusterSelector,omitempty"`

	// ClusterRefs identifies clusters to associate to.
	// +optional
	ClusterRefs []corev1.ObjectReference `json:"clusterRefs,omitempty"`

	// OpenAPIValidationRefs is a list of OpenAPI validations. In the matching clusters, add-ons
	// will be deployed only if all validations pass.
	OpenAPIValidationRefs []OpenAPIValidationRef `json:"openAPIValidationRefs,omitempty"`
}

// AddonConstraintStatus defines the observed state of AddonConstraint
type AddonConstraintStatus struct {
	// MatchingClusterRefs reference all the clusters currently matching
	// ClusterSelector
	MatchingClusterRefs []corev1.ObjectReference `json:"matchingClusters,omitempty"`

	// OpenapiValidations contains all validations collected from all existing
	// referenced resources
	OpenapiValidations map[string][]byte `json:"openapiValidations,omitempty"`

	// FailureMessage provides more information if an error occurs.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=addonconstraints,scope=Cluster
//+kubebuilder:subresource:status

// AddonConstraint is the Schema for the AddonConstraint API
type AddonConstraint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonConstraintSpec   `json:"spec,omitempty"`
	Status AddonConstraintStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AddonConstraintList contains a list of AddonConstraints
type AddonConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddonConstraint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AddonConstraint{}, &AddonConstraintList{})
}
