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

//nolint:lll // This file has long lines due to signed licenses
package license_test

import (
	"context"
	"encoding/base64"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
	license "github.com/projectsveltos/libsveltos/lib/licenses"
)

const (
	kubeSystemNamespace = "kube-system"
	//nolint: gosec // this is a Secret name, not a credential
	licenseSecretName = "sveltos-license"

	// testLicenseDataKey/testLicenseSignatureKey/testLabelKey/testLabelValue are used by the
	// MigrateLicenseSecretType tests below, which don't need a real signed license, just data
	// to assert gets carried over unchanged.
	testLicenseDataKey        = "licenseData"
	testLicenseSignatureKey   = "licenseSignature"
	testLicenseDataValue      = "data"
	testLicenseSignatureValue = "signature"
	testLabelKey              = "foo"
	testLabelValue            = "bar"

	// validLicenseDataB64 / validLicenseSignatureB64 are the licenseData/licenseSignature
	// values of a valid, non-expired license (Enterprise plan, PullMode feature, MaxClusters 1).
	// Shared by multiple tests below so this fixture isn't duplicated further.
	validLicenseDataB64      = "eyJpZCI6IjdmNDU0Mjk4LWUyYjItNDFhOC05MWRjLTc4ZjU5MjE4Nzk5MiIsImN1c3RvbWVyTmFtZSI6IkFjbWUgSW5jIiwiZmVhdHVyZXMiOlsiUHVsbE1vZGUiXSwicGxhbiI6IkVudGVycHJpc2UiLCJleHBpcmF0aW9uRGF0ZSI6IjIwMjctMDctMjZUMTM6NDc6MzEuODUyMjE2WiIsImdyYWNlUGVyaW9kRGF5cyI6NywibWF4Q2x1c3RlcnMiOjEsImlzc3VlZEF0IjoiMjAyNi0wNy0yNlQxMzo0NzozMS44NTIyMTZaIn0="
	validLicenseSignatureB64 = "BjYc6SDAcfrj4rBc4Fi0LfcdkU5qtAEojunwFV/0Llpsgx+IYi+XDyiq6VhZFDGl24tMcKOSHyPoTA3/5s5IIeZVpDRSd2+BXXFd9ccBScfyKRzDlO8Cg5rs0ejNwzrJKTNPxjorB7WxB3WK7ad6hbrmU6/PI6vdER7XjPsb3BuaDpzA5Z3wiuoyzB+BFJ9fbeVSDL2XxaZ+M/ifaM8/bGKsj7dUXwrP+ArNouObikxBsJsqZ9n1mgQx6WZm8fJOxct79Qmva1ys4O8QcP4MYY6rsbfag0xshpKKaZKMO10XugqtDYWz14pDV9vMKWEM0jMa4oHZmebMd4Wq+F/tB+GVIyYU8aWroYVKU5kkWFdFOcQozdNmyyLOn1umdJd2JouEWkcDHIRKfA1TpIfwaeKHB6JjW5WpKzbFfnrZUW9LWVOSegmv/HpgvXMiAXUyvVlhPHArPjBJyKhb5FNcb2kaYAk8ouFv/ydhFFcerTJMWp7bqkeLWlgMfUtw/oC0GKBLM090ovCkYAeaxjgWqm9sHVJCTFGeeqh2hff98UfAoQ5g9R9mP8e1rXNtWtdWK2UIaxcwymTKS7RNe2K3TpVORE4ESuOpSJMjUut7w2oAo1O7C2ZAlfoTi4JQmqBU9S+XG4AoEBkFPPbIlA8ak0xZU9Hh3Ony7WGr59tmkOU="
)

