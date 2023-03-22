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
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
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

	return getSecretData(ctx, logger, c, clusterNamespace, clusterName, capiKubeconfigSecretNamePostfix)
}

// GetSveltosKubernetesRestConfig returns rest.Config for a Sveltos Cluster clusterNamespace/clusterName
// c is the client to access management cluster
func GetSveltosKubernetesRestConfig(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) (*rest.Config, error) {

	kubeconfigContent, err := GetSveltosSecretData(ctx, logger, c, clusterNamespace, clusterName)
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

// GetSveltosSecretData verifies Cluster exists and returns the content of secret containing
// the kubeconfig for Sveltos cluster
func GetSveltosSecretData(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName string) ([]byte, error) {

	logger.WithValues("namespace", clusterNamespace, "cluster", clusterName)
	logger.V(logs.LogVerbose).Info("Get secret")
	key := client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      clusterName,
	}

	cluster := libsveltosv1alpha1.SveltosCluster{}
	if err := c.Get(ctx, key, &cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("SveltosCluster does not exist")
			return nil, errors.Wrap(err,
				fmt.Sprintf("SveltosCluster %s/%s does not exist",
					clusterNamespace,
					clusterName,
				))
		}
		return nil, err
	}

	return getSecretData(ctx, logger, c, clusterNamespace, clusterName, sveltosKubeconfigSecretNamePostfix)
}

// IsClusterReadyToBeConfigured returns true if cluster is ready to be configured
func IsClusterReadyToBeConfigured(
	ctx context.Context, c client.Client,
	cluster *corev1.ObjectReference, logger logr.Logger,
) (bool, error) {

	if cluster.Kind == libsveltosv1alpha1.SveltosClusterKind {
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

	sveltosCluster := &libsveltosv1alpha1.SveltosCluster{}
	err := c.Get(ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}, sveltosCluster)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to get SveltosCluster %v", err))
		return false, err
	}

	return sveltosCluster.Status.Ready, nil
}

// isCAPIClusterReadyToBeConfigured gets all Machines for a given CAPI Cluster and returns true
// if at least one control plane machine is in running phase
func isCAPIClusterReadyToBeConfigured(
	ctx context.Context, c client.Client,
	cluster *corev1.ObjectReference, logger logr.Logger,
) (bool, error) {

	machineList, err := GetMachinesForCluster(ctx, c, cluster, logger)
	if err != nil {
		return false, err
	}

	for i := range machineList.Items {
		if util.IsControlPlaneMachine(&machineList.Items[i]) {
			if machineList.Items[i].Status.GetTypedPhase() == clusterv1.MachinePhaseRunning ||
				machineList.Items[i].Status.GetTypedPhase() == clusterv1.MachinePhaseProvisioned {

				return true, nil
			}
		}
	}

	return false, nil
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

func getSecretData(ctx context.Context, logger logr.Logger, c client.Client,
	clusterNamespace, clusterName, postfix string) ([]byte, error) {

	secretName := clusterName + postfix
	logger = logger.WithValues("secret", secretName)

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

	for k, contents := range secret.Data {
		logger.V(logs.LogVerbose).Info("Reading secret", "key", k)
		return contents, nil
	}

	return nil, nil
}
