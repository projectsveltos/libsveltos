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
