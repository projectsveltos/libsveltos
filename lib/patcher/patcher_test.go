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
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
	"github.com/projectsveltos/libsveltos/lib/patcher"
)

var (
	podYAML = `apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  containers:
  - name: mycontainer
    image: myimage
`
)

var _ = Describe("decodeJSONPointerToken", func() {
	DescribeTable("decodes RFC 6902 escape sequences",
		func(input, expected string) {
			Expect(patcher.DecodeJSONPointerToken(input)).To(Equal(expected))
		},
		Entry("~1 becomes /", "velero.io~1exclude-from-backup", "velero.io/exclude-from-backup"),
		Entry("~0 becomes ~", "foo~0bar", "foo~bar"),
		Entry("~01 becomes ~1 (spec order matters)", "~01", "~1"),
		Entry("plain token unchanged", "metadata", "metadata"),
		Entry("empty string unchanged", "", ""),
	)
})

var _ = Describe("pathExistsInObject", func() {
	var obj *unstructured.Unstructured

	BeforeEach(func() {
		obj = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name": "mypod",
					"labels": map[string]interface{}{
						"velero.io/exclude-from-backup": "true",
						"env":                           "prod",
					},
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
		}
	})

	It("returns true for a simple path that exists", func() {
		Expect(patcher.PathExistsInObject(obj, "/metadata/name")).To(BeTrue())
	})

	It("returns false for a simple path that does not exist", func() {
		Expect(patcher.PathExistsInObject(obj, "/metadata/annotations")).To(BeFalse())
	})

	It("returns true for a label key containing a slash encoded as ~1", func() {
		Expect(patcher.PathExistsInObject(obj, "/metadata/labels/velero.io~1exclude-from-backup")).To(BeTrue())
	})

	It("returns false for a ~1-encoded path whose key does not exist", func() {
		Expect(patcher.PathExistsInObject(obj, "/metadata/labels/velero.io~1missing")).To(BeFalse())
	})

	It("returns true for a valid array index path", func() {
		Expect(patcher.PathExistsInObject(obj, "/spec/containers/0/name")).To(BeTrue())
	})

	It("returns false for an out-of-bounds array index", func() {
		Expect(patcher.PathExistsInObject(obj, "/spec/containers/5/name")).To(BeFalse())
	})

	It("returns false for a non-numeric array segment", func() {
		Expect(patcher.PathExistsInObject(obj, "/spec/containers/notanindex/name")).To(BeFalse())
	})
})

