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

package clusterproxy

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

const (
	//nolint: gosec // CAPI secret postfix
	kubeconfigSecretNamePostfix = "-kubeconfig"
)

// GetKubernetesRestConfig returns rest.Config for a CAPI Cluster clusterNamespace/clusterName
// c is the client to access management cluster
func GetKubernetesRestConfig(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) (*rest.Config, error) {

	kubeconfigContent, err := GetSecretData(ctx, logger, c, clusterNamespace, clusterName)
	if err != nil {
		return nil, err
	}

	kubeconfig, err := CreateKubeconfig(logger, kubeconfigContent)
	if err != nil {
		return nil, err
	}
	defer os.Remove(kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Error(err, "BuildConfigFromFlags")
		return nil, errors.Wrap(err, "BuildConfigFromFlags")
	}

	return config, nil
}

// GetKubernetesClient returns a client to access CAPI Cluster clusterNamespace/clusterName
// c is the client to access management cluster
func GetKubernetesClient(ctx context.Context, logger logr.Logger, c client.Client,
	s *runtime.Scheme, clusterNamespace, clusterName string) (client.Client, error) {

	config, err := GetKubernetesRestConfig(ctx, logger, c, clusterNamespace, clusterName)
	if err != nil {
		return nil, err
	}
	logger.V(logs.LogVerbose).Info("return new client")
	return client.New(config, client.Options{Scheme: s})
}

// GetSecretData verifies Cluster exists and returns the content of secret containing
// the kubeconfig for CAPI cluster
func GetSecretData(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) ([]byte, error) {

	logger.WithValues("namespace", clusterNamespace, "cluster", clusterName)
	logger.V(logs.LogVerbose).Info("Get secret")
	key := client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      clusterName,
	}

	cluster := clusterv1.Cluster{}
	if err := c.Get(ctx, key, &cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Cluster does not exist")
			return nil, errors.Wrap(err,
				fmt.Sprintf("Cluster %s/%s does not exist",
					clusterNamespace,
					clusterName,
				))
		}
		return nil, err
	}

	secretName := cluster.Name + kubeconfigSecretNamePostfix
	logger = logger.WithValues("secret", secretName)

	secret := &corev1.Secret{}
	key = client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      secretName,
	}

	if err := c.Get(ctx, key, secret); err != nil {
		logger.Error(err, "failed to get secret")
		return nil, errors.Wrap(err,
			fmt.Sprintf("Failed to get secret %s/%s",
				clusterNamespace, secretName))
	}

	for k, contents := range secret.Data {
		logger.V(logs.LogVerbose).Info("Reading secret", "key", k)
		return contents, nil
	}

	return nil, nil
}

// CreateKubeconfig creates a temporary file with the Kubeconfig to access CAPI cluster
func CreateKubeconfig(logger logr.Logger, kubeconfigContent []byte) (string, error) {
	tmpfile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		logger.Error(err, "failed to create temporary file")
		return "", errors.Wrap(err, "os.CreateTemp")
	}
	defer tmpfile.Close()

	if _, err := tmpfile.Write(kubeconfigContent); err != nil {
		logger.Error(err, "failed to write to temporary file")
		return "", errors.Wrap(err, "failed to write to temporary file")
	}

	return tmpfile.Name(), nil
}
