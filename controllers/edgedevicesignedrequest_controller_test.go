package controllers_test

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/controllers"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevicesignedrequest"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("EdgeDeviceSignedRequest controller", func() {
	const (
		targetNamespace = "NewNamespace"
	)

	var (
		edgeDeviceRepoMock   *edgedevice.MockRepository
		edgeDeviceSRRepoMock *edgedevicesignedrequest.MockRepository

		k8sManager manager.Manager
		reconciler *controllers.EdgeDeviceSignedRequestReconciler
		req        ctrl.Request
		edsr       *v1alpha1.EdgeDeviceSignedRequest
	)

	BeforeEach(func() {

		k8sManager = getK8sManager(cfg)
		mockCtrl := gomock.NewController(GinkgoT())

		edgeDeviceRepoMock = edgedevice.NewMockRepository(mockCtrl)
		edgeDeviceSRRepoMock = edgedevicesignedrequest.NewMockRepository(mockCtrl)
		reconciler = &controllers.EdgeDeviceSignedRequestReconciler{
			Client:                            k8sClient,
			Scheme:                            k8sManager.GetScheme(),
			EdgedeviceSignedRequestRepository: edgeDeviceSRRepoMock,
			EdgeDeviceRepository:              edgeDeviceRepoMock,
			MaxConcurrentReconciles:           1,
		}

		req = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test",
				Namespace: "test",
			},
		}

		edsr = &v1alpha1.EdgeDeviceSignedRequest{
			ObjectMeta: v1.ObjectMeta{Name: req.Name, Namespace: req.Namespace},
			Spec: v1alpha1.EdgeDeviceSignedRequestSpec{
				TargetNamespace: targetNamespace,
				Approved:        false,
			},
		}
	})

	It("Cannot retrieve current values", func() {
		// given
		edgeDeviceSRRepoMock.EXPECT().
			Read(gomock.Any(), req.Name, req.Namespace).
			Return(nil, fmt.Errorf("INVALID")).
			Times(1)
		// when
		result, err := reconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).To(HaveOccurred())
		Expect(result.Requeue).To(BeTrue())
	})

	It("Cannot find current endpoint", func() {
		// given
		returnErr := errors.NewNotFound(
			schema.GroupResource{Group: "", Resource: "notfound"},
			"notfound")

		edgeDeviceSRRepoMock.EXPECT().
			Read(gomock.Any(), req.Name, req.Namespace).
			Return(nil, returnErr).
			Times(1)
		// when
		result, err := reconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Requeue).To(BeFalse())
	})

	It("Device Signed request is not yet approved", func() {

		// given
		edgeDeviceSRRepoMock.EXPECT().
			Read(gomock.Any(), req.Name, req.Namespace).
			Return(edsr, nil).
			Times(1)

		edgeDeviceSRRepoMock.EXPECT().
			PatchStatus(gomock.Any(), edsr, gomock.Any()).
			Do(func(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) {
				Expect(edgedeviceSignedRequest.Spec.Approved).To(BeFalse())
			}).
			Return(nil).
			Times(1)

		// when
		result, err := reconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Requeue).To(BeFalse())
		Expect(edsr.Spec.Approved).To(BeFalse())
	})

	It("Device Signed request is not yet approved on AutoApproval", func() {

		// given
		reconciler.AutoApproval = true

		edgeDeviceSRRepoMock.EXPECT().
			Read(gomock.Any(), req.Name, req.Namespace).
			Return(edsr, nil).
			Times(1)

		edgeDeviceSRRepoMock.EXPECT().
			Patch(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, old, new *v1alpha1.EdgeDeviceSignedRequest) {
				Expect(new.Spec.Approved).To(BeTrue())
				Expect(old.Spec.Approved).To(BeFalse())
			}).
			Return(nil).
			Times(1)

		// when
		result, err := reconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Requeue).To(BeTrue())
	})

	It("Device Signed request is not yet approved but fail on patch", func() {

		// given
		reconciler.AutoApproval = true

		edgeDeviceSRRepoMock.EXPECT().
			Read(gomock.Any(), req.Name, req.Namespace).
			Return(edsr, nil).
			Times(1)

		edgeDeviceSRRepoMock.EXPECT().
			Patch(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, old, new *v1alpha1.EdgeDeviceSignedRequest) {
				Expect(new.Spec.Approved).To(BeTrue())
				Expect(old.Spec.Approved).To(BeFalse())
			}).
			Return(fmt.Errorf("Fail")).
			Times(1)

		// when
		result, err := reconciler.Reconcile(context.TODO(), req)

		// then
		Expect(err).To(HaveOccurred())
		Expect(result.Requeue).To(BeTrue())
		Expect(edsr.Spec.Approved).To(BeFalse())
	})

	Context("Is approved", func() {
		BeforeEach(func() {
			edsr.Spec.Approved = true

			edgeDeviceSRRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(edsr, nil).
				Times(1)
		})

		It("Device is already created", func() {

			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, targetNamespace).
				Return(nil, nil).
				Times(1)

			edgeDeviceSRRepoMock.EXPECT().
				PatchStatus(gomock.Any(), edsr, gomock.Any()).
				Do(func(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) {
					Expect(edgedeviceSignedRequest.Spec.Approved).To(BeTrue())
					Expect(edgedeviceSignedRequest.Status.Conditions).To(HaveLen(1))
					Expect(edgedeviceSignedRequest.Status.Conditions[0].Type).To(Equal(v1alpha1.EdgeDeviceSignedRequestStatusApproved))
				}).
				Return(nil).
				Times(1)

			// when
			result, err := reconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(edsr.Spec.Approved).To(BeTrue())
		})

		It("Device is marked as pending", func() {
			// given

			// first loop
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, targetNamespace).
				Return(nil, nil).
				Times(1)

			edgeDeviceSRRepoMock.EXPECT().
				PatchStatus(gomock.Any(), edsr, gomock.Any()).
				Do(func(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) {
					Expect(edgedeviceSignedRequest.Spec.Approved).To(BeTrue())
					Expect(edgedeviceSignedRequest.Status.Conditions).To(HaveLen(1))
					Expect(edgedeviceSignedRequest.Status.Conditions[0].Type).To(Equal(v1alpha1.EdgeDeviceSignedRequestStatusApproved))
				}).
				Return(nil).
				Times(1)

			result, err := reconciler.Reconcile(context.TODO(), req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(edsr.Spec.Approved).To(BeTrue())

			// second loop
			edsr.Spec.Approved = false

			edgeDeviceSRRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(edsr, nil).
				Times(1)

			edgeDeviceSRRepoMock.EXPECT().
				PatchStatus(gomock.Any(), edsr, gomock.Any()).
				Do(func(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) {
					Expect(edgedeviceSignedRequest.Spec.Approved).To(BeFalse())
					Expect(edgedeviceSignedRequest.Status.Conditions).To(HaveLen(2))
					Expect(edgedeviceSignedRequest.Status.Conditions[0].Type).To(Equal(v1alpha1.EdgeDeviceSignedRequestStatusApproved))
					Expect(edgedeviceSignedRequest.Status.Conditions[0].Status).To(Equal(v1.ConditionStatus("false")))
					Expect(edgedeviceSignedRequest.Status.Conditions[1].Type).To(Equal(v1alpha1.EdgeDeviceSignedRequestStatusPending))
					Expect(edgedeviceSignedRequest.Status.Conditions[1].Status).To(Equal(v1.ConditionStatus("true")))
				}).
				Return(nil).
				Times(1)

			// when
			result, err = reconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(edsr.Spec.Approved).To(BeFalse())

		})

		It("Creates device correctly", func() {

			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, targetNamespace).
				Return(nil, fmt.Errorf("cannot retrieve device")).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) {
					Expect(edgeDevice.Name).To(Equal(req.Name))
				}).
				Return(nil).
				Times(1)

			edgeDeviceSRRepoMock.EXPECT().
				PatchStatus(gomock.Any(), edsr, gomock.Any()).
				Do(func(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) {
					Expect(edgedeviceSignedRequest.Spec.Approved).To(BeTrue())
					Expect(edgedeviceSignedRequest.Status.Conditions).To(HaveLen(1))
					Expect(edgedeviceSignedRequest.Status.Conditions[0].Type).To(Equal(v1alpha1.EdgeDeviceSignedRequestStatusApproved))
				}).
				Return(nil).
				Times(1)

			// when
			result, err := reconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(edsr.Spec.Approved).To(BeTrue())
		})

		It("Creates device correctly with the right set", func() {

			// given
			edsr.Spec.TargetSet = "foo-group"

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, targetNamespace).
				Return(nil, fmt.Errorf("cannot retrieve device")).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) {
					Expect(edgeDevice.Name).To(Equal(req.Name))
					Expect(edgeDevice.ObjectMeta.Labels).To(HaveLen(2))
					Expect(edgeDevice.ObjectMeta.Labels).To(HaveKeyWithValue("edgedeviceSignedRequest", "true"))
					Expect(edgeDevice.ObjectMeta.Labels).To(HaveKeyWithValue("flotta/member-of", "foo-group"))
				}).
				Return(nil).
				Times(1)

			edgeDeviceSRRepoMock.EXPECT().
				PatchStatus(gomock.Any(), edsr, gomock.Any()).
				Do(func(ctx context.Context, edgedeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, patch *client.Patch) {
					Expect(edgedeviceSignedRequest.Spec.Approved).To(BeTrue())
					Expect(edgedeviceSignedRequest.Status.Conditions).To(HaveLen(1))
					Expect(edgedeviceSignedRequest.Status.Conditions[0].Type).To(Equal(v1alpha1.EdgeDeviceSignedRequestStatusApproved))
				}).
				Return(nil).
				Times(1)

			// when
			result, err := reconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(edsr.Spec.Approved).To(BeTrue())
		})

		It("cannot create device", func() {

			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, targetNamespace).
				Return(nil, fmt.Errorf("cannot retrieve device")).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				Return(fmt.Errorf("Invalid")).
				Times(1)

			// when
			result, err := reconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())
		})

		It("Creates device correctly but fails to patch status", func() {

			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, targetNamespace).
				Return(nil, fmt.Errorf("cannot retrieve device")).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) {
					Expect(edgeDevice.Name).To(Equal(req.Name))
				}).
				Return(nil).
				Times(1)

			edgeDeviceSRRepoMock.EXPECT().
				PatchStatus(gomock.Any(), edsr, gomock.Any()).
				Return(fmt.Errorf("Invalid status")).
				Times(1)

			// when
			result, err := reconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())
		})

	})

})
