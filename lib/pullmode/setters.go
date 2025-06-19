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

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

type Options struct {
	SourceRef              *corev1.ObjectReference
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

	confGroup.Spec.ContinueOnError = c.ContinueOnError
	confGroup.Spec.ContinueOnConflict = c.ContinueOnConflict
	confGroup.Spec.MaxConsecutiveFailures = c.MaxConsecutiveFailures

	confGroup.Spec.RequestorHash = c.RequestorHash

	return confGroup
}
