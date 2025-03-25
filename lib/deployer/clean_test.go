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

package deployer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
)

var _ = Describe("Clean utils", func() {
	It("handleResourceDelete leaves policies on Cluster when mode is LeavePolicies", func() {
		randomKey := randomString()
		randomValue := randomString()
		depl := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Labels: map[string]string{
					deployer.ReferenceKindLabel:      randomString(),
					deployer.ReferenceNameLabel:      randomString(),
					deployer.ReferenceNamespaceLabel: randomString(),
					randomKey:                        randomValue,
				},
			},
		}
		Expect(addTypeInformationToObject(scheme, depl)).To(Succeed())

		initObjects := []client.Object{depl}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		leavePolicies := true
		Expect(deployer.HandleResourceDelete(ctx, c, depl, leavePolicies,
			textlogger.NewLogger(textlogger.NewConfig()))).To(Succeed())

		currentDepl := &appsv1.Deployment{}
		Expect(c.Get(context.TODO(), types.NamespacedName{Namespace: depl.Namespace, Name: depl.Name}, currentDepl)).To(Succeed())
		Expect(len(currentDepl.Labels)).To(Equal(1))
		v, ok := currentDepl.Labels[randomKey]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(randomValue))
	})

	It("handleResourceDelete removes policies from Cluster when mode is not set to leave policies", func() {
		randomKey := randomString()
		randomValue := randomString()
		depl := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Labels: map[string]string{
					deployer.ReferenceKindLabel:      randomString(),
					deployer.ReferenceNameLabel:      randomString(),
					deployer.ReferenceNamespaceLabel: randomString(),
					randomKey:                        randomValue,
				},
			},
		}
		Expect(addTypeInformationToObject(scheme, depl)).To(Succeed())

		initObjects := []client.Object{depl}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		leavePolicies := false
		Expect(deployer.HandleResourceDelete(ctx, c, depl, leavePolicies,
			textlogger.NewLogger(textlogger.NewConfig()))).To(Succeed())

		currentDepl := &appsv1.Deployment{}
		err := c.Get(context.TODO(), types.NamespacedName{Namespace: depl.Namespace, Name: depl.Name}, currentDepl)
		Expect(err).ToNot(BeNil())
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("canDelete returns false when ClusterProfile is not referencing the policies anymore", func() {
		depl := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, depl)).To(Succeed())

		Expect(deployer.CanDelete(depl, map[string]libsveltosv1beta1.Resource{})).To(BeTrue())

		name := deployer.GetPolicyInfo(&libsveltosv1beta1.Resource{
			Kind:      depl.GetObjectKind().GroupVersionKind().Kind,
			Group:     depl.GetObjectKind().GroupVersionKind().Group,
			Name:      depl.GetName(),
			Namespace: depl.GetNamespace(),
		})
		Expect(deployer.CanDelete(depl, map[string]libsveltosv1beta1.Resource{name: {}})).To(BeFalse())
	})
})
