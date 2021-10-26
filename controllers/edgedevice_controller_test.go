package controllers_test

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/controllers"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/storage"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("EdgeDevice controller", func() {
	var (
		edgeDeviceReconciler *controllers.EdgeDeviceReconciler
		err                  error
		cancelContext        context.CancelFunc
		signalContext        context.Context

		edgeDeviceRepoMock *edgedevice.MockRepository
		k8sManager         manager.Manager
	)

	BeforeEach(func() {
		k8sManager = getK8sManager(cfg)

		edgeDeviceRepository := edgedevice.NewEdgeDeviceRepository(k8sClient)
		edgeDeviceReconciler = &controllers.EdgeDeviceReconciler{
			Client:               k8sClient,
			Scheme:               k8sManager.GetScheme(),
			EdgeDeviceRepository: edgeDeviceRepository,
			Claimer:              storage.NewClaimer(k8sClient),
			ObcAutoCreate:        false,
		}
		err = edgeDeviceReconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		signalContext, cancelContext = context.WithCancel(context.TODO())
		go func() {
			err = k8sManager.Start(signalContext)
			Expect(err).ToNot(HaveOccurred())
		}()

		mockCtrl := gomock.NewController(GinkgoT())
		edgeDeviceRepoMock = edgedevice.NewMockRepository(mockCtrl)

	})

	AfterEach(func() {
		cancelContext()
		edgeDeviceReconciler.ObcAutoCreate = false
	})

	Context("Reconcile", func() {
		var (
			req ctrl.Request
		)

		BeforeEach(func() {
			req = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test",
					Namespace: "test",
				},
			}

			edgeDeviceReconciler = &controllers.EdgeDeviceReconciler{
				Client:               k8sClient,
				Scheme:               k8sManager.GetScheme(),
				EdgeDeviceRepository: edgeDeviceRepoMock,
				Claimer:              storage.NewClaimer(k8sClient),
				ObcAutoCreate:        false,
			}
		})

		getDevice := func(name string) *v1alpha1.EdgeDevice {
			return &v1alpha1.EdgeDevice{
				ObjectMeta: v1.ObjectMeta{
					Name:      name,
					Namespace: "default",
				},
				Spec: v1alpha1.EdgeDeviceSpec{
					OsImageId:   "test",
					RequestTime: &v1.Time{},
					Heartbeat:   &v1alpha1.HeartbeatConfiguration{},
				},
			}
		}
		It("Edgedevice does not exists on CRD", func() {
			// given
			returnErr := errors.NewNotFound(schema.GroupResource{Group: "", Resource: "notfound"}, "notfound")
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(nil, returnErr).
				Times(1)

			// when
			res, err := edgeDeviceReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
		})

		It("Cannot get edgedevice", func() {
			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(nil, fmt.Errorf("failed")).
				Times(1)

			// when
			res, err := edgeDeviceReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
		})

		It("No ObcAutoCreate", func() {
			// given
			device := getDevice("test")
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(device, nil).
				Times(1)

			edgeDeviceReconciler.ObcAutoCreate = false
			// when
			res, err := edgeDeviceReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
		})

		It("Cannot attach OBC status to device", func() {
			// given
			device := getDevice("test")
			device.Namespace = "test" // to force fail on createOrgetOBC

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(device, nil).
				Times(1)

			edgeDeviceReconciler.ObcAutoCreate = true

			// when
			res, err := edgeDeviceReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
		})

		It("Failed to add OBC reference to device", func() {
			// given
			device := getDevice("test")

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(device, nil).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
					Expect(edgeDevice.Name).To(Equal("test"))
				}).
				Return(fmt.Errorf("failed")).
				Times(1)

			edgeDeviceReconciler.ObcAutoCreate = true

			// when
			res, err := edgeDeviceReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).To(HaveOccurred())
			Expect(res.Requeue).To(BeTrue())
		})

		It("Added OBC reference to device", func() {
			// given
			device := getDevice("test")

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(device, nil).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
					Expect(edgeDevice.Name).To(Equal("test"))
				}).
				Return(nil).
				Times(1)

			edgeDeviceReconciler.ObcAutoCreate = true

			// when
			res, err := edgeDeviceReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Requeue).To(BeFalse())
		})

	})

	It("should not attach OBC to EdgeDevice when OBC auto-creation is disabled", func() {
		// given
		edgeDeviceReconciler.ObcAutoCreate = false

		ctx := context.Background()
		now := metav1.Now()
		edgeDevice := v1alpha1.EdgeDevice{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "no-obc-device",
				Namespace:    "default",
			},
			Spec: v1alpha1.EdgeDeviceSpec{
				RequestTime: &now,
			},
		}

		// when
		err := k8sClient.Create(ctx, &edgeDevice)

		// then
		Expect(err).ToNot(HaveOccurred())
		Consistently(func() *string {
			var ed v1alpha1.EdgeDevice
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&edgeDevice), &ed)
			if err != nil {
				errorString := err.Error()
				return &errorString
			}
			return ed.Status.DataOBC
		}, 6*time.Second, time.Second).Should(BeNil())
	})

	It("should attach OBC to EdgeDevice when OBC auto-creation is enabled", func() {
		// given
		edgeDeviceReconciler.ObcAutoCreate = true

		ctx := context.Background()
		now := metav1.Now()
		edgeDevice := v1alpha1.EdgeDevice{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "no-obc-device",
				Namespace:    "default",
			},
			Spec: v1alpha1.EdgeDeviceSpec{
				RequestTime: &now,
			},
		}

		// when
		err := k8sClient.Create(ctx, &edgeDevice)

		// then
		Expect(err).ToNot(HaveOccurred())

		edgeDeviceKey := client.ObjectKeyFromObject(&edgeDevice)
		Eventually(func() *string {
			var ed v1alpha1.EdgeDevice
			err := k8sClient.Get(ctx, edgeDeviceKey, &ed)
			if err != nil {
				return nil
			}
			return ed.Status.DataOBC
		}, 10*time.Second, 10*time.Millisecond).ShouldNot(BeNil())

		ed := getExpectedEdgeDevice(ctx, edgeDeviceKey)
		var obc obv1.ObjectBucketClaim
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: ed.GetNamespace(), Name: *ed.Status.DataOBC}, &obc)
		Expect(err).ToNot(HaveOccurred())
		Expect(obc.Spec.StorageClassName).To(BeEquivalentTo("openshift-storage.noobaa.io"))
	})
})

func getExpectedEdgeDevice(ctx context.Context, objectKey client.ObjectKey) v1alpha1.EdgeDevice {
	var ed v1alpha1.EdgeDevice
	err := k8sClient.Get(ctx, objectKey, &ed)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return ed
}
