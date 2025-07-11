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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EventReportKind = "EventReport"

	// EventReportFinalizer allows EventReportReconciler to clean up resources associated with
	// EventReport before removing it from the apiserver.
	EventReportFinalizer = "eventreport.finalizer.projectsveltos.io"

	// EventSourceNameLabel is added to each EventReport generated
	// for a EventSource instance
	EventSourceNameLabel = "projectsveltos.io/eventsource-name"

	// EventReportClusterNameLabel is added to each EventReport
	EventReportClusterNameLabel = "eventreport.projectsveltos.io/cluster-name"

	// EventReportClusterTypeLabel is added to each EventReport
	EventReportClusterTypeLabel = "eventreport.projectsveltos.io/cluster-type"

	// EventReportPullModeLabel is added to each EventReport from clusters in pull mode
	EventReportPullModeLabel = "eventreport.projectsveltos.io/pullmode"
)

func GetEventReportName(healthName, clusterName string, clusterType *ClusterType) string {
	// TODO: shorten this
	return fmt.Sprintf("%s--%s--%s",
		strings.ToLower(string(*clusterType)), healthName, clusterName)
}

func GetEventReportLabels(eventSourceName, clusterName string, clusterType *ClusterType) map[string]string {
	return map[string]string{
		EventSourceNameLabel:        eventSourceName,
		EventReportClusterNameLabel: clusterName,
		EventReportClusterTypeLabel: strings.ToLower(string(*clusterType)),
	}
}

type EventReportSpec struct {
	// ClusterNamespace is the namespace of the Cluster this
	// EventReport is for.
	ClusterNamespace string `json:"clusterNamespace"`

	// ClusterName is the name of the Cluster this EventReport
	// is for.
	ClusterName string `json:"clusterName"`

	// ClusterType is the type of Cluster this EventReport
	// is for.
	ClusterType ClusterType `json:"clusterType"`

	// EventSourceName is the name of the EventSource instance this report
	// is for.
	EventSourceName string `json:"eventSourceName"`

	// MatchingResources contains a list of resources matching an EventSource
	// +optional
	MatchingResources []corev1.ObjectReference `json:"matchingResources,omitempty"`

	// If EventSource Spec.CollectResources is set to true, all matching resources
	// will be collected and contained in the Resources field.
	// +optional
	Resources []byte `json:"resources,omitempty"`

	// CloudEvents contains a list of CloudEvents matching an EventSource
	// +optional
	CloudEvents [][]byte `json:"cloudEvents,omitempty"`
}

// EventReportStatus defines the observed state of EventReport
type EventReportStatus struct {
	// Phase represents the current phase of report.
	// +optional
	Phase *ReportPhase `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=eventreports,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// EventReport is the Schema for the EventReport API
type EventReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventReportSpec   `json:"spec,omitempty"`
	Status EventReportStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventReportList contains a list of EventReport
type EventReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventReport{}, &EventReportList{})
}
