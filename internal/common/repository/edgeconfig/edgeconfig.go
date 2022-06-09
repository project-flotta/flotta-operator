package edgeconfig

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/indexer"
)

//go:generate mockgen -package=edgeconfig -destination=mock_edgeconfig.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeConfig, error)
	Patch(ctx context.Context, old, new *v1alpha1.EdgeConfig) error
	RemoveFinalizer(ctx context.Context, edgeConfig *v1alpha1.EdgeConfig, finalizer string) error
	ListByLabel(ctx context.Context, labelName, labelValue string, namespace string) ([]v1alpha1.EdgeConfig, error)
}

type CRRepository struct {
	client client.Client
}

func NewEdgeConfigRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeConfig, error) {
	edgeConfig := v1alpha1.EdgeConfig{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeConfig)
	return &edgeConfig, err
}

func (r *CRRepository) Patch(ctx context.Context, old, new *v1alpha1.EdgeConfig) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}

func (r *CRRepository) RemoveFinalizer(ctx context.Context, edgeConfig *v1alpha1.EdgeConfig, finalizer string) error {
	cp := edgeConfig.DeepCopy()

	var finalizers []string
	for _, f := range cp.Finalizers {
		if f != finalizer {
			finalizers = append(finalizers, f)
		}
	}
	cp.Finalizers = finalizers

	err := r.Patch(ctx, edgeConfig, cp)
	if err == nil {
		edgeConfig.Finalizers = cp.Finalizers
	}

	return nil
}

func (r *CRRepository) ListByLabel(ctx context.Context, labelName, labelValue string, namespace string) ([]v1alpha1.EdgeConfig, error) {
	edgeConfigs := v1alpha1.EdgeConfigList{}
	err := r.client.List(ctx, &edgeConfigs,
		client.MatchingLabels{indexer.DeviceByConfigIndexKey: indexer.CreateDeviceConfigIndexKey(labelValue)},
		client.InNamespace(namespace),
	)
	if err != nil {
		return nil, err
	}
	return edgeConfigs.Items, nil
}
