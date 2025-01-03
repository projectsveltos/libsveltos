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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TechsupportFinalizer allows TechsupportReconciler to clean up resources associated with
	// Techsupport instance before removing it from the apiserver.
	TechsupportFinalizer = "techsupportfinalizer.projectsveltos.io"
)

// LogFilter allows to select which logs to collect
type Log struct {
	// Namespace of the pods deployed in the Cluster.
	// An empty string "" indicates all namespaces.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the pods deployed in the Cluster.
	// +optional
	Name string `json:"name,omitempty"`

	// LabelFilters allows to filter pods based on current labels.
	// +optional
	LabelFilters []LabelFilter `json:"labelFilters,omitempty"`

	// A relative time in seconds before the current time from which to collect logs.
	// If this value precedes the time a pod was started, only logs since the pod
	// start will be returned.
	// +optional
	SinceSeconds *int64 `json:"sinceSeconds,omitempty"`
}

// EventType represents the possible types of events.
type EventType string

const (
	EventTypeNormal  EventType = "Normal"
	EventTypeWarning EventType = "Warning"
)

type Event struct {
	// Namespace of the events.
	// An empty string "" indicates all namespaces.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Type filters events based on the type of the events (Normal, Warning),
	// +kubebuilder:validation:Enum=Normal;Warning
	// +optional
	Type string `json:"type,omitempty"`
}

type FromManagement struct {
	// Resources indicates what resorces to collect
	// +optional
	Resources []ResourceSelector `json:"resources,omitempty"`

	// Logs indicates what pods' log to collect
	// +optional
	Logs []Log `json:"logs,omitempty"`

	// Events indicates what events to collect
	// +optional
	Events []Event `json:"events,omitempty"`
}

type FromManaged struct {
	// ClusterSelector identifies clusters to collect techsupport from.
	// +optional
	ClusterSelector Selector `json:"clusterSelector,omitempty"`

	// ClusterRefs identifies clusters to collect techsupport from.
	// +optional
	ClusterRefs []corev1.ObjectReference `json:"clusterRefs,omitempty"`

	// Resources indicates what resorces to collect
	// +optional
	Resources []ResourceSelector `json:"resources,omitempty"`

	// Logs indicates what pods' log to collect
	// +optional
	Logs []Log `json:"logs,omitempty"`

	// Events indicates what events to collect
	// +optional
	Events []Event `json:"events,omitempty"`
}

type SchedulingConfig struct {
	// Schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`
}

// +kubebuilder:validation:Enum:=Collected;InProgress;Failed
type CollectionStatus string

const (
	// CollectionStatusStatusInProgress indicates that collection is being collected
	CollectionStatusInProgress = CollectionStatus("InProgress")

	// CollectionStatusStatusCollected indicates that collection succeeded
	CollectionStatusCollected = CollectionStatus("Collected")

	// CollectionStatusStatusFailed indicates that last collection failed
	CollectionStatusFailed = CollectionStatus("Failed")
)

// TechsupportSpec defines the desired state of Techsupport
type TechsupportSpec struct {
	// FromManagement identifies which resources and logs to collect
	// from the management cluster
	// +optional
	FromManagement FromManagement `json:"fromManagement,omitempty"`

	// FromManaged specifies which resources and logs to collect from
	// matching managed cluster.
	// +optional
	FromManaged FromManaged `json:"fromManaged,omitempty"`

	// OnDemand indicates if tech support should be collected immediately.
	// +optional
	OnDemand bool `json:"onDemand,omitempty"`

	// SchedulingConfig defines a schedule options for recurring tech support
	// collections.
	// +optional
	SchedulingConfig *SchedulingConfig `json:"schedulingConfig,omitempty"`

	// Notification is a list of notification mechanisms.
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	Notifications []Notification `json:"notifications"`
}

// TechsupportStatus defines the observed state of Techsupport
type TechsupportStatus struct {
	// Information when next techsupport is scheduled
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`

	// Information when was the last time a techsupport was successfully scheduled.
	// +optional
	LastRunTime *metav1.Time `json:"lastRunTime,omitempty"`

	// Status indicates what happened to last techsupport collection.
	LastRunStatus *CollectionStatus `json:"lastRunStatus,omitempty"`

	// FailureMessage provides more information about the error, if
	// any occurred
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Hash represents of a unique value for techsupport Spec at a fixed point in
	// time
	// +optional
	Hash []byte `json:"hash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=techsupports,scope=Cluster
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Techsupport is the Schema for the techsupport API
type Techsupport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TechsupportSpec   `json:"spec,omitempty"`
	Status TechsupportStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TechsupportList contains a list of Techsupport instances
type TechsupportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Techsupport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Techsupport{}, &TechsupportList{})
}
