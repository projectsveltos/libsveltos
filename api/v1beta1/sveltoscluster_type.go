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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	SveltosClusterKind = "SveltosCluster"
)

const (
	ShardAnnotation = "sharding.projectsveltos.io/key"
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

	// KubeconfigKeyName is the name of the key in the Secret where the renewed
	// kubeconfig will be stored.
	// If nil, Sveltos defaults to "re-kubeconfig" and updates SveltosCluster.Spec.KubeconfigKeyName.
	// If set, Sveltos writes to this key. To prevent GitOps drift, set this to the
	// same value as SveltosCluster.Spec.KubeconfigKeyName.
	// +optional
	KubeconfigKeyName *string `json:"kubeconfigKeyName,omitempty"`
}

// WorkloadIdentityProvider identifies the cloud provider for workload identity.
// +kubebuilder:validation:Enum=AWS;GCP;Azure
type WorkloadIdentityProvider string

const (
	WorkloadIdentityProviderAWS   WorkloadIdentityProvider = "AWS"
	WorkloadIdentityProviderGCP   WorkloadIdentityProvider = "GCP"
	WorkloadIdentityProviderAzure WorkloadIdentityProvider = "Azure"
)

// AWSWorkloadIdentityConfig holds AWS-specific workload identity configuration.
type AWSWorkloadIdentityConfig struct {
	// ClusterName is the name of the EKS cluster. Required: it is embedded in
	// the bearer token header (x-k8s-aws-id) sent to the EKS API server.
	// +kubebuilder:validation:MinLength=1
	ClusterName string `json:"clusterName"`

	// RoleARN is the ARN of the IAM role Sveltos should assume before generating
	// the EKS bearer token. If empty, the pod's own IRSA credentials are used directly.
	// +optional
	RoleARN string `json:"roleARN,omitempty"`

	// Region is the AWS region of the EKS cluster. If empty, the AWS_REGION
	// environment variable injected by IRSA is used.
	// +optional
	Region string `json:"region,omitempty"`
}

// GCPWorkloadIdentityConfig holds GCP-specific workload identity configuration.
// Fields are kept for future auto-discovery support; the current implementation
// obtains tokens via Application Default Credentials without calling the GKE API.
type GCPWorkloadIdentityConfig struct {
	// ProjectID is the GCP project ID that contains the GKE cluster.
	// +kubebuilder:validation:MinLength=1
	ProjectID string `json:"projectID"`

	// ClusterName is the name of the GKE cluster.
	// +kubebuilder:validation:MinLength=1
	ClusterName string `json:"clusterName"`

	// Location is the GCP region or zone of the GKE cluster (e.g. "us-central1").
	// +kubebuilder:validation:MinLength=1
	Location string `json:"location"`
}

// AzureWorkloadIdentityConfig holds Azure-specific workload identity configuration.
type AzureWorkloadIdentityConfig struct {
	// TenantID is the Azure Active Directory tenant ID.
	// +kubebuilder:validation:MinLength=1
	TenantID string `json:"tenantID"`

	// ClientID is the client ID of the managed identity or app registration
	// federated with the management cluster service account.
	// +kubebuilder:validation:MinLength=1
	ClientID string `json:"clientID"`

	// SubscriptionID is the Azure subscription containing the AKS cluster.
	// Kept for future auto-discovery support.
	// +optional
	SubscriptionID string `json:"subscriptionID,omitempty"`

	// ResourceGroup is the Azure resource group containing the AKS cluster.
	// Kept for future auto-discovery support.
	// +optional
	ResourceGroup string `json:"resourceGroup,omitempty"`

	// ClusterName is the name of the AKS cluster.
	// Kept for future auto-discovery support.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
}

// WorkloadIdentityConfig specifies how Sveltos authenticates to the managed
// cluster using the cloud provider's workload identity mechanism instead of a
// static kubeconfig Secret.
// +kubebuilder:validation:XValidation:rule="(self.provider == 'AWS') == has(self.aws)",message="aws must be set if and only if provider is AWS"
// +kubebuilder:validation:XValidation:rule="(self.provider == 'GCP') == has(self.gcp)",message="gcp must be set if and only if provider is GCP"
// +kubebuilder:validation:XValidation:rule="(self.provider == 'Azure') == has(self.azure)",message="azure must be set if and only if provider is Azure"
type WorkloadIdentityConfig struct {
	// Provider is the cloud provider implementing the workload identity mechanism.
	// +kubebuilder:validation:Required
	Provider WorkloadIdentityProvider `json:"provider"`

	// Endpoint is the API server endpoint of the managed cluster (e.g. https://…).
	// +kubebuilder:validation:MinLength=1
	Endpoint string `json:"endpoint"`

	// CASecretRef references a Secret in the management cluster containing the
	// CA certificate of the managed cluster's API server under the key "ca.crt".
	// If not set, the system certificate pool is used.
	// +optional
	CASecretRef *corev1.LocalObjectReference `json:"caSecretRef,omitempty"`

	// AWS contains configuration specific to AWS IRSA / EKS Pod Identity.
	// Required when Provider is AWS.
	// +optional
	AWS *AWSWorkloadIdentityConfig `json:"aws,omitempty"`

	// GCP contains configuration specific to GCP Workload Identity Federation.
	// Required when Provider is GCP.
	// +optional
	GCP *GCPWorkloadIdentityConfig `json:"gcp,omitempty"`

	// Azure contains configuration specific to Azure Workload Identity.
	// Required when Provider is Azure.
	// +optional
	Azure *AzureWorkloadIdentityConfig `json:"azure,omitempty"`
}

// SveltosClusterSpec defines the desired state of SveltosCluster
// +kubebuilder:validation:XValidation:rule="!has(self.workloadIdentity) || !has(self.kubeconfigName)",message="workloadIdentity, kubeconfigName: conflict"
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

	// WorkloadIdentity configures authentication to the managed cluster via the
	// cloud provider's workload identity mechanism. When set, Sveltos does not
	// read a kubeconfig Secret; instead it obtains short-lived credentials from
	// the cloud provider at runtime.
	// Mutually exclusive with KubeconfigName/KubeconfigKeyName.
	// +optional
	WorkloadIdentity *WorkloadIdentityConfig `json:"workloadIdentity,omitempty"`
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

	// ActiveWindowHash is the hash of the SveltosCluster Spec.ActiveWindow.
	// This is used to detect if the ActiveWindow configuration has changed
	// and requires recalculation of NextPause and NextUnpause.
	// +optional
	ActiveWindowHash []byte `json:"activeWindowHash,omitempty"`

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
//+kubebuilder:printcolumn:name="Shard",type="string",JSONPath=".metadata.annotations['sharding\\.projectsveltos\\.io/key']",description="Cluster Shard"
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
	SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(GroupVersion,
			&SveltosCluster{},
			&SveltosClusterList{},
		)
		return nil
	})
}
