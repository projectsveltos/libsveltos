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

package sveltos_upgrade

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Sveltos-agent evaluates classifier, healthCheck, eventSource, and reloader
// instances within managed clusters. Other Sveltos microservices in the management
// cluster consume these evaluation results. During upgrades, management cluster
// services should pause fetching sveltos-agent evaluation results until the
// sveltos-agent version aligns with the versions of the Sveltos microservices in
// the management cluster.
// Sveltos-agent creates a ConfigMap in each managed cluster to store its version.
// Sveltos services in the management cluster then reference this ConfigMap to verify
// version compatibility.
// This compatibility check is especially important when Sveltos CRD versions change
// after upgrades. For example, if we transition from v1alpha1 to v1beta1, without this
// check, Sveltos microservices might attempt to fetch v1beta1 instances while sveltos-agent
// and the CRDs in the managed cluster are still on the older version.

// This package exposes methods to:
// - create/update ConfigMap
// - verify versions are compatible

const (
	configMapNamespace          = "projectsveltos"
	sveltosAgentConfigMapName   = "sveltos-agent-version"
	driftDetectionConfigMapName = "drift-detection-version"
	configMapKey                = "version"
)

// IsSveltosAgentVersionCompatible returns true if Sveltos-agent running in a managed cluster is compatible
// with the provided version.

// It takes three arguments:
//   - ctx (context.Context): Context for the function call
//   - c (client.Client): Kubernetes client used to interact with the API server
//   - version (string): Version to compare against the sveltos-agent version

func IsSveltosAgentVersionCompatible(ctx context.Context, c client.Client, version string) bool {
	cm := &corev1.ConfigMap{}

	const timeout = 10 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := c.Get(ctxWithTimeout, types.NamespacedName{Namespace: configMapNamespace, Name: sveltosAgentConfigMapName}, cm)
	if err != nil {
		return false
	}

	if cm.Data == nil {
		return false
	}

	return cm.Data[configMapKey] == version
}

// IsDriftDetectionVersionCompatible returns true if drift-detection-manager running in a managed cluster
// is compatible with the provided version.

// It takes three arguments:
//   - ctx (context.Context): Context for the function call
//   - c (client.Client): Kubernetes client used to interact with the API server
//   - version (string): Version to compare against the drift-detection-manager version

func IsDriftDetectionVersionCompatible(ctx context.Context, c client.Client, version string) bool {
	cm := &corev1.ConfigMap{}

	err := c.Get(ctx, types.NamespacedName{Namespace: configMapNamespace, Name: driftDetectionConfigMapName}, cm)
	if err != nil {
		return false
	}

	if cm.Data == nil {
		return false
	}

	return cm.Data[configMapKey] == version
}

// StoreSveltosAgentVersion stores the provided Sveltos-agent version in a ConfigMap.
// It takes three arguments:
//   - ctx (context.Context): Context for the function call
//   - c (client.Client): Kubernetes client used to interact with the API server
//   - version (string): Version of the Sveltos-agent to be stored
func StoreSveltosAgentVersion(ctx context.Context, c client.Client, version string) error {
	cm := &corev1.ConfigMap{}

	err := c.Get(ctx, types.NamespacedName{Namespace: configMapNamespace, Name: sveltosAgentConfigMapName}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return createConfigMap(ctx, c, sveltosAgentConfigMapName, version)
		}
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[configMapKey] = version
	return c.Update(ctx, cm)
}

// StoreDriftDetectionVersion stores the provided drift-detection-manager version in a ConfigMap.
// It takes three arguments:
//   - ctx (context.Context): Context for the function call
//   - c (client.Client): Kubernetes client used to interact with the API server
//   - version (string): Version of the drift-detection-manager to be stored
func StoreDriftDetectionVersion(ctx context.Context, c client.Client, version string) error {
	cm := &corev1.ConfigMap{}

	err := c.Get(ctx, types.NamespacedName{Namespace: configMapNamespace, Name: driftDetectionConfigMapName}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return createConfigMap(ctx, c, driftDetectionConfigMapName, version)
		}
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[configMapKey] = version
	return c.Update(ctx, cm)
}

func createConfigMap(ctx context.Context, c client.Client, name, version string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: configMapNamespace,
			Name:      name,
		},
		Data: map[string]string{
			configMapKey: version,
		},
	}
	return c.Create(ctx, cm)
}
