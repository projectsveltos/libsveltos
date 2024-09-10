package notifications

import (
	corev1 "k8s.io/api/core/v1"
)

// NotificationType specifies different type of notifications
// +kubebuilder:validation:Enum:=SMTP
type NotificationType string

const (
	// NotificationTypeSMTP refers to generating an Email message
	NotificationTypeSMTP = NotificationType("SMTP")
)

type Notification struct {
	// Name of the notification check.
	// Must be a DNS_LABEL and unique within the list of notifications.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// NotificationType specifies the type of notification
	Type NotificationType `json:"type"`

	// NotificationRef is a reference to a notification-specific resource that holds
	// the details for the notification.
	// +optional
	NotificationRef *corev1.ObjectReference `json:"notificationRef,omitempty"`
}
