package controllers_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/controllers"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeconfig"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/playbookexecution"
)

var _ = Describe("EdgeConfig controller", func() {
	var (
		edgeConfigReconciler *controllers.EdgeConfigReconciler
		err                  error
		cancelContext        context.CancelFunc
		signalContext        context.Context

		edgeConfigRepoMock   *edgeconfig.MockRepository
		edgeDeviceRepoMock   *edgedevice.MockRepository
		playbookExecRepoMock *playbookexecution.MockRepository
		k8sManager           manager.Manager
		edgeConfigName       = "edgeconfig-test"
		edgeConfigLabel      = map[string]string{"config/device-by-config": edgeConfigName}
		namespace            = "test"
	)

	BeforeEach(func() {
		k8sManager = getK8sManager(cfg)
		mockCtrl := gomock.NewController(GinkgoT())
		edgeConfigRepository := edgeconfig.NewEdgeConfigRepository(k8sClient)
		edgeDeviceRepository := edgedevice.NewEdgeDeviceRepository(k8sClient)
		playbookExecRepository := playbookexecution.NewPlaybookExecutionRepository(k8sClient)
		edgeConfigReconciler = &controllers.EdgeConfigReconciler{
			Client:                      k8sClient,
			Scheme:                      k8sManager.GetScheme(),
			EdgeConfigRepository:        edgeConfigRepository,
			EdgeDeviceRepository:        edgeDeviceRepository,
			PlaybookExecutionRepository: playbookExecRepository,
			Concurrency:                 1,
			MaxConcurrentReconciles:     1,
		}
		err = edgeConfigReconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		signalContext, cancelContext = context.WithCancel(context.TODO())
		go func() {
			err = k8sManager.Start(signalContext)
			Expect(err).ToNot(HaveOccurred())
		}()

		edgeConfigRepoMock = edgeconfig.NewMockRepository(mockCtrl)
		edgeDeviceRepoMock = edgedevice.NewMockRepository(mockCtrl)
		playbookExecRepoMock = playbookexecution.NewMockRepository(mockCtrl)
	})
	AfterEach(func() {
		cancelContext()
	})

	Context("Reconcile", func() {
		var (
			req ctrl.Request
		)

		BeforeEach(func() {
			req = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      edgeConfigName,
					Namespace: namespace,
				},
			}

			edgeConfigReconciler = &controllers.EdgeConfigReconciler{
				Client:                      k8sClient,
				Scheme:                      k8sManager.GetScheme(),
				EdgeConfigRepository:        edgeConfigRepoMock,
				EdgeDeviceRepository:        edgeDeviceRepoMock,
				PlaybookExecutionRepository: playbookExecRepoMock,
				Concurrency:                 1,
				MaxConcurrentReconciles:     1,
			}
		})

		// edgePlaybookSpec := v1alpha1.EdgePlaybookSpec{
		// 	Playbooks: []v1alpha1.Playbook{
		// 		{
		// 			Content: []byte("test"),
		// 		},
		// 	},
		// }

		// getEdgeConfig := func(name string) *v1alpha1.EdgeConfig {
		// 	return &v1alpha1.EdgeConfig{
		// 		ObjectMeta: v1.ObjectMeta{
		// 			Name:      name,
		// 			Namespace: namespace,
		// 		},
		// 		Spec: v1alpha1.EdgeConfigSpec{
		// 			EdgePlaybook: &edgePlaybookSpec,
		// 		},
		// 	}
		// }
		It("EdgeConfig does not exists on CRD", func() {
			// given
			returnErr := errors.NewNotFound(schema.GroupResource{Group: "", Resource: "notfound"}, "notfound")
			edgeConfigRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(nil, returnErr).
				Times(1)

			// when
			res, err := edgeConfigReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
		})
		It("Cannot get edgeconfig", func() {
			// given
			edgeConfigRepoMock.EXPECT().
				Read(gomock.Any(), req.Name, req.Namespace).
				Return(nil, fmt.Errorf("failed")).
				Times(1)

			// when
			res, err := edgeConfigReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
		})

		Context("edgeDevice selection", func() {
			var (
				edgeConfigData *v1alpha1.EdgeConfig
				device         *v1alpha1.EdgeDevice
				namespace      = "default"
			)
			getDevice := func(name string) *v1alpha1.EdgeDevice {
				return &v1alpha1.EdgeDevice{
					ObjectMeta: v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: v1alpha1.EdgeDeviceSpec{
						RequestTime: &v1.Time{},
						Heartbeat:   &v1alpha1.HeartbeatConfiguration{},
					},
				}
			}
			BeforeEach(func() {
				edgeConfigData = &v1alpha1.EdgeConfig{
					ObjectMeta: v1.ObjectMeta{
						Name:       edgeConfigName,
						Namespace:  namespace,
						Finalizers: []string{controllers.YggdrasilDeviceReferenceFinalizer},
						Labels:     edgeConfigLabel,
					},
					Spec: v1alpha1.EdgeConfigSpec{
						EdgePlaybook: &v1alpha1.EdgePlaybookSpec{
							Playbooks: []v1alpha1.Playbook{
								{
									Content: []byte("test"),
								},
							},
						},
					}}

				edgeConfigRepoMock.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(edgeConfigData, nil).Times(1)

				device = getDevice("testdevice")
			})
			It("Cannot get edgedevices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForEdgeConfig(gomock.Any(), gomock.Any(), namespace).
					Return(nil, fmt.Errorf("err")).
					Times(1)
				// when
				res, err := edgeConfigReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})
			It("Create PlaybookExecution for a devices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForEdgeConfig(gomock.Any(), gomock.Any(), namespace).
					Return([]v1alpha1.EdgeDevice{*device}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Status.PlaybookExecutions).To(HaveLen(1))
					}).
					Return(nil).
					Times(1)

				playbookExecRepoMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// when
				res, err := edgeConfigReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

		})

	})

})

func getExpectedEdgeConfig(ctx context.Context, objectKey client.ObjectKey) v1alpha1.EdgeConfig {
	var ed v1alpha1.EdgeConfig
	err := k8sClient.Get(ctx, objectKey, &ed)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return ed
}
