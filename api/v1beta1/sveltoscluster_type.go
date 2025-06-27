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
	SveltosClusterKind = "SveltosCluster"
)

type ActiveWindow struct {
	// From in Cron format, see https://en.wikipedia.org/wiki/Cron.
	// Indicates when to un-pause the cluster (cluster in paused state receives no update from sveltos).
	// +kubebuilder:validation:MinLength=1
	From string `json:"from"`

	// To in Cron format, see https://en.wikipedia.org/Cron.
	// Indicates when to pause the cluster (cluster in paused state receives no update from sveltos).
	// +kubebuilder:validation:MinLength=1
	To string `json:"to"`
}

// ConnectionStatus specifies whether connecting to managed cluster is healthy or not
// +kubebuilder:validation:Enum:=Healthy;Down
type ConnectionStatus string

const (
	// ConnectionHealthy indicates connection from management cluster to managed cluster is healthy
	ConnectionHealthy = ConnectionStatus("Healthy")

	// ConnectionDown indicates connection from management cluster to managed cluster is down
	ConnectionDown = ConnectionStatus("Down")
)

type ClusterCheck struct {
	// Name of the cluster check.
	// Must be a DNS_LABEL and unique within the ClusterChecks.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// ResourceSelectors identifies what Kubernetes resources to select
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

	// This field is  used to specify a Lua function that will be used to evaluate
	// this check.
	// The function will receive the array of resources selected by ResourceSelectors.
	// The Lua function name is evaluate and must return a struct with:
	// - "result" field: boolean indicating whether check passed or failed;
	// - "message" field: (optional) message.
	Condition string `json:"condition"`
}

type TokenRequestRenewalOption struct {
	// RenewTokenRequestInterval is the interval at which to renew the TokenRequest
	RenewTokenRequestInterval metav1.Duration `json:"renewTokenRequestInterval"`

	// TokenDuration is the duration the requested token will be valid for.
	// If not specified, the value of RenewTokenRequestInterval will be used.
	// This allows the token to remain valid beyond the renewal interval,
	// providing a buffer in case of connectivity loss.
	// Example: renew every hour, token lasts 3 hours (buffer for disconnection)
	//   renewTokenRequestInterval: 1h
	//   tokenDuration: 3h
	// +optional
	TokenDuration metav1.Duration `json:"tokenDuration,omitempty"`

	// SANamespace is the namespace of the ServiceAccount to renew the token for.
	// If specified, ServiceAccount must exist in the managed cluster.
	// If not specified, sveltos will try to deduce it from current kubeconfig
	// +optional
	SANamespace string `json:"saNamespace,omitempty"`

	// SAName is name of the ServiceAccount to renew the token for.
	// If specified, ServiceAccount must exist in the managed cluster.
	// If not specified, sveltos will try to deduce it from current kubeconfig
	// +optional
	SAName string `json:"saName,omitempty"`
}

// SveltosClusterSpec defines the desired state of SveltosCluster
type SveltosClusterSpec struct {
	// KubeconfigName allows overriding the default Sveltos convention which expected a valid kubeconfig
	// to be hosted in a secret with the pattern ${sveltosClusterName}-sveltos-kubeconfig.
	//
	// When a value is specified, the referenced Kubernetes Secret object must exist,
	// and will be used to connect to the Kubernetes cluster.
	// +optional
	KubeconfigName string `json:"kubeconfigName,omitempty"`

	// KubeconfigKeyName specifies the key within the Secret that holds the kubeconfig.
	// If not specified, Sveltos will use first key in the Secret.
	// +optional
	KubeconfigKeyName string `json:"kubeconfigKeyName,omitempty"`

	// Paused can be used to prevent controllers from processing the
	// SveltosCluster and all its associated objects.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// TokenRequestRenewalOption contains options describing how to renew TokenRequest
	// +optional
	TokenRequestRenewalOption *TokenRequestRenewalOption `json:"tokenRequestRenewalOption,omitempty"`

	// ArbitraryData allows for arbitrary nested structures
	// +optional
	ArbitraryData map[string]string `json:"data,omitempty"`

	// ActiveWindow is an optional field for automatically pausing and unpausing
	// the cluster.
	// If not specified, the cluster will not be paused or unpaused automatically.
	ActiveWindow *ActiveWindow `json:"activeWindow,omitempty"`

	// ConsecutiveFailureThreshold is the maximum number of consecutive connection
	// failures before setting the problem status in Status.ConnectionStatus
	// +kubebuilder:default:=3
	// +optional
	ConsecutiveFailureThreshold int `json:"consecutiveFailureThreshold,omitempty"`

	// ReadinessChecks is an optional list of custom checks to verify cluster
	// readiness
	// +optional
	ReadinessChecks []ClusterCheck `json:"readinessChecks,omitempty"`

	// LivenessChecks is an optional list of custom checks to verify cluster
	// is healthy
	// +optional
	LivenessChecks []ClusterCheck `json:"livenessChecks,omitempty"`

	// PullMode indicates whether the cluster is in pull mode.
	// If true, the agent in the managed cluster will fetch the configuration.
	// If false (default), the management cluster will push the configuration.
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:default:=false
	// +optional
	PullMode bool `json:"pullMode,omitempty"`
}

// SveltosClusterStatus defines the status of SveltosCluster
type SveltosClusterStatus struct {
	// The Kubernetes version of the cluster.
	// +optional
	Version string `json:"version,omitempty"`

	// Ready is the state of the cluster.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ConnectionStatus indicates whether connection from the management cluster
	// to the managed cluster is healthy
	// +optional
	ConnectionStatus ConnectionStatus `json:"connectionStatus,omitempty"`

	// FailureMessage is a human consumable message explaining the
	// misconfiguration
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// LastReconciledTokenRequestAt is the last time the TokenRequest
	// was renewed.
	// +optional
	LastReconciledTokenRequestAt string `json:"lastReconciledTokenRequestAt,omitempty"`

	// AgentLastReportTime indicates the last time the Sveltos agent in the managed cluster
	// successfully reported its status to the management cluster's API server.
	// This field is updated exclusively when Sveltos operates in pull mode,
	// serving as a heartbeat from the agent's perspective.
	// +optional
	AgentLastReportTime *metav1.Time `json:"agentLastReportTime,omitempty"`

	// Information when next unpause cluster is scheduled
	// +optional
	NextUnpause *metav1.Time `json:"nextUnpause,omitempty"`

	// Information when next pause cluster is scheduled
	// +optional
	NextPause *metav1.Time `json:"nextPause,omitempty"`

	// connectionFailures is the number of consecutive failed attempts to connect
	// to the remote cluster.
	// +kubebuilder:default:=0
	// +optional
	ConnectionFailures int `json:"connectionFailures,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=sveltosclusters,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Indicates whether cluster is ready to be managed by sveltos"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Kubernetes version associated with this Cluster"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of SveltosCluster"
//+kubebuilder:storageversion

// SveltosCluster is the Schema for the SveltosCluster API
type SveltosCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SveltosClusterSpec   `json:"spec,omitempty"`
	Status SveltosClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SveltosClusterList contains a list of SveltosCluster
type SveltosClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SveltosCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SveltosCluster{}, &SveltosClusterList{})
}
