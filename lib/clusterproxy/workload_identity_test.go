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
	"encoding/pem"
	"net/http"
	"net/http/httptest"
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
	wiTestEndpoint       = "https://example.com"
	wiTestGCPProjectID   = "proj"
	wiTestGCPCluster     = "cluster"
	wiTestGCPLocation    = "us-central1"
	wiTestCADataKey      = "ca.crt"
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
		cfg := &rest.Config{Host: wiTestEndpoint}
		wi := &libsveltosv1beta1.WorkloadIdentityConfig{
			Provider: libsveltosv1beta1.WorkloadIdentityProviderGCP,
			Endpoint: wiTestEndpoint,
		}
		clusterproxy.StoreTestWiCache(ns, name, cfg, time.Now().Add(time.Hour), wi)

		_, _, ok := clusterproxy.LoadTestWiCache(ns, name)
		Expect(ok).To(BeTrue())

		clusterproxy.EvictWorkloadIdentityCache(ns, name)

		_, _, ok = clusterproxy.LoadTestWiCache(ns, name)
		Expect(ok).To(BeFalse())
	})

	It("GetSveltosKubernetesRestConfig returns cached config when not near expiry", func() {
		wi := &libsveltosv1beta1.WorkloadIdentityConfig{
			Provider: libsveltosv1beta1.WorkloadIdentityProviderGCP,
			Endpoint: wiTestCachedEndpoint,
			GCP: &libsveltosv1beta1.GCPWorkloadIdentityConfig{
				ProjectID:   wiTestGCPProjectID,
				ClusterName: wiTestGCPCluster,
				Location:    wiTestGCPLocation,
			},
		}

		cached := &rest.Config{Host: wiTestCachedEndpoint}
		clusterproxy.StoreTestWiCache(ns, name, cached, time.Now().Add(time.Hour), wi)

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

	It("GetSveltosKubernetesRestConfig does not reuse a cached config once workloadIdentity spec changes", func() {
		oldWi := &libsveltosv1beta1.WorkloadIdentityConfig{
			Provider: libsveltosv1beta1.WorkloadIdentityProviderGCP,
			Endpoint: wiTestCachedEndpoint,
			GCP: &libsveltosv1beta1.GCPWorkloadIdentityConfig{
				ProjectID:   wiTestGCPProjectID,
				ClusterName: wiTestGCPCluster,
				Location:    wiTestGCPLocation,
			},
		}
		cached := &rest.Config{Host: wiTestCachedEndpoint}
		clusterproxy.StoreTestWiCache(ns, name, cached, time.Now().Add(time.Hour), oldWi)

		// Same cluster, but the CA secret reference was added/changed: this must be treated
		// as a different spec, not the one the cached entry was built from.
		newWi := oldWi.DeepCopy()
		newWi.CASecretRef = &corev1.LocalObjectReference{Name: wiTestCASecretName}

		sveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
			Spec: libsveltosv1beta1.SveltosClusterSpec{
				WorkloadIdentity: newWi,
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster).Build()
		logger := logr.Discard()

		// The stale cached config must not be returned. Without real cloud credentials in this
		// test environment the slow path is expected to fail obtaining a token, which is itself
		// proof the fast path was correctly bypassed.
		got, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, ns, name)
		Expect(err).ToNot(BeNil())
		Expect(got).ToNot(Equal(cached))
	})
})

