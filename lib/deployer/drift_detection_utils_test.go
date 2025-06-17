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

package deployer_test

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
)

var _ = Describe("Drift DetectionUtils", func() {
	It("deployResourceSummaryInstance updates ResourceSummary instance", func() {
		resources := []libsveltosv1beta1.Resource{
			{
				Name:      randomString(),
				Namespace: randomString(),
				Group:     randomString(),
				Kind:      randomString(),
				Version:   randomString(),
			},
		}
		namespace := randomString()
		name := randomString()

		// are created
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		err := testEnv.Create(context.TODO(), ns)
		if err != nil {
			Expect(apierrors.IsAlreadyExists(err)).To(BeTrue())
		}
		Expect(waitForObject(context.TODO(), testEnv.Client, ns)).To(Succeed())

		clusterSummaryNamespace := namespace
		clusterSummaryName := randomString()
		annotations := map[string]string{
			libsveltosv1beta1.ClusterSummaryNameAnnotation:      clusterSummaryName,
			libsveltosv1beta1.ClusterSummaryNamespaceAnnotation: clusterSummaryNamespace,
		}

		Expect(deployer.DeployResourceSummaryInstance(ctx, testEnv.Client, resources, nil, nil,
			namespace, name, nil, annotations, nil, textlogger.NewLogger(textlogger.NewConfig()))).
			To(Succeed())

		currentResourceSummary := &libsveltosv1beta1.ResourceSummary{}

		const pollingInterval = 5 * time.Second
		Eventually(func() error {
			return testEnv.Get(context.TODO(),
				types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
				currentResourceSummary)
		}, time.Minute, pollingInterval).Should(BeNil())

		Expect(testEnv.Get(context.TODO(),
			types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
			currentResourceSummary)).To(Succeed())

		Expect(currentResourceSummary.Annotations).ToNot(BeNil())
		v, ok := currentResourceSummary.Annotations[libsveltosv1beta1.ClusterSummaryNameAnnotation]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(clusterSummaryName))

		v, ok = currentResourceSummary.Annotations[libsveltosv1beta1.ClusterSummaryNamespaceAnnotation]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(clusterSummaryNamespace))

		Expect(reflect.DeepEqual(currentResourceSummary.Spec.Resources, resources)).To(BeTrue())
	})
})
