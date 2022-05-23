package edgeworkload

import (
	"context"
	"github.com/project-flotta/flotta-operator/internal/common/indexer"

	_ "github.com/golang/mock/mockgen/model"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
)

//go:generate mockgen -package=edgeworkload -destination=mock_edgeworkload.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeWorkload, error)
	Patch(ctx context.Context, old, new *v1alpha1.EdgeWorkload) error
	RemoveFinalizer(ctx context.Context, edgeWorkload *v1alpha1.EdgeWorkload, finalizer string) error
	ListByLabel(ctx context.Context, labelName, labelValue string, namespace string) ([]v1alpha1.EdgeWorkload, error)
}

type CRRepository struct {
	client client.Client
}

func NewEdgeWorkloadRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeWorkload, error) {
	edgeWorkload := v1alpha1.EdgeWorkload{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeWorkload)
	return &edgeWorkload, err
}

func (r *CRRepository) Patch(ctx context.Context, old, new *v1alpha1.EdgeWorkload) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}

func (r *CRRepository) RemoveFinalizer(ctx context.Context, edgeWorkload *v1alpha1.EdgeWorkload, finalizer string) error {
	cp := edgeWorkload.DeepCopy()

	var finalizers []string
	for _, f := range cp.Finalizers {
		if f != finalizer {
			finalizers = append(finalizers, f)
		}
	}
	cp.Finalizers = finalizers

	err := r.Patch(ctx, edgeWorkload, cp)
	if err == nil {
		edgeWorkload.Finalizers = cp.Finalizers
	}

	return nil
}

func (r *CRRepository) ListByLabel(ctx context.Context, labelName, labelValue string, namespace string) ([]v1alpha1.EdgeWorkload, error) {
	edgeWorkloads := v1alpha1.EdgeWorkloadList{}
	err := r.client.List(ctx, &edgeWorkloads,
		client.MatchingFields{indexer.WorkloadByDeviceIndexKey: indexer.CreateWorkloadIndexKey(labelName, labelValue)},
		client.InNamespace(namespace),
	)
	if err != nil {
		return nil, err
	}
	return edgeWorkloads.Items, nil
}
