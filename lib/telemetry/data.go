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

package telemetry

// Cluster represents the telemetry data Sveltos collects from each
// management cluster
type Cluster struct {
	UUID                   string `json:"uuid"`
	SveltosVersion         string `json:"sveltosVersion"`
	ManagedSveltosClusters int    `json:"sveltosClusters"`
	ReadySveltosClusters   int    `json:"readySveltosClusters"`
	ManagedCAPIClusters    int    `json:"capiClusters"`
	ClusterProfiles        int    `json:"clusterProfiles"`
	Profiles               int    `json:"profiles"`
	ClusterSummaries       int    `json:"clusterSummaries"`
}
