package edgedevice

import (
	"context"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=edgedevice -destination=mock_edgedevice.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDevice, error)
	Create(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) error
	PatchStatus(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) error
	Patch(ctx context.Context, old, new *v1alpha1.EdgeDevice) error
	ListForSelector(ctx context.Context, selector *metav1.LabelSelector, namespace string) ([]v1alpha1.EdgeDevice, error)
	RemoveFinalizer(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, finalizer string) error
}

type CRRepository struct {
	client client.Client
}

func NewEdgeDeviceRepository(client client.Client) *CRRepository {
	return &CRRepository{client: client}
}

func (r *CRRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDevice, error) {
	edgeDevice := v1alpha1.EdgeDevice{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeDevice)
	return &edgeDevice, err
}

func (r *CRRepository) Create(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) error {
	return r.client.Create(ctx, edgeDevice)
}

func (r *CRRepository) PatchStatus(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) error {
	return r.client.Status().Patch(ctx, edgeDevice, *patch)
}

func (r *CRRepository) Patch(ctx context.Context, old, new *v1alpha1.EdgeDevice) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}

func (r CRRepository) ListForSelector(ctx context.Context, selector *metav1.LabelSelector, namespace string) ([]v1alpha1.EdgeDevice, error) {
	s, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}
	options := client.ListOptions{
		Namespace:     namespace,
		LabelSelector: s,
	}
	var edl v1alpha1.EdgeDeviceList
	err = r.client.List(ctx, &edl, &options)
	if err != nil {
		return nil, err
	}

	return edl.Items, nil
}

func (r *CRRepository) RemoveFinalizer(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, finalizer string) error {
	cp := edgeDevice.DeepCopy()

	var finalizers []string
	for _, f := range cp.Finalizers {
		if f != finalizer {
			finalizers = append(finalizers, f)
		}
	}
	cp.Finalizers = finalizers

	err := r.Patch(ctx, edgeDevice, cp)
	if err == nil {
		edgeDevice.Finalizers = cp.Finalizers
	}

	return nil
}
