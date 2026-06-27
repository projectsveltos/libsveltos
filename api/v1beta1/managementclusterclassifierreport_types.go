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

package v1beta1

import (
	"crypto/sha256"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	// ManagementClusterClassifierNameLabel is added to each ManagementClusterClassifierReport
	// to identify the ManagementClusterClassifier that created it.
	// `:` is not a valid character in Kubernetes resource names, so this prefix cannot
	// collide with any Classifier name when both are registered in the keymanager.
	ManagementClusterClassifierNameLabel = "projectsveltos.io/managementclusterclassifier-name"

	ManagementClusterClassifierReportKind = "ManagementClusterClassifierReport"
)

// GetManagementClusterClassifierReportName returns a deterministic, fixed-length name
// for a ManagementClusterClassifierReport derived from a SHA-256 hash of the three
// identifying components. The name is opaque; labels carry the human-readable context.
func GetManagementClusterClassifierReportName(classifierName, clusterName string, clusterType *ClusterType) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s:%s:%s", strings.ToLower(string(*clusterType)), classifierName, clusterName)
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// GetManagementClusterClassifierReportLabels returns the labels applied to a
// ManagementClusterClassifierReport. The three labels together allow efficient
// queries in both directions: all reports for a given classifier, and all reports
// targeting a given cluster.
func GetManagementClusterClassifierReportLabels(classifierName, clusterName string, clusterType *ClusterType) map[string]string {
	return map[string]string{
		ManagementClusterClassifierNameLabel: classifierName,
		ClassifierReportClusterNameLabel:     clusterName,
		ClassifierReportClusterTypeLabel:     strings.ToLower(string(*clusterType)),
	}
}

// ManagementClusterClassifierReportSpec identifies the classifier and cluster this
// report belongs to.
type ManagementClusterClassifierReportSpec struct {
	// ClassifierName is the name of the ManagementClusterClassifier this report is for.
	ClassifierName string `json:"classifierName"`

	// ClusterNamespace is the namespace of the cluster this report is for.
	ClusterNamespace string `json:"clusterNamespace"`

	// ClusterName is the name of the cluster this report is for.
	ClusterName string `json:"clusterName"`

	// ClusterType is the type of the cluster this report is for.
	ClusterType ClusterType `json:"clusterType"`
}

// ManagementClusterClassifierReportStatus is written by the controller after processing.
type ManagementClusterClassifierReportStatus struct {
	// ManagedLabels lists the label keys this ManagementClusterClassifier is
	// actively managing on the cluster.
	// +optional
	ManagedLabels []string `json:"managedLabels,omitempty"`

	// UnManagedLabels lists label keys this ManagementClusterClassifier would like
	// to manage but cannot because another Classifier or ManagementClusterClassifier
	// already owns them.
	// +optional
	UnManagedLabels []UnManagedLabel `json:"unmanagedLabels,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=managementclusterclassifierreports,scope=Namespaced
//+kubebuilder:subresource:status

// ManagementClusterClassifierReport is created by the ManagementClusterClassifier
// controller — one per (ManagementClusterClassifier, cluster) pair — to track which
// labels are managed on each cluster and to enable conflict detection via the keymanager.
type ManagementClusterClassifierReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagementClusterClassifierReportSpec   `json:"spec,omitempty"`
	Status ManagementClusterClassifierReportStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagementClusterClassifierReportList contains a list of ManagementClusterClassifierReport.
type ManagementClusterClassifierReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagementClusterClassifierReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(GroupVersion,
			&ManagementClusterClassifierReport{},
			&ManagementClusterClassifierReportList{},
		)
		return nil
	})
}
