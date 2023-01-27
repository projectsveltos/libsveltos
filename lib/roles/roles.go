/*
Copyright 2023. projectsveltos.io. All rights reserved.

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

package roles

import (
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	sveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/deployer"
)

const (
	clusterNameLabel = "projectsveltos.io/role-cluster"

	serviceAccountNameLabel = "projectsveltos.io/role-service-account"

	key = "kubeconfig"
)

// Platform admin can create RoleRequests to grant other admins
// permission in managed cluster.
// When RoleRequest is created, Sveltos deploys ClusterRoles/Roles
// with corresponding ClusterRoleBindings/RoleBindings and ServiceAccounts
// in managed clusters.
// Finally, Kubeconfig for each Cluster/ServiceAccount is taken and stored
// in the management cluster in a Secret.
//
// Here there are utilities to work with those secrets.

// GetSecret returns the secret to be used to store kubeconfig for serviceAccountName
// in cluster. It returns nil if it does not exist yet.
func GetSecret(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, serviceAccountName string, clusterType sveltosv1alpha1.ClusterType) (*corev1.Secret, error) {

	secretList := &corev1.SecretList{}
	err := c.List(ctx, secretList, getListOptionsForSecret(clusterNamespace, clusterName, serviceAccountName)...)
	if err != nil {
		return nil, err
	}

	switch len(secretList.Items) {
	case 0:
		return nil, nil
	case 1:
		return &secretList.Items[0], nil
	default:
		return nil, fmt.Errorf("found more than one existing secret for %s in cluster %s/%s",
			serviceAccountName, clusterNamespace, clusterName)
	}
}

// CreateSecret returns the secret to be used to store kubeconfig for serviceAccountName
// in cluster. It does create it if it does not exist yet.
// If Secret already exists, updates Data section if necessary (kubeconfig is different)
func CreateSecret(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, serviceAccountName string, clusterType sveltosv1alpha1.ClusterType,
	kubeconfig []byte, ownerReference metav1.Object) (*corev1.Secret, error) {

	secretList := &corev1.SecretList{}
	err := c.List(ctx, secretList, getListOptionsForSecret(clusterNamespace, clusterName, serviceAccountName)...)
	if err != nil {
		return nil, err
	}

	switch len(secretList.Items) {
	case 0:
		return createSecret(ctx, c, clusterNamespace, clusterName, serviceAccountName, kubeconfig, ownerReference)
	case 1:
		if shouldUpdate(&secretList.Items[0], kubeconfig) {
			return updateSecretData(ctx, c, &secretList.Items[0], kubeconfig)
		}
		return &secretList.Items[0], nil
	default:
		return nil, fmt.Errorf("found more than one existing secret for %s in cluster %s/%s",
			serviceAccountName, clusterNamespace, clusterName)
	}
}

// DeleteSecret finds Secret used to store kubeconfig for serviceAccountName in cluster.
// Removes owner as one of the OwnerReferences for secret. If no more OwnerReferences are left, deletes secret.
func DeleteSecret(ctx context.Context, c client.Client, clusterNamespace, clusterName, serviceAccountName string,
	clusterType sveltosv1alpha1.ClusterType, owner client.Object) error {

	secretList := &corev1.SecretList{}
	err := c.List(ctx, secretList, getListOptionsForSecret(clusterNamespace, clusterName, serviceAccountName)...)
	if err != nil {
		return nil
	}

	for i := range secretList.Items {
		deployer.RemoveOwnerReference(&secretList.Items[i], owner)

		if len(secretList.Items[i].GetOwnerReferences()) != 0 {
			err = c.Update(ctx, &secretList.Items[i])
			if err != nil {
				return err
			}
			// Other resources are still deploying this very same policy
			continue
		}

		err = c.Delete(ctx, &secretList.Items[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func ListSecretForOwnner(ctx context.Context, c client.Client, owner client.Object) ([]corev1.Secret, error) {
	listOption := []client.ListOption{
		client.MatchingLabels{
			sveltosv1alpha1.RoleRequestLabel: "ok",
		},
	}

	secretList := &corev1.SecretList{}
	err := c.List(ctx, secretList, listOption...)
	if err != nil {
		return nil, err
	}

	results := make([]corev1.Secret, 0)

	for i := range secretList.Items {
		secret := &secretList.Items[i]
		if secret.Labels == nil {
			continue
		}
		if _, ok := secret.Labels[sveltosv1alpha1.RoleRequestLabel]; !ok {
			continue
		}
		if deployer.IsOwnerReference(secret, owner) {
			results = append(results, *secret)
		}
	}

	return results, nil
}

func getSha256(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash)
}

func createSecret(ctx context.Context, c client.Client, namespace, clusterName, serviceAccountName string,
	kubeconfig []byte, ownerReference metav1.Object) (*corev1.Secret, error) {

	var config string
	config += clusterName
	config += serviceAccountName
	name := fmt.Sprintf("sveltos-%s", getSha256(config))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				clusterNameLabel:                 clusterName,
				serviceAccountNameLabel:          serviceAccountName,
				sveltosv1alpha1.RoleRequestLabel: "ok",
			},
		},
		Data: map[string][]byte{
			key: kubeconfig,
		},
	}

	if err := controllerutil.SetOwnerReference(ownerReference, secret, c.Scheme()); err != nil {
		return nil, err
	}

	if err := c.Create(ctx, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func getListOptionsForSecret(clusterNamespace, clusterName, serviceAccountName string) []client.ListOption {
	return []client.ListOption{
		client.InNamespace(clusterNamespace),
		client.MatchingLabels{
			clusterNameLabel:        clusterName,
			serviceAccountNameLabel: serviceAccountName,
		},
	}
}

// shouldUpdate returns true if secret needs to be updated, which happens
// when kubeconfig stored in the secret and the new kubeconfig are different.
func shouldUpdate(secret *corev1.Secret, kubeconfig []byte) bool {
	if secret.Data == nil {
		return true
	}

	return !reflect.DeepEqual(secret.Data[key], kubeconfig)
}

// updateSecretData updates secret data section
func updateSecretData(ctx context.Context, c client.Client,
	secret *corev1.Secret, kubeconfig []byte) (*corev1.Secret, error) {

	secret.Data = map[string][]byte{
		key: kubeconfig,
	}

	err := c.Update(ctx, secret)
	return secret, err
}
