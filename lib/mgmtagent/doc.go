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

package mgmtagent

// Package mgmtagent provides utility functions and types specifically designed
// for Sveltos agents running within the management cluster.

// In this deployment model, EventSource, HealthCheck, and Reloader instances
// are not propagated to managed clusters. Instead, the sveltos-agent running
// in the management cluster is responsible for monitoring these resources.
// However, not all such resources are relevant to the agent's specific managed
// cluster.
//
// To address this, other Sveltos components within the management cluster
// (such as the event manager and healthcheck manager) maintain per-cluster
// ConfigMaps. These ConfigMaps contain the names of the EventSource,
// HealthCheck, and Reloader instances that the sveltos-agent for a particular
// managed cluster needs to process.
