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
	ClusterKind = "Cluster"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// Paused can be used to prevent controllers from processing the Cluster and all its associated objects.
	// +optional
	Paused bool `json:"paused,omitempty"`
}

// ClusterStatus defines the status of Cluster
type ClusterStatus struct {
	// Ready is the state of the cluster.
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=clusters,scope=Namespaced
//+kubebuilder:subresource:status

// Cluster is the Schema for the cluster API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
