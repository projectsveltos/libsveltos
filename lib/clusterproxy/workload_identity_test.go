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

package clusterproxy_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/clusterproxy"
)

const (
	wiTestCachedEndpoint = "https://cached.example.com"
	wiTestCASecretName   = "my-ca"
)

var _ = Describe("WorkloadIdentity cache", func() {
	const (
		ns   = "default"
		name = "test-cluster"
	)

	AfterEach(func() {
		clusterproxy.EvictWorkloadIdentityCache(ns, name)
	})

	It("EvictWorkloadIdentityCache removes the entry", func() {
		cfg := &rest.Config{Host: "https://example.com"}
		clusterproxy.StoreTestWiCache(ns, name, cfg, time.Now().Add(time.Hour))

		_, _, ok := clusterproxy.LoadTestWiCache(ns, name)
		Expect(ok).To(BeTrue())

		clusterproxy.EvictWorkloadIdentityCache(ns, name)

		_, _, ok = clusterproxy.LoadTestWiCache(ns, name)
		Expect(ok).To(BeFalse())
	})

	It("GetSveltosKubernetesRestConfig returns cached config when not near expiry", func() {
		cached := &rest.Config{Host: wiTestCachedEndpoint}
		clusterproxy.StoreTestWiCache(ns, name, cached, time.Now().Add(time.Hour))

		wi := &libsveltosv1beta1.WorkloadIdentityConfig{
			Provider: libsveltosv1beta1.WorkloadIdentityProviderGCP,
			Endpoint: wiTestCachedEndpoint,
			GCP: &libsveltosv1beta1.GCPWorkloadIdentityConfig{
				ProjectID:   "proj",
				ClusterName: "cluster",
				Location:    "us-central1",
			},
		}
		sveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
			Spec: libsveltosv1beta1.SveltosClusterSpec{
				WorkloadIdentity: wi,
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster).Build()
		logger := logr.Discard()

		got, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, ns, name)
		Expect(err).To(BeNil())
		Expect(got).To(Equal(cached))
	})
})

var _ = Describe("WorkloadIdentity CA secret", func() {
	It("returns CA bytes from a Secret via getCAData", func() {
		caData := []byte("-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      wiTestCASecretName,
			},
			Data: map[string][]byte{
				"ca.crt": caData,
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
		logger := logr.Discard()

		ref := &corev1.LocalObjectReference{Name: wiTestCASecretName}
		got, err := clusterproxy.GetCADataForTest(ctx, c, "default", ref, logger)
		Expect(err).To(BeNil())
		Expect(got).To(Equal(caData))
	})

	It("returns nil when caSecretRef is nil", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		logger := logr.Discard()

		got, err := clusterproxy.GetCADataForTest(ctx, c, "default", nil, logger)
		Expect(err).To(BeNil())
		Expect(got).To(BeNil())
	})

	It("returns error when CA Secret does not exist", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		logger := logr.Discard()

		ref := &corev1.LocalObjectReference{Name: "missing-ca"}
		_, err := clusterproxy.GetCADataForTest(ctx, c, "default", ref, logger)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("CA secret"))
		Expect(err.Error()).To(ContainSubstring("missing-ca"))
	})
})
