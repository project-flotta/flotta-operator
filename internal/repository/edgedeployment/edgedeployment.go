package edgedeployment

import (
	"context"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	_ "github.com/golang/mock/mockgen/model"
)

//go:generate mockgen -package=edgedeployment -destination=mock_edgedeployment.go . Repository
type Repository interface {
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
