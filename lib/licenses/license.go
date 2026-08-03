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

const (
	// LicensePayloadAnnotation is the annotation key under which a signed license's payload
	// bytes (base64-encoded) can be relayed onto another object (e.g. SveltosCluster), for
	// components that cannot reach the sveltos-license Secret directly.
	LicensePayloadAnnotation = "license.projectsveltos.io/payload"

	// LicenseSignatureAnnotation is the annotation key under which a signed license's
	// signature bytes (base64-encoded) can be relayed alongside LicensePayloadAnnotation.
	LicenseSignatureAnnotation = "license.projectsveltos.io/signature"
)

// Features is all features requiring a license
// +kubebuilder:validation:Enum:=PullMode;MCP
type Features string

const (
	// FeaturePullMode is the ability to manage cluster behing firewalls
	FeaturePullMode = Features("PullMode")

	// FeatureMCP is the ability to use Sveltos MCP Server
	FeatureMCP = Features("MCP")

	// FeaturePromotion is the ability to use Sveltos ClusterPromotion
	FeaturePromotion = Features("Promotion")

	// FeatureJobHealthCheck is the ability to use a Job as a ValidateHealth check
	FeatureJobHealthCheck = Features("JobHealthCheck")
)

// +kubebuilder:validation:Enum:=Enterprise;EnterprisePlus
type Plan string