var _ = Describe("filterPatchOperations", func() {
	var obj *unstructured.Unstructured

	BeforeEach(func() {
		obj = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name": "mypod",
					"labels": map[string]interface{}{
						"velero.io/exclude-from-backup": "true",
						"existing-label":                "value",
					},
				},
			},
		}
	})

	It("keeps an SM patch unchanged", func() {
		p := sveltosv1beta1.Patch{
			Patch: `apiVersion: v1
kind: Pod
metadata:
  name: mypod
  labels:
    foo: bar`,
		}
		result, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeTrue())
		Expect(result.Patch).To(Equal(p.Patch))
	})

	It("keeps a remove op when the path exists", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: remove
  path: /metadata/labels/existing-label`,
		}
		result, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeTrue())
		Expect(patcher.IsJSONPatch(result.Patch)).To(BeTrue())
	})

	It("drops a remove op when the path does not exist", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: remove
  path: /metadata/labels/nonexistent`,
		}
		_, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeFalse())
	})

	It("keeps an add op regardless of whether the path exists", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: add
  path: /metadata/labels/new-label
  value: foo`,
		}
		_, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeTrue())
	})

	It("strips only the remove op targeting a missing path from a multi-op patch", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: add
  path: /metadata/labels/new-label
  value: foo
- op: remove
  path: /metadata/labels/nonexistent`,
		}
		result, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeTrue())
		// Rebuilt patch should contain only the add operation
		Expect(result.Patch).To(ContainSubstring(`"add"`))
		Expect(result.Patch).NotTo(ContainSubstring(`"remove"`))
	})

	It("keeps both ops when both remove paths exist", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: remove
  path: /metadata/labels/existing-label
- op: remove
  path: /metadata/labels/velero.io~1exclude-from-backup`,
		}
		result, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeTrue())
		// Patch should be returned as-is (both ops kept)
		Expect(result.Patch).To(Equal(p.Patch))
	})

	It("keeps a remove op for a ~1-encoded path that exists", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: remove
  path: /metadata/labels/velero.io~1exclude-from-backup`,
		}
		_, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeTrue())
	})

	It("drops a remove op for a ~1-encoded path that does not exist", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: remove
  path: /metadata/labels/velero.io~1missing`,
		}
		_, keep := patcher.FilterPatchOperations(p, obj)
		Expect(keep).To(BeFalse())
	})
})

var _ = Describe("validatePatch", func() {
	It("accepts a JSON patch without metadata.name", func() {
		p := sveltosv1beta1.Patch{
			Patch: `- op: add
  path: /metadata/labels/velero.io~1exclude-from-backup
  value: "true"`,
			Target: &sveltosv1beta1.PatchSelector{
				Kind:    "PersistentVolumeClaim",
				Version: "v1",
				Name:    "data-thanos-.*",
			},
		}
		Expect(patcher.ValidatePatch(p)).To(Succeed())
	})

	It("accepts an SM patch that has metadata.name", func() {
		p := sveltosv1beta1.Patch{
			Patch: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-thanos-0
  labels:
    velero.io/exclude-from-backup: "true"`,
		}
		Expect(patcher.ValidatePatch(p)).To(Succeed())
	})

	It("returns a clear error for an SM patch without metadata.name", func() {
		p := sveltosv1beta1.Patch{
			Patch: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
    velero.io/exclude-from-backup: "true"`,
			Target: &sveltosv1beta1.PatchSelector{
				Kind:    "PersistentVolumeClaim",
				Version: "v1",
				Name:    "data-thanos-.*",
			},
		}
		err := patcher.ValidatePatch(p)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("strategic merge patch"))
		Expect(err.Error()).To(ContainSubstring("metadata.name"))
		Expect(err.Error()).To(ContainSubstring("JSON patch"))
	})
})

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

		renderedManifests = bytes.NewBufferString(podYAML)

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

		It("adds a label whose key contains a slash (RFC 6902 ~1 encoding)", func() {
			pvc := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "PersistentVolumeClaim",
					"metadata": map[string]interface{}{
						"name":      "data-thanos-ruler-0",
						"namespace": "monitoring",
						"labels":    map[string]interface{}{},
					},
					"spec": map[string]interface{}{},
				},
			}

			r := &patcher.CustomPatchPostRenderer{
				Patches: []sveltosv1beta1.Patch{
					{
						Patch: `- op: add
  path: /metadata/labels/velero.io~1exclude-from-backup
  value: "true"`,
						Target: &sveltosv1beta1.PatchSelector{
							Version: "v1",
							Kind:    "PersistentVolumeClaim",
							Name:    "data-thanos-.*",
						},
					},
				},
			}

			result, err := r.RunUnstructured([]*unstructured.Unstructured{pvc})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].GetLabels()).To(HaveKey("velero.io/exclude-from-backup"))
			Expect(result[0].GetLabels()["velero.io/exclude-from-backup"]).To(Equal("true"))
		})

		It("removes a label whose key contains a slash (RFC 6902 ~1 encoding)", func() {
			pvc := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "PersistentVolumeClaim",
					"metadata": map[string]interface{}{
						"name":      "data-thanos-ruler-0",
						"namespace": "monitoring",
						"labels": map[string]interface{}{
							"velero.io/exclude-from-backup": "true",
						},
					},
					"spec": map[string]interface{}{},
				},
			}

			r := &patcher.CustomPatchPostRenderer{
				Patches: []sveltosv1beta1.Patch{
					{
						Patch: `- op: remove
  path: /metadata/labels/velero.io~1exclude-from-backup`,
						Target: &sveltosv1beta1.PatchSelector{
							Version: "v1",
							Kind:    "PersistentVolumeClaim",
							Name:    "data-thanos-.*",
						},
					},
				},
			}

			result, err := r.RunUnstructured([]*unstructured.Unstructured{pvc})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].GetLabels()).ToNot(HaveKey("velero.io/exclude-from-backup"))
		})

		It("with multiple resources correctly apply patches to unstructured objects and return modified objects", func() {
			nsYAML := `apiVersion: v1
kind: Namespace
metadata:
  name: my-test`

			saYAML := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-serviceaccount
  namespace: my-test`

			namespace, err := k8s_utils.GetUnstructured([]byte(nsYAML))
			Expect(err).To(BeNil())

			sa, err := k8s_utils.GetUnstructured([]byte(saYAML))
			Expect(err).To(BeNil())

			pod, err := k8s_utils.GetUnstructured([]byte(podYAML))
			Expect(err).To(BeNil())

			outputObjects, err := renderer.RunUnstructured([]*unstructured.Unstructured{namespace, sa, pod})
			Expect(err).ToNot(HaveOccurred())
			Expect(outputObjects).ToNot(BeNil())
			Expect(outputObjects).To(HaveLen(3))

			// Validate the output object
			obj := outputObjects[2]
			Expect(obj.GetAPIVersion()).To(Equal("v1"))
			Expect(obj.GetKind()).To(Equal("Pod"))
			Expect(obj.GetName()).To(Equal("mypod"))
			Expect(obj.GetLabels()["test"]).To(Equal("value"))
			Expect(obj.GetLabels()["environment"]).To(Equal("production"))
		})
	})
})
