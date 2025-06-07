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

package cue_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2/textlogger"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/cue"
)

var _ = Describe("utils ", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
	})

	It("CUE Rule marks object as a match", func() {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})
		obj.SetName("test-deployment")
		obj.SetNamespace("default")
		obj.SetLabels(map[string]string{
			"env": "prod",
		})

		// Rule that should match (label "env" == "prod")
		matchRule := libsveltosv1beta1.CUERule{
			Name: "match-env-prod",
			Rule: `resource: metadata: labels: env: "prod"`,
		}

		matched, err := cue.EvaluateRules(obj, []libsveltosv1beta1.CUERule{matchRule}, logger)
		Expect(err).To(BeNil())
		Expect(matched).To(BeTrue())

		noMatchRule := libsveltosv1beta1.CUERule{
			Name: "no-match-env-staging",
			Rule: `resource: metadata: labels: env: "staging"`,
		}

		matched, err = cue.EvaluateRules(obj, []libsveltosv1beta1.CUERule{noMatchRule}, logger)
		Expect(err).To(BeNil())
		Expect(matched).To(BeFalse())
	})

	It("CUE Rule marks object as a match when object matches at least one rule", func() {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "ConfigMap",
		})
		obj.SetName("my-config")
		obj.SetNamespace("kube-system")
		obj.Object["data"] = map[string]interface{}{
			"config.yaml": "enabled: true",
		}
		obj.SetLabels(map[string]string{
			"owner": "admin",
		})

		// Rule 1: Wrong label value — should not match
		rule1 := libsveltosv1beta1.CUERule{
			Name: "wrong-owner",
			Rule: `resource: metadata: labels: owner: "user"`,
		}

		// Rule 2: Wrong namespace — should not match
		rule2 := libsveltosv1beta1.CUERule{
			Name: "wrong-namespace",
			Rule: `resource: metadata: namespace: "default"`,
		}

		// Rule 3: Correct label value — should match
		rule3 := libsveltosv1beta1.CUERule{
			Name: "correct-owner",
			Rule: `resource: metadata: labels: owner: "admin"`,
		}

		matched, err := cue.EvaluateRules(obj,
			[]libsveltosv1beta1.CUERule{rule1, rule2, rule3}, logger)
		Expect(err).To(BeNil())
		Expect(matched).To(BeTrue())
	})

	It("CUE Rule verify deployment active replicas match expected replicas", func() {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})
		obj.SetName("test-deployment")
		obj.SetNamespace("default")
		obj.SetLabels(map[string]string{
			"env": "prod",
		})

		// Set spec.replicas
		Expect(unstructured.SetNestedField(obj.Object, int64(3), "spec", "replicas")).To(Succeed())

		// Set status.activeReplicas
		Expect(unstructured.SetNestedField(obj.Object, int64(1), "status", "activeReplicas")).To(Succeed())

		// Rule that should match (label "env" == "prod")
		matchRule := libsveltosv1beta1.CUERule{
			Name: "match-env-prod",
			Rule: `resource: {
    spec: replicas: int
    status: activeReplicas: spec.replicas
}`,
		}

		matched, err := cue.EvaluateRules(obj,
			[]libsveltosv1beta1.CUERule{matchRule}, logger)
		Expect(err).To(BeNil())
		Expect(matched).To(BeFalse())

		// Set status.activeReplicas
		Expect(unstructured.SetNestedField(obj.Object, int64(3), "status", "activeReplicas")).To(Succeed())

		matched, err = cue.EvaluateRules(obj,
			[]libsveltosv1beta1.CUERule{matchRule}, logger)
		Expect(err).To(BeNil())
		Expect(matched).To(BeTrue())
	})
})
