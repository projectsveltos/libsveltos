/*
Copyright 2026. projectsveltos.io. All rights reserved.

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

// Package eventreasons defines the vocabulary Sveltos components use when they record
// Kubernetes events: the reason, the action and the maximum length of the note.
//
// Conventions for every Sveltos component:
//   - The reason is CamelCase and names the cause, not the component. Use one of the
//     constants in this package, so the same cause has the same reason everywhere.
//   - The event type is Warning for anything an operator must act on (failure, conflict,
//     drift, degraded, stalled) and Normal for a state change that completed. The type
//     each reason is meant to be used with is stated in its comment.
//   - The regarding object is the Sveltos resource the user knows (ClusterSummary,
//     ClusterProfile, SveltosCluster, EventTrigger, Classifier, ...).
//   - The note names the cluster, the feature or profile, and the cause. It does not
//     repeat the reason. It is passed through TruncateNote.
//   - Record an event the first time a cause is seen and when the cause changes, not on
//     every retry.
package eventreasons

import (
	"unicode/utf8"
)

// Reasons for events about the deployment of a profile on a cluster. They are recorded on
// the ClusterSummary.
const (
	// FeatureDeployed is used, with a Normal event, when a feature (Helm, Resources,
	// Kustomize) was deployed on a cluster.
	FeatureDeployed = "FeatureDeployed"
	// FeatureRemoved is used, with a Normal event, when a feature was removed from a cluster.
	FeatureRemoved = "FeatureRemoved"
	// FeatureFailed is used, with a Warning event, when a feature failed to deploy or to be
	// removed and the operation is retried. Record it on the first failure and when the
	// error changes.
	FeatureFailed = "FeatureFailed"
	// FeatureFailedNonRetriable is used, with a Warning event, when a feature failed and
	// will not be retried until something changes.
	FeatureFailedNonRetriable = "FeatureFailedNonRetriable"
	// ProfileConflict is used, with a Warning event, when a resource a profile wants to
	// deploy is already managed by another profile.
	ProfileConflict = "ProfileConflict"
	// DependencyNotReady is used, with a Warning event, when a profile is waiting for a
	// profile in its DependsOn list.
	DependencyNotReady = "DependencyNotReady"
	// UpdatePending is used, with a Warning event, when an update is not being delivered to
	// a cluster because the cluster is paused, its active window is closed, or the MaxUpdate
	// throttle has been reached.
	UpdatePending = "UpdatePending"
	// ClusterNotReady is used, with a Warning event, when a cluster cannot be configured
	// because it is not ready.
	ClusterNotReady = "ClusterNotReady"
	// AgentUnhealthy is used, with a Warning event, when the Sveltos agent of a cluster is
	// not healthy.
	AgentUnhealthy = "AgentUnhealthy"
)

// Reasons for events about configuration drift.
const (
	// DriftDetected is used, with a Warning event, when a resource deployed by Sveltos was
	// changed or deleted in a managed cluster.
	DriftDetected = "DriftDetected"
	// DriftCorrected is used, with a Normal event, when Sveltos deployed again a resource
	// that had drifted.
	DriftCorrected = "DriftCorrected"
)

// Reasons for events about health checks.
const (
	// HealthCheckDegraded is used, with a Warning event, when a HealthCheck evaluated a
	// resource as degraded.
	HealthCheckDegraded = "HealthCheckDegraded"
	// LivenessCheckFailed is used, with a Warning event, when the liveness checks of a
	// ClusterHealthCheck are not passing.
	LivenessCheckFailed = "LivenessCheckFailed"
	// ClusterHealthCheckFailed is used, with a Warning event, when a ClusterHealthCheck could not be
	// deployed to a cluster, removed from it, or its liveness checks could not be evaluated.
	ClusterHealthCheckFailed = "ClusterHealthCheckFailed"
	// NotificationFailed is used, with a Warning event, when a notification of a ClusterHealthCheck
	// could not be delivered.
	NotificationFailed = "NotificationFailed"
)

// Reasons for events about event driven deployments.
const (
	// EventTriggerFailed is used, with a Warning event, when an EventTrigger could not
	// create or update the ClusterProfile it generates.
	EventTriggerFailed = "EventTriggerFailed"
	// EventSourceInvalid is used, with a Warning event, when an EventSource cannot be
	// evaluated, for instance because its Lua script does not compile.
	EventSourceInvalid = "EventSourceInvalid"
)

// Reasons for events about classifiers.
const (
	// ClassifierFailed is used, with a Warning event, when a Classifier or a
	// ManagementClusterClassifier could not be evaluated or could not label a cluster.
	ClassifierFailed = "ClassifierFailed"
	// ClassifierLabelConflict is used, with a Warning event, when more than one classifier
	// wants to manage the same label key on a cluster.
	ClassifierLabelConflict = "ClassifierLabelConflict"
)

// Reasons for events about managed clusters and licensing.
const (
	// ClusterUnreachable is used, with a Warning event, when the management cluster cannot
	// connect to a managed cluster.
	ClusterUnreachable = "ClusterUnreachable"
	// KubeconfigInvalid is used, with a Warning event, when the kubeconfig used to reach a
	// managed cluster is missing, malformed, or expired.
	KubeconfigInvalid = "KubeconfigInvalid"
	// LicenseInvalid is used, with a Warning event, when a feature needs a license and the
	// license is missing, invalid, or expired.
	LicenseInvalid = "LicenseInvalid"
	// ClusterRecovered is used, with a Normal event, when a cluster that was reported as
	// unreachable (ClusterUnreachable) or with an unhealthy agent (AgentUnhealthy) is healthy again.
	ClusterRecovered = "ClusterRecovered"
)

// Reasons for events about access and role requests.
const (
	// AccessRequestFailed is used, with a Warning event, when an AccessRequest could not be
	// fulfilled.
	AccessRequestFailed = "AccessRequestFailed"
	// RoleRequestFailed is used, with a Warning event, when a RoleRequest could not be
	// fulfilled.
	RoleRequestFailed = "RoleRequestFailed"
)

// Reasons for events about ClusterPromotion.
const (
	// StageStarted is used, with a Normal event, when a stage of a promotion started.
	StageStarted = "StageStarted"
	// PromotionCompleted is used, with a Normal event, when all the stages of a promotion
	// completed.
	PromotionCompleted = "PromotionCompleted"
	// StageFailed is used, with a Warning event, when a stage of a promotion failed.
	StageFailed = "StageFailed"
	// ApprovalRequired is used, with a Warning event, when a stage is waiting for a manual
	// approval.
	ApprovalRequired = "ApprovalRequired"
)

// Actions describe what the component was doing when the event was recorded. They are
// passed as the action argument of events.EventRecorder.Eventf. The object the event is
// about is the regarding argument, so an action is never a kind.
const (
	ActionDeploying   = "Deploying"
	ActionRemoving    = "Removing"
	ActionEvaluating  = "Evaluating"
	ActionReconciling = "Reconciling"
	ActionPromoting   = "Promoting"
	ActionConnecting  = "Connecting"
	ActionNotifying   = "Notifying"
)

const (
	// NoteMaxLength matches the Kubernetes API server's validation limit on the note of an
	// events.k8s.io/v1 Event, in bytes. An Event over this length is rejected outright by
	// the API server, it is not truncated.
	NoteMaxLength = 1024

	truncatedSuffix = "...(truncated)"
)

// TruncateNote caps note to NoteMaxLength bytes, so that an oversized message is recorded,
// with the cut called out, instead of being dropped by the API server. The result is
// always valid UTF-8.
func TruncateNote(note string) string {
	if len(note) <= NoteMaxLength {
		return note
	}

	truncated := note[:NoteMaxLength-len(truncatedSuffix)]
	// A byte index can land in the middle of a multi-byte character. Back up to the last
	// complete one, so the result never contains invalid UTF-8.
	for truncated != "" && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}

	return truncated + truncatedSuffix
}
