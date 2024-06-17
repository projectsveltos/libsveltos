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
	"crypto/sha256"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReloaderReportKind = "ReloaderReport"

	// ReloaderReportFinalizer allows ReloaderReportReconciler to clean up resources associated with
	// ReloaderReport before removing it from the apiserver.
	ReloaderReportFinalizer = "reloaderreport.finalizer.projectsveltos.io"

	// ReloaderReportClusterNameLabel is added to each ReloaderReport
	ReloaderReportClusterNameLabel = "reloaderreport.projectsveltos.io/cluster-name"

	// ReloaderReportClusterTypeLabel is added to each ReloaderReport
	ReloaderReportClusterTypeLabel = "reloaderreport.projectsveltos.io/cluster-type"

	// ReloaderReportResourceKindAnnotation is added to each ReloaderReport
	ReloaderReportResourceKindAnnotation = "reloaderreport.projectsveltos.io/resource-kind"

	// ReloaderReportResourceNamespaceAnnotation is added to each ReloaderReport
	ReloaderReportResourceNamespaceAnnotation = "reloaderreport.projectsveltos.io/resource-namespace"

	// ReloaderReportResourceNameAnnotation is added to each ReloaderReport
	ReloaderReportResourceNameAnnotation = "reloaderreport.projectsveltos.io/resource-name"
)

// mountedResourcekind is the kind of the resource being mounted as volume (either ConfigMap or Secret)
// mountedResourceNamespace/mountedResourceName is the namespace/name of the resource being mounted as volume
// clusterName and clusterType identify the managed cluster
func GetReloaderReportName(mountedResourcekind, mountedResourceNamespace, mountedResourceName, clusterName string,
	clusterType *ClusterType) string {

	h := sha256.New()
	fmt.Fprintf(h, "%s--%s--%s--%s--%s", mountedResourcekind, mountedResourceNamespace, mountedResourceName,
		clusterName, string(*clusterType))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash)
}

func GetReloaderReportLabels(clusterName string, clusterType *ClusterType) map[string]string {
	return map[string]string{
		ReloaderReportClusterNameLabel: clusterName,
		ReloaderReportClusterTypeLabel: strings.ToLower(string(*clusterType)),
	}
}

// GetReloaderReportAnnotations returns the annotation to add to ReloaderReport
// kind, namespace, name identify mounted resource (ConfigMap or Secret) which was modified
// causing a reload
func GetReloaderReportAnnotations(kind, namespace, name string) map[string]string {
	return map[string]string{
		ReloaderReportResourceKindAnnotation:      strings.ToLower(kind),
		ReloaderReportResourceNamespaceAnnotation: namespace,
		ReloaderReportResourceNameAnnotation:      name,
	}
}

type ReloaderReportSpec struct {
	// ClusterNamespace is the namespace of the Cluster this
	// ReloaderReport is for.
	ClusterNamespace string `json:"clusterNamespace"`

	// ClusterName is the name of the Cluster this ReloaderReport
	// is for.
	ClusterName string `json:"clusterName"`

	// ClusterType is the type of Cluster this ReloaderReport
	// is for.
	ClusterType ClusterType `json:"clusterType"`

	// ResourcesToReload contains a list of resources that requires
	// rolling upgrade
	// +optional
	ResourcesToReload []ReloaderInfo `json:"resourcesToReload,omitempty"`
}

// ReloaderReportStatus defines the observed state of ReloaderReport
type ReloaderReportStatus struct {
	// Phase represents the current phase of report.
	// +optional
	Phase *ReportPhase `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=reloaderreports,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// ReloaderReport is the Schema for the ReloaderReport API
type ReloaderReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReloaderReportSpec   `json:"spec,omitempty"`
	Status ReloaderReportStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ReloaderReportList contains a list of ReloaderReport
type ReloaderReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReloaderReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReloaderReport{}, &ReloaderReportList{})
}
