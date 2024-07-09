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

package patcher_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	sveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/patcher"
)

var _ = Describe("CustomPatchPostRenderer", func() {
	var renderer *patcher.CustomPatchPostRenderer
	var renderedManifests *bytes.Buffer
	var unstructuredObjs []*unstructured.Unstructured

	BeforeEach(func() {
		renderer = &patcher.CustomPatchPostRenderer{
			Patches: []sveltosv1beta1.Patch{
				{
					Patch: `apiVersion: v1
kind: Pod
metadata:
  name: patch
  labels:
    test: value`,
					Target: &sveltosv1beta1.PatchSelector{Kind: "Pod"},
				},
				{
					Patch: `- op: add
  path: /metadata/labels/environment
  value: production`,
					Target: &sveltosv1beta1.PatchSelector{Kind: "Pod"},
				},
			},
		}

		renderedManifests = bytes.NewBufferString(`
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  containers:
  - name: mycontainer
    image: myimage
`)

		unstructuredObjs = []*unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "mypod",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "mycontainer",
								"image": "myimage",
							},
						},
					},
				},
			},
		}
	})

	Describe("Run", func() {
		It("should correctly apply patches and return modified manifests", func() {
			modifiedManifests, err := renderer.Run(renderedManifests)
			Expect(err).ToNot(HaveOccurred())
			Expect(modifiedManifests).ToNot(BeNil())

			parsedObjects, err := patcher.ParseYAMLToUnstructured(modifiedManifests)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedObjects).ToNot(BeNil())
			Expect(parsedObjects).To(HaveLen(1))

			// Validate the output object
			obj := parsedObjects[0]
			Expect(obj.GetAPIVersion()).To(Equal("v1"))
			Expect(obj.GetKind()).To(Equal("Pod"))
			Expect(obj.GetName()).To(Equal("mypod"))
			Expect(obj.GetLabels()["test"]).To(Equal("value"))
			Expect(obj.GetLabels()["environment"]).To(Equal("production"))
		})
	})

	Describe("RunUnstructured", func() {
		It("should correctly apply patches to unstructured objects and return modified objects", func() {
			outputObjects, err := renderer.RunUnstructured(unstructuredObjs)
			Expect(err).ToNot(HaveOccurred())
			Expect(outputObjects).ToNot(BeNil())
			Expect(outputObjects).To(HaveLen(1))

			// Validate the output object
			obj := outputObjects[0]
			Expect(obj.GetAPIVersion()).To(Equal("v1"))
			Expect(obj.GetKind()).To(Equal("Pod"))
			Expect(obj.GetName()).To(Equal("mypod"))
			Expect(obj.GetLabels()["test"]).To(Equal("value"))
			Expect(obj.GetLabels()["environment"]).To(Equal("production"))
		})
	})
})
