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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	utilkubeconfig "sigs.k8s.io/cluster-api/util/kubeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

const (
	//nolint: gosec // CAPI secret postfix
	capiKubeconfigSecretNamePostfix = "-kubeconfig"

	//nolint: gosec // Sveltos secret postfix
	sveltosKubeconfigSecretNamePostfix = "-sveltos-kubeconfig"
)

// GetCAPIKubernetesRestConfig returns rest.Config for a CAPI Cluster clusterNamespace/clusterName
// c is the client to access management cluster
func GetCAPIKubernetesRestConfig(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) (*rest.Config, error) {

	kubeconfigContent, err := GetCAPISecretData(ctx, logger, c, clusterNamespace, clusterName)
	if err != nil {
		return nil, err
	}

	kubeconfig, closer, err := CreateKubeconfig(logger, kubeconfigContent)
	if err != nil {
		return nil, err
	}
	defer closer()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Error(err, "BuildConfigFromFlags")
		return nil, errors.Wrap(err, "BuildConfigFromFlags")
	}

	return config, nil
}

// GetCAPIKubernetesClient returns a client to access CAPI Cluster clusterNamespace/clusterName
// c is the client to access management cluster
func GetCAPIKubernetesClient(ctx context.Context, logger logr.Logger, c client.Client,
	s *runtime.Scheme, clusterNamespace, clusterName string) (client.Client, error) {

	config, err := GetCAPIKubernetesRestConfig(ctx, logger, c, clusterNamespace, clusterName)
	if err != nil {
		return nil, err
	}
	logger.V(logs.LogVerbose).Info("return new client")
	return client.New(config, client.Options{Scheme: s})
}

// GetCAPISecretData verifies Cluster exists and returns the content of secret containing
// the kubeconfig for CAPI cluster
func GetCAPISecretData(ctx context.Context, logger logr.Logger, c client.Client,
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

	return utilkubeconfig.FromSecret(ctx, c, key)
}

// GetSveltosKubernetesRestConfig returns rest.Config for a Sveltos Cluster clusterNamespace/clusterName
// c is the client to access management cluster
func GetSveltosKubernetesRestConfig(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) (*rest.Config, error) {

	kubeconfigContent, err := GetSveltosSecretData(ctx, logger, c, clusterNamespace, clusterName)
	if err != nil {
		return nil, err
	}

	kubeconfig, closer, err := CreateKubeconfig(logger, kubeconfigContent)
	if err != nil {
		return nil, err
	}
	defer closer()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Error(err, "BuildConfigFromFlags")
		return nil, errors.Wrap(err, "BuildConfigFromFlags")
	}

	return config, nil
}

// GetSveltosKubernetesClient returns a client to access Sveltos Cluster clusterNamespace/clusterName
// c is the client to access management cluster
func GetSveltosKubernetesClient(ctx context.Context, logger logr.Logger, c client.Client,
	s *runtime.Scheme, clusterNamespace, clusterName string) (client.Client, error) {

	config, err := GetSveltosKubernetesRestConfig(ctx, logger, c, clusterNamespace, clusterName)
	if err != nil {
		return nil, err
	}
	logger.V(logs.LogVerbose).Info("return new client")
	return client.New(config, client.Options{Scheme: s})
}

// GetSveltosSecretNameAndKey returns the name of the Secret containing the Kubeconfig
// for the SveltosCluster. If a key is specified, returns the name of the key to use.
func GetSveltosSecretNameAndKey(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) (secretName, secretkey string, err error) {

	logger.WithValues("namespace", clusterNamespace, "cluster", clusterName)
	logger.V(logs.LogVerbose).Info("Get secret name")
	key := client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      clusterName,
	}

	cluster := libsveltosv1beta1.SveltosCluster{}
	if err := c.Get(ctx, key, &cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("SveltosCluster does not exist")
			return "", "", errors.Wrap(err,
				fmt.Sprintf("SveltosCluster %s/%s does not exist",
					clusterNamespace,
					clusterName,
				))
		}
		return "", "", err
	}

	secretName = cluster.Spec.KubeconfigName
	if secretName == "" {
		secretName = fmt.Sprintf("%s%s", cluster.Name, sveltosKubeconfigSecretNamePostfix)
	}

	return secretName, cluster.Spec.KubeconfigKeyName, nil
}

// GetSveltosSecretData verifies Cluster exists and returns the content of secret containing
// the kubeconfig for Sveltos cluster.
func GetSveltosSecretData(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) ([]byte, error) {

	logger.WithValues("namespace", clusterNamespace, "cluster", clusterName)
	logger.V(logs.LogVerbose).Info("Get secret")
	secretName, secretKey, err := GetSveltosSecretNameAndKey(ctx, logger, c, clusterNamespace, clusterName)
	if err != nil {
		return nil, err
	}

	data, err := getSecretData(ctx, logger, c, clusterNamespace, secretName, secretKey)
	return data, err
}

