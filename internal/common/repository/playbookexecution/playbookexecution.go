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
