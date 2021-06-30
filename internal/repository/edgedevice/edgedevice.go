package edgedevice

import (
	"context"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Repository struct {
	client client.Client
}

func NewEdgeDeviceRepository(client client.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDevice, error) {
	edgeDevice := v1alpha1.EdgeDevice{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeDevice)
	return &edgeDevice, err
}

func (r *Repository) Create(ctx context.Context, edgeDevice v1alpha1.EdgeDevice) (*v1alpha1.EdgeDevice, error) {
	edgeDevice.GenerateName = "ed-"
	edgeDevice.Name = ""
	err := r.client.Create(ctx, &edgeDevice)
	return &edgeDevice, err
}
