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

package v1alpha1

// ReferencedResourceKind is a string representation of allowed kind of resources
// that can be referenced in a ClusterProfile
type ReferencedResourceKind string

// Define the ReferencedResourceKind constants.
const (
	SecretReferencedResourceKind    ReferencedResourceKind = "Secret"
	ConfigMapReferencedResourceKind ReferencedResourceKind = "ConfigMap"
)

type Selector string