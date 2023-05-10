package edgeautoconfig

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
)

//go:generate mockgen -package=EdgeAutoConfig -destination=mock_EdgeAutoConfig.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeAutoConfig, error)
	ReadNS(ctx context.Context, namespace string) (*v1alpha1.EdgeAutoConfig, error)
	Create(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig) error
	PatchStatus(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig, patch *client.Patch) error
	Patch(ctx context.Context, old *v1alpha1.EdgeAutoConfig, new *v1alpha1.EdgeAutoConfig) error
	Delete(ctx context.Context, obj *v1alpha1.EdgeAutoConfig) error
}

type CRRepository struct {
	client client.Client
}

type EdgeAutoConfigRepository struct {
	client client.Client
}

func NewEdgeAutoConfigRepository(client client.Client) *EdgeAutoConfigRepository {
	return &EdgeAutoConfigRepository{client: client}
}

func (esr *EdgeAutoConfigRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeAutoConfig, error) {
	deviceSignedRequest := &v1alpha1.EdgeAutoConfig{}
	err := esr.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deviceSignedRequest)
	return deviceSignedRequest, err
}

func (esr *EdgeAutoConfigRepository) Create(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig) error {
	return esr.client.Create(ctx, EdgeAutoConfig)
}

func (esr *EdgeAutoConfigRepository) PatchStatus(ctx context.Context, EdgeAutoConfig *v1alpha1.EdgeAutoConfig, patch *client.Patch) error {
	return esr.client.Status().Patch(ctx, EdgeAutoConfig, *patch)
}

func (esr *EdgeAutoConfigRepository) Patch(ctx context.Context, old *v1alpha1.EdgeAutoConfig, new *v1alpha1.EdgeAutoConfig) error {
	patch := client.MergeFrom(old)
	return esr.client.Patch(ctx, new, patch)
}

func (esr *EdgeAutoConfigRepository) Delete(ctx context.Context, obj *v1alpha1.EdgeAutoConfig) error {
	return esr.client.Delete(ctx, obj)
}
