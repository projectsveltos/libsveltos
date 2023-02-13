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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterHealthCheckFinalizer allows ClusterHealthCheckReconciler to clean up resources associated with
	// ClusterHealthCheck before removing it from the apiserver.
	ClusterHealthCheckFinalizer = "clusterhcfinalizer.projectsveltos.io"

	ClusterHealthCheckKind = "ClusterHealthCheck"
)

// ConditionSeverity expresses the severity of a Condition Type failing.
type ConditionSeverity string

const (
	// ConditionSeverityError specifies that a condition with `Status=False` is an error.
	ConditionSeverityError ConditionSeverity = "Error"

	// ConditionSeverityWarning specifies that a condition with `Status=False` is a warning.
	ConditionSeverityWarning ConditionSeverity = "Warning"

	// ConditionSeverityInfo specifies that a condition with `Status=False` is informative.
	ConditionSeverityInfo ConditionSeverity = "Info"

	// ConditionSeverityNone should apply only to conditions with `Status=True`.
	ConditionSeverityNone ConditionSeverity = ""
)

// ConditionType is a valid value for Condition.Type.
type ConditionType string

// Condition defines an observation of a Cluster API resource operational state.
type Condition struct {
	// Type of condition in CamelCase or in foo.example.com/CamelCase.
	Type ConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`

	// Severity provides an explicit classification of Reason code, so the users or machines can immediately
	// understand the current situation and act accordingly.
	// The Severity field MUST be set only when Status=False.
	// +optional
	Severity ConditionSeverity `json:"severity,omitempty"`

	// Last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed. If that is not known, then using the time when
	// the API field changed is acceptable.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// The reason for the condition's last transition in CamelCase.
	// The specific API may choose whether or not this field is considered a guaranteed API.
	// This field may not be empty.
	// +optional
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	// This field may be empty.
	// +optional
	Message string `json:"message,omitempty"`
}

type ClusterCondition struct {
	// ClusterNamespace is the namespace of the Cluster this
	// Condition is for.
	ClusterNamespace string `json:"clusterNamespace"`

	// ClusterName is the name of the Cluster this Condition is for.
	ClusterName string `json:"clusterName"`

	// ClusterType is the type of Cluster this Condition is for.
	ClusterType ClusterType `json:"clusterType"`

	// Cluster conditions.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

// Event specifies different type of liveness checks
// +kubebuilder:validation:Enum:=Addons
type LivenessType string

const (
	// LivenessTypeAddons refers to add-ons deployment state.
	LivenessTypeAddons = LivenessType("Addons")
)

type LivenessCheck struct {
	// Type specifies the type of liveness
	Type LivenessType `json:"type"`

	// LivenessSourceRef is a reference to a liveness-specific resource that holds
	// the details for the liveness check.
	// +optional
	LivenessSourceRef *corev1.ObjectReference `json:"livenessSourceRef,omitempty"`
}

// NotificationType specifies different type of notifications
// +kubebuilder:validation:Enum:=KubernetesEvent
type NotificationType string

const (
	// NotificationTypeKubernetesEvent refers to generating a Kubernetes event
	NotificationTypeKubernetesEvent = NotificationType("KubernetesEvent")
)

type Notification struct {
	// NotificationType specifies the type of notification
	Type NotificationType `json:"type"`

	// NotificationRef is a reference to a notification-specific resource that holds
	// the details for the notification.
	// +optional
	NotificationRef *corev1.ObjectReference `json:"notificationRef,omitempty"`
}

// ClusterHealthCheckSpec defines the desired state of ClusterHealthCheck
type ClusterHealthCheckSpec struct {
	// ClusterSelector identifies clusters to associate to.
	ClusterSelector Selector `json:"clusterSelector"`

	// LivenessChecks is a list of source of liveness checks to evaluate.
	// Anytime one of those changes, notifications will be sent
	LivenessChecks []LivenessCheck `json:"livenessChecks"`

	// Notification is a list of source of events to evaluate.
	Notifications []Notification `json:"notifications"`
}

type ClusterHealthCheckStatus struct {
	// ClusterConditions contains conditions for all clusters matching
	// ClusterHealthCheck instance
	// +optional
	ClusterConditions []ClusterCondition `json:"clusterCondition,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=clusterhealthchecks,scope=Cluster

// ClusterHealthCheck is the Schema for the clusterhealthchecks API
type ClusterHealthCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterHealthCheckSpec   `json:"spec,omitempty"`
	Status ClusterHealthCheckStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterHealthCheckList contains a list of ClusterHealthChecks
type ClusterHealthCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterHealthCheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterHealthCheck{}, &ClusterHealthCheckList{})
}