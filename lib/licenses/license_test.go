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

//nolint:lll,gosec // This file has long lines due to signed licenses
package license_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
	license "github.com/projectsveltos/libsveltos/lib/licenses"
)

var _ = Describe("License", func() {
	var logger logr.Logger

	secretInfo := types.NamespacedName{Namespace: "projectsveltos", Name: "sveltos-license"}

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
	})

	It("Detect Expired License", func() {
		// Expired license
		secret := `apiVersion: v1
kind: Secret
metadata:
  name: sveltos-license
  namespace: projectsveltos
type: Opaque
data:
  licenseData: eyJpZCI6IjY0Mzk5ZTY3LTdhMjctNDY5MS05YzU1LWY4NmY0YjQ4MGFkOSIsImN1c3RvbWVyTmFtZSI6IkFjbWUgSW5jIiwiZmVhdHVyZXMiOlsiUHVsbE1vZGUiXSwiZXhwaXJhdGlvbkRhdGUiOiIyMDI0LTA3LTI1VDExOjU3OjIxLjk2MTYwOFoiLCJpc3N1ZWRBdCI6IjIwMjUtMDctMjVUMTE6NTc6MjEuOTYxNjA4WiJ9
  licenseSignature: Nk+Q3x/ZBg2DydTMcAhGzi8+xCBma4bsLfKXlN5f217/OqJVcfFDqlG3Q46nVRI92i/hOvXVAeEOnBpv8/0iDbUvSZB1fBilkyzglcH00hC7Y3CFF9CnxcmLlqWBl5ucL+MTmzCsgxMHhzklOF4oCMAAbigfty9xVCXE81rQN0jKPktZcVui15uubs7PVgXkvc7+NZrmmchXnECXz912S8ayllRWcgKL482xi8bf9XsKubg+mzQm/S4KvPBR1R8Yugnp1byyZmpzQmNMF1KYC5YT/vVqk7ojVZTPVG9y1SxnpFXGVO+4HRBnbEWoVnifg5U74FcU3kiIgOxpUoylsX88PCfZXdaJT5Mh65cZJVRx1RTYLgnBX260gzaLzuPF33uu5IZ1J182Si5RatkvNdPQd7mtLC2T/lyQK4gMqS2g0iidlxA2iwEeqC/UV42aeXrel3KRJ38TL0SNiCpMLly3ueC5sftdvRWARNel7aV/DAE+nfANIBO9YuLpiJY9EMndr1mpGclMZF6KbXkzOnEqbsiNmXANl7Y2lAKORWElC58IznD0WKFoFuc1ZltUDecGEFoExkdstrIPJ8HYi0dJ0OBaHfQNlo7MjEuHWkmZ1XoeUqMPxjFBrULlX74Lbowqif1lDnZhmZTTJs+qqGYLz424HtcVmir8UD5IboQ=`

		u, err := k8s_utils.GetUnstructured([]byte(secret))
		Expect(err).To(BeNil())

		initObjects := []client.Object{
			u,
		}

		c := fake.NewClientBuilder().WithObjects(initObjects...).Build()

		publicKey, err := license.GetPublicKeyFromString()
		Expect(err).To(BeNil())

		_, err = license.VerifyLicenseSecret(context.TODO(), c, secretInfo, publicKey, logger)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("License has fully expired and is enforced"))
	})

	It("Get Features from license", func() {
		// This contains a valid license (valid for one managed cluster only)
		secret := `apiVersion: v1
kind: Secret
metadata:
  name: sveltos-license
  namespace: projectsveltos
type: Opaque
data:
  licenseData: eyJpZCI6IjY0MzNmYzk0LTE2MDYtNGZkOC04YTFkLThjYzg5NjRjOTk0MyIsImN1c3RvbWVyTmFtZSI6IkFjbWUgSW5jIiwiZmVhdHVyZXMiOlsiUHVsbE1vZGUiXSwiZXhwaXJhdGlvbkRhdGUiOiIyMDI2LTA3LTI1VDExOjU2OjQyLjcyNDIyM1oiLCJtYXhDbHVzdGVycyI6MSwiaXNzdWVkQXQiOiIyMDI1LTA3LTI1VDExOjU2OjQyLjcyNDIyM1oifQ==
  licenseSignature: ghCbge6VIPr2O0Tvu8jMoEa7HeFBIWxBMOU6L1yq6p1vH7fRQBWGJSEYRH+byUlyXc3MU/mMfmRyphM+X8Birqr25JszIo7n8cps1Ec5FekHo7xBKwXnHp/HWOm6NMohtonCcjU76sd7QTsYbLHujC6NhEObjZzBxBN6TP+m0hYYTufh1gOYBH4BdVFNGlFkqslk1bL4pQo6573okkYilRg+WF+vOKdlidz+pruUcqGvByRsL3OcENN9NItSyP9S2Hkz7Wb0ENfaMhND5jqH8NIfsjNTTE8TeUtTphmUwiRyXV65Tkdv3nLA1ektoSS+fc318KEV0EjIwZPo5Yq7KZls9l7mBKZXj4YSy2Rhj6cD5NOMoAKdt5S82t2amQhWbShCCgWyDOmrBRXiyD9OK8y3TKQ6zQYGCwOYtpCZV+uD1WBdTwhrG0lnGZkfdzkkt4pftMx6YFGopFMLc5/REOPaSWtWGoR9/bwS4s0EWkDCyFrAatxtwhEbOV4pOx5uV2ap/j4v7Ag0PpHJnlnYkW/Z6q5VCov8U1I6OFaCcBoI1MDruubD0qiL2eE25kpTT//cWA/3qxEyh54UqQPeZo6Lno3dW3YlZr62uPXniOzHuOlGWikKK/Wly1gUHfjOUR2PyohC9u3BL9HAZp5IRz4/vXflVqAlv0yWaEbaXHQ=`

		u, err := k8s_utils.GetUnstructured([]byte(secret))
		Expect(err).To(BeNil())

		initObjects := []client.Object{
			u,
		}

		c := fake.NewClientBuilder().WithObjects(initObjects...).Build()

		publicKey, err := license.GetPublicKeyFromString()
		Expect(err).To(BeNil())

		payload, err := license.VerifyLicenseSecret(context.TODO(), c, secretInfo, publicKey, logger)
		Expect(err).To(BeNil())

		Expect(payload).ToNot(BeNil())
		Expect(payload.Features).ToNot(BeNil())
		Expect(payload.Features).To(ContainElements(license.FeaturePullMode))

		Expect(payload.MaxClusters).To(Equal(1))
	})

	It("Verifies Cluster Fingerprint", func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
				UID:  "000cbaab-0898-4932-a066-8c5cff6c9752",
			},
		}

		// This contains a valid license (valid for one managed cluster only) that contains a Cluster fingerprint
		secret := `apiVersion: v1
kind: Secret
metadata:
  name: sveltos-license
  namespace: projectsveltos
type: Opaque
data:
  licenseData: eyJpZCI6IjI3NWU2MzUyLTFiYzUtNDUyOC04OTUwLWZhY2Q0MzdlMTI0YyIsImN1c3RvbWVyTmFtZSI6IkFjbWUgSW5jIiwiZmVhdHVyZXMiOlsiUHVsbE1vZGUiXSwiZXhwaXJhdGlvbkRhdGUiOiIyMDI2LTA3LTI1VDExOjU2OjE2Ljk4MzYxOVoiLCJtYXhDbHVzdGVycyI6MSwiaXNzdWVkQXQiOiIyMDI1LTA3LTI1VDExOjU2OjE2Ljk4MzYxOVoiLCJjbHVzdGVyRmluZ2VycHJpbnQiOiIwMDBjYmFhYi0wODk4LTQ5MzItYTA2Ni04YzVjZmY2Yzk3NTIifQ==
  licenseSignature: U2LQnSviNdmc7z44DYedZCdg0IGUXvOzk+9rJWu4Rqtph/6e8qBdk2suy0QgftT+NJOB3hdzw1QHSYauw1AZ5EAPMcooSa6P0VVt/Fk9ceJgP+xS63SNogqbSaQadM1GoG5EzF5DpuNvNMxFLXJ9uXCOdt4N/G4k5O7oLTArnkp8cF0897SHnaeUuQBJzsswL4ZYbB3ikLRdXpr/ZW7+rmYc7Y/KpCcAb65LxPWCy28gJm1GP1olM9WkKuiR1z1jOLSgx0UhyATaDxNa60Jkk4RWaTlJMjXNxuEaZ7ptFlrcpV+d/3mSbWlNFKu5/u1iEMO9Jw4BHWKUvT8fpXw1FY07CRdcbHvtOn7Brm2SZCrTmzjFvS1STtgVLErd12f0dqpQ70XNz0q1BlHLVroyMzDFyksonne1KA9GhCNtjRzsXQzImdU/rJYv1bxoWK1c7hSJB4Bht4Gg75WVsjoLIhW0ME9b0E+j/LFukFIVeqb4j+QGDmDJhgvwLvxHl8XZqH/8kyn6rK29sro0Y7Kiqr3NkEsiwQSqj3Rf0nVyutpQBhW5aqEM5eyh1IY8nLYX7gGS3h7FfiPw61tj664w/PwTHIjwsvkpJ9hGbikadDBd3Ihj9Al9x5kCTkNmBpM07+UK3vK2HFYFDSZdAjWmMQx0O1suANKvl0kUu8ptYsw=`

		u, err := k8s_utils.GetUnstructured([]byte(secret))
		Expect(err).To(BeNil())

		initObjects := []client.Object{
			u, ns,
		}

		c := fake.NewClientBuilder().WithObjects(initObjects...).Build()

		publicKey, err := license.GetPublicKeyFromString()
		Expect(err).To(BeNil())

		payload, err := license.VerifyLicenseSecret(context.TODO(), c, secretInfo, publicKey, logger)
		Expect(err).To(BeNil())

		Expect(payload).ToNot(BeNil())
		Expect(payload.MaxClusters).To(Equal(1))

		currentNs := &corev1.Namespace{}
		Expect(c.Get(context.TODO(), types.NamespacedName{Name: "kube-system"}, currentNs)).To(Succeed())
		currentNs.UID = "000cbaab-1234-4932-a111-8c5cff6c9752"
		Expect(c.Update(context.TODO(), currentNs)).To(Succeed())

		_, err = license.VerifyLicenseSecret(context.TODO(), c, secretInfo, publicKey, logger)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("license is not valid for this cluster (fingerprint mismatch)"))
	})
})
