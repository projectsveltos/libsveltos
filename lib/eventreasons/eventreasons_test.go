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

package eventreasons_test

import (
	"regexp"
	"strings"
	"unicode/utf8"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/projectsveltos/libsveltos/lib/eventreasons"
)

var camelCase = regexp.MustCompile(`^[A-Z][a-zA-Z]+$`)

// allReasons lists every reason constant. A new reason must be added here.
func allReasons() []string {
	return []string{
		eventreasons.FeatureDeployed,
		eventreasons.FeatureRemoved,
		eventreasons.FeatureFailed,
		eventreasons.FeatureFailedNonRetriable,
		eventreasons.ProfileConflict,
		eventreasons.DependencyNotReady,
		eventreasons.UpdatePending,
		eventreasons.ClusterNotReady,
		eventreasons.AgentUnhealthy,
		eventreasons.DriftDetected,
		eventreasons.DriftCorrected,
		eventreasons.HealthCheckDegraded,
		eventreasons.LivenessCheckFailed,
		eventreasons.ClusterHealthCheckFailed,
		eventreasons.NotificationFailed,
		eventreasons.EventTriggerFailed,
		eventreasons.EventSourceInvalid,
		eventreasons.ClassifierFailed,
		eventreasons.ClassifierLabelConflict,
		eventreasons.ClusterUnreachable,
		eventreasons.KubeconfigInvalid,
		eventreasons.LicenseInvalid,
		eventreasons.ClusterRecovered,
		eventreasons.AccessRequestFailed,
		eventreasons.RoleRequestFailed,
		eventreasons.StageStarted,
		eventreasons.PromotionCompleted,
		eventreasons.StageFailed,
		eventreasons.ApprovalRequired,
	}
}

func allActions() []string {
	return []string{
		eventreasons.ActionDeploying,
		eventreasons.ActionRemoving,
		eventreasons.ActionEvaluating,
		eventreasons.ActionReconciling,
		eventreasons.ActionPromoting,
		eventreasons.ActionConnecting,
		eventreasons.ActionNotifying,
	}
}

var _ = Describe("EventReasons", func() {
	It("every reason is CamelCase", func() {
		for _, reason := range allReasons() {
			Expect(reason).To(MatchRegexp(camelCase.String()), "reason %q", reason)
		}
	})

	It("every reason is unique", func() {
		seen := map[string]bool{}
		for _, reason := range allReasons() {
			Expect(seen).ToNot(HaveKey(reason), "reason %q is defined twice", reason)
			seen[reason] = true
		}
	})

	It("every action is CamelCase and is not also a reason", func() {
		reasons := map[string]bool{}
		for _, reason := range allReasons() {
			reasons[reason] = true
		}

		for _, action := range allActions() {
			Expect(action).To(MatchRegexp(camelCase.String()), "action %q", action)
			Expect(reasons).ToNot(HaveKey(action), "action %q is also a reason", action)
		}
	})

	It("TruncateNote returns a short note unchanged", func() {
		note := "Feature Helm deployed to cluster Cluster default/prod"
		Expect(eventreasons.TruncateNote(note)).To(Equal(note))
	})

	It("TruncateNote returns a note of exactly NoteMaxLength unchanged", func() {
		note := strings.Repeat("a", eventreasons.NoteMaxLength)
		Expect(eventreasons.TruncateNote(note)).To(Equal(note))
	})

	It("TruncateNote caps a long note and says it was cut", func() {
		note := strings.Repeat("a", eventreasons.NoteMaxLength*2)

		result := eventreasons.TruncateNote(note)
		Expect(len(result)).To(Equal(eventreasons.NoteMaxLength))
		Expect(result).To(HaveSuffix("...(truncated)"))
	})

	It("TruncateNote never returns invalid UTF-8", func() {
		// One ASCII byte followed by two byte characters: the cut point falls in the middle of a character.
		note := "a" + strings.Repeat("é", eventreasons.NoteMaxLength)

		result := eventreasons.TruncateNote(note)
		Expect(utf8.ValidString(result)).To(BeTrue())
		Expect(len(result)).To(BeNumerically("<=", eventreasons.NoteMaxLength))
		Expect(result).To(HaveSuffix("...(truncated)"))
	})
})
