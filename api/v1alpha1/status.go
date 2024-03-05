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
	corev1 "k8s.io/api/core/v1"
)

// Status defines the observed state of ClusterSet/Set
type Status struct {
	// MatchingClusterRefs reference all the clusters currently matching
	// ClusterSet/Set ClusterSelector
	MatchingClusterRefs []corev1.ObjectReference `json:"matchingClusters,omitempty"`

	// SelectedClusters reference all the cluster currently selected among
	// all the ones matching
	SelectedClusters []corev1.ObjectReference `json:"selectedClusters,omitempty"`
}
