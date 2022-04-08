package edgedevicegroup

import (
	"context"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=edgedevicegroup -destination=mock_edgedevicegroup.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeviceGroup, error)
}

type CRRepository struct {
	client client.Client
}

func NewEdgeDeviceGroupRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeviceGroup, error) {
	edgeDeviceGroup := v1alpha1.EdgeDeviceGroup{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeDeviceGroup)
	return &edgeDeviceGroup, err
}
