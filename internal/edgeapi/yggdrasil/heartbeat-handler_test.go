package yggdrasil_test

import (
	"context"
	"fmt"
	"github.com/project-flotta/flotta-operator/models"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/yggdrasil"
)

var _ = Describe("Heartbeat handler", func() {

	const (
		deviceID        = "dev-ns"
		deviceNamespace = "dev-ns"
	)
	var (
		mockCtrl     *gomock.Controller
		mockDelegate *yggdrasil.MockStatusUpdater
		handler      *yggdrasil.RetryingDelegatingHandler
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDelegate = yggdrasil.NewMockStatusUpdater(mockCtrl)

		handler = yggdrasil.NewRetryingDelegatingHandler(mockDelegate)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should call delegate", func() {
		// given
		ctx := context.TODO()
		initialContext := context.WithValue(ctx, backend.RetryContextKey, false)
		heartbeat := &models.Heartbeat{}

		mockDelegate.EXPECT().
			UpdateStatus(initialContext, deviceID, deviceNamespace, heartbeat).
			Return(false, nil)

		// when
		err := handler.Process(ctx, deviceID, deviceNamespace, heartbeat)

		// then
		Expect(err).ToNot(HaveOccurred())
		// mock verification
	})

	It("should retry calling delegate on error and succeed", func() {
		// given
		ctx := context.TODO()
		heartbeat := &models.Heartbeat{}
		initialContext := context.WithValue(ctx, backend.RetryContextKey, false)
		retryContext := context.WithValue(ctx, backend.RetryContextKey, true)
		errorCall := mockDelegate.EXPECT().
			UpdateStatus(initialContext, deviceID, deviceNamespace, heartbeat).
			Return(true, fmt.Errorf("boom"))
		mockDelegate.EXPECT().
			UpdateStatus(retryContext, deviceID, deviceNamespace, heartbeat).
			Return(false, nil).
			After(errorCall)

		// when
		err := handler.Process(ctx, deviceID, deviceNamespace, heartbeat)

		// then
		Expect(err).ToNot(HaveOccurred())
	})

	It("should retry calling delegate on error and eventually fail", func() {
		// given
		ctx := context.TODO()
		initialContext := context.WithValue(ctx, backend.RetryContextKey, false)
		retryCtx := context.WithValue(ctx, backend.RetryContextKey, true)
		heartbeat := &models.Heartbeat{}

		initialCall := mockDelegate.EXPECT().
			UpdateStatus(initialContext, deviceID, deviceNamespace, gomock.AssignableToTypeOf(heartbeat)).
			Return(true, fmt.Errorf("boom"))

		mockDelegate.EXPECT().
			UpdateStatus(retryCtx, deviceID, deviceNamespace, gomock.AssignableToTypeOf(heartbeat)).
			Return(true, fmt.Errorf("boom")).
			Times(3).
			After(initialCall)

		// when
		err := handler.Process(ctx, deviceID, deviceNamespace, heartbeat)

		// then
		Expect(err).To(HaveOccurred())
	})
})
