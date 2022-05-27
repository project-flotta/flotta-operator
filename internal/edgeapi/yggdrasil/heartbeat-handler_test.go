package yggdrasil_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/yggdrasil"
)

var _ = Describe("Heartbeat handler", func() {

	var (
		mockCtrl     *gomock.Controller
		mockDelegate *backend.MockHeartbeatHandler
		handler      *yggdrasil.RetryingDelegatingHandler
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDelegate = backend.NewMockHeartbeatHandler(mockCtrl)

		handler = yggdrasil.NewRetryingDelegatingHandler(mockDelegate)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should call delegate", func() {
		// given
		ctx := context.TODO()
		notification := backend.Notification{DeviceID: "1234"}

		mockDelegate.EXPECT().
			Process(ctx, notification).
			Return(false, nil)

		// when
		err := handler.Process(ctx, notification)

		// then
		Expect(err).ToNot(HaveOccurred())
		// mock verification
	})

	It("should retry calling delegate on error and succeed", func() {
		// given
		ctx := context.TODO()
		notification := backend.Notification{DeviceID: "1234"}
		retryNotification := backend.Notification{DeviceID: "1234", Retry: 1}
		errorCall := mockDelegate.EXPECT().
			Process(ctx, notification).
			Return(true, fmt.Errorf("boom"))
		mockDelegate.EXPECT().
			Process(ctx, retryNotification).
			Return(false, nil).
			After(errorCall)

		// when
		err := handler.Process(ctx, notification)

		// then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should retry calling delegate on error and eventually fail", func() {
		// given
		ctx := context.TODO()
		notification := backend.Notification{DeviceID: "1234"}

		mockDelegate.EXPECT().
			Process(ctx, gomock.AssignableToTypeOf(notification)).
			Return(true, fmt.Errorf("boom")).
			Times(4)

		// when
		err := handler.Process(ctx, notification)

		// then
		Expect(err).To(HaveOccurred())
	})
})
