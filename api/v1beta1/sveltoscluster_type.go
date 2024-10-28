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

type TokenRequestRenewalOption struct {
	// RenewTokenRequestInterval is the interval at which to renew the TokenRequest
	RenewTokenRequestInterval metav1.Duration `json:"renewTokenRequestInterval"`

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

	// Information when next unpause cluster is scheduled
	// +optional
	NextUnpause *metav1.Time `json:"nextUnpause,omitempty"`

	// Information when next pause cluster is scheduled
	// +optional
	NextPause *metav1.Time `json:"nextPause,omitempty"`

	// connectionFailures is the number of consecutive failed attempts to connect
	// to the remote cluster.
	// +optional
	ConnectionFailures int `json:"connectionFailures,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=sveltosclusters,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Indicates whether cluster is ready to be managed by sveltos"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Kubernetes version associated with this Cluster"
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
