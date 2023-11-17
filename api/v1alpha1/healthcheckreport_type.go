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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HealthCheckReportKind = "HealthCheckReport"

	// HealthCheckReportFinalizer allows HealthReconciler to clean up resources associated with
	// HealthCheckReport before removing it from the apiserver.
	HealthCheckReportFinalizer = "healthcheckreport.finalizer.projectsveltos.io"

	// HealthCheckNameLabel is added to each HealthCheckReport generated
	// for a HealthCheck instance
	HealthCheckNameLabel = "projectsveltos.io/healthcheck-name"

	// HealthCheckReportClusterNameLabel is added to each HealthCheckReport
	HealthCheckReportClusterNameLabel = "healthcheckreport.projectsveltos.io/cluster-name"

	// HealthCheckReportClusterTypeLabel is added to each HealthCheckReport
	HealthCheckReportClusterTypeLabel = "healthcheckreport.projectsveltos.io/cluster-type"
)

func GetHealthCheckReportName(healthCheckName, clusterName string, clusterType *ClusterType) string {
	// TODO: shorten this
	return fmt.Sprintf("%s--%s--%s",
		strings.ToLower(string(*clusterType)), healthCheckName, clusterName)
}

func GetHealthCheckReportLabels(healthCheckName, clusterName string, clusterType *ClusterType) map[string]string {
	return map[string]string{
		HealthCheckNameLabel:              healthCheckName,
		HealthCheckReportClusterNameLabel: clusterName,
		HealthCheckReportClusterTypeLabel: strings.ToLower(string(*clusterType)),
	}
}

// +kubebuilder:validation:Enum:=Healthy;Progressing;Degraded;Suspended
type HealthStatus string

const (
	// HealthStatusHealthy indicates healthy status
	HealthStatusHealthy = HealthStatus("Healthy")

	// HealthStatusProgressing indicates resource is not healthy yet but
	// it is progressing
	HealthStatusProgressing = HealthStatus("Progressing")

	// HealthStatusDegraded indicates resource is degraded
	HealthStatusDegraded = HealthStatus("Degraded")

	// HealthStatusSuspended indicates resource is suspended
	HealthStatusSuspended = HealthStatus("Suspended")
)

type ResourceStatus struct {
	// ObjectRef for which status is reported
	ObjectRef corev1.ObjectReference `json:"objectRef"`

	// If HealthCheck Spec.CollectResources is set to true, resource
	// will be collected and contained in the Resource field.
	// +optional
	Resource []byte `json:"resource,omitempty"`

	// HealthStatus is the health status of the object
	HealthStatus HealthStatus `json:"healthStatus"`

	// Message is an extra message for human consumption
	// +optional
	Message string `json:"message,omitempty"`
}

type HealthCheckReportSpec struct {
	// ClusterNamespace is the namespace of the Cluster this
	// HealthCheckReport is for.
	ClusterNamespace string `json:"clusterNamespace"`

	// ClusterName is the name of the Cluster this HealthCheckReport
	// is for.
	ClusterName string `json:"clusterName"`

	// ClusterType is the type of Cluster this HealthCheckReport
	// is for.
	ClusterType ClusterType `json:"clusterType"`

	// HealthName is the name of the HealthCheck instance this report
	// is for.
	HealthCheckName string `json:"healthCheckName"`

	// ResourceStatuses contains a list of resources with their status
	// +optional
	ResourceStatuses []ResourceStatus `json:"resourceStatuses,omitempty"`
}

// HealthCheckReportStatus defines the observed state of HealthCheckReport
type HealthCheckReportStatus struct {
	// Phase represents the current phase of report.
	// +optional
	Phase *ReportPhase `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=healthcheckreports,scope=Namespaced
//+kubebuilder:subresource:status

// HealthCheckReport is the Schema for the HealthCheckReport API
type HealthCheckReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HealthCheckReportSpec   `json:"spec,omitempty"`
	Status HealthCheckReportStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HealthCheckReportList contains a list of HealthCheckReport
type HealthCheckReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealthCheckReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HealthCheckReport{}, &HealthCheckReportList{})
}
