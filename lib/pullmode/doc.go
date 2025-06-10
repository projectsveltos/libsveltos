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

package pullmode

// Package pullmode provides functions to manage the deployment of resources to SveltosClusters
// in pull mode. It offers mechanisms to register resources for immediate deployment
// (RecordResourcesForDeployment) and to stage resources for later deployment
// (StageResourcesForDeployment, CommitStagedResourcesForDeployment, DiscardStagedResourcesForDeployment).
//
// Overview:
//
// In pull mode, an agent running on managed clusters fetch resources for deployment from
// the management cluster. This package provides the necessary functions for management
// cluster components to register these resources.
//
// Resource Registration Methods:
//
// - RecordResourcesForDeployment: Registers resources that should be deployed immediately.
//   When this method is called, the resources are made available for the agent to fetch.
//
// - StageResourcesForDeployment: Registers resources that are being prepared for deployment.
//   Unlike RecordResourcesForDeployment, this method does not immediately make the resources
//   available. It's intended for scenarios where a component needs to prepare resources
//   over multiple steps.
//
// Staging Workflow:
//
// The typical workflow for staging resources involves:
//
// 1. Calling StageResourcesForDeployment one or more times to register the  resources as they
// 	  are prepared.
//
// 2. Calling CommitStagedResourcesForDeployment when all resources are ready. This makes
//    the staged resources available for the agent to fetch.
//
// 3. (Optional) Calling DiscardStagedResourcesForDeployment if the resource preparation
//    process fails and the staged resources should be discarded.
//
// Common Parameters:
//
// All methods in this package that interact with resources for a specific cluster and component
// use the following common parameters:
//
// - ctx context.Context: For managing the request lifecycle.
// - c client.Client: Kubernetes client to interact with the management cluster.
// - clusterNamespace string: Namespace of the target SveltosCluster.
// - clusterName string: Name of the target SveltosCluster.
// - requestorKind string: Kind of the component in the management cluster making the request (e.g., ClusterSummary).
// - requestorName string: Name of the component in the management cluster making the request.
// - requestorFeature string: Specific feature within the component making the request (can be empty or indicate the
//   type of feature like helmcharts vs policyrefs vs kustomizationrefs)
//
// Setters for Resource Deployment:
// Sveltos offers different modes when deploying resources:
//
// - DriftDetection: Instructs Sveltos to actively monitor for configuration drift in deployed resources.
//
// - DryRun:  Prevents actual deployment changes. Instead, Sveltos generates a report detailing the
// changes that would occur on the managed clusters.
//
// - Reloader:  Indicates whether Sveltos should automatically restart Deployments, StatefulSets, or DaemonSets
// via a rolling upgrade when their mounted ConfigMaps or Secrets are modified. Setting this to true ensures
// that any change in a mounted ConfigMap or Secret prompts an automatic rolling upgrade.
//
//
// Utility Functions:
//
// - GetClusterLabels: This method is made available to the agent running in the managed
//   cluster. When the agent starts a watcher to look for ConfigurationGroups intended
//   for its cluster, it uses the labels returned by this function to filter the resources.
//   It takes the cluster namespace and cluster name as input and returns a map of labels
//   used to identify ConfigurationGroups relevant to that specific cluster.
//
