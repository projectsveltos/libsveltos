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
	RoleRequestFinalizer = "rolerequestfinalizer.projectsveltos.io"

	RoleRequestKind = "RoleRequest"

	// RoleRequestLabel is added to each object generated for a RoleRequest
	// in both management and managed clusters
	RoleRequestLabel = "projectsveltos.io/role-request-name"

	FeatureRoleRequest = "RoleRequest"
)

// RoleRequestSpec defines the desired state of RoleRequest
type RoleRequestSpec struct {
	// ClusterSelector identifies clusters where permissions requestes
	// in this instance will be granted
	ClusterSelector Selector `json:"clusterSelector"`

	// RoleRefs references all the Secret/ConfigMaps containing kubernetes
	// Roles/ClusterRoles that need to be deployed in the matching clusters.
	// +optional
	RoleRefs []PolicyRef `json:"roleRefs,omitempty"`

	// ExpirationSeconds is the requested duration of validity of the TokenRequest
	// associated to ServiceAccount. If not specified, default value is used
	// +optional
	ExpirationSeconds *int64 `json:"expirationSeconds,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount representing a tenant admin for which
	// those permissions are requested
	ServiceAccountName string `json:"serviceAccountName"`

	// ServiceAccountNamespace is the name of the ServiceAccount representing a tenant admin
	// for which those permissions are requested
	ServiceAccountNamespace string `json:"serviceAccountNamespace"`
}

// RoleRequestStatus defines the status of RoleRequest
type RoleRequestStatus struct {
	// MatchingClusterRefs reference all the cluster currently matching
	// RoleRequest ClusterSelector
	MatchingClusterRefs []corev1.ObjectReference `json:"matchingClusters,omitempty"`

	// ClusterInfo represents the hash of the ClusterRoles/Roles deployed in
	// a matching cluster for the admin.
	// +optional
	ClusterInfo []ClusterInfo `json:"clusterInfo,omitempty"`

	// FailureMessage provides more information if an error occurs.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=rolerequests,scope=Cluster
//+kubebuilder:subresource:status

// RoleRequest is the Schema for the rolerequest API
type RoleRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleRequestSpec   `json:"spec,omitempty"`
	Status RoleRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RoleRequestList contains a list of RoleRequest
type RoleRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RoleRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RoleRequest{}, &RoleRequestList{})
}
