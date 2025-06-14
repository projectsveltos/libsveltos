/*
Copyright 2025. projectsveltos.io. All rights reserved.

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
	ConfigurationBundleKind = "ConfigurationBundle"
)

type ConfigurationBundleSpec struct {
	// Resources contains all resources that need to be deployed.
	// Content is either YAML or JSON
	// +listType=atomic
	// +optional
	Resources []string `json:"resources,omitempty"`

	// NotTracked, when true, signifies that the resources managed by the
	// ConfigurationBundles should not be tracked for conflicts
	// with other configurations and will not be automatically removed when the
	// ConfigurationGroup is deleted. This is intended for resources like
	// Sveltos CRDs or the agents Sveltos deploys in the managed clusters.
	NotTracked bool `json:"notTracked,omitempty"`
}

type ConfigurationBundleStatus struct {
	// Hash represents of a unique value for the content stored in
	// the ConfigurationBundle
	// +optional
	Hash []byte `json:"hash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=configurationbundles,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// ConfigurationBundle is the Schema for the configurationbundle API
type ConfigurationBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationBundleSpec   `json:"spec,omitempty"`
	Status ConfigurationBundleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigurationBundleList contains a list of ConfigurationBundle
type ConfigurationBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigurationBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigurationBundle{}, &ConfigurationBundleList{})
}
