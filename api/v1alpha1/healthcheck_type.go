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
	// HealthCheckFinalizer allows HealthReconciler to clean up resources associated with
	// HealthCheck before removing it from the apiserver.
	HealthCheckFinalizer = "healthcheck.finalizer.projectsveltos.io"

	HealthCheckKind = "HealthCheck"
)

// HealthCheckSpec defines the desired state of HealthCheck
type HealthCheckSpec struct {
	// ResourceSelectors identifies what resources to select
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

	// This field is optional and can be used to specify a Lua function
	// that will be used to further analyze a subset of the resources that
	// have already been selected using the ResourceSelector field.
	// The function will receive the array of resources selected by ResourceSelectors.
	// If this field is not specified, all resources selected by the ResourceSelector
	// field will be considered.
	// This field allows to perform more complex analysys  on the resources, looking
	// at all resources together.
	// This can be useful for more sophisticated tasks, such as identifying resources
	// that are related to each other or that have similar properties.
	// The Lua function must return a struct with:
	// - "resources" field: slice of resorces;
	// - "message" field: (optional) message.
	AggregatedAnalysis string `json:"aggregatedAnalysis,omitempty"`

	// CollectResources indicates whether matching resources need
	// to be collected and added to EventReport.
	// +kubebuilder:default:=false
	// +optional
	CollectResources bool `json:"collectResources,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=healthchecks,scope=Cluster

// HealthCheck is the Schema for the HealthCheck API
type HealthCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HealthCheckSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// HealthCheckList contains a list of Event
type HealthCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealthCheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HealthCheck{}, &HealthCheckList{})
}
