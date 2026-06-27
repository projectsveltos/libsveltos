/*
Copyright 2026. projectsveltos.io. All rights reserved.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	// ManagementClusterClassifierFinalizer allows ManagementClusterClassifierReconciler
	// to clean up resources before removing the CR from the apiserver.
	ManagementClusterClassifierFinalizer = "managementclusterclassifierfinalizer.projectsveltos.io"

	ManagementClusterClassifierKind = "ManagementClusterClassifier"
)

// ManagementClusterClassifierSpec defines the desired state of ManagementClusterClassifier.
type ManagementClusterClassifierSpec struct {
	// MatchResources lists the management-cluster resource types to watch.
	// Reuses the existing ResourceSelector type. Label/name/namespace filters
	// and per-resource Lua (ResourceSelector.Evaluate) all apply: only objects
	// where Evaluate returns {matching: true} are included in the resources array
	// passed to ClassificationLua. Same behavior as the existing Classifier.
	MatchResources []ResourceSelector `json:"matchResources"`

	// ClassificationLua is a Lua function called once per reconcile with the full
	// set of resources matched by MatchResources. It receives the array as resources
	// and must return an array of cluster references.
	// Each entry must have fields: namespace (string), name (string), kind (string).
	// kind must be "SveltosCluster" or "Cluster". Entries with a missing or unrecognized
	// kind are skipped and recorded in status.failureMessage.
	// Passing all resources at once allows the function to make decisions that
	// depend on the relationship between multiple resources.
	ClassificationLua string `json:"classificationLua"`

	// ClassifierLabels is the set of labels to add to every cluster returned by
	// ClassificationLua. Same type as Classifier.spec.classifierLabels.
	ClassifierLabels []ClassifierLabel `json:"classifierLabels"`
}

// ManagementClusterClassifierStatus defines the observed state of ManagementClusterClassifier.
type ManagementClusterClassifierStatus struct {
	// FailureMessage reports the error from the last reconcile, if any.
	// Set when resource collection or Lua evaluation fails; cleared on success.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=managementclusterclassifiers,scope=Cluster
//+kubebuilder:subresource:status

// ManagementClusterClassifier evaluates resources on the management cluster and
// labels managed clusters accordingly. Unlike Classifier, no deployment to managed
// clusters is required: the Lua function runs entirely on the management cluster and
// returns the list of clusters to label.
type ManagementClusterClassifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagementClusterClassifierSpec   `json:"spec,omitempty"`
	Status ManagementClusterClassifierStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagementClusterClassifierList contains a list of ManagementClusterClassifier.
type ManagementClusterClassifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagementClusterClassifier `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(GroupVersion,
			&ManagementClusterClassifier{},
			&ManagementClusterClassifierList{},
		)
		return nil
	})
}
