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

package license

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

const (
	licenseKey   = "licenseData"
	signatureKey = "licenseSignature"
)

// Features is all features requiring a license
// +kubebuilder:validation:Enum:=PullMode
type Features string

const (
	// FeaturePullMode is the ability to manage cluster behing firewalls
	FeaturePullMode = Features("PullMode")
)

// +kubebuilder:validation:Enum:=Enterprise;EnterprisePlus
type Plan string

const (
	PlanEnterprise = Features("Enterprise")

	PlanEnterprisePlus = Features("EnterprisePlus")
)

// LicensePayload defines the internal structure of the data that gets signed
// and embedded within the Kubernetes Secret.
type LicensePayload struct {
	// ID is a unique identifier for this specific license.
	ID string `json:"id"`

	// CustomerName is the name of the customer the license is issued to.
	CustomerName string `json:"customerName"`

	// Features is a list of feature strings enabled by this license.
	Features []Features `json:"features"`

	// Specify the type of plan
	// +optional
	Plan Plan `json:"plan,omitempty"`

	// ExpirationDate is the exact time when the license expires.
	ExpirationDate time.Time `json:"expirationDate"`

	// GracePeriodDays specifies the number of days the license remains functional
	// after its expiration date, during which warnings are issued.
	// +optional
	GracePeriodDays int `json:"gracePeriodDays,omitempty"`

	// MaxClusters is the maximum number of clusters allowed for this license (optional).
	// +optional
	MaxClusters int `json:"maxClusters,omitempty"`

	// IssuedAt is the timestamp when the license was generated and signed.
	IssuedAt time.Time `json:"issuedAt"`

	// ClusterFingerprint is a unique identifier derived from the target Kubernetes cluster.
	// +optional
	ClusterFingerprint string `json:"clusterFingerprint,omitempty"`
}

// Custom error types to differentiate license states
type LicenseError struct {
	Status  LicenseStatus
	Message string
	Err     error
}

func (e *LicenseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s (status: %s): %v", e.Message, e.Status.String(), e.Err)
	}
	return fmt.Sprintf("%s (status: %s)", e.Message, e.Status.String())
}

func (e *LicenseError) Unwrap() error {
	return e.Err
}

// LicenseStatus represents the state of the license (copy from previous example or import)
type LicenseStatus int

const (
	LicenseStatusValid LicenseStatus = iota
	LicenseStatusExpiredGracePeriod
	LicenseStatusExpiredEnforced
)

// String method for better printing of status
func (s LicenseStatus) String() string {
	switch s {
	case LicenseStatusValid:
		return "Valid"
	case LicenseStatusExpiredGracePeriod:
		return "Expired (Grace Period)"
	case LicenseStatusExpiredEnforced:
		return "Expired (Enforced)"
	default:
		return "Unknown"
	}
}

// GetActualGracePeriod calculates the effective grace period duration.
// It uses GracePeriodDays if set and positive, otherwise it uses the DefaultGracePeriod.
func getActualGracePeriod(lp *LicensePayload) time.Duration {
	// defaultGracePeriod is the fallback duration if GracePeriodDays is not specified or is 0.
	const defaultGracePeriod = time.Hour * 24 * 7 // One week

	if lp.GracePeriodDays > 0 {
		return time.Duration(lp.GracePeriodDays) * 24 * time.Hour
	}
	return defaultGracePeriod
}

