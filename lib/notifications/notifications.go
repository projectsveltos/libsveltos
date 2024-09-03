package notifications

import (
	"context"
	"fmt"

	"github.com/projectsveltos/libsveltos/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getSecret(ctx context.Context, c client.Client, notification *v1beta1.Notification) (*corev1.Secret, error) {
	if notification.NotificationRef == nil {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	if notification.NotificationRef.Kind != "Secret" {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	if notification.NotificationRef.APIVersion != "v1" {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	secret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{
		Namespace: notification.NotificationRef.Namespace,
		Name:      notification.NotificationRef.Name,
	}, secret)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("notification must reference v1 secret containing smtp configuration")
	}

	return secret, nil
}
