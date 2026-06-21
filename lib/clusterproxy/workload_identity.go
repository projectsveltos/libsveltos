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

package clusterproxy

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/aws/aws-sdk-go-v2/aws"
	signerv4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"golang.org/x/sync/singleflight"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

const (
	// wiRefreshThreshold is how much time before expiry we proactively refresh.
	wiRefreshThreshold = 5 * time.Minute

	//nolint:gosec // this is a token format prefix, not a credential
	eksBearerTokenPrefix = "k8s-aws-v1."

	// eksTokenTTL is the fixed TTL of EKS pre-signed STS tokens.
	eksTokenTTL = 15 * time.Minute

	// aksScope is the AAD scope required to authenticate against an AKS API server.
	aksScope = "6dae42f8-4368-4678-94ff-3960e28e3630/.default"

	gcpCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	caSecretKey = "ca.crt"
)

type cachedRestConfig struct {
	config    *rest.Config
	expiresAt time.Time
}

var (
	wiCache sync.Map           // map[string]cachedRestConfig
	wiGroup singleflight.Group // deduplicates concurrent refreshes for the same cluster
)

// EvictWorkloadIdentityCache removes the cached rest.Config for the given
// SveltosCluster. Call this from the SveltosCluster delete handler in any
// component that uses workload identity.
func EvictWorkloadIdentityCache(namespace, name string) {
	wiCache.Delete(wiCacheKey(namespace, name))
}

func wiCacheKey(namespace, name string) string {
	return namespace + "/" + name
}

// getWorkloadIdentityRestConfig returns a rest.Config for a SveltosCluster that
// uses cloud provider workload identity. Results are cached and proactively
// refreshed when they approach expiry; concurrent calls for the same cluster are
// collapsed into a single cloud API call via singleflight.
func getWorkloadIdentityRestConfig(
	ctx context.Context,
	c client.Client,
	clusterNamespace, clusterName string,
	wi *libsveltosv1beta1.WorkloadIdentityConfig,
	logger logr.Logger,
) (*rest.Config, error) {

	key := wiCacheKey(clusterNamespace, clusterName)

	// Fast path: valid cached entry.
	if v, ok := wiCache.Load(key); ok {
		entry := v.(cachedRestConfig)
		if time.Until(entry.expiresAt) > wiRefreshThreshold {
			return entry.config, nil
		}
	}

	// Slow path: fetch from cloud provider, deduplicated across concurrent callers.
	type result struct {
		cfg *rest.Config
	}
	val, err, _ := wiGroup.Do(key, func() (interface{}, error) {
		caData, err := getCAData(ctx, c, clusterNamespace, wi.CASecretRef, logger)
		if err != nil {
			return nil, err
		}

		var cfg *rest.Config
		var expiresAt time.Time

		switch wi.Provider {
		case libsveltosv1beta1.WorkloadIdentityProviderAWS:
			cfg, expiresAt, err = getAWSRestConfig(ctx, wi, caData, logger)
		case libsveltosv1beta1.WorkloadIdentityProviderGCP:
			cfg, expiresAt, err = getGCPRestConfig(ctx, wi, caData, logger)
		case libsveltosv1beta1.WorkloadIdentityProviderAzure:
			cfg, expiresAt, err = getAzureRestConfig(ctx, wi, caData, logger)
		default:
			err = fmt.Errorf("unknown workload identity provider %q", wi.Provider)
		}
		if err != nil {
			return nil, err
		}

		wiCache.Store(key, cachedRestConfig{config: cfg, expiresAt: expiresAt})
		return result{cfg: cfg}, nil
	})
	if err != nil {
		return nil, err
	}

	return val.(result).cfg, nil
}

// getCAData fetches the CA certificate bytes from the referenced Secret.
// If caSecretRef is nil, nil is returned and the system certificate pool is used.
func getCAData(
	ctx context.Context,
	c client.Client,
	namespace string,
	caSecretRef *corev1.LocalObjectReference,
	logger logr.Logger,
) ([]byte, error) {

	if caSecretRef == nil {
		return nil, nil
	}

	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: caSecretRef.Name}
	if err := c.Get(ctx, key, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errors.Wrap(err,
				fmt.Sprintf("CA secret %s/%s not found", namespace, caSecretRef.Name))
		}
		return nil, errors.Wrap(err,
			fmt.Sprintf("failed to get CA secret %s/%s", namespace, caSecretRef.Name))
	}

	ca, ok := secret.Data[caSecretKey]
	if !ok {
		return nil, fmt.Errorf("CA secret %s/%s has no %q key", namespace, caSecretRef.Name, caSecretKey)
	}
	logger.V(logs.LogDebug).Info("loaded CA from secret", "secret", caSecretRef.Name)
	return ca, nil
}

// buildRestConfig assembles a rest.Config from the common fields.
func buildRestConfig(endpoint, bearerToken string, caData []byte) *rest.Config {
	cfg := &rest.Config{
		Host:        endpoint,
		BearerToken: bearerToken,
	}
	cfg.CAData = caData
	return cfg
}

// ── AWS ──────────────────────────────────────────────────────────────────────

