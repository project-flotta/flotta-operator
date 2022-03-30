package informers_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/informers"
	mtrcs "github.com/project-flotta/flotta-operator/internal/metrics"
	"k8s.io/client-go/tools/cache"
)

var _ = Describe("EdgeDevice informer event handler", func() {

	var (
		mockCtrl *gomock.Controller

		handler     cache.ResourceEventHandler
		metricsMock *mtrcs.MockMetrics
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		metricsMock = mtrcs.NewMockMetrics(mockCtrl)
		handler = informers.NewEdgeDeviceEventHandler(metricsMock)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should register device metric on add", func() {
		// given
		device := v1alpha1.EdgeDevice{}
		device.Name = "test"
		device.Namespace = "test-ns"

		// then
		metricsMock.EXPECT().RegisterDeviceCounter(device.Namespace, device.Name)

		// when
		handler.OnAdd(&device)
	})

	It("should register device metric on delete", func() {
		// given
		device := v1alpha1.EdgeDevice{}
		device.Name = "test"
		device.Namespace = "test-ns"

		// then
		metricsMock.EXPECT().RemoveDeviceCounter(device.Namespace, device.Name)

		// when
		handler.OnDelete(&device)
	})
})
