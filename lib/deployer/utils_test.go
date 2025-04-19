/*
Copyright 2023. projectsveltos.io. All rights reserved.

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

package deployer_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
)

const (
	nsTemplate = `apiVersion: v1
kind: Namespace
metadata:
  name: %s`

	viewClusterRole = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s
rules:
- apiGroups: [""] # "" indicates the core API group
  resources: ["pods"]
  verbs: ["get", "watch", "list"]`

	clusterProfile = `apiVersion: config.projectsveltos.io/v1beta1
kind: ClusterProfile
metadata:
  name: deploy-resources
  uid: ef15985d-045b-496c-92d9-e31e99dc13ee`

	profile = `apiVersion: config.projectsveltos.io/v1beta1
kind: Profile
metadata:
  name: deploy-resources
  namespace: default
  uid: ef15985d-045b-496c-92d9-e31e99dc13ee`
)

var _ = Describe("Client", func() {
	It("ValidateObjectForUpdate returns error when resource is already installed because of different ConfigMap (OwnerReference)",
		func() {
			name := randomString()

			cp, err := k8s_utils.GetUnstructured([]byte(clusterProfile))
			Expect(err).To(BeNil())

			nsInstance := fmt.Sprintf(nsTemplate, name)

			configMapNs := randomString()
			configMapName := randomString()
			policyHash := randomString()

			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						deployer.ReferenceKindLabel:      string(libsveltosv1beta1.ConfigMapReferencedResourceKind),
						deployer.ReferenceNameLabel:      configMapName,
						deployer.ReferenceNamespaceLabel: configMapNs,
					},
					Annotations: map[string]string{
						deployer.PolicyHash: policyHash,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "ClusterProfile",
							APIVersion: "config.projectsveltos.io/v1beta1",
							Name:       cp.GetName(),
							UID:        cp.GetUID(),
						},
					},
				},
			}

			Expect(testEnv.Create(context.TODO(), ns)).To(Succeed())
			Expect(waitForObject(context.TODO(), testEnv.Client, ns)).To(Succeed())

			Expect(addTypeInformationToObject(scheme, ns))

			dr, err := k8s_utils.GetDynamicResourceInterface(testEnv.Config, ns.GroupVersionKind(), "")
			Expect(err).To(BeNil())

			var u *unstructured.Unstructured
			u, err = k8s_utils.GetUnstructured([]byte(nsInstance))
			Expect(err).To(BeNil())

			// If different configMap, return error
			_, err = deployer.ValidateObjectForUpdate(context.TODO(), dr, u, string(libsveltosv1beta1.ConfigMapReferencedResourceKind),
				randomString(), randomString(), cp)
			Expect(err).ToNot(BeNil())

			// If same configMap, return no error
			var resourceInfo *deployer.ResourceInfo
			resourceInfo, err = deployer.ValidateObjectForUpdate(context.TODO(), dr, u, string(libsveltosv1beta1.ConfigMapReferencedResourceKind),
				configMapNs, configMapName, cp)
			Expect(err).To(BeNil())
			Expect(resourceInfo.CurrentResource).ToNot(BeNil())
			Expect(resourceInfo.CurrentResource.GetResourceVersion()).ToNot(BeEmpty())
			Expect(resourceInfo.Hash).To(Equal(policyHash))
			Expect(resourceInfo.CurrentResource).ToNot(BeNil())
			Expect(resourceInfo.CurrentResource.GetName()).To(Equal(ns.Name))
		})

	It("ValidateObjectForUpdate returns error when resource is already installed because of different ConfigMap (annotations)",
		func() {
			name := randomString()

			cp, err := k8s_utils.GetUnstructured([]byte(clusterProfile))
			Expect(err).To(BeNil())

			nsInstance := fmt.Sprintf(nsTemplate, name)

			configMapNs := randomString()
			configMapName := randomString()
			policyHash := randomString()

			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						deployer.ReferenceKindLabel:      string(libsveltosv1beta1.ConfigMapReferencedResourceKind),
						deployer.ReferenceNameLabel:      configMapName,
						deployer.ReferenceNamespaceLabel: configMapNs,
					},
					Annotations: map[string]string{
						deployer.PolicyHash: policyHash,
						deployer.OwnerKind:  "ClusterProfile",
						deployer.OwnerName:  cp.GetName(),
					},
				},
			}

			Expect(testEnv.Create(context.TODO(), ns)).To(Succeed())
			Expect(waitForObject(context.TODO(), testEnv.Client, ns)).To(Succeed())

			Expect(addTypeInformationToObject(scheme, ns))

			dr, err := k8s_utils.GetDynamicResourceInterface(testEnv.Config, ns.GroupVersionKind(), "")
			Expect(err).To(BeNil())

			var u *unstructured.Unstructured
			u, err = k8s_utils.GetUnstructured([]byte(nsInstance))
			Expect(err).To(BeNil())

			// If different configMap, return error
			_, err = deployer.ValidateObjectForUpdate(context.TODO(), dr, u, string(libsveltosv1beta1.ConfigMapReferencedResourceKind),
				randomString(), randomString(), cp)
			Expect(err).ToNot(BeNil())

			// If different profile, return err
			p, err := k8s_utils.GetUnstructured([]byte(profile))
			Expect(err).To(BeNil())
			_, err = deployer.ValidateObjectForUpdate(context.TODO(), dr, u, string(libsveltosv1beta1.ConfigMapReferencedResourceKind),
				configMapNs, configMapName, p)
			Expect(err).ToNot(BeNil())

			// If same configMap, return no error
			var resourceInfo *deployer.ResourceInfo
			resourceInfo, err = deployer.ValidateObjectForUpdate(context.TODO(), dr, u, string(libsveltosv1beta1.ConfigMapReferencedResourceKind),
				configMapNs, configMapName, cp)
			Expect(err).To(BeNil())
			Expect(resourceInfo.CurrentResource).ToNot(BeNil())
			Expect(resourceInfo.CurrentResource.GetResourceVersion()).ToNot(BeEmpty())
			Expect(resourceInfo.Hash).To(Equal(policyHash))
			Expect(resourceInfo.CurrentResource).ToNot(BeNil())
			Expect(resourceInfo.CurrentResource.GetName()).To(Equal(ns.Name))
		})

	It("addOwnerReference adds an OwnerReference to an object. removeOwnerReference removes it", func() {
		roleRequest := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		policy, err := k8s_utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())
		Expect(policy.GetKind()).To(Equal("ClusterRole"))

		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		k8s_utils.AddOwnerReference(policy, roleRequest)

		Expect(policy.GetOwnerReferences()).ToNot(BeNil())
		Expect(len(policy.GetOwnerReferences())).To(Equal(1))
		Expect(policy.GetOwnerReferences()[0].Kind).To(Equal(libsveltosv1beta1.RoleRequestKind))
		Expect(policy.GetOwnerReferences()[0].Name).To(Equal(roleRequest.Name))

		k8s_utils.RemoveOwnerReference(policy, roleRequest)
		Expect(len(policy.GetOwnerReferences())).To(Equal(0))
	})

	It("IsOnlyOwnerReference returns true when only one Owner is present", func() {
		roleRequest := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		policy, err := k8s_utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())
		Expect(policy.GetKind()).To(Equal("ClusterRole"))

		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		k8s_utils.AddOwnerReference(policy, roleRequest)

		Expect(k8s_utils.IsOnlyOwnerReference(policy, roleRequest)).To(BeTrue())

		roleRequest2 := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest2)).To(Succeed())
		k8s_utils.AddOwnerReference(policy, roleRequest2)
		Expect(k8s_utils.IsOnlyOwnerReference(policy, roleRequest)).To(BeFalse())
	})

	It("IsOwnerReference returns true when owner is present", func() {
		roleRequest := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		roleRequest2 := &libsveltosv1beta1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest2)).To(Succeed())

		policy, err := k8s_utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())
		Expect(policy.GetKind()).To(Equal("ClusterRole"))

		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		k8s_utils.AddOwnerReference(policy, roleRequest)

		Expect(k8s_utils.IsOwnerReference(policy, roleRequest)).To(BeTrue())
		Expect(k8s_utils.IsOwnerReference(policy, roleRequest2)).To(BeFalse())

		k8s_utils.AddOwnerReference(policy, roleRequest2)
		Expect(k8s_utils.IsOwnerReference(policy, roleRequest)).To(BeTrue())
	})
})