// eksHTTPPresigner is a standalone SigV4 presigner that adds the x-k8s-aws-id
// header required by the EKS authentication flow before signing.
// It does not wrap the default STS presigner — instead it uses signerv4.Signer
// directly so it never holds a nil interface (the default presigner is only
// initialized later in the STS presign middleware stack).
type eksHTTPPresigner struct {
	signer      *signerv4.Signer
	clusterName string
}

//nolint:gocritic // creds is passed by value to satisfy the sts.HTTPPresignerV4 interface
func (p *eksHTTPPresigner) PresignHTTP(
	ctx context.Context,
	creds aws.Credentials,
	r *http.Request,
	payloadHash, service, region string,
	signingTime time.Time,
	optFns ...func(*signerv4.SignerOptions),
) (string, http.Header, error) {

	// X-Amz-Expires must be in the signed URL; the STS middleware does not add
	// it when a custom presigner is used. EKS rejects tokens without it.
	q := r.URL.Query()
	q.Set("X-Amz-Expires", "900")
	r.URL.RawQuery = q.Encode()

	r.Header.Set("x-k8s-aws-id", p.clusterName)
	return p.signer.PresignHTTP(ctx, creds, r, payloadHash, service, region, signingTime, optFns...)
}

func getAWSRestConfig(
	ctx context.Context,
	wi *libsveltosv1beta1.WorkloadIdentityConfig,
	caData []byte,
	logger logr.Logger,
) (*rest.Config, time.Time, error) {

	awsCfg := wi.AWS
	region := awsCfg.Region
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			return nil, time.Time{}, errors.New("AWS region not specified and AWS_REGION env var is not set")
		}
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, time.Time{}, errors.Wrap(err, "failed to load AWS config")
	}

	// If a role ARN is provided, assume that role before generating the EKS token.
	if awsCfg.RoleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		provider := stscreds.NewAssumeRoleProvider(stsClient, awsCfg.RoleARN)
		cfg.Credentials = aws.NewCredentialsCache(provider)
		logger.V(logs.LogDebug).Info("assuming AWS role", "roleARN", awsCfg.RoleARN)
	}

	stsOpts := sts.Options{
		Region:      region,
		Credentials: cfg.Credentials,
	}
	presignClient := sts.NewPresignClient(sts.New(stsOpts), func(po *sts.PresignOptions) {
		po.Presigner = &eksHTTPPresigner{
			signer:      signerv4.NewSigner(),
			clusterName: awsCfg.ClusterName,
		}
	})

	presigned, err := presignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, time.Time{}, errors.Wrap(err, "failed to presign EKS bearer token")
	}

	token := eksBearerTokenPrefix + base64.RawURLEncoding.EncodeToString([]byte(presigned.URL))
	expiresAt := time.Now().Add(eksTokenTTL)

	logger.V(logs.LogDebug).Info("obtained AWS EKS bearer token",
		"cluster", awsCfg.ClusterName, "expiresAt", expiresAt)
	return buildRestConfig(wi.Endpoint, token, caData), expiresAt, nil
}

// ── GCP ──────────────────────────────────────────────────────────────────────

func getGCPRestConfig(
	ctx context.Context,
	wi *libsveltosv1beta1.WorkloadIdentityConfig,
	caData []byte,
	logger logr.Logger,
) (*rest.Config, time.Time, error) {

	src, err := google.DefaultTokenSource(ctx, gcpCloudPlatformScope)
	if err != nil {
		return nil, time.Time{}, errors.Wrap(err, "failed to obtain GCP token source")
	}

	token, err := src.Token()
	if err != nil {
		return nil, time.Time{}, errors.Wrap(err, "failed to obtain GCP access token")
	}

	logger.V(logs.LogDebug).Info("obtained GCP access token", "expiresAt", token.Expiry)
	return buildRestConfig(wi.Endpoint, token.AccessToken, caData), token.Expiry, nil
}

// ── Azure ─────────────────────────────────────────────────────────────────────

func getAzureRestConfig(
	ctx context.Context,
	wi *libsveltosv1beta1.WorkloadIdentityConfig,
	caData []byte,
	logger logr.Logger,
) (*rest.Config, time.Time, error) {

	azureCfg := wi.Azure

	// The Azure workload identity webhook injects AZURE_FEDERATED_TOKEN_FILE.
	tokenFile := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	if tokenFile == "" {
		return nil, time.Time{}, errors.New(
			"AZURE_FEDERATED_TOKEN_FILE env var is not set; " +
				"ensure the pod is configured with Azure Workload Identity")
	}

	getAssertion := func(_ context.Context) (string, error) {
		//nolint:gosec // path comes from the Azure workload identity webhook, not user input
		b, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", fmt.Errorf("failed to read federated token file %s: %w", tokenFile, err)
		}
		return string(b), nil
	}

	cred, err := azidentity.NewClientAssertionCredential(
		azureCfg.TenantID, azureCfg.ClientID, getAssertion, nil)
	if err != nil {
		return nil, time.Time{}, errors.Wrap(err, "failed to create Azure client assertion credential")
	}

	tokenResp, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{aksScope},
	})
	if err != nil {
		return nil, time.Time{}, errors.Wrap(err, "failed to obtain Azure AAD token for AKS")
	}

	logger.V(logs.LogDebug).Info("obtained Azure AAD token", "expiresAt", tokenResp.ExpiresOn)
	return buildRestConfig(wi.Endpoint, tokenResp.Token, caData), tokenResp.ExpiresOn, nil
}