var _ = Describe("License", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
	})

	It("Detect Expired License", func() {
		sveltosNamespace := randomString()
		// Expired license
		secret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: sveltos-license
  namespace: %s
type: Opaque
data:
  licenseData: eyJpZCI6IjY0Mzk5ZTY3LTdhMjctNDY5MS05YzU1LWY4NmY0YjQ4MGFkOSIsImN1c3RvbWVyTmFtZSI6IkFjbWUgSW5jIiwiZmVhdHVyZXMiOlsiUHVsbE1vZGUiXSwiZXhwaXJhdGlvbkRhdGUiOiIyMDI0LTA3LTI1VDExOjU3OjIxLjk2MTYwOFoiLCJpc3N1ZWRBdCI6IjIwMjUtMDctMjVUMTE6NTc6MjEuOTYxNjA4WiJ9
  licenseSignature: Nk+Q3x/ZBg2DydTMcAhGzi8+xCBma4bsLfKXlN5f217/OqJVcfFDqlG3Q46nVRI92i/hOvXVAeEOnBpv8/0iDbUvSZB1fBilkyzglcH00hC7Y3CFF9CnxcmLlqWBl5ucL+MTmzCsgxMHhzklOF4oCMAAbigfty9xVCXE81rQN0jKPktZcVui15uubs7PVgXkvc7+NZrmmchXnECXz912S8ayllRWcgKL482xi8bf9XsKubg+mzQm/S4KvPBR1R8Yugnp1byyZmpzQmNMF1KYC5YT/vVqk7ojVZTPVG9y1SxnpFXGVO+4HRBnbEWoVnifg5U74FcU3kiIgOxpUoylsX88PCfZXdaJT5Mh65cZJVRx1RTYLgnBX260gzaLzuPF33uu5IZ1J182Si5RatkvNdPQd7mtLC2T/lyQK4gMqS2g0iidlxA2iwEeqC/UV42aeXrel3KRJ38TL0SNiCpMLly3ueC5sftdvRWARNel7aV/DAE+nfANIBO9YuLpiJY9EMndr1mpGclMZF6KbXkzOnEqbsiNmXANl7Y2lAKORWElC58IznD0WKFoFuc1ZltUDecGEFoExkdstrIPJ8HYi0dJ0OBaHfQNlo7MjEuHWkmZ1XoeUqMPxjFBrULlX74Lbowqif1lDnZhmZTTJs+qqGYLz424HtcVmir8UD5IboQ=`,
			sveltosNamespace)

		u, err := k8s_utils.GetUnstructured([]byte(secret))
		Expect(err).To(BeNil())

		initObjects := []client.Object{
			u,
		}

		c := fake.NewClientBuilder().WithObjects(initObjects...).Build()

		publicKey, err := license.GetPublicKey()
		Expect(err).To(BeNil())

		licenseVerificationResult := license.VerifyLicenseSecret(context.TODO(), c, sveltosNamespace, publicKey, logger)
		Expect(licenseVerificationResult.RawError).To(BeNil())
		Expect(licenseVerificationResult.IsExpired).To(BeTrue())
	})

	It("Get Features from license", func() {
		sveltosNamespace := randomString()
		// This contains a valid license (valid for one managed cluster only)
		secret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: sveltos-license
  namespace: %s
type: Opaque
data:
  licenseData: %s
  licenseSignature: %s`,
			sveltosNamespace, validLicenseDataB64, validLicenseSignatureB64)

		u, err := k8s_utils.GetUnstructured([]byte(secret))
		Expect(err).To(BeNil())

		initObjects := []client.Object{
			u,
		}

		c := fake.NewClientBuilder().WithObjects(initObjects...).Build()

		publicKey, err := license.GetPublicKey()
		Expect(err).To(BeNil())

		licenseVerificationResult := license.VerifyLicenseSecret(context.TODO(), c, sveltosNamespace, publicKey, logger)
		Expect(licenseVerificationResult.RawError).To(BeNil())
		Expect(licenseVerificationResult.IsExpired).To(BeFalse())
		Expect(licenseVerificationResult.IsInGracePeriod).To(BeFalse())

		Expect(licenseVerificationResult.Payload).ToNot(BeNil())
		Expect(licenseVerificationResult.Payload.Features).ToNot(BeNil())
		Expect(licenseVerificationResult.Payload.Features).To(ContainElements(license.FeaturePullMode))

		Expect(licenseVerificationResult.Payload.MaxClusters).To(Equal(1))
	})

	It("Verifies Cluster Fingerprint", func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubeSystemNamespace,
				UID:  "000cbaab-0898-4932-a066-8c5cff6c9752",
			},
		}

		sveltosNamespace := randomString()
		// This contains a valid license (valid for one managed cluster only) that contains a Cluster fingerprint
		secret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: sveltos-license
  namespace: %s
