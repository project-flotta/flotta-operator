package edgeautoconfig

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
)

//go:generate mockgen -package=edgeautoconfig -destination=mock_edgeautoconfig.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeAutoConfig, error)
	ReadNS(ctx context.Context, namespace string) (*v1alpha1.EdgeAutoConfig, error)
	Create(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig) error
	PatchStatus(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig, patch *client.Patch) error
	Patch(ctx context.Context, old *v1alpha1.EdgeAutoConfig, new *v1alpha1.EdgeAutoConfig) error
	Delete(ctx context.Context, obj *v1alpha1.EdgeAutoConfig) error
	ListByNamespace(ctx context.Context, namespace string) ([]v1alpha1.EdgeAutoConfig, error)
}

type EdgeAutoConfigRepository struct {
	client client.Client
}

func NewEdgeAutoConfigRepository(client client.Client) *EdgeAutoConfigRepository {
	return &EdgeAutoConfigRepository{client: client}
}

func (r *EdgeAutoConfigRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeAutoConfig, error) {
	deviceAutoConfig := &v1alpha1.EdgeAutoConfig{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deviceAutoConfig)
	return deviceAutoConfig, err
}

func (r *EdgeAutoConfigRepository) Create(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig) error {
	return r.client.Create(ctx, EdgeAutoConfig)
}

func (r *EdgeAutoConfigRepository) PatchStatus(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig, patch *client.Patch) error {
	return r.client.Status().Patch(ctx, EdgeAutoConfig, *patch)
}

func (r *EdgeAutoConfigRepository) Patch(ctx context.Context, old *v1alpha1.EdgeAutoConfig, new *v1alpha1.EdgeAutoConfig) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}

func (r *EdgeAutoConfigRepository) Delete(ctx context.Context, obj *v1alpha1.EdgeAutoConfig) error {
	return r.client.Delete(ctx, obj)
}

func (r *EdgeAutoConfigRepository) ReadNS(ctx context.Context, namespace string) (*v1alpha1.EdgeAutoConfig, error) {
	deviceAutoConfig := &v1alpha1.EdgeAutoConfig{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace}, deviceAutoConfig)
	return deviceAutoConfig, err
}

func (r *EdgeAutoConfigRepository) ListByNamespace(ctx context.Context, namespace string) ([]v1alpha1.EdgeAutoConfig, error) {
	edgeautocfg := v1alpha1.EdgeAutoConfigList{}
	err := r.client.List(ctx, &edgeautocfg,
		client.InNamespace(namespace),
	)
	if err != nil {
		return nil, err
	}
	return edgeautocfg.Items, nil
}
