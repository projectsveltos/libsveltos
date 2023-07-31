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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReloaderFinalizer finalizer
	ReloaderFinalizer = "reloader.finalizer.projectsveltos.io"

	ReloaderKind = "Reloader"
)

// ReloaderInfo represents a resource that need to be reloaded
// if any mounted ConfigMap/Secret changes.
type ReloaderInfo struct {
	// Namespace of the referenced resource.
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// Name of the referenced resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Kind of the resource. Supported kinds are: Deployment StatefulSet DaemonSet.
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;DaemonSet
	Kind string `json:"kind"`

	// +optional
	Value string `json:"value,omitempty"`
}

// ReloaderSpec defines the desired state of Reloader
type ReloaderSpec struct {
	// +optional
	ReloaderInfo []ReloaderInfo `json:"reloaderInfo,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=reloaders,scope=Cluster

// Reloader is the Schema for the Reloader API
type Reloader struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ReloaderSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ReloaderList contains a list of Reloader
type ReloaderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Reloader `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Reloader{}, &ReloaderList{})
}
