package edgedeviceset

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
)

//go:generate mockgen -package=edgedeviceset -destination=mock_edgedeviceset.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeviceSet, error)
}

type CRRepository struct {
	client client.Client
}

func NewEdgeDeviceSetRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeviceSet, error) {
	edgeDeviceSet := v1alpha1.EdgeDeviceSet{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeDeviceSet)
	return &edgeDeviceSet, err
}
