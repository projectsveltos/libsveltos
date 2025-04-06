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

import (
	"fmt"
	"strings"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

// Returns the name of the ConfigMap that holds the names of resources
// (EventSource, HealthCheck, Reloader) intended for a specific managed cluster.
//
// The naming convention for this ConfigMap is consistent across Sveltos
// components running in the management cluster.
func GetConfigMapName(clusterName string, clusterType libsveltosv1beta1.ClusterType) string {
	return fmt.Sprintf("%s-%s", clusterName, clusterType)
}

const (
	eventSourcePrefix = "eventsource-" // eventSourcePrefix is the prefix used for keys related to EventSource resources in ConfigMaps.
	healthCheckPrefix = "healthcheck-" // healthCheckPrefix is the prefix used for keys related to HealthCheck resources in ConfigMaps.
	reloaderPrefix    = "reloader-"    // reloaderPrefix is the prefix used for keys related to Reloader resources in ConfigMaps.
)

// IsEventSourceEntry checks if a given string `k` starts with the `eventSourcePrefix`.
// This is used to identify entries in ConfigMaps that refer to EventSource resources.
func IsEventSourceEntry(k string) bool {
	return strings.HasPrefix(k, eventSourcePrefix)
}

// IsHealthCheckEntry checks if a given string `k` starts with the `healthCheckPrefix`.
// This is used to identify entries in ConfigMaps that refer to HealthCheck resources.
func IsHealthCheckEntry(k string) bool {
	return strings.HasPrefix(k, healthCheckPrefix)
}

// IsReloaderEntry checks if a given string `k` starts with the `reloaderPrefix`.
// This is used to identify entries in ConfigMaps that refer to Reloader resources.
func IsReloaderEntry(k string) bool {
	return strings.HasPrefix(k, reloaderPrefix)
}

// GetKeyForEventSource generates a key for an EventSource resource by prepending
// the `eventSourcePrefix` to the provided `eventSourceName`. This key is typically
// used when storing or retrieving EventSource information in ConfigMaps.
func GetKeyForEventSource(eventTriggerName, eventSourceName string) string {
	return eventSourcePrefix + eventTriggerName + "-" + eventSourceName
}

// GetKeyForHealthCheck generates a key for a HealthCheck resource by prepending
// the `healthCheckPrefix` to the provided `eventSourceName`. This key is typically
// used when storing or retrieving HealthCheck information in ConfigMaps.
func GetKeyForHealthCheck(clusterHealthCheckName, healthcheckName string) string {
	return healthCheckPrefix + clusterHealthCheckName + "-" + healthcheckName
}

// GetKeyForReloader generates a key for a Reloader resource by prepending
// the `reloaderPrefix` to the provided `reloaderName`. This key is typically
// used when storing or retrieving Reloader information in ConfigMaps.
func GetKeyForReloader(reloaderName string) string {
	return reloaderPrefix + reloaderName
}
