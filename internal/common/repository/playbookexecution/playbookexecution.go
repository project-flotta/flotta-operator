package playbookexecution

import (
	"context"

	_ "github.com/golang/mock/mockgen/model"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
)

//go:generate mockgen -package=playbookexecution -destination=mock_playbookexecution.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.PlaybookExecution, error)
	Create(ctx context.Context, playbookExecution *v1alpha1.PlaybookExecution) error
	Patch(ctx context.Context, old, new *v1alpha1.PlaybookExecution) error
	RemoveFinalizer(ctx context.Context, playbookExecution *v1alpha1.PlaybookExecution, finalizer string) error
	ListByLabel(ctx context.Context, labelName, labelValue string, namespace string) ([]v1alpha1.PlaybookExecution, error)
}

type CRRepository struct {
	client client.Client
}

func NewPlaybookExecutionRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.PlaybookExecution, error) {
	playbookExecution := v1alpha1.PlaybookExecution{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &playbookExecution)
	return &playbookExecution, err
}

func (r *CRRepository) Create(ctx context.Context, playbookExecution *v1alpha1.PlaybookExecution) error {
	return r.client.Create(ctx, playbookExecution)
}

func (r *CRRepository) Patch(ctx context.Context, old, new *v1alpha1.PlaybookExecution) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}

func (r *CRRepository) RemoveFinalizer(ctx context.Context, playbookExecution *v1alpha1.PlaybookExecution, finalizer string) error {
	cp := playbookExecution.DeepCopy()

	var finalizers []string
	for _, f := range cp.Finalizers {
		if f != finalizer {
			finalizers = append(finalizers, f)
		}
	}
	cp.Finalizers = finalizers

	err := r.Patch(ctx, playbookExecution, cp)
	if err == nil {
		playbookExecution.Finalizers = cp.Finalizers
	}

	return nil
}

func (r *CRRepository) ListByLabel(ctx context.Context, labelName, labelValue string, namespace string) ([]v1alpha1.PlaybookExecution, error) {
	playbookExecutions := v1alpha1.PlaybookExecutionList{}
	err := r.client.List(ctx, &playbookExecutions,
		// client.MatchingLabels{indexer.DeviceByConfigIndexKey: indexer.CreateDeviceConfigIndexKey(labelValue)},
		client.InNamespace(namespace),
	)
	if err != nil {
		return nil, err
	}
	return playbookExecutions.Items, nil
}
