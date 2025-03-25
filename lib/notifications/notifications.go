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

package notifications

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getSecret(ctx context.Context, c client.Client, secretRef *corev1.ObjectReference) (*corev1.Secret, error) {
	if secretRef == nil {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	if secretRef.Kind != "Secret" {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	if secretRef.APIVersion != "v1" {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	secret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{
		Namespace: secretRef.Namespace,
		Name:      secretRef.Name,
	}, secret)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	return secret, nil
}
