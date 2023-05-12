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

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
	"github.com/projectsveltos/libsveltos/lib/utils"
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
)

var _ = Describe("Client", func() {
	It("ValidateObjectForUpdate returns error when resource is already installed because of different ConfigMap",
		func() {
			name := randomString()

			nsInstance := fmt.Sprintf(nsTemplate, name)

			configMapNs := randomString()
			configMapName := randomString()
			policyHash := randomString()

			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						deployer.ReferenceKindLabel:      string(libsveltosv1alpha1.ConfigMapReferencedResourceKind),
						deployer.ReferenceNameLabel:      configMapName,
						deployer.ReferenceNamespaceLabel: configMapNs,
					},
					Annotations: map[string]string{
						deployer.PolicyHash: policyHash,
					},
				},
			}

			Expect(testEnv.Create(context.TODO(), ns)).To(Succeed())
			Expect(waitForObject(context.TODO(), testEnv.Client, ns)).To(Succeed())

			Expect(addTypeInformationToObject(scheme, ns))

			dr, err := utils.GetDynamicResourceInterface(testEnv.Config, ns.GroupVersionKind(), "")
			Expect(err).To(BeNil())

			var u *unstructured.Unstructured
			u, err = utils.GetUnstructured([]byte(nsInstance))
			Expect(err).To(BeNil())

			// If different configMap, return error
			_, _, err = deployer.ValidateObjectForUpdate(context.TODO(), dr, u, string(libsveltosv1alpha1.ConfigMapReferencedResourceKind),
				randomString(), randomString())
			Expect(err).ToNot(BeNil())

			// If same configMap, return no error
			var exist bool
			var hash string
			exist, hash, err = deployer.ValidateObjectForUpdate(context.TODO(), dr, u, string(libsveltosv1alpha1.ConfigMapReferencedResourceKind),
				configMapNs, configMapName)
			Expect(err).To(BeNil())
			Expect(exist).To(BeTrue())
			Expect(hash).To(Equal(policyHash))
		})

	It("addOwnerReference adds an OwnerReference to an object. removeOwnerReference removes it", func() {
		roleRequest := &libsveltosv1alpha1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		policy, err := utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())
		Expect(policy.GetKind()).To(Equal("ClusterRole"))

		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		deployer.AddOwnerReference(policy, roleRequest)

		Expect(policy.GetOwnerReferences()).ToNot(BeNil())
		Expect(len(policy.GetOwnerReferences())).To(Equal(1))
		Expect(policy.GetOwnerReferences()[0].Kind).To(Equal(libsveltosv1alpha1.RoleRequestKind))
		Expect(policy.GetOwnerReferences()[0].Name).To(Equal(roleRequest.Name))

		deployer.RemoveOwnerReference(policy, roleRequest)
		Expect(len(policy.GetOwnerReferences())).To(Equal(0))
	})

	It("IsOnlyOwnerReference returns true when only one Owner is present", func() {
		roleRequest := &libsveltosv1alpha1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		policy, err := utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())
		Expect(policy.GetKind()).To(Equal("ClusterRole"))

		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		deployer.AddOwnerReference(policy, roleRequest)

		Expect(deployer.IsOnlyOwnerReference(policy, roleRequest)).To(BeTrue())

		roleRequest2 := &libsveltosv1alpha1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest2)).To(Succeed())
		deployer.AddOwnerReference(policy, roleRequest2)
		Expect(deployer.IsOnlyOwnerReference(policy, roleRequest)).To(BeFalse())
	})

	It("IsOwnerReference returns true when owner is present", func() {
		roleRequest := &libsveltosv1alpha1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		roleRequest2 := &libsveltosv1alpha1.RoleRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest2)).To(Succeed())

		policy, err := utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())
		Expect(policy.GetKind()).To(Equal("ClusterRole"))

		Expect(addTypeInformationToObject(testEnv.Scheme(), roleRequest)).To(Succeed())

		deployer.AddOwnerReference(policy, roleRequest)

		Expect(deployer.IsOwnerReference(policy, roleRequest)).To(BeTrue())
		Expect(deployer.IsOwnerReference(policy, roleRequest2)).To(BeFalse())

		deployer.AddOwnerReference(policy, roleRequest2)
		Expect(deployer.IsOwnerReference(policy, roleRequest)).To(BeTrue())
	})
})
