package devicemetrics

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/k8sclient"
	"github.com/project-flotta/flotta-operator/models"
)

//go:generate mockgen -package=devicemetrics -destination=mock_allowlists.go . AllowListGenerator
type AllowListGenerator interface {
	GenerateFromConfigMap(ctx context.Context, name, namespace string) (*models.MetricsAllowList, error)
}

type allowListGenerator struct {
	client k8sclient.K8sClient
}

func NewAllowListGenerator(client k8sclient.K8sClient) AllowListGenerator {
	return &allowListGenerator{client: client}
}

func (g *allowListGenerator) GenerateFromConfigMap(ctx context.Context, name, namespace string) (*models.MetricsAllowList, error) {
	cm := corev1.ConfigMap{}
	err := g.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &cm)
	if err != nil {
		return nil, err
	}
	metricsList, ok := cm.Data["metrics_list.yaml"]
	if !ok {
		return nil, fmt.Errorf("metrics_list.yaml not found in %s/%s config map", namespace, name)
	}
	mal := &models.MetricsAllowList{}
	err = yaml.Unmarshal([]byte(metricsList), mal)
	if err != nil {
		return nil, err
	}
	return mal, nil
}
