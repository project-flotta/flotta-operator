package edgedevicesignedrequest

import (
	"context"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=edgedevicesignedrequest -destination=mock_edgedeviceSignedRequest.go . Repository
type Repository interface {
	Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeviceSignedRequest, error)
	Create(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest) error
	PatchStatus(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) error
	Patch(ctx context.Context, old *v1alpha1.EdgeDeviceSignedRequest, new *v1alpha1.EdgeDeviceSignedRequest) error
	Delete(ctx context.Context, obj *v1alpha1.EdgeDeviceSignedRequest) error
}

type EdgedeviceSignedRequestRepository struct {
	client client.Client
}

func NewEdgedeviceSignedRequestRepository(client client.Client) *EdgedeviceSignedRequestRepository {
	return &EdgedeviceSignedRequestRepository{client: client}
}

func (esr *EdgedeviceSignedRequestRepository) Read(ctx context.Context, name string, namespace string) (*v1alpha1.EdgeDeviceSignedRequest, error) {
	deviceSignedRequest := &v1alpha1.EdgeDeviceSignedRequest{}
	err := esr.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deviceSignedRequest)
	return deviceSignedRequest, err
}

func (esr *EdgedeviceSignedRequestRepository) Create(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest) error {
	return esr.client.Create(ctx, edgedeviceSignedRequest)
}

func (esr *EdgedeviceSignedRequestRepository) PatchStatus(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) error {
	return esr.client.Status().Patch(ctx, edgedeviceSignedRequest, *patch)
}

func (esr *EdgedeviceSignedRequestRepository) Patch(ctx context.Context, old *v1alpha1.EdgeDeviceSignedRequest, new *v1alpha1.EdgeDeviceSignedRequest) error {
	patch := client.MergeFrom(old)
	return esr.client.Patch(ctx, new, patch)
}

func (esr *EdgedeviceSignedRequestRepository) Delete(ctx context.Context, obj *v1alpha1.EdgeDeviceSignedRequest) error {
	return esr.client.Delete(ctx, obj)
}
