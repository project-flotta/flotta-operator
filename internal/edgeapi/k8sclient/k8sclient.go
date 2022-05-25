package k8sclient

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=k8sclient -destination=mock_k8sclient.go . K8sClient
type K8sClient interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object) error
}

func NewK8sClient(client client.Client) *k8sClient {
	return &k8sClient{client: client}
}

type k8sClient struct {
	client client.Client
}

func (c *k8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return c.client.Get(ctx, key, obj)
}
