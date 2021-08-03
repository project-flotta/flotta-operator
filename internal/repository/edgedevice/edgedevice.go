package edgedevice

import (
	"context"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Repository struct {
	client client.Client
}

func NewEdgeDeviceRepository(client client.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDevice, error) {
	edgeDevice := v1alpha1.EdgeDevice{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &edgeDevice)
	return &edgeDevice, err
}

func (r *Repository) Create(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) error {
	return r.client.Create(ctx, edgeDevice)
}

func (r *Repository) UpdateStatus(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) error {
	return r.client.Status().Update(ctx, edgeDevice)
}

func (r *Repository) Patch(ctx context.Context, old, new *v1alpha1.EdgeDevice) error {
	patch := client.MergeFrom(old)
	return r.client.Patch(ctx, new, patch)
}

func (r Repository) ListForSelector(ctx context.Context, selector *metav1.LabelSelector, namespace string) ([]v1alpha1.EdgeDevice, error) {
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

func (r *Repository) RemoveFinalizer(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, finalizer string) error {
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
