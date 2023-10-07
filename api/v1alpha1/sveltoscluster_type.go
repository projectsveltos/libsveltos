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
	SveltosClusterKind = "SveltosCluster"
)

// SveltosClusterSpec defines the desired state of SveltosCluster
type SveltosClusterSpec struct {
	// KubeconfigName allows overriding the default Sveltos convention which expected a valid kubeconfig
	// to be hosted in a secret with the pattern ${sveltosClusterName}-sveltos-kubeconfig.
	//
	// When a value is specified, the referenced Kubernetes Secret object must exist,
	// and will be used to connect to the Kubernetes cluster.
	// +optional
	KubeconfigName string `json:"kubeconfigName,omitempty"`
	// Paused can be used to prevent controllers from processing the
	// SveltosCluster and all its associated objects.
	// +optional
	Paused bool `json:"paused,omitempty"`
}

// SveltosClusterStatus defines the status of SveltosCluster
type SveltosClusterStatus struct {
	// The Kubernetes version of the cluster.
	// +optional
	Version string `json:"version,omitempty"`

	// Ready is the state of the cluster.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// FailureMessage is a human consumable message explaining the
	// misconfiguration
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=sveltosclusters,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Indicates whether cluster is ready to be managed by sveltos"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Kubernetes version associated with this Cluster"

// SveltosCluster is the Schema for the SveltosCluster API
type SveltosCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SveltosClusterSpec   `json:"spec,omitempty"`
	Status SveltosClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SveltosClusterList contains a list of SveltosCluster
type SveltosClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SveltosCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SveltosCluster{}, &SveltosClusterList{})
}