type: Opaque
data:
  licenseData: eyJpZCI6ImRkY2JjNDVjLTI5NTgtNDM0Mi1hOWY5LTgzYzEzZmJjYjk5MyIsImN1c3RvbWVyTmFtZSI6IkFjbWUgSW5jIiwiZmVhdHVyZXMiOlsiUHVsbE1vZGUiXSwicGxhbiI6IkVudGVycHJpc2UiLCJleHBpcmF0aW9uRGF0ZSI6IjIwMjctMDctMjZUMTM6NTM6NDIuNjAzNjEyWiIsImdyYWNlUGVyaW9kRGF5cyI6NywibWF4Q2x1c3RlcnMiOjEsImlzc3VlZEF0IjoiMjAyNi0wNy0yNlQxMzo1Mzo0Mi42MDM2MTJaIiwiY2x1c3RlckZpbmdlcnByaW50IjoiMDAwY2JhYWItMDg5OC00OTMyLWEwNjYtOGM1Y2ZmNmM5NzUyIn0=
  licenseSignature: BYUSUEDLPCKDB4U4KvcL2iVUl/WooEVFPUpglwvq83Ts0sRhUqTtcmxsPr8t4Z6Vp7xuXP3qvXd6ke+xUCq0JO/xfk1UCXuABkBki8gwrLL5EUyZpjtzr/ULQpkFFa0ElufBbkGHe5Bs/M6d2MuDLeY8l917aubE+dYyKTX7J0j1o8YZ/Ko327YHhwwjoP96YTALCCcu4LpxJ8WROrcoE3ngT/nWcfrpn/A5u7HI2/WPPMJpicIRxG3r3nPIyv3v71XrAaBi3CHpepoyJQmFoA4hLUn4KXfJWnB5DQ1FnHrf/KpMoVzVR2vGuWSo7Gh05KaSgdO1fjI+vs/iPhzCVErS5Lk6MmmSnpDJYSIl1T+XIQmOVhw7l+iyd5/AznKvBbYc5qCsz9F6zhLAck38U063iy+zUx+tt3o5kJLfWpHJMkcXJkA6XkxACGYrRILAB2HoPkUrkIj+J7sSHRprJj+vhEiU4gWVl/BNowWcFlPppKHQoNFanfdPwxELlOnn+6YjhFnaOUGUnbTyupjmYfuB8vPK9cHXuwMjGXjky2zMDvki+1DITc3YKzcpJlXrHkSyVEx+//wmHwq/yOAS8pBqqSWDKX+cfZtLpZHdl3NFmhO+FzBKvOejuw0eg+nSK/rnSXiyofLY6W7cgNTKemruF3hMjpNDZjklQrrOQVQ=`,
			sveltosNamespace)

		u, err := k8s_utils.GetUnstructured([]byte(secret))
		Expect(err).To(BeNil())

		initObjects := []client.Object{
			u, ns,
		}

		c := fake.NewClientBuilder().WithObjects(initObjects...).Build()

		publicKey, err := license.GetPublicKey()
		Expect(err).To(BeNil())

		licenseVerificationResult := license.VerifyLicenseSecret(context.TODO(), c, sveltosNamespace, publicKey, logger)
		Expect(licenseVerificationResult.RawError).To(BeNil())
		Expect(licenseVerificationResult.IsExpired).To(BeFalse())
		Expect(licenseVerificationResult.IsInGracePeriod).To(BeFalse())

		Expect(licenseVerificationResult.Payload).ToNot(BeNil())
		Expect(licenseVerificationResult.Payload.MaxClusters).To(Equal(1))

		currentNs := &corev1.Namespace{}
		Expect(c.Get(context.TODO(), types.NamespacedName{Name: kubeSystemNamespace}, currentNs)).To(Succeed())
		currentNs.UID = "000cbaab-1234-4932-a111-8c5cff6c9752"
		Expect(c.Update(context.TODO(), currentNs)).To(Succeed())

		licenseVerificationResult = license.VerifyLicenseSecret(context.TODO(), c, sveltosNamespace, publicKey, logger)
		Expect(licenseVerificationResult.RawError).ToNot(BeNil())
		Expect(licenseVerificationResult.RawError.Error()).To(ContainSubstring("License is not valid for this cluster (fingerprint mismatch)"))
	})
})

var _ = Describe("VerifyLicensePayload", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
	})

	It("verifies a valid payload/signature pair without reading a Secret", func() {
		payloadData, err := base64.StdEncoding.DecodeString(validLicenseDataB64)
		Expect(err).To(BeNil())
		signatureData, err := base64.StdEncoding.DecodeString(validLicenseSignatureB64)
		Expect(err).To(BeNil())

		publicKey, err := license.GetPublicKey()
		Expect(err).To(BeNil())

		result := license.VerifyLicensePayload(payloadData, signatureData, publicKey, logger)
		Expect(result.RawError).To(BeNil())
		Expect(result.IsExpired).To(BeFalse())
		Expect(result.IsInGracePeriod).To(BeFalse())

		Expect(result.Payload).ToNot(BeNil())
		Expect(result.Payload.Features).To(ContainElements(license.FeaturePullMode))
		Expect(result.Payload.MaxClusters).To(Equal(1))

		Expect(result.PayloadData).To(Equal(payloadData))
		Expect(result.SignatureData).To(Equal(signatureData))
	})

	It("rejects a payload that does not match the signature", func() {
		payloadData, err := base64.StdEncoding.DecodeString(validLicenseDataB64)
		Expect(err).To(BeNil())
		signatureData, err := base64.StdEncoding.DecodeString(validLicenseSignatureB64)
		Expect(err).To(BeNil())

		// Tamper with the payload after it was signed.
		payloadData = append(payloadData, []byte("tampered")...)

		publicKey, err := license.GetPublicKey()
		Expect(err).To(BeNil())

		result := license.VerifyLicensePayload(payloadData, signatureData, publicKey, logger)
		Expect(result.RawError).ToNot(BeNil())
		Expect(result.Payload).To(BeNil())
		Expect(result.IsExpired).To(BeTrue())
		Expect(result.IsEnforced).To(BeTrue())
	})
})

