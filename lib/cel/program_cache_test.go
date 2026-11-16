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

package cel_test

import (
	"fmt"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2/textlogger"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/cel"
)

const (
	envProdRule = `resource.metadata.labels.env == "prod"`
	envDevRule  = `resource.metadata.labels.env == "dev"`
	invalidRule = `resource.metadata.labels.env ==`

	appsGroup      = "apps"
	deploymentKind = "Deployment"
	envLabel       = "env"
	prodValue      = "prod"
	sameRuleName   = "same-name-rule"
)

var _ = Describe("program cache", Serial, func() {
	var logger logr.Logger
	var obj *unstructured.Unstructured

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
		cel.ResetSharedCache()

		obj = &unstructured.Unstructured{}
		obj.SetGroupVersionKind(schema.GroupVersionKind{Group: appsGroup, Version: v1Version, Kind: deploymentKind})
		obj.SetName("test-deployment")
		obj.SetNamespace("default")
		obj.SetLabels(map[string]string{envLabel: prodValue})
	})

	It("compiles an expression once and reuses it for every object", func() {
		rules := []libsveltosv1beta1.CELRule{{Name: "match-prod", Rule: envProdRule}}

		for i := 0; i < 10; i++ {
			matched, err := cel.EvaluateRules(obj, rules, logger)
			Expect(err).To(BeNil())
			Expect(matched).To(BeTrue())
		}

		Expect(cel.SharedCacheLen()).To(Equal(1))
	})

	It("treats an edited rule as a new expression and evaluates the new text", func() {
		matched, err := cel.EvaluateRules(obj,
			[]libsveltosv1beta1.CELRule{{Name: sameRuleName, Rule: envProdRule}}, logger)
		Expect(err).To(BeNil())
		Expect(matched).To(BeTrue())

		// Same rule name, different text
		matched, err = cel.EvaluateRules(obj,
			[]libsveltosv1beta1.CELRule{{Name: sameRuleName, Rule: envDevRule}}, logger)
		Expect(err).To(BeNil())
		Expect(matched).To(BeFalse())

		Expect(cel.SharedCacheLen()).To(Equal(2))
	})

	It("keeps returning the same error for an invalid expression, compiling it once", func() {
		rules := []libsveltosv1beta1.CELRule{{Name: "invalid", Rule: invalidRule}}

		for i := 0; i < 3; i++ {
			matched, err := cel.EvaluateRules(obj, rules, logger)
			// EvaluateRules skips rules that fail and reports no match
			Expect(err).To(BeNil())
			Expect(matched).To(BeFalse())
		}

		Expect(cel.SharedCacheLen()).To(Equal(1))

		cache := cel.NewProgramCache(10, 1024)
		err := cache.Get(invalidRule)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to parse CEL expression"))
		Expect(cache.Get(invalidRule)).To(Equal(err))
		Expect(cache.Len()).To(Equal(1))
	})

	It("evicts the least recently used expression when it holds too many", func() {
		cache := cel.NewProgramCache(2, 1024)

		Expect(cache.Get(`1 == 1`)).To(BeNil())
		Expect(cache.Get(`2 == 2`)).To(BeNil())
		// Using the first one makes the second the least recently used
		Expect(cache.Get(`1 == 1`)).To(BeNil())
		Expect(cache.Get(`3 == 3`)).To(BeNil())

		Expect(cache.Len()).To(Equal(2))
		Expect(cache.Contains(`1 == 1`)).To(BeTrue())
		Expect(cache.Contains(`2 == 2`)).To(BeFalse())
		Expect(cache.Contains(`3 == 3`)).To(BeTrue())
	})

	It("evicts expressions when their total size exceeds the limit", func() {
		first := `1 == 1`
		second := `2 == 2`
		third := `3 == 3`
		cache := cel.NewProgramCache(100, len(first)+len(second))

		Expect(cache.Get(first)).To(BeNil())
		Expect(cache.Get(second)).To(BeNil())
		Expect(cache.ExpressionBytes()).To(Equal(len(first) + len(second)))

		Expect(cache.Get(third)).To(BeNil())
		Expect(cache.Contains(first)).To(BeFalse())
		Expect(cache.Contains(second)).To(BeTrue())
		Expect(cache.Contains(third)).To(BeTrue())
		Expect(cache.ExpressionBytes()).To(Equal(len(second) + len(third)))
	})

	It("evaluates but does not cache an expression larger than the whole size limit", func() {
		cache := cel.NewProgramCache(100, 8)

		Expect(cache.Get(`1 == 1`)).To(BeNil())
		Expect(cache.Get(`"aaaaaaaaaa" == "aaaaaaaaaa"`)).To(BeNil())

		Expect(cache.Len()).To(Equal(1))
		Expect(cache.Contains(`1 == 1`)).To(BeTrue())
	})

	It("is safe for concurrent use and always returns the right answer", func() {
		const workers = 16
		const iterations = 200

		var wg sync.WaitGroup
		failures := make(chan string, workers)

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(worker int) {
				defer wg.Done()
				defer GinkgoRecover()

				// A mix of shared expressions (cache hits) and one per worker (cache misses)
				ownRule := fmt.Sprintf(`resource.metadata.labels.env == "worker-%d"`, worker)
				for i := 0; i < iterations; i++ {
					matched, err := cel.EvaluateRules(obj, []libsveltosv1beta1.CELRule{
						{Name: "dev", Rule: envDevRule},
						{Name: "own", Rule: ownRule},
						{Name: "prod", Rule: envProdRule},
					}, logr.Discard())
					if err != nil || !matched {
						failures <- fmt.Sprintf("worker %d iteration %d: matched=%t err=%v", worker, i, matched, err)
						return
					}
				}
			}(w)
		}

		wg.Wait()
		close(failures)

		var failed []string
		for f := range failures {
			failed = append(failed, f)
		}
		Expect(failed).To(BeEmpty(), strings.Join(failed, "\n"))

		// envDevRule + envProdRule + one per worker, each compiled and stored once
		Expect(cel.SharedCacheLen()).To(Equal(2 + workers))
	})
})
