package edgedeployment

import (
	"context"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Repository struct {
	client client.Client
}

func NewEdgeDeploymentRepository(client client.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) ListForEdgeDevice(ctx context.Context, name string, namespace string) ([]v1alpha1.EdgeDeployment, error) {
	edl := v1alpha1.EdgeDeploymentList{}

	selector, err := fields.ParseSelector("spec.device=" + name)
	if err != nil {
		return nil, err
	}
	options := client.ListOptions{
		Namespace:     namespace,
		FieldSelector: selector,
	}
	err = r.client.List(ctx, &edl, &options)
	if err != nil {
		return nil, err
	}
	return edl.Items, nil
}

func (r *Repository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeployment, error) {
	edgeDeployment := v1alpha1.EdgeDeployment{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeDeployment)
	return &edgeDeployment, err
}

func (r *Repository) UpdateStatus(ctx context.Context, edgeDeployment v1alpha1.EdgeDeployment) (*v1alpha1.EdgeDeployment, error) {
	err := r.client.Status().Update(ctx, &edgeDeployment)
	return &edgeDeployment, err
}
