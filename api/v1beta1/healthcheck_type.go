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
	// HealthCheckFinalizer allows HealthReconciler to clean up resources associated with
	// HealthCheck before removing it from the apiserver.
	HealthCheckFinalizer = "healthcheck.finalizer.projectsveltos.io"

	HealthCheckKind = "HealthCheck"
)

// HealthCheckSpec defines the desired state of HealthCheck
type HealthCheckSpec struct {
	// ResourceSelectors identifies what resources to select to evaluate health
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

	// The EvaluateHealth field specifies a Lua function responsible for evaluating the
	// health of the resources selected by resourceSelectors.
	// This function can assess the health of each resource independently or consider inter-resource relationships.
	// The function must be named *evaluate* and can access all objects identified by resourceSelectors using
	// the *resources* variable. It should return an array of structured instances, each containing the following fields:
	// - resource: The resource being evaluated
	// - healthStatus: The health status of the resource, which can be one of "Healthy", "Progressing", "Degraded", or "Suspended"
	// - message: An optional message providing additional information about the health status
	// +kubebuilder:validation:MinLength=1
	EvaluateHealth string `json:"evaluateHealth"`

	// CollectResources indicates whether matching resources need
	// to be collected and added to HealthReport.
	// +kubebuilder:default:=false
	// +optional
	CollectResources bool `json:"collectResources,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=healthchecks,scope=Cluster
//+kubebuilder:storageversion

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
