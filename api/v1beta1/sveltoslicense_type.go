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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SveltosLicenseKind = "SveltosLicense"
)

// SveltosLicenseStatus defines the observed state of SveltosLicense
type SveltosLicenseStatus struct {
	// Status indicates the overall state of the license.
	// Possible values:
	// - Valid: License secret found, valid signature, not expired.
	// - Expired: License secret found, valid signature, but expired.
	// - Invalid: License secret found, but signature invalid or data malformed.
	// - NotFound: No license secret found at the specified reference.
	// +kubebuilder:validation:Enum=Valid;Expired;Invalid;NotFound
	Status LicenseStatusType `json:"status,omitempty"`

	// Message provides a human-readable explanation of the current license status.
	Message string `json:"message,omitempty"`

	// ExpirationDate is the exact expiration timestamp from the license payload,
	// if a license was found and successfully parsed. This field will be present
	// even if the license is expired or invalid, as long as the date could be extracted.
	// +kubebuilder:validation:Format=date-time
	ExpirationDate *metav1.Time `json:"expirationDate,omitempty"`

	// Features is a list of feature strings enabled by this license.
	// +optional
	Features []string `json:"features,omitempty"`

	// MaxClusters is the maximum number of clusters allowed for this license.
	// +optional
	MaxClusters *int `json:"maxClusters,omitempty"`
}

// LicenseStatusType defines the type for the license status.
type LicenseStatusType string

const (
	// LicenseStatusValid indicates the license secret was found,
	// its signature is valid, and it has not expired.
	LicenseStatusValid LicenseStatusType = "Valid"
	// LicenseStatusExpired indicates the license secret was found,
	// its signature is valid, but it has expired.
	LicenseStatusExpired LicenseStatusType = "Expired"
	// LicenseStatusInvalid indicates the license secret was found,
	// but its signature is invalid or the data is malformed.
	LicenseStatusInvalid LicenseStatusType = "Invalid"
	// LicenseStatusNotFound indicates no license secret was found at the
	// specified reference.
	LicenseStatusNotFound LicenseStatusType = "NotFound"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=sveltoslicenses,scope=Cluster
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// SveltosLicense is the Schema for the clustersets API
type SveltosLicense struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status SveltosLicenseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SveltosLicenseList contains a list of SveltosLicense
type SveltosLicenseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SveltosLicense `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SveltosLicense{}, &SveltosLicenseList{})
}
