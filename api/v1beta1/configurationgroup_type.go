/*
Copyright 2025. projectsveltos.io. All rights reserved.

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
	ConfigurationGroupKind = "ConfigurationGroup"
)

type ConfigurationItem struct {
	// ContentRef references the Kubernetes resource containing
	// the content to deploy.
	// +optional
	ContentRef *corev1.ObjectReference `json:"contentRef,omitempty"`

	// Hash represents of a unique value for the content stored in
	// the referenced contentRef
	// +optional
	Hash []byte `json:"hash,omitempty"`
}

// +kubebuilder:validation:Enum:=Deploy;Remove
type Action string

const (
	// ActionDeploy indicates to deploy the content referenced by the ConfigurationGroup
	ActionDeploy = Action("Deploy")

	// ActionRemove indicates to withdraw the content deployed because of the ConfigurationGroup
	ActionRemove = Action("Remove")
)

// +kubebuilder:validation:Enum:=Ready;Preparing
type UpdatePhase string

const (
	// UpdatePhaseReady indicates the ConfigurationGroup is ready for deployment
	UpdatePhaseReady = UpdatePhase("Ready")

	// UpdatePhasePreparing indicates a new version of the ConfigurationGroup is being prepared
	UpdatePhasePreparing = UpdatePhase("Preparing")
)

type ConfigurationGroupSpec struct {
	// +kubebuilder:default:=Deploy
	Action Action `json:"action,omitempty"`

	// SourceRef is the user facing Sveltos resource that caused this ConfigurationGroup to be
	// created. For instance, when a ClusterSummary creates a ConfigurationGroup, the user
	// facing resource is either a ClusterProfile or a Profile.
	// +optional
	SourceRef *corev1.ObjectReference `json:"sourceRef,omitempty"`

	// ConfigurationItems represents a list of configurations to deploy
	// +optional
	ConfigurationItems []ConfigurationItem `json:"configurationItem,omitempty"`

	// Tier controls the order of deployment for ClusterProfile or Profile resources targeting
	// the same cluster resources.
	// Imagine two configurations (ClusterProfiles or Profiles) trying to deploy the same resource (a Kubernetes
	// resource or an helm chart). By default, the first one to reach the cluster "wins" and deploys it.
	// Tier allows you to override this. When conflicts arise, the ClusterProfile or Profile with the **lowest**
	// Tier value takes priority and deploys the resource.
	// Higher Tier values represent lower priority. The default Tier value is 100.
	// Using Tiers provides finer control over resource deployment within your cluster, particularly useful
	// when multiple configurations manage the same resources.
	// +kubebuilder:default:=100
	// +kubebuilder:validation:Minimum=1
	// +optional
	Tier int32 `json:"tier,omitempty"`

	// DryRun means no change will be propagated to matching cluster. A report
	// instead will be generated summarizing what would happen in any matching cluster
	// because of the changes made by this ConfigurationGroup
	// +kubebuilder:default:=false
	// +optional
	DryRun bool `json:"dryRun,omitempty"`

	// Reloader indicates whether Deployment/StatefulSet/DaemonSet instances deployed
	// by Sveltos and part of this ConfigurationGroup need to be restarted via rolling upgrade
	// when a ConfigMap/Secret instance mounted as volume is modified.
	// When set to true, when any mounted ConfigMap/Secret is modified, Sveltos automatically
	// starts a rolling upgrade for Deployment/StatefulSet/DaemonSet instances mounting it.
	// +kubebuilder:default:=false
	// +optional
	Reloader bool `json:"reloader,omitempty"`

	// DriftDetection indicates Sveltos must monitors deployed for resources configuration
	// drift. If drift is detected, a reconciliation is triggered to ensure the managed
	// cluster's configuration aligns with the expected configuration defined in the management
	// cluster
	// +kubebuilder:default:=false
	// +optional
	DriftDetection bool `json:"driftDetection,omitempty"`

	// DriftExclusions is a list of configuration drift exclusions to be applied when syncMode is
	// set to ContinuousWithDriftDetection. Each exclusion specifies JSON6902 paths to ignore
	// when evaluating drift, optionally targeting specific resources and features.
	// +listType=atomic
	// +optional
	DriftExclusions []DriftExclusion `json:"driftExclusions,omitempty"`

	// By default (when ContinueOnConflict is unset or set to false), Sveltos stops deployment after
	// encountering the first conflict (e.g., another ClusterProfile already deployed the resource).
	// If set to true, Sveltos will attempt to deploy remaining resources in the ClusterProfile even
	// if conflicts are detected for previous resources.
	// +kubebuilder:default:=false
	// +optional
	ContinueOnConflict bool `json:"continueOnConflict,omitempty"`

	// By default (when ContinueOnError is unset or set to false), Sveltos stops deployment after
	// encountering the first error.
	// If set to true, Sveltos will attempt to deploy remaining resources in the ClusterProfile even
	// if errors are detected for previous resources.
	// +kubebuilder:default:=false
	// +optional
	ContinueOnError bool `json:"continueOnError,omitempty"`

	// The maximum number of consecutive deployment failures that Sveltos will permit.
	// After this many consecutive failures, the deployment will be considered failed, and Sveltos will stop retrying.
	// This setting applies only to feature deployments, not resource removal.
	// This field is optional. If not set, Sveltos default behavior is to keep retrying.
	// +optional
	MaxConsecutiveFailures *uint `json:"maxConsecutiveFailures,omitempty"`

	// Indicates whether deployed resources must be left on the managed cluster.
	// +kubebuilder:default:=false
	// +optional
	LeavePolicies bool `json:"leavePolicies,omitempty"`

	// ValidateHealths is a slice of Lua functions to run against
	// the managed cluster to validate the state of those add-ons/applications
	// is healthy
	// +listType=atomic
	// +optional
	ValidateHealths []ValidateHealth `json:"validateHealths,omitempty"`

	// DeployedGroupVersionKind contains all GroupVersionKinds deployed in either
	// the workload cluster or the management cluster because of this feature.
	// Each element has format kind.version.group
	// +optional
	DeployedGroupVersionKind []string `json:"deployedGroupVersionKind,omitempty"`

	// UpdatePhase indicates the current phase of configuration updates. When set to "Preparing",
	// it signals that a new version of this ConfigurationGroup is being prepared, forcing
	// metadata.generation to advance. This allows detection of differences between
	// status.observedGeneration and metadata.generation when agents process an older version,
	// enabling tracking of update propagation across managed clusters.
	// +kubebuilder:default:=Ready
	// +optional
	UpdatePhase UpdatePhase `json:"updatePhase,omitempty"`

	// RequestorHash represents a hash of the state of the requestor that created this ConfigurationGroup.
	// This field is optional and can be used to determine when the creating resource's state has
	// changed and the ConfigurationGroup needs to be updated accordingly.
	// +optional
	RequestorHash []byte `json:"requestorHash,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to impersonate when applying
	// the configuration. If empty, the default ServiceAccount for the Sveltos-applier
	// will be used.
	// The ServiceAccount must exist in the managed cluster.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// ServiceAccountNamespace is the namespace of the ServiceAccount to impersonate when applying
	// the configuration. If empty, the default namespace for the Sveltos-applier
	// will be used (typically the same namespace where the Sveltos-applier is deployed).
	// The ServiceAccount must exist in the managed cluster.
	// +optional
	ServiceAccountNamespace string `json:"serviceAccountNamespace,omitempty"`
}

type ConfigurationGroupStatus struct {
	// DeployedGroupVersionKind contains all GroupVersionKinds deployed because of
	// the ConfigurationGroup.
	// Each element has format kind.version.group
	// +optional
	DeployedGroupVersionKind []string `json:"deployedGroupVersionKind,omitempty"`

	// Status represents the state of the feature in the workload cluster
	// +optional
	DeploymentStatus *FeatureStatus `json:"status,omitempty"`

	// FailureMessage provides more information about the error.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// LastAppliedTime is the time feature was last reconciled
	// +optional
	LastAppliedTime *metav1.Time `json:"lastAppliedTime,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed ConfigurationGroup.
	// When this value matches the ConfigurationGroup's metadata.generation, it indicates that the
	// status reflects the latest desired specification. If observedGeneration is less than generation,
	// it means the controller has not yet processed the latest changes to the specification.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ObservedRequestorHash reflects the requestor hash that was last processed by the controller.
	// When this value matches the spec.requestorHash, it indicates that the controller has
	// processed the latest state of the creating resource. If observedRequestorHash differs
	// from spec.requestorHash, it means the requestor has changed and reconciliation is needed.
	// +optional
	ObservedRequestorHash []byte `json:"observedRequestorHash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=configurationgroups,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// ConfigurationGroup is the Schema for the configurationgroup API
type ConfigurationGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationGroupSpec   `json:"spec,omitempty"`
	Status ConfigurationGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigurationGroupList contains a list of ConfigurationGroup
type ConfigurationGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigurationGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigurationGroup{}, &ConfigurationGroupList{})
}
