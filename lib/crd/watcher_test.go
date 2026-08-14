/*
Copyright 2022. projectsveltos.io. All rights reserved.

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

package crd_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectsveltos/libsveltos/lib/crd"
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
)

var (
	handlerCalled bool

	crdYAML = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  # name must match the spec fields below, and be in the form: <plural>.<group>
  name: crontabs.stable.example.com
spec:
  # group name to use for REST API: /apis/<group>/<version>
  group: stable.example.com
  # list of versions supported by this CustomResourceDefinition
  versions:
    - name: v1
      # Each version can be enabled/disabled by Served flag.
      served: true
      # One and only one version must be marked as the storage version.
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
  # either Namespaced or Cluster
  scope: Namespaced
  names:
    # plural name to be used in the URL: /apis/<group>/<version>/<plural>
    plural: crontabs
    # singular name to be used as an alias on the CLI and for display
    singular: crontab
    # kind is normally the CamelCased singular type. Your resource manifests use this.
    kind: CronTab
    # shortNames allow shorter string to match your resource on the CLI
    shortNames:
    - ct`

	// A second, distinct CRD (different name/group) so this spec does not collide
	// with the CRD left behind (envtest is shared across specs) by the spec above.
	widgetCrdYAML = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.other.example.com
spec:
  group: other.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size:
                  type: string
  scope: Namespaced
  names:
    plural: widgets
    singular: widget
    kind: Widget`
)

func handler(gvk *schema.GroupVersionKind, _ crd.ChangeType) {
	handlerCalled = true
}

var _ = Describe("WatchCustomResourceDefinition", func() {
	It("WatchCustomResourceDefinition registers handlers and starts watcher", func() {
		var err error
		scheme, err = setupScheme()
		Expect(err).ToNot(HaveOccurred())

		logger := textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))

		watcherCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go crd.WatchCustomResourceDefinition(watcherCtx, testEnv.Config, handler, logger)

		crdInstance, err := k8s_utils.GetUnstructured([]byte(crdYAML))
		Expect(err).To(BeNil())

		Expect(testEnv.Create(watcherCtx, crdInstance)).To(Succeed())

		Eventually(func() bool {
			return handlerCalled
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("WatchCustomResourceDefinition reports Modify and Delete for the same GVK", func() {
		var err error
		scheme, err = setupScheme()
		Expect(err).ToNot(HaveOccurred())

		logger := textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))

		var mu sync.Mutex
		seen := make(map[crd.ChangeType]bool)
		recorder := func(gvk *schema.GroupVersionKind, action crd.ChangeType) {
			if gvk.Group != "other.example.com" {
				// Ignore CRDs created/left over by other specs in this suite.
				return
			}
			mu.Lock()
			defer mu.Unlock()
			seen[action] = true
		}

		watcherCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go crd.WatchCustomResourceDefinition(watcherCtx, testEnv.Config, recorder, logger)

		crdInstance, err := k8s_utils.GetUnstructured([]byte(widgetCrdYAML))
		Expect(err).To(BeNil())
		Expect(testEnv.Create(watcherCtx, crdInstance)).To(Succeed())

		Eventually(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return seen[crd.Add]
		}, time.Minute, time.Second).Should(BeTrue())

		currentCrd, err := k8s_utils.GetUnstructured([]byte(widgetCrdYAML))
		Expect(err).To(BeNil())
		Expect(testEnv.Get(watcherCtx, client.ObjectKeyFromObject(currentCrd), currentCrd)).To(Succeed())
		labels := currentCrd.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels["updated"] = "true"
		currentCrd.SetLabels(labels)
		Expect(testEnv.Update(watcherCtx, currentCrd)).To(Succeed())

		Eventually(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return seen[crd.Modify]
		}, time.Minute, time.Second).Should(BeTrue())

		Expect(testEnv.Delete(watcherCtx, currentCrd)).To(Succeed())

		Eventually(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return seen[crd.Delete]
		}, time.Minute, time.Second).Should(BeTrue())
	})
})
