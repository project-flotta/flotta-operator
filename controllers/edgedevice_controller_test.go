package controllers_test

import (
	"context"
	"time"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/controllers"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/storage"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EdgeDevice controller", func() {
	var (
		edgeDeviceReconciler *controllers.EdgeDeviceReconciler
		err                  error
		cancelContext        context.CancelFunc
		signalContext        context.Context
	)

	BeforeEach(func() {
		k8sManager := getK8sManager(cfg)

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
	})

	AfterEach(func() {
		cancelContext()
		edgeDeviceReconciler.ObcAutoCreate = false
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
