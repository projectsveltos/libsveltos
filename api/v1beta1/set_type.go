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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SetFinalizer allows SetReconciler to clean up resources associated with
	// Set before removing it from the apiserver.
	SetFinalizer = "setfinalizer.projectsveltos.io"

	SetKind = "Set"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=sets,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Set is the Schema for the sets API
type Set struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Spec   `json:"spec,omitempty"`
	Status Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SetList contains a list of Set
type SetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Set `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Set{}, &SetList{})
}
