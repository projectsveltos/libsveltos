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

// PolicyRef specifies a resource containing one or more policy
// to deploy in matching Clusters.
type PolicyRef struct {
	// Namespace of the referenced resource.
	// Namespace can be left empty. In such a case, namespace will
	// be implicit set to cluster's namespace.
	Namespace string `json:"namespace"`

	// Name of the rreferenced resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Kind of the resource. Supported kinds are: Secrets and ConfigMaps.
	// +kubebuilder:validation:Enum=Secret;ConfigMap
	Kind string `json:"kind"`
}

func (r PolicyRef) String() string {
	return r.Kind + "-" + r.Namespace + "-" + r.Name
}
