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
	// Group of the resource deployed in the Cluster.
	Group string `json:"group"`

	// Version of the resource deployed in the Cluster.
	Version string `json:"version"`

	// Kind of the resource deployed in the Cluster.
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// LabelFilters allows to filter resources based on current labels.
	// +optional
	LabelFilters []LabelFilter `json:"labelFilters,omitempty"`

	// Namespace of the resource deployed in the  Cluster.
	// Empty for resources scoped at cluster level.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Script is a text containing a lua script.
	// Must return a struct with field "status"
	// set to one of the possible value of HealthStatus.
	Script string `json:"script,omitempty"`

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
