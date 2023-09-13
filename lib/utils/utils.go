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

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	apiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/kubectl/pkg/scheme"

	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

var (
	serverWithPortRegExp    = regexp.MustCompile(`^https://[0-9a-zA-Z][0-9a-zA-Z-.]+[0-9a-zA-Z]:\d+$`)
	serverWithoutPortRegExp = regexp.MustCompile(`^https://[0-9a-zA-Z][0-9a-zA-Z-.]+[0-9a-zA-Z]$`)
)

// GetUnstructured returns an unstructured given a []bytes containing it
func GetUnstructured(object []byte) (*unstructured.Unstructured, error) {
	request := &unstructured.Unstructured{}
	universalDeserializer := scheme.Codecs.UniversalDeserializer()
	_, _, err := universalDeserializer.Decode(object, nil, request)
	if err != nil {
		return nil, fmt.Errorf("failed to decode k8s resource %.50s. Err: %w",
			string(object), err)
	}

	return request, nil
}

// GetDynamicResourceInterface returns a dynamic ResourceInterface for the policy's GroupVersionKind
func GetDynamicResourceInterface(config *rest.Config, gvk schema.GroupVersionKind,
	namespace string) (dynamic.ResourceInterface, error) {

	if config == nil {
		return nil, fmt.Errorf("rest.Config is nil")
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = dynClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		// for cluster-wide resources
		dr = dynClient.Resource(mapping.Resource)
	}

	return dr, nil
}

// GetKubeconfigWithUserToken() accepts a k8s client interface, OIDC IDToken,
// username (e.g. user@example.org) and server with scheme, host, and port
// (e.g.  https://127.0.0.1:1234) to return a kube-config with credentials set
// for the user.
func GetKubeconfigWithUserToken(ctx context.Context, idToken, caData []byte,
	userID, server string) ([]byte, error) {

	// Input validations.
	if userID == "" || len(idToken) == 0 {
		return nil,
			fmt.Errorf("userID and IDToken cannot be empty")
	}

	return getUserOrSAKubeconfig(idToken, caData, userID, server)
}

// getUserOrSAKubeconfig() is a helper to return kubeconfig given the required
// details for ServiceAccount or User/Token use-case.
func getUserOrSAKubeconfig(
	token, caData []byte,
	user, server string,
) ([]byte, error) {

	if server == "" {
		return nil, fmt.Errorf("server cannot be empty")
	}

	if !serverWithPortRegExp.MatchString(server) && !serverWithoutPortRegExp.MatchString(server) {
		return nil,
			fmt.Errorf("server value is invalid. valid values e.g. https://127.0.0.1:123, https://hostname:321, https://127.0.0.1")
	}

	config := apiv1.Config{
		Kind:       "Config",
		APIVersion: "v1",
		AuthInfos: []apiv1.NamedAuthInfo{
			{
				Name: user,
				AuthInfo: apiv1.AuthInfo{
					Token: string(token),
				},
			},
		},
		Clusters: []apiv1.NamedCluster{
			{
				Name: user,
				Cluster: apiv1.Cluster{
					Server:                   server,
					CertificateAuthorityData: caData,
				},
			},
		},
		Contexts: []apiv1.NamedContext{
			{
				Name: user,
				Context: apiv1.Context{
					Cluster:  user,
					AuthInfo: user,
				},
			},
		},
		CurrentContext: user,
		Preferences:    apiv1.Preferences{},
	}

	if caData == nil {
		return nil, fmt.Errorf("empty CA data")
	}

	kubeconfig, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("error creating kubeconfig: %w", err)
	}

	return kubeconfig, nil
}

func GetKubernetesVersion(ctx context.Context, cfg *rest.Config, logger logr.Logger) (string, error) {
	discClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get discovery client: %v", err))
		return "", err
	}

	var k8sVersion *version.Info
	k8sVersion, err = discClient.ServerVersion()
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to get version from discovery client: %v", err))
		return "", err
	}

	logger.V(logs.LogDebug).Info(fmt.Sprintf("cluster version: %s", k8sVersion.String()))
	return k8sVersion.String(), nil
}
