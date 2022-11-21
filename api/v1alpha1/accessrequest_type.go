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
	AccessRequestFinalizer = "accessrequestfinalizer.projectsveltos.io"

	AccessRequestKind = "AccessRequest"
)

// RequestType specifies the type of AccessRequest
// +kubebuilder:validation:Enum:=ClassifierAgent;Different
type RequestType string

const (
	// ClassifierAgent is the request type to generate kubeconfig
	// for classifier agent
	ClassifierAgentRequest = RequestType("ClassifierAgent")
)

// AccessRequestSpec defines the desired state of AccessRequest
type AccessRequestSpec struct {
	// Namespace is the namespace of the service account created
	// for this AccessRequest
	Namespace string `json:"namespace"`

	// Name is the name of the service account created
	// for this AccessRequest
	Name string `json:"name"`

	// Type represent the type of the request
	Type RequestType `json:"type"`
}

// AccessRequestStatus defines the status of AccessRequest
type AccessRequestStatus struct {
	// SecretRef points to the Secret containing Kubeconfig
	// +optional
	SecretRef *corev1.ObjectReference `json:"secretRef,omitempty"`

	// FailureMessage provides more information if an error occurs.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=accessrequests,scope=Cluster

// AccessRequest is the Schema for the accessrequest API
type AccessRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessRequestSpec   `json:"spec,omitempty"`
	Status AccessRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AccessRequestList contains a list of AccessRequest
type AccessRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AccessRequest{}, &AccessRequestList{})
}
