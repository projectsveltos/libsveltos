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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

type Options struct {
	SourceRef              *corev1.ObjectReference
	SourceStatus           libsveltosv1beta1.SourceStatus
	Tier                   int32
	DryRun                 bool
	Reloader               bool
	DriftDetection         bool
	DriftExclusion         []libsveltosv1beta1.DriftExclusion
	ContinueOnConflict     bool
	ContinueOnError        bool
	MaxConsecutiveFailures *uint
	LeavePolicies          bool
	ValidateHealths        []libsveltosv1beta1.ValidateHealth
	DeployedGVKs           []string
	Annotations            map[string]string
	RequestorHash          []byte
	ServiceAccount         types.NamespacedName
}

type Option func(*Options)

func WithRequestorHash(hash []byte) Option {
	return func(args *Options) {
		args.RequestorHash = hash
	}
}

func WithLeavePolicies() Option {
	return func(args *Options) {
		args.LeavePolicies = true
	}
}

func WithDryRun() Option {
	return func(args *Options) {
		args.DryRun = true
	}
}

func WithDriftDetection() Option {
	return func(args *Options) {
		args.DriftDetection = true
	}
}

func WithDriftDetectionPatches(driftExclusions []libsveltosv1beta1.DriftExclusion) Option {
	return func(args *Options) {
		args.DriftExclusion = driftExclusions
	}
}

func WithValidateHealths(validateHealths []libsveltosv1beta1.ValidateHealth) Option {
	return func(args *Options) {
		args.ValidateHealths = validateHealths
	}
}

func WithReloader() Option {
	return func(args *Options) {
		args.Reloader = true
	}
}

func WithTier(tier int32) Option {
	return func(args *Options) {
		args.Tier = tier
	}
}

func WithSourceRef(sourceRef *corev1.ObjectReference) Option {
	return func(args *Options) {
		args.SourceRef = sourceRef
	}
}

func WithSourceStatus(sourceStatus libsveltosv1beta1.SourceStatus) Option {
	return func(args *Options) {
		args.SourceStatus = sourceStatus
	}
}

func WithContinueOnError(continueOnError bool) Option {
	return func(args *Options) {
		args.ContinueOnError = continueOnError
	}
}

func WithContinueOnConflict(continueOnConflict bool) Option {
	return func(args *Options) {
		args.ContinueOnConflict = continueOnConflict
	}
}

func WithMaxConsecutiveFailures(maxConsecutiveFailures uint) Option {
	return func(args *Options) {
		args.MaxConsecutiveFailures = &maxConsecutiveFailures
	}
}

func WithDeployedGVKs(gvks []string) Option {
	return func(args *Options) {
		args.DeployedGVKs = gvks
	}
}

func WithAnnotations(annotations map[string]string) Option {
	return func(args *Options) {
		args.Annotations = annotations
	}
}

func WithServiceAccount(namespace, name string) Option {
	return func(args *Options) {
		args.ServiceAccount = types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
	}
}

func applySetters(confGroup *libsveltosv1beta1.ConfigurationGroup, setters ...Option,
) *libsveltosv1beta1.ConfigurationGroup {

	c := &Options{}
	for _, setter := range setters {
		setter(c)
	}

	confGroup.Spec.Tier = c.Tier

	if c.DriftDetection {
		confGroup.Spec.DriftDetection = true
	}
	confGroup.Spec.DriftExclusions = c.DriftExclusion

	confGroup.Spec.ValidateHealths = c.ValidateHealths

	if c.DryRun {
		confGroup.Spec.DryRun = true
	}

	if c.LeavePolicies {
		confGroup.Spec.LeavePolicies = true
	}

	if c.Reloader {
		confGroup.Spec.Reloader = true
	}

	confGroup.Annotations = c.Annotations

	confGroup.Spec.DeployedGroupVersionKind = c.DeployedGVKs

	confGroup.Spec.SourceRef = c.SourceRef
	confGroup.Spec.SourceStatus = c.SourceStatus

	confGroup.Spec.ContinueOnError = c.ContinueOnError
	confGroup.Spec.ContinueOnConflict = c.ContinueOnConflict
	confGroup.Spec.MaxConsecutiveFailures = c.MaxConsecutiveFailures

	confGroup.Spec.RequestorHash = c.RequestorHash
	confGroup.Spec.ServiceAccountNamespace = c.ServiceAccount.Namespace
	confGroup.Spec.ServiceAccountName = c.ServiceAccount.Name

	return confGroup
}

type BundleOptions struct {
	Timeout                   *metav1.Duration
	ReleaseNamespace          string
	ReleaseName               string
	ChartVersion              string
	Icon                      string
	RepoURL                   string
	UninstallRealease         bool
	IsLastHelmReleaseBundle   bool
	ReferencedObjectKind      string
	ReferencedObjectNamespace string
	ReferencedObjectName      string
	ReferencedTier            int32
}

type BundleOption func(*BundleOptions)

func WithTimeout(timeout *metav1.Duration) BundleOption {
	return func(args *BundleOptions) {
		args.Timeout = timeout
	}
}

func WithReleaseInfo(namespace, name, repoURL, chartVersion, icon string,
	uninstall, isLast bool) BundleOption {

	return func(args *BundleOptions) {
		args.ReleaseNamespace = namespace
		args.ReleaseName = name
		args.ChartVersion = chartVersion
		args.RepoURL = repoURL
		args.Icon = icon
		args.UninstallRealease = uninstall
		args.IsLastHelmReleaseBundle = isLast
	}
}

func WithResourceInfo(kind, namespace, name string,
	tier int32) BundleOption {

	return func(args *BundleOptions) {
		args.ReferencedObjectKind = kind
		args.ReferencedObjectNamespace = namespace
		args.ReferencedObjectName = name
		args.ReferencedTier = tier
	}
}

func applyBundleSetters(confBundle *libsveltosv1beta1.ConfigurationBundle, setters ...BundleOption,
) *libsveltosv1beta1.ConfigurationBundle {

	if len(setters) == 0 {
		// No options were passed
		return confBundle
	}

	c := &BundleOptions{}
	for _, setter := range setters {
		setter(c)
	}

	confBundle.Spec.Timeout = c.Timeout
	confBundle.Spec.HelmReleaseNamespace = c.ReleaseNamespace
	confBundle.Spec.HelmReleaseName = c.ReleaseName
	confBundle.Spec.HelmChartVersion = c.ChartVersion
	confBundle.Spec.HelmIcon = c.Icon
	confBundle.Spec.HelmRepoURL = c.RepoURL
	confBundle.Spec.HelmReleaseUninstall = c.UninstallRealease
	confBundle.Spec.IsLastHelmReleaseBundle = c.IsLastHelmReleaseBundle

	confBundle.Spec.ReferencedObjectKind = c.ReferencedObjectKind
	confBundle.Spec.ReferencedObjectNamespace = c.ReferencedObjectNamespace
	confBundle.Spec.ReferencedObjectName = c.ReferencedObjectName
	confBundle.Spec.ReferenceTier = c.ReferencedTier

	return confBundle
}