var _ = Describe("workloadIdentityConfigHash", func() {
	It("returns the same hash for equal configs and a different hash when the spec changes", func() {
		wi := &libsveltosv1beta1.WorkloadIdentityConfig{
			Provider: libsveltosv1beta1.WorkloadIdentityProviderGCP,
			Endpoint: wiTestCachedEndpoint,
			GCP: &libsveltosv1beta1.GCPWorkloadIdentityConfig{
				ProjectID:   wiTestGCPProjectID,
				ClusterName: wiTestGCPCluster,
				Location:    wiTestGCPLocation,
			},
		}

		h1, err := clusterproxy.WorkloadIdentityConfigHashForTest(wi)
		Expect(err).To(BeNil())

		h2, err := clusterproxy.WorkloadIdentityConfigHashForTest(wi.DeepCopy())
		Expect(err).To(BeNil())
		Expect(h1).To(Equal(h2))

		withCA := wi.DeepCopy()
		withCA.CASecretRef = &corev1.LocalObjectReference{Name: wiTestCASecretName}
		h3, err := clusterproxy.WorkloadIdentityConfigHashForTest(withCA)
		Expect(err).To(BeNil())
		Expect(h3).ToNot(Equal(h1))
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
				wiTestCADataKey: caData,
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

var _ = Describe("WorkloadIdentity OIDC", func() {
	const (
		oidcNS          = "default"
		oidcClusterName = "oidc-test-cluster"
		//nolint:gosec // test fixture name, not a credential
		oidcSecretName    = "oidc-creds"
		oidcClientID      = "test-client-id"
		oidcClientSecret  = "test-client-secret"
		oidcClientIDKey   = "client_id"
		oidcSecretDataKey = "client_secret"
	)

	AfterEach(func() {
		clusterproxy.EvictWorkloadIdentityCache(oidcNS, oidcClusterName)
	})

	buildOIDCCluster := func(secretRef corev1.SecretReference, tokenURL string) *libsveltosv1beta1.SveltosCluster {
		wi := &libsveltosv1beta1.WorkloadIdentityConfig{
			Provider: libsveltosv1beta1.WorkloadIdentityProviderOIDC,
			Endpoint: wiTestEndpoint,
			OIDC: &libsveltosv1beta1.OIDCWorkloadIdentityConfig{
				TokenURL:  tokenURL,
				SecretRef: secretRef,
			},
		}
		return &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: oidcNS,
				Name:      oidcClusterName,
			},
			Spec: libsveltosv1beta1.SveltosClusterSpec{
				WorkloadIdentity: wi,
			},
		}
	}

	It("obtains a token via the standard client credentials grant and uses it as the bearer token", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
		}))
		defer server.Close()

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: oidcNS,
				Name:      oidcSecretName,
			},
			Data: map[string][]byte{
				oidcClientIDKey:   []byte(oidcClientID),
				oidcSecretDataKey: []byte(oidcClientSecret),
			},
		}

		sveltosCluster := buildOIDCCluster(
			corev1.SecretReference{Name: oidcSecretName, Namespace: oidcNS}, server.URL)

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster, secret).Build()
		logger := logr.Discard()

		got, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, oidcNS, oidcClusterName)
		Expect(err).To(BeNil())
		Expect(got).ToNot(BeNil())
		Expect(got.BearerToken).To(Equal("test-access-token"))
		Expect(got.Host).To(Equal(wiTestEndpoint))
	})

	It("defaults the credentials Secret namespace to the cluster's namespace when SecretRef.Namespace is empty", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"another-token","token_type":"Bearer","expires_in":3600}`))
		}))
		defer server.Close()

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: oidcNS,
				Name:      oidcSecretName,
			},
			Data: map[string][]byte{
				oidcClientIDKey:   []byte(oidcClientID),
				oidcSecretDataKey: []byte(oidcClientSecret),
			},
		}

		// SecretRef.Namespace intentionally left empty: getOIDCCredentials must fall back
		// to the cluster's own namespace rather than erroring or defaulting elsewhere.
		sveltosCluster := buildOIDCCluster(corev1.SecretReference{Name: oidcSecretName}, server.URL)

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster, secret).Build()
		logger := logr.Discard()

		got, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, oidcNS, oidcClusterName)
		Expect(err).To(BeNil())
		Expect(got.BearerToken).To(Equal("another-token"))
	})

	It("returns an error when the credentials Secret does not exist", func() {
		sveltosCluster := buildOIDCCluster(
			corev1.SecretReference{Name: "missing-secret", Namespace: oidcNS}, "https://idp.example.com/token")

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster).Build()
		logger := logr.Discard()

		_, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, oidcNS, oidcClusterName)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("OIDC credentials secret"))
	})

	It("returns an error when the Secret is missing the client_secret key", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: oidcNS,
				Name:      oidcSecretName,
			},
			Data: map[string][]byte{
				oidcClientIDKey: []byte(oidcClientID),
			},
		}

		sveltosCluster := buildOIDCCluster(
			corev1.SecretReference{Name: oidcSecretName, Namespace: oidcNS}, "https://idp.example.com/token")

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster, secret).Build()
		logger := logr.Discard()

		_, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, oidcNS, oidcClusterName)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring(oidcSecretDataKey))
	})

	It("trusts the IdP's token endpoint when its CA is provided via CASecretRef", func() {
		const idpCASecretName = "oidc-idp-ca" //nolint:gosec // test fixture name, not a credential

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tls-token","token_type":"Bearer","expires_in":3600}`))
		}))
		defer server.Close()

		serverCert := server.Certificate()
		caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Raw})

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: oidcNS,
				Name:      oidcSecretName,
			},
			Data: map[string][]byte{
				oidcClientIDKey:   []byte(oidcClientID),
				oidcSecretDataKey: []byte(oidcClientSecret),
			},
		}
		caSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: oidcNS,
				Name:      idpCASecretName,
			},
			Data: map[string][]byte{
				wiTestCADataKey: caPEM,
			},
		}

		sveltosCluster := buildOIDCCluster(
			corev1.SecretReference{Name: oidcSecretName, Namespace: oidcNS}, server.URL)
		sveltosCluster.Spec.WorkloadIdentity.OIDC.CASecretRef =
			&corev1.LocalObjectReference{Name: idpCASecretName}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster, secret, caSecret).Build()
		logger := logr.Discard()

		got, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, oidcNS, oidcClusterName)
		Expect(err).To(BeNil())
		Expect(got.BearerToken).To(Equal("tls-token"))
	})

	It("fails to reach a TLS IdP token endpoint when no CASecretRef is provided", func() {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tls-token","token_type":"Bearer","expires_in":3600}`))
		}))
		defer server.Close()

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: oidcNS,
				Name:      oidcSecretName,
			},
			Data: map[string][]byte{
				oidcClientIDKey:   []byte(oidcClientID),
				oidcSecretDataKey: []byte(oidcClientSecret),
			},
		}

		sveltosCluster := buildOIDCCluster(
			corev1.SecretReference{Name: oidcSecretName, Namespace: oidcNS}, server.URL)

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sveltosCluster, secret).Build()
		logger := logr.Discard()

		_, err := clusterproxy.GetSveltosKubernetesRestConfig(ctx, logger, c, oidcNS, oidcClusterName)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to obtain OIDC access token"))
	})
})
