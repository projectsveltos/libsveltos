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
	// ClassifierLabelName is added to each ClassifierReport generated
	// for a Classifier instance
	ClassifierLabelName = "projectsveltos.io/classifier-name"
)

// ClassifierReportSpec defines the desired state of ClassifierReport
type ClassifierReportSpec struct {
	// ClusterNamespace is the namespace of the CAPI Cluster this
	// ClusterReport is for.
	ClusterNamespace string `json:"clusterNamespace"`

	// ClusterName is the name of the CAPI Cluster this ClusterReport
	// is for.
	ClusterName string `json:"clusterName"`

	// ClassifierName is the name of the Classifier instance this report
	// is for.
	ClassifierName string `json:"classifierName"`

	// Match indicates whether CAPI Cluster is currently a match for
	// the Classifier instance this report is for
	Match bool `json:"match"`
}

// ClassifierReportStatus defines the observed state of ClassifierReport
type ClassifierReportStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=classifierreports,scope=Namespaced
//+kubebuilder:subresource:status

// ClassifierReport is the Schema for the classifierreports API
type ClassifierReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClassifierReportSpec   `json:"spec,omitempty"`
	Status ClassifierReportStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClassifierReportList contains a list of ClassifierReport
type ClassifierReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClassifierReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClassifierReport{}, &ClassifierReportList{})
}