const (
	PlanEnterprise = Plan("Enterprise")

	PlanEnterprisePlus = Plan("EnterprisePlus")
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

// HasFeature reports whether feature is allowed by this license.
//
// An empty Features list means the license predates (or was issued without) per-feature
// restriction: every feature is allowed, and callers should fall back to their own Plan/
// MaxClusters-based checks. A non-empty Features list is an explicit allowlist — it must
// contain feature — and this takes precedence over any other bypass (e.g. MaxClusters == 0
// for unlimited clusters, or Plan tier): a license scoped to specific features stays scoped
// to them regardless of what other entitlements it grants.
func (lp *LicensePayload) HasFeature(feature Features) bool {
	if len(lp.Features) == 0 {
		return true
	}

	for i := range lp.Features {
		if lp.Features[i] == feature {
			return true
		}
	}

	return false
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

// LicenseVerificationResult encapsulates the outcome of the license verification.
type LicenseVerificationResult struct {
	Payload         *LicensePayload // The decoded license payload if found and unmarshaled
	PayloadData     []byte          // The exact signed payload bytes, set once the signature has verified
	SignatureData   []byte          // The signature bytes matching PayloadData
	IsValid         bool            // True if license is fully valid
	IsExpired       bool            // True if license is expired (either grace or enforced)
	IsInGracePeriod bool            // True if license is expired but within grace period
	IsEnforced      bool            // True if license is expired and fully enforced
	Message         string          // A human-readable message about the license status
	RawError        error           // The underlying error (e.g., secret not found, unmarshal error, signature error)
}

// VerifyLicenseSecret attempts to decode and verify the license secret.
// It returns a LicenseVerificationResult struct containing the license payload (if found)
// and various booleans indicating its validity status, along with a human-readable message.
// The RawError field will contain any technical errors encountered during the process.
// Requires permission to read Secret in projectsveltos namespace.
func VerifyLicenseSecret(ctx context.Context, c client.Client, sveltosNamespace string,
	publicKey *rsa.PublicKey, logger logr.Logger) LicenseVerificationResult {

	secretNsName := types.NamespacedName{
		Namespace: sveltosNamespace,
		Name:      "sveltos-license",
	}

	result := LicenseVerificationResult{}

	licenseSecret := &corev1.Secret{}
	err := c.Get(ctx, secretNsName, licenseSecret)
	if err != nil {
		result.IsExpired = true
		result.IsEnforced = true
		result.RawError = err
		if apierrors.IsNotFound(err) {
			result.Message = fmt.Sprintf("License secret '%s' not found.", secretNsName.String())
		} else {
			result.Message = fmt.Sprintf("Failed to get license secret '%s': %v", secretNsName.String(), err)
		}
		logger.V(logs.LogInfo).Info(fmt.Sprintf("%s: %v", result.Message, err))
		return result
	}

	payloadData, ok := licenseSecret.Data[licenseKey]
	if !ok {
		result.IsExpired = true
		result.IsEnforced = true
		result.Message = fmt.Sprintf("License secret '%s' is missing the %q key", secretNsName.String(), licenseKey)
		result.RawError = errors.New(result.Message)
		logger.V(logs.LogInfo).Info(result.Message)
		return result
	}

	signatureData, ok := licenseSecret.Data[signatureKey]
	if !ok {
		result.IsExpired = true
		result.IsEnforced = true
		result.Message = fmt.Sprintf("License secret '%s' is missing the %q key", secretNsName.String(), signatureKey)
		result.RawError = errors.New(result.Message)
		logger.V(logs.LogInfo).Info(result.Message)
		return result
	}

	result = VerifyLicensePayload(payloadData, signatureData, publicKey, logger)
	if result.Payload == nil {
		// Signature verification or unmarshaling failed. Reword the generic message to
		// mention the secret this payload came from.
		result.Message = fmt.Sprintf("%s (secret %s)", result.Message, secretNsName.String())
		return result
	}

	verifyClusterFingerprint(ctx, c, result.Payload, &result, logger)
	return result
}

// VerifyLicensePayload verifies a license's digital signature against publicKey and, if valid,
// unmarshals and checks its expiration. Unlike VerifyLicenseSecret, it does not read a Secret
// and does not check the cluster fingerprint (that check needs a client into the specific
// cluster the license is meant for) — callers that need fingerprint validation should use
// VerifyLicenseSecret, or call verifyClusterFingerprint-equivalent logic themselves.
// This is meant for components that receive the payload/signature bytes relayed from
// elsewhere (e.g. via LicensePayloadAnnotation/LicenseSignatureAnnotation) rather than
// reading the sveltos-license Secret directly.
func VerifyLicensePayload(payloadData, signatureData []byte, publicKey *rsa.PublicKey,
	logger logr.Logger) LicenseVerificationResult {

	result := LicenseVerificationResult{}

	// Verify the digital signature
	hashedPayload := sha256.Sum256(payloadData)
	err := rsa.VerifyPSS(publicKey, crypto.SHA256, hashedPayload[:], signatureData, nil)
	if err != nil {
		result.IsExpired = true
		result.IsEnforced = true
		result.Message = fmt.Sprintf("Digital signature verification failed for license: %v", err)
		result.RawError = err
		logger.V(logs.LogInfo).Info(result.Message, "error", err)
		return result
	}

	logger.V(logs.LogDebug).Info("Digital signature successfully verified for license")

	result.PayloadData = payloadData
	result.SignatureData = signatureData

	// Unmarshal the LicensePayload from the verified data
	var verifiedPayload LicensePayload
	if err := json.Unmarshal(payloadData, &verifiedPayload); err != nil {
		result.IsExpired = true
		result.IsEnforced = true
		result.Message = fmt.Sprintf("Failed to unmarshal verified license payload: %v", err)
		result.RawError = err
		logger.V(logs.LogInfo).Info(fmt.Sprintf("%s: %v", err, result.Message))
		return result
	}
	result.Payload = &verifiedPayload

	verifyExpirationDate(&verifiedPayload, &result, logger)
	return result
}

func verifyExpirationDate(verifiedPayload *LicensePayload, result *LicenseVerificationResult,
	logger logr.Logger) {

	// --- License Expiration and Grace Period Logic ---
	now := time.Now()
	actualGracePeriod := getActualGracePeriod(verifiedPayload)
	enforcementDate := verifiedPayload.ExpirationDate.Add(actualGracePeriod)

	if now.Before(verifiedPayload.ExpirationDate) {
		// License is valid
		result.IsValid = true
		result.Message = fmt.Sprintf("License is valid (expires %s).",
			verifiedPayload.ExpirationDate.Format(time.RFC3339))
		logger.V(logs.LogDebug).Info(result.Message)
	} else if now.After(verifiedPayload.ExpirationDate) && now.Before(enforcementDate) {
		daysLeftInGrace := enforcementDate.Sub(now).Hours() / 24
		result.IsExpired = true
		result.IsInGracePeriod = true
		result.Message = fmt.Sprintf(`License expired on %s, but is within grace period.
        Functionality will be enforced in approximately %.0f days.`,
			verifiedPayload.ExpirationDate.Format(time.RFC3339), daysLeftInGrace)
		logger.V(logs.LogInfo).Info(result.Message,
			"expiration_date", verifiedPayload.ExpirationDate.Format(time.RFC3339),
			"enforcement_date", enforcementDate.Format(time.RFC3339))
	} else {
		// License is fully expired and enforced
		result.IsExpired = true
		result.IsEnforced = true
		result.Message = fmt.Sprintf(`License has fully expired and is enforced.
            Expired %s, grace period ended %s.`,
			verifiedPayload.ExpirationDate.Format(time.RFC3339),
			enforcementDate.Format(time.RFC3339))
		logger.V(logs.LogInfo).Info(result.Message)
	}
}

func verifyClusterFingerprint(ctx context.Context, c client.Client, verifiedPayload *LicensePayload,
	result *LicenseVerificationResult, logger logr.Logger) {

	// --- Cluster Fingerprint Validation ---
	if result.IsValid && verifiedPayload.ClusterFingerprint != "" &&
		!isClusterFingerprintValid(ctx, c, verifiedPayload.ClusterFingerprint, logger) {
		// If cluster fingerprint is invalid, it overrides previous valid status
		result.IsValid = false
		result.IsExpired = true // Treat fingerprint mismatch as enforced expiration
		result.IsInGracePeriod = false
		result.IsEnforced = true
		result.Message = "License is not valid for this cluster (fingerprint mismatch)."
		// Clear RawError if it was nil, or wrap original error if present.
		if result.RawError == nil {
			result.RawError = errors.New(result.Message)
		} else {
			result.RawError = fmt.Errorf("%s: %w", result.Message, result.RawError)
		}
		logger.V(logs.LogInfo).Info(fmt.Sprintf("%s: %v", result.RawError, result.Message))
	}
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

func GetPublicKey() (*rsa.PublicKey, error) {
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
