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

package v1alpha1_test

import (
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

var _ = Describe("Conversion", func() {
	Context("Convert from v1alpha1 to v1beta1 and back", func() {
		It("ClusterHealthCheck conversion", func() {
			key := randomString()
			value := randomString()

			clusterHealthCheck := libsveltosv1alpha1.ClusterHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name: randomString(),
				},
				Spec: libsveltosv1alpha1.ClusterHealthCheckSpec{
					ClusterSelector: libsveltosv1alpha1.Selector(fmt.Sprintf("%s=%s", key, value)),
					LivenessChecks: []libsveltosv1alpha1.LivenessCheck{
						{
							Name: randomString(),
							Type: libsveltosv1alpha1.LivenessTypeHealthCheck,
							LivenessSourceRef: &corev1.ObjectReference{
								APIVersion: libsveltosv1alpha1.GroupVersion.String(),
								Name:       randomString(),
								Kind:       libsveltosv1alpha1.HealthCheckKind,
								Namespace:  randomString(),
							},
						},
					},
					Notifications: []libsveltosv1alpha1.Notification{
						{
							Name: randomString(),
							Type: libsveltosv1alpha1.NotificationTypeDiscord,
							NotificationRef: &corev1.ObjectReference{
								Namespace: randomString(),
								Name:      randomString(),
								Kind:      string(libsveltosv1alpha1.SecretReferencedResourceKind),
							},
						},
					},
				},
			}

			dst := &libsveltosv1beta1.ClusterHealthCheck{}
			Expect(clusterHealthCheck.ConvertTo(dst)).To(Succeed())

			Expect(len(dst.Spec.ClusterSelector.LabelSelector.MatchLabels)).To(Equal(1))
			Expect(dst.Spec.ClusterSelector.LabelSelector.MatchLabels[key]).To(Equal(value))

			Expect(len(dst.Spec.LivenessChecks)).ToNot(BeZero())
			for i := range dst.Spec.LivenessChecks {
				lc := &dst.Spec.LivenessChecks[i]
				Expect(lc.LivenessSourceRef).ToNot(BeNil())
				Expect(lc.LivenessSourceRef.APIVersion).To(Equal(libsveltosv1beta1.GroupVersion.String()))
			}

			final := &libsveltosv1alpha1.ClusterHealthCheck{}
			Expect(final.ConvertFrom(dst)).To(Succeed())

			Expect(reflect.DeepEqual(final.ObjectMeta, clusterHealthCheck.ObjectMeta)).To(BeTrue())
			Expect(reflect.DeepEqual(final.Spec.LivenessChecks, clusterHealthCheck.Spec.LivenessChecks)).To(BeTrue())
			Expect(reflect.DeepEqual(final.Spec.Notifications, clusterHealthCheck.Spec.Notifications)).To(BeTrue())
			Expect(reflect.DeepEqual(final.Status, clusterHealthCheck.Status)).To(BeTrue())
		})

		It("ClusterSet conversion", func() {
			key := randomString()
			value := randomString()

			clusterSet := libsveltosv1alpha1.ClusterSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: randomString(),
				},
				Spec: libsveltosv1alpha1.Spec{
					ClusterSelector: libsveltosv1alpha1.Selector(fmt.Sprintf("%s=%s", key, value)),
					ClusterRefs: []corev1.ObjectReference{
						{
							Kind:      libsveltosv1alpha1.SveltosClusterKind,
							Namespace: randomString(),
							Name:      randomString(),
						},
					},
					MaxReplicas: 1,
				},
			}

			dst := &libsveltosv1beta1.ClusterSet{}
			Expect(clusterSet.ConvertTo(dst)).To(Succeed())

			Expect(len(dst.Spec.ClusterSelector.LabelSelector.MatchLabels)).To(Equal(1))
			Expect(dst.Spec.ClusterSelector.LabelSelector.MatchLabels[key]).To(Equal(value))

			final := &libsveltosv1alpha1.ClusterSet{}
			Expect(final.ConvertFrom(dst)).To(Succeed())

			Expect(reflect.DeepEqual(final.ObjectMeta, clusterSet.ObjectMeta)).To(BeTrue())
			Expect(reflect.DeepEqual(final.Spec.ClusterRefs, clusterSet.Spec.ClusterRefs)).To(BeTrue())
			Expect(final.Spec.MaxReplicas).To(Equal(clusterSet.Spec.MaxReplicas))
			Expect(reflect.DeepEqual(final.Status, clusterSet.Status)).To(BeTrue())
		})

		It("ClusterSet conversion", func() {
			key1 := randomString()
			value1 := randomString()
			key2 := randomString()
			value2 := randomString()

			set := libsveltosv1alpha1.Set{
				ObjectMeta: metav1.ObjectMeta{
					Name: randomString(),
				},
				Spec: libsveltosv1alpha1.Spec{
					ClusterSelector: libsveltosv1alpha1.Selector(fmt.Sprintf("%s=%s,%s=%s", key1, value1, key2, value2)),
					ClusterRefs: []corev1.ObjectReference{
						{
							Kind:      libsveltosv1alpha1.SveltosClusterKind,
							Namespace: randomString(),
							Name:      randomString(),
						},
					},
					MaxReplicas: 1,
				},
			}

			dst := &libsveltosv1beta1.Set{}
			Expect(set.ConvertTo(dst)).To(Succeed())

			Expect(len(dst.Spec.ClusterSelector.LabelSelector.MatchLabels)).To(Equal(2))
			Expect(dst.Spec.ClusterSelector.LabelSelector.MatchLabels[key1]).To(Equal(value1))
			Expect(dst.Spec.ClusterSelector.LabelSelector.MatchLabels[key2]).To(Equal(value2))

			final := &libsveltosv1alpha1.Set{}
			Expect(final.ConvertFrom(dst)).To(Succeed())

			Expect(reflect.DeepEqual(final.ObjectMeta, set.ObjectMeta)).To(BeTrue())
			Expect(reflect.DeepEqual(final.Spec.ClusterRefs, set.Spec.ClusterRefs)).To(BeTrue())
			Expect(final.Spec.MaxReplicas).To(Equal(set.Spec.MaxReplicas))
			Expect(reflect.DeepEqual(final.Status, set.Status)).To(BeTrue())
		})

		It("RoleRequest conversion", func() {
			key := randomString()
			value := randomString()

			expirationSeconds := int64(600)

			roleRequest := libsveltosv1alpha1.RoleRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: randomString(),
				},
				Spec: libsveltosv1alpha1.RoleRequestSpec{
					ClusterSelector: libsveltosv1alpha1.Selector(fmt.Sprintf("%s=%s", key, value)),
					RoleRefs: []libsveltosv1alpha1.PolicyRef{
						{
							Kind:      string(libsveltosv1alpha1.ConfigMapReferencedResourceKind),
							Namespace: randomString(),
							Name:      randomString(),
						},
						{
							Kind:      string(libsveltosv1alpha1.SecretReferencedResourceKind),
							Namespace: randomString(),
							Name:      randomString(),
						},
					},
					ExpirationSeconds:       &expirationSeconds,
					ServiceAccountName:      randomString(),
					ServiceAccountNamespace: randomString(),
				},
			}

			dst := &libsveltosv1beta1.RoleRequest{}
			Expect(roleRequest.ConvertTo(dst)).To(Succeed())

			Expect(len(dst.Spec.ClusterSelector.LabelSelector.MatchLabels)).To(Equal(1))
			Expect(dst.Spec.ClusterSelector.LabelSelector.MatchLabels[key]).To(Equal(value))

			final := &libsveltosv1alpha1.RoleRequest{}
			Expect(final.ConvertFrom(dst)).To(Succeed())

			Expect(reflect.DeepEqual(final.ObjectMeta, roleRequest.ObjectMeta)).To(BeTrue())
			Expect(reflect.DeepEqual(final.Spec.RoleRefs, roleRequest.Spec.RoleRefs)).To(BeTrue())
			Expect(final.Spec.ExpirationSeconds).To(Equal(roleRequest.Spec.ExpirationSeconds))
			Expect(final.Spec.ServiceAccountName).To(Equal(roleRequest.Spec.ServiceAccountName))
			Expect(final.Spec.ServiceAccountNamespace).To(Equal(roleRequest.Spec.ServiceAccountNamespace))
			Expect(reflect.DeepEqual(final.Status, roleRequest.Status)).To(BeTrue())
		})
	})
})

func randomString() string {
	const length = 10
	return util.RandomString(length)
}
