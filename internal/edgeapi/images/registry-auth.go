package images

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=images -destination=mock_registry-auth.go . RegistryAuthAPI
type RegistryAuthAPI interface {
	GetAuthFileFromSecret(ctx context.Context, namespace, name string) (string, error)
}

type RegistryAuth struct {
	client client.Client
}

func NewRegistryAuth(k8sClient client.Client) *RegistryAuth {
	return &RegistryAuth{client: k8sClient}
}

func (r *RegistryAuth) GetAuthFileFromSecret(ctx context.Context, namespace, name string) (string, error) {
	secret := v1.Secret{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &secret)
	if err != nil {
		return "", err
	}
	authFile, found := secret.Data[".dockerconfigjson"]
	if !found {
		return "", fmt.Errorf(".dockerconfigjson not found in %s/%s Secret", namespace, name)
	}

	return string(authFile), nil
}