// UpdateSveltosSecretData updates the content of the secret containing the
// the kubeconfig for Sveltos cluster
func UpdateSveltosSecretData(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName, kubeconfig, kubeconfigKey string) error {

	logger.WithValues("namespace", clusterNamespace, "cluster", clusterName)
	logger.V(logs.LogVerbose).Info("Get secret")
	key := client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      clusterName,
	}

	cluster := libsveltosv1beta1.SveltosCluster{}
	if err := c.Get(ctx, key, &cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("SveltosCluster does not exist")
			return errors.Wrap(err,
				fmt.Sprintf("SveltosCluster %s/%s does not exist",
					clusterNamespace,
					clusterName,
				))
		}
		return err
	}

	secretName := cluster.Spec.KubeconfigName
	if secretName == "" {
		secretName = fmt.Sprintf("%s%s", cluster.Name, sveltosKubeconfigSecretNamePostfix)
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      secretName,
	}

	if err := c.Get(ctx, secretKey, secret); err != nil {
		logger.Error(err, "failed to get secret")
		return errors.Wrap(err,
			fmt.Sprintf("Failed to get secret %s/%s",
				clusterNamespace, secretName))
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data[kubeconfigKey] = []byte(kubeconfig)

	return c.Update(ctx, secret)
}

// IsClusterReadyToBeConfigured returns true if cluster is ready to be configured
func IsClusterReadyToBeConfigured(
	ctx context.Context, c client.Client,
	cluster *corev1.ObjectReference, logger logr.Logger,
) (bool, error) {

	if cluster.Kind == libsveltosv1beta1.SveltosClusterKind {
		return isSveltosClusterReadyToBeConfigured(ctx, c, cluster, logger)
	}

	return isCAPIClusterReadyToBeConfigured(ctx, c, cluster, logger)
}

// isSveltosClusterReadyToBeConfigured  returns true if SveltosCluster
// Status.Ready is set to true
func isSveltosClusterReadyToBeConfigured(
	ctx context.Context, c client.Client,
	cluster *corev1.ObjectReference, logger logr.Logger,
) (bool, error) {

	sveltosCluster := &libsveltosv1beta1.SveltosCluster{}
	err := c.Get(ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}, sveltosCluster)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to get SveltosCluster %v", err))
		return false, err
	}

	return isSveltosClusterStatusReady(sveltosCluster), nil
}

func isSveltosClusterStatusReady(sveltosCluster *libsveltosv1beta1.SveltosCluster) bool {
	return sveltosCluster.Status.Ready
}

// isCAPIClusterReadyToBeConfigured checks whether Cluster:
// - ControlPlaneInitialized condition is set to true on Cluster object or
// - Status.ControlPlaneReady is set to true
func isCAPIClusterReadyToBeConfigured(
	ctx context.Context, c client.Client,
	cluster *corev1.ObjectReference, logger logr.Logger,
) (bool, error) {

	capiCluster := &clusterv1.Cluster{}
	err := c.Get(ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}, capiCluster)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to get Cluster %v", err))
		return false, err
	}

	return isCAPIControlPlaneReady(capiCluster), nil
}

func isCAPIControlPlaneReady(capiCluster *clusterv1.Cluster) bool {
	for i := range capiCluster.Status.Conditions {
		c := capiCluster.Status.Conditions[i]
		if c.Type == clusterv1.ControlPlaneInitializedCondition &&
			c.Status == corev1.ConditionTrue {

			return true
		}
	}

	return capiCluster.Status.ControlPlaneReady
}

// GetMachinesForCluster find all Machines for a given CAPI Cluster.
func GetMachinesForCluster(
	ctx context.Context, c client.Client,
	cluster *corev1.ObjectReference, logger logr.Logger,
) (*clusterv1.MachineList, error) {

	listOptions := []client.ListOption{
		client.InNamespace(cluster.Namespace),
		client.MatchingLabels{clusterv1.ClusterNameLabel: cluster.Name},
	}
	var machineList clusterv1.MachineList
	if err := c.List(ctx, &machineList, listOptions...); err != nil {
		logger.Error(err, fmt.Sprintf("unable to list Machines for CAPI Cluster %s/%s",
			cluster.Namespace, cluster.Name))
		return nil, err
	}
	logger.V(logs.LogDebug).Info(fmt.Sprintf("Found %d machine", len(machineList.Items)))

	return &machineList, nil
}

// CreateKubeconfig creates a temporary file with the Kubeconfig to access CAPI cluster
func CreateKubeconfig(logger logr.Logger, kubeconfigContent []byte) (fileName string, closer func(), err error) {
	var tmpfile *os.File
	tmpfile, err = os.CreateTemp("", "kubeconfig")
	if err != nil {
		logger.Error(err, "failed to create temporary file")
		err = errors.Wrap(err, "os.CreateTemp")
		return fileName, closer, err
	}
	defer tmpfile.Close()

	_, err = tmpfile.Write(kubeconfigContent)
	if err != nil {
		logger.Error(err, "failed to write to temporary file")
		err = errors.Wrap(err, "failed to write to temporary file")
		return fileName, closer, err
	}

	fileName = tmpfile.Name()
	closer = func() { os.Remove(tmpfile.Name()) }

	return fileName, closer, err
}

func getSecretData(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, secretName, secretKey string) ([]byte, error) {

	logger = logger.WithValues("secret", secretName, "key", secretKey)

	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      secretName,
	}

	if err := c.Get(ctx, key, secret); err != nil {
		logger.Error(err, "failed to get secret")
		return nil, errors.Wrap(err,
			fmt.Sprintf("Failed to get secret %s/%s",
				clusterNamespace, secretName))
	}

	if secret.Data == nil {
		return nil, errors.New("data section is empty")
	}

	if secretKey != "" {
		content, ok := secret.Data[secretKey]
		if !ok {
			return nil, errors.New(fmt.Sprintf("data section does not contain key: %s", secretKey))
		}

		return content, nil
	}

	for k, content := range secret.Data {
		logger.V(logs.LogVerbose).Info("Reading secret", "key", k)
		return content, nil
	}

	return nil, nil
}
