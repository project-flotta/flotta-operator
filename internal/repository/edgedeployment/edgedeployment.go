package edgedeployment

import (
	"context"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Repository interface {
	ListForEdgeDevice(ctx context.Context, name string, namespace string) ([]v1alpha1.EdgeDeployment, error)
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeployment, error)
	Patch(ctx context.Context, old, new *v1alpha1.EdgeDeployment) error
	RemoveFinalizer(ctx context.Context, edgeDeployment *v1alpha1.EdgeDeployment, finalizer string) error
}

type CRRespository struct {
	client client.Client
}

func NewEdgeDeploymentRepository(client client.Client) *CRRespository {
	return &CRRespository{client: client}
}

func (r *CRRespository) ListForEdgeDevice(ctx context.Context, name string, namespace string) ([]v1alpha1.EdgeDeployment, error) {
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

func (r *CRRespository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeployment, error) {
	edgeDeployment := v1alpha1.EdgeDeployment{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeDeployment)
	return &edgeDeployment, err
}

func (r *CRRespository) Patch(ctx context.Context, old, new *v1alpha1.EdgeDeployment) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}

func (r *CRRespository) RemoveFinalizer(ctx context.Context, edgeDeployment *v1alpha1.EdgeDeployment, finalizer string) error {
	cp := edgeDeployment.DeepCopy()

	var finalizers []string
	for _, f := range cp.Finalizers {
		if f != finalizer {
			finalizers = append(finalizers, f)
		}
	}
	cp.Finalizers = finalizers

	err := r.Patch(ctx, edgeDeployment, cp)
	if err == nil {
		edgeDeployment.Finalizers = cp.Finalizers
	}

	return nil
}