// Returns the decoded and verified LicensePayload, or a LicenseError if validation fails,
// indicating the license status.
func VerifyLicenseSecret(ctx context.Context, c client.Client, secretNsName types.NamespacedName,
	publicKey *rsa.PublicKey, logger logr.Logger) (*LicensePayload, error) {

	licenseSecret := &corev1.Secret{}
	err := c.Get(ctx, secretNsName, licenseSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get license secret '%s': %w", secretNsName.String(), err)
	}

	payload, ok := licenseSecret.Data[licenseKey]
	if !ok {
		msg := fmt.Sprintf("license secret '%s' is missing the %q key",
			secretNsName.String(), licenseKey)
		return nil, &LicenseError{
			Status:  LicenseStatusExpiredEnforced,
			Message: msg,
			Err:     errors.New(msg),
		}
	}

	signature, ok := licenseSecret.Data[signatureKey]
	if !ok {
		msg := fmt.Sprintf("license secret '%s' is missing the %q key",
			secretNsName.String(), signatureKey)
		return nil, &LicenseError{
			Status:  LicenseStatusExpiredEnforced,
			Message: msg,
			Err:     errors.New(msg),
		}
	}

	// Verify the digital signature
	hashedPayload := sha256.Sum256(payload)
	err = rsa.VerifyPSS(publicKey, crypto.SHA256, hashedPayload[:], signature, nil)
	if err != nil {
		logger.V(logs.LogInfo).Info(
			fmt.Sprintf("Digital signature verification failed for license from secret %s: %v",
				secretNsName.String(), err))
		return nil, &LicenseError{
			Status:  LicenseStatusExpiredEnforced,
			Message: fmt.Sprintf("digital signature verification failed for secret '%s'", secretNsName.String()),
			Err:     err,
		}
	}

	logger.V(logs.LogInfo).Info("Digital signature successfully verified for license from secret")

	var verifiedPayload LicensePayload
	if err := json.Unmarshal(payload, &verifiedPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal verified license payload from secret %q: %w",
			secretNsName.String(), err)
	}

	now := time.Now()

	// Calculate the enforcement date (ExpirationDate + GracePeriod)
	enforcementDate := verifiedPayload.ExpirationDate.Add(getActualGracePeriod(&verifiedPayload))

	if now.Before(verifiedPayload.ExpirationDate) {
		logger.V(logs.LogInfo).Info("License is valid",
			"expires", verifiedPayload.ExpirationDate.Format(time.RFC3339))
	} else if now.After(verifiedPayload.ExpirationDate) && now.Before(enforcementDate) {
		daysLeftInGrace := enforcementDate.Sub(now).Hours() / 24
		msg := fmt.Sprintf(`License expired on %s, but is within grace period.
		Functionality will be enforced in approximately %.0f days.`,
			verifiedPayload.ExpirationDate.Format(time.RFC3339), daysLeftInGrace)
		logger.V(logs.LogInfo).Info(msg,
			"expiration_date", verifiedPayload.ExpirationDate.Format(time.RFC3339),
			"enforcement_date", enforcementDate.Format(time.RFC3339))

		return &verifiedPayload, &LicenseError{
			Status:  LicenseStatusExpiredGracePeriod,
			Message: msg,
		}
	} else {
		msg := fmt.Sprintf(`License has fully expired and is enforced.
			Expired %s, grace period ended %s.`,
			verifiedPayload.ExpirationDate.Format(time.RFC3339),
			enforcementDate.Format(time.RFC3339))
		logger.V(logs.LogInfo).Error(nil, msg)
		return &verifiedPayload, &LicenseError{
			Status:  LicenseStatusExpiredEnforced,
			Message: msg,
		}
	}

	if verifiedPayload.ClusterFingerprint != "" &&
		!isClusterFingerprintValid(ctx, c, verifiedPayload.ClusterFingerprint, logger) {

		return &verifiedPayload, &LicenseError{
			Status:  LicenseStatusExpiredEnforced,
			Message: "license is not valid for this cluster (fingerprint mismatch)",
		}
	}

	return &verifiedPayload, nil
}

func isClusterFingerprintValid(ctx context.Context, c client.Client, clusterFingerprint string,
	logger logr.Logger) bool {

	ns := &corev1.Namespace{}
	err := c.Get(ctx, types.NamespacedName{Name: "kube-system"}, ns)
	if err != nil {
		logger.V(logs.LogInfo).Info("failed to get kube-system namespace: %v", err)
		return false
	}

	return ns.UID == types.UID(clusterFingerprint)
}

func GetPublicKeyFromString() (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicLicense))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from public key string")
	}

	var pubKey interface{}
	var err error

	// Try parsing as PKCS#1 public key (BEGIN RSA PUBLIC KEY)
	pubKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		// If PKCS#1 fails, try parsing as PKCS#8 public key (BEGIN PUBLIC KEY)
		pubKey, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: neither PKCS#1 nor PKCS#8 format recognized: %w", err)
		}
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("parsed public key is not an RSA public key")
	}

	return rsaPubKey, nil
}

var (
	publicLicense = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA5VYCI5qa5MdKf70OmaGI
o/tNkzPuQh8VwW7NMFkJheRlYbFxixFZYUG8vFsWasw3/RnHgc3Y5V7biJxKjCQo
w6m8J3d0IFB5nqaZjzGog0zTuMX5t3G3GdgGK3LpBfcmHJiRbVM+pEFTEL6TgBeV
lh/cIvN9rKqrYg3vTwTl7B0XYlgiiUOKE5CczL0Tj9SjMRUoFdeXEPgzFpVwSS4H
kQMQiswRsHOzF8tRj6GmFo3A8EJO56vBNwlDyYn14brHfcyu5cKI9YMBXVXjkQH0
EeDvC72vCueiD7Q8lqNSSavmyCm4yJX+q0E9MEdIpbfVI4h6s5/LHFCYEDGQ8zAp
nFdOAn9JL0F+GGDmeBljTTWyjdUu55UUWDYjXfSViYe+ZrER92BU13F4OhJUZOK3
/fD/DsLHyeN4GPcAkwRPsG45if1rjmL+26ymUxkSTdP4dHNvSQRM9NMmx/PlVh/Q
ItaaVm0heRxhNqm48ex0c2IdR4PLGIvtAnejv6y2rY8tzprd7Ktq/dvNGxDSrapc
BbX1LW0IG5XbgSJhLHOEzr/RpmN5otEcmhokX1aniYwyhS62UrRtYEijxLe0R7Ri
b+8ezq9wt48da3VJOmTiiB3tu/SLVPKIxvgnKq49x3azPL4Qzl5bqL9iRT6LgCMM
NkuUxS7aVFAQrSvFFxc3c9kCAwEAAQ==
-----END PUBLIC KEY-----`
)
