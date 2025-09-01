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

package k8s_utils

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	apiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

var (
	serverWithPortRegExp    = regexp.MustCompile(`^https://[0-9a-zA-Z][0-9a-zA-Z-.]+[0-9a-zA-Z]:\d+$`)
	serverWithoutPortRegExp = regexp.MustCompile(`^https://[0-9a-zA-Z][0-9a-zA-Z-.]+[0-9a-zA-Z]$`)
)

// GetUnstructured returns an unstructured given a []bytes containing it
func GetUnstructured(object []byte) (*unstructured.Unstructured, error) {
	var out map[string]interface{}
	err := yaml.Unmarshal(object, &out)
	if err != nil {
		return nil, fmt.Errorf("failed to decode k8s resource %.50s. Err: %w",
			string(object), err)
	}
	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(out)

	return u, nil
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

// AddOwnerReference adds Sveltos resource owning a resource as an object's OwnerReference.
// OwnerReferences are used as ref count. Different Sveltos resources might match same cluster and
// reference same ConfigMap. This means a policy contained in a ConfigMap is deployed in a Cluster
// because of different Sveltos resources.
// When cleaning up, a policy can be removed only if no more Sveltos resources are listed as OwnerReferences.
func AddOwnerReference(object, owner client.Object) {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		onwerReferences = make([]metav1.OwnerReference, 0)
	}

	for i := range onwerReferences {
		ref := &onwerReferences[i]
		if ref.Kind == owner.GetObjectKind().GroupVersionKind().Kind &&
			ref.Name == owner.GetName() {

			return
		}
	}

	apiVersion, kind := owner.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()

	onwerReferences = append(onwerReferences,
		metav1.OwnerReference{
			APIVersion: apiVersion,
			Kind:       kind,
			Name:       owner.GetName(),
			UID:        owner.GetUID(),
		},
	)

	object.SetOwnerReferences(onwerReferences)
}

// RemoveOwnerReference removes Sveltos resource as an OwnerReference from object.
// OwnerReferences are used as ref count. Different Sveltos resources might match same cluster and
// reference same ConfigMap. This means a policy contained in a ConfigMap is deployed in a Cluster
// because of different SveltosResources. When cleaning up, a policy can be removed only if no more
// SveltosResources are listed as OwnerReferences.
func RemoveOwnerReference(object, owner client.Object) {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		return
	}

	_, kind := owner.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()

	for i := range onwerReferences {
		ref := &onwerReferences[i]
		if ref.Kind == kind &&
			ref.Name == owner.GetName() {

			onwerReferences[i] = onwerReferences[len(onwerReferences)-1]
			onwerReferences = onwerReferences[:len(onwerReferences)-1]
			break
		}
	}

	object.SetOwnerReferences(onwerReferences)
}

// IsOnlyOwnerReference returns true if clusterprofile is the only ownerreference for object
func IsOnlyOwnerReference(object, owner client.Object) bool {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		return false
	}

	if len(onwerReferences) != 1 {
		return false
	}

	kind := owner.GetObjectKind().GroupVersionKind().Kind

	ref := &onwerReferences[0]
	return ref.Kind == kind &&
		ref.Name == owner.GetName()
}

// IsOwnerReference returns true is owner is one of the OwnerReferences
// for object
func IsOwnerReference(object, owner client.Object) bool {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		return false
	}

	kind := owner.GetObjectKind().GroupVersionKind().Kind

	for i := range onwerReferences {
		ref := &onwerReferences[i]
		if ref.Kind == kind &&
			ref.Name == owner.GetName() {

			return true
		}
	}

	return false
}

// HasSveltosResourcesAsOwnerReference returns true if at least one
// of current OwnerReferences is a Sveltos resource
func HasSveltosResourcesAsOwnerReference(object client.Object) bool {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		return false
	}

	for i := range onwerReferences {
		ref := &onwerReferences[i]
		if strings.Contains(ref.APIVersion, "projectsveltos.io") {
			return true
		}
	}

	return false
}
