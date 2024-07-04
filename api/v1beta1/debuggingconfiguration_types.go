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
	DebuggingConfigurationKind = "DebuggingConfiguration"
)

// +kubebuilder:validation:Enum:=LogLevelNotSet;LogLevelInfo;LogLevelDebug;LogLevelVerbose
type LogLevel string

const (
	// LogLevelNotSet indicates log severity is not set. Default configuration will apply.
	LogLevelNotSet = LogLevel("LogLevelNotSet")

	// LogLevelInfo indicates log severity info (V(0)) is set
	LogLevelInfo = LogLevel("LogLevelInfo")

	// LogLevelDebug indicates log severity debug (V(5)) is set
	LogLevelDebug = LogLevel("LogLevelDebug")

	// LogLevelVerbose indicates log severity debug (V(10)) is set
	LogLevelVerbose = LogLevel("LogLevelVerbose")
)

//nolint:lll // kubebuilder marker
// +kubebuilder:validation:Enum:=AddonManager;Classifier;ClassifierAgent;SveltosClusterManager;DriftDetectionManager;AccessManager;HealthCheckManager;EventManager;ShardController;UIBackend

type Component string

const (
	// ComponentAddonManager is the addon-manager pod
	ComponentAddonManager = Component("AddonManager")

	// Classifier is the classifier pod
	ComponentClassifier = Component("Classifier")

	// ClassifierAgent is the classifier agent pod
	ComponentClassifierAgent = Component("ClassifierAgent")

	// ComponentSveltosClusterManager is the sveltoscluster-manager pod
	ComponentSveltosClusterManager = Component("SveltosClusterManager")

	// ComponentDriftDetectionManager is the drift-detection-manager pod
	ComponentDriftDetectionManager = Component("DriftDetectionManager")

	// ComponentAccessManager is the access-manager pod
	ComponentAccessManager = Component("AccessManager")

	// ComponentHealthCheckManager is the healthcheck-manager pod
	ComponentHealthCheckManager = Component("HealthCheckManager")

	// ComponentEventManager is the event-manager pod
	ComponentEventManager = Component("EventManager")

	// ComponentShardController is the shard-controller pod
	ComponentShardController = Component("ShardController")

	// ComponentUIBackend is the ui backend pod
	ComponentUIBackend = Component("UIBaeckend")
)

// ComponentConfiguration is the debugging configuration to be applied to a Sveltos component.
type ComponentConfiguration struct {
	// Component indicates which Sveltos component the configuration applies to.
	Component Component `json:"component"`

	// LogLevel is the log severity above which logs are sent to the stdout. [Default: Info]
	LogLevel LogLevel `json:"logLevel,omitempty"`
}

// DebuggingConfigurationSpec defines the desired state of DebuggingConfiguration
type DebuggingConfigurationSpec struct {
	// Configuration contains debugging configuration as granular as per component.
	// +listType=atomic
	// +optional
	Configuration []ComponentConfiguration `json:"configuration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=debuggingconfigurations,scope=Cluster
//+kubebuilder:storageversion

// DebuggingConfiguration is the Schema for the debuggingconfigurations API
type DebuggingConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DebuggingConfigurationSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// DebuggingConfigurationList contains a list of DebuggingConfiguration
type DebuggingConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DebuggingConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DebuggingConfiguration{}, &DebuggingConfigurationList{})
}