var _ = Describe("MigrateLicenseSecretType", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
	})

	It("does nothing when the license Secret does not exist", func() {
		sveltosNamespace := randomString()

		c := fake.NewClientBuilder().Build()

		Expect(license.MigrateLicenseSecretType(context.TODO(), c, sveltosNamespace, logger)).To(Succeed())

		currentSecret := &corev1.Secret{}
		err := c.Get(context.TODO(), types.NamespacedName{Namespace: sveltosNamespace, Name: licenseSecretName},
			currentSecret)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("does nothing when the license Secret already has the correct type", func() {
		sveltosNamespace := randomString()

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sveltosNamespace,
				Name:      licenseSecretName,
				Labels:    map[string]string{testLabelKey: testLabelValue},
			},
			Type: libsveltosv1beta1.ClusterProfileSecretType,
			Data: map[string][]byte{
				testLicenseDataKey:      []byte(testLicenseDataValue),
				testLicenseSignatureKey: []byte(testLicenseSignatureValue),
			},
		}

		c := fake.NewClientBuilder().WithObjects(secret).Build()

		Expect(license.MigrateLicenseSecretType(context.TODO(), c, sveltosNamespace, logger)).To(Succeed())

		currentSecret := &corev1.Secret{}
		Expect(c.Get(context.TODO(), types.NamespacedName{Namespace: sveltosNamespace, Name: licenseSecretName},
			currentSecret)).To(Succeed())
		Expect(currentSecret.Type).To(Equal(libsveltosv1beta1.ClusterProfileSecretType))
		Expect(currentSecret.Data).To(Equal(secret.Data))
		// unchanged: still the exact same object, never deleted/recreated
		Expect(currentSecret.UID).To(Equal(secret.UID))
	})

	It("recreates an Opaque license Secret with the ClusterProfileSecretType type, preserving data/labels/annotations", func() {
		sveltosNamespace := randomString()

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   sveltosNamespace,
				Name:        licenseSecretName,
				Labels:      map[string]string{testLabelKey: testLabelValue},
				Annotations: map[string]string{"baz": "qux"},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				testLicenseDataKey:      []byte(testLicenseDataValue),
				testLicenseSignatureKey: []byte(testLicenseSignatureValue),
			},
		}

		c := fake.NewClientBuilder().WithObjects(secret).Build()

		Expect(license.MigrateLicenseSecretType(context.TODO(), c, sveltosNamespace, logger)).To(Succeed())

		currentSecret := &corev1.Secret{}
		Expect(c.Get(context.TODO(), types.NamespacedName{Namespace: sveltosNamespace, Name: licenseSecretName},
			currentSecret)).To(Succeed())
		Expect(currentSecret.Type).To(Equal(libsveltosv1beta1.ClusterProfileSecretType))
		Expect(currentSecret.Data).To(Equal(secret.Data))
		Expect(currentSecret.Labels).To(Equal(secret.Labels))
		Expect(currentSecret.Annotations).To(Equal(secret.Annotations))
	})
})

var _ = Describe("HasFeature", func() {
	It("returns true for any feature when Features is empty", func() {
		lp := &license.LicensePayload{}
		Expect(lp.HasFeature(license.FeaturePullMode)).To(BeTrue())
		Expect(lp.HasFeature(license.FeatureMCP)).To(BeTrue())
		Expect(lp.HasFeature(license.FeaturePromotion)).To(BeTrue())
	})

	It("only allows features explicitly listed when Features is non-empty", func() {
		lp := &license.LicensePayload{Features: []license.Features{license.FeaturePullMode}}
		Expect(lp.HasFeature(license.FeaturePullMode)).To(BeTrue())
		Expect(lp.HasFeature(license.FeatureMCP)).To(BeFalse())
		Expect(lp.HasFeature(license.FeaturePromotion)).To(BeFalse())
	})

	It("allows multiple listed features", func() {
		lp := &license.LicensePayload{
			Features: []license.Features{license.FeatureMCP, license.FeaturePromotion},
		}
		Expect(lp.HasFeature(license.FeatureMCP)).To(BeTrue())
		Expect(lp.HasFeature(license.FeaturePromotion)).To(BeTrue())
		Expect(lp.HasFeature(license.FeaturePullMode)).To(BeFalse())
	})
})
