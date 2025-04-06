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

package mgmtagent_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/projectsveltos/libsveltos/lib/mgmtagent"
)

var _ = Describe("Mgmtagent", func() {
	Context("Resource Key Handling", func() {
		const (
			testEventSourceName = "my-event-source"
			testHealthCheckName = "my-health-check"
			testReloaderName    = "my-reloader"
		)

		It("should correctly identify EventSource keys", func() {
			eventSourceKey := mgmtagent.GetKeyForEventSource(testEventSourceName)
			Expect(strings.HasPrefix(eventSourceKey, "eventsource-")).Should(BeTrue())
			Expect(mgmtagent.IsEventSourceEntry(eventSourceKey)).Should(BeTrue())
			Expect(mgmtagent.IsEventSourceEntry("other-" + testEventSourceName)).Should(BeFalse())
		})

		It("should correctly identify HealthCheck keys", func() {
			healthCheckKey := mgmtagent.GetKeyForHealthCheck(testHealthCheckName)
			Expect(strings.HasPrefix(healthCheckKey, "healthcheck-")).Should(BeTrue())
			Expect(mgmtagent.IsHealthCheckEntry(healthCheckKey)).Should(BeTrue())
			Expect(mgmtagent.IsHealthCheckEntry("other-" + testHealthCheckName)).Should(BeFalse())
		})

		It("should correctly identify Reloader keys", func() {
			reloaderKey := mgmtagent.GetKeyForReloader(testReloaderName)
			Expect(strings.HasPrefix(reloaderKey, "reloader-")).Should(BeTrue())
			Expect(mgmtagent.IsReloaderEntry(reloaderKey)).Should(BeTrue())
			Expect(mgmtagent.IsReloaderEntry("other-" + testReloaderName)).Should(BeFalse())
		})

		It("should generate the correct key for EventSource", func() {
			expectedKey := "eventsource-" + testEventSourceName
			actualKey := mgmtagent.GetKeyForEventSource(testEventSourceName)
			Expect(actualKey).To(Equal(expectedKey))
		})

		It("should generate the correct key for HealthCheck", func() {
			expectedKey := "healthcheck-" + testHealthCheckName
			actualKey := mgmtagent.GetKeyForHealthCheck(testHealthCheckName)
			Expect(actualKey).To(Equal(expectedKey))
		})

		It("should generate the correct key for Reloader", func() {
			expectedKey := "reloader-" + testReloaderName
			actualKey := mgmtagent.GetKeyForReloader(testReloaderName)
			Expect(actualKey).To(Equal(expectedKey))
		})
	})
})
