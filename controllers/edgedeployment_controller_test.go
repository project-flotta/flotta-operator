package controllers_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/controllers"
	"github.com/project-flotta/flotta-operator/internal/labels"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedeployment"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedevice"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers", func() {

	var (
		edgeDeploymentReconciler *controllers.EdgeDeploymentReconciler
		mockCtrl                 *gomock.Controller
		deployRepoMock           *edgedeployment.MockRepository
		edgeDeviceRepoMock       *edgedevice.MockRepository
		cancelContext            context.CancelFunc
		signalContext            context.Context
		err                      error
		req                      ctrl.Request
	)

	BeforeEach(func() {

		k8sManager := getK8sManager(cfg)

		mockCtrl = gomock.NewController(GinkgoT())
		deployRepoMock = edgedeployment.NewMockRepository(mockCtrl)

		edgeDeviceRepoMock = edgedevice.NewMockRepository(mockCtrl)

		edgeDeploymentReconciler = &controllers.EdgeDeploymentReconciler{
			Client:                   k8sClient,
			Scheme:                   k8sManager.GetScheme(),
			EdgeDeploymentRepository: deployRepoMock,
			EdgeDeviceRepository:     edgeDeviceRepoMock,
			Concurrency:              1,
			ExecuteConcurrent:        controllers.ExecuteConcurrent,
		}

		signalContext, cancelContext = context.WithCancel(context.TODO())
		go func() {
			err = k8sManager.Start(signalContext)
			Expect(err).ToNot(HaveOccurred())
		}()

		req = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test",
				Namespace: "test",
			},
		}
	})

	AfterEach(func() {
		cancelContext()
		mockCtrl.Finish()
	})

	getDevice := func(name string) *v1alpha1.EdgeDevice {
		return &v1alpha1.EdgeDevice{
			ObjectMeta: v1.ObjectMeta{
				Name:      name,
				Namespace: "test",
			},
			Spec: v1alpha1.EdgeDeviceSpec{
				RequestTime: &v1.Time{},
				Heartbeat:   &v1alpha1.HeartbeatConfiguration{},
			},
		}
	}

	Context("Reconcile", func() {
		It("Return nil if no edgedeployment found", func() {
			// given
			returnErr := errors.NewNotFound(schema.GroupResource{Group: "", Resource: "notfound"}, "notfound")
			deployRepoMock.EXPECT().Read(gomock.Any(), req.Name, req.Namespace).Return(nil, returnErr).AnyTimes()
			// when
			res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
		})

		It("Return error if edgedeployment retrieval failed", func() {
			// given
			deployRepoMock.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("Failed")).AnyTimes()

			// when
			res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

			// then
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
		})

		Context("Finalizers", func() {
			var (
				deploymentData *v1alpha1.EdgeDeployment
				finalizers     = []string{"yggdrasil-device-reference-finalizer"}
			)

			BeforeEach(func() {
				deploymentData = &v1alpha1.EdgeDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.EdgeDeploymentSpec{
						DeviceSelector: &v1.LabelSelector{
							MatchLabels: map[string]string{"test": "test"},
						},
						Device: "test",
						Type:   "pod",
						Pod:    v1alpha1.Pod{},
						Data:   &v1alpha1.DataConfiguration{},
					}}

				deployRepoMock.EXPECT().Read(gomock.Any(), deploymentData.Name, deploymentData.Namespace).
					Return(deploymentData, nil).Times(1)
			})

			It("Added finalizer requeue correctly", func() {
				// given
				deployRepoMock.EXPECT().Patch(gomock.Any(), deploymentData, gomock.Any()).
					Return(nil).Do(func(ctx context.Context, old, new *v1alpha1.EdgeDeployment) {
					Expect(new.Finalizers).To(HaveLen(1))
					Expect(new.Finalizers).To(Equal(finalizers))
					Expect(old.Finalizers).To(HaveLen(0))
				}).Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})

			It("Added finalizer failed requeue with error", func() {
				// given
				deployRepoMock.EXPECT().Patch(gomock.Any(), deploymentData, gomock.Any()).
					Return(nil).Do(func(ctx context.Context, old, new *v1alpha1.EdgeDeployment) {
					Expect(new.Finalizers).To(HaveLen(1))
					Expect(new.Finalizers).To(Equal(finalizers))
					Expect(old.Finalizers).To(HaveLen(0))
				}).Return(fmt.Errorf("error")).Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})
		})

		Context("devices selector section", func() {
			var (
				deploymentData *v1alpha1.EdgeDeployment
				device         *v1alpha1.EdgeDevice
			)

			BeforeEach(func() {
				deploymentData = &v1alpha1.EdgeDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Namespace:  "test",
						Finalizers: []string{controllers.YggdrasilDeviceReferenceFinalizer},
						Labels:     map[string]string{labels.CreateSelectorLabel("test"): "true"},
					},
					Spec: v1alpha1.EdgeDeploymentSpec{
						DeviceSelector: &v1.LabelSelector{
							MatchLabels: map[string]string{"test": "test"},
						},
						Type: "test",
						Pod:  v1alpha1.Pod{},
						Data: &v1alpha1.DataConfiguration{},
					}}

				deployRepoMock.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(deploymentData, nil).Times(1)

				device = getDevice("testdevice")
			})

			It("Cannot get edgedevices", func() {

				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("err")).
					Times(1)
				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})

			It("edgedevices return 404", func() {
				// given
				ReturnErr := errors.NewNotFound(
					schema.GroupResource{Group: "", Resource: "notfound"},
					"notfound")

				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{}, ReturnErr).
					Times(2)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

			It("cannot retrieve edgedevices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{}, fmt.Errorf("Invalid")).
					Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})

			It("Add deployments to devices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*device}, nil).
					Times(2)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Patch(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, old, new *v1alpha1.EdgeDevice) {
						Expect(new.Labels).To(Equal(map[string]string{"workload/test": "true"}))
					}).
					Return(nil).
					Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

			It("Only correct devices got deployments", func() {
				// When  running workloads, the Reconcile got all edgedevices that have
				// the label workload/name and all matching devices. If one device does
				// not apply, it'll remove the workload labels
				deviceToDelete := getDevice("todelete")
				deviceToDelete.Status.Deployments = []v1alpha1.Deployment{
					{Name: "test"},
					{Name: "otherWorkload"},
				}

				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, selector *metav1.LabelSelector, namespace string) {
						Expect(selector.MatchLabels).To(Equal(map[string]string{"workload/test": "true"}))
					}).
					Return([]v1alpha1.EdgeDevice{*device, *deviceToDelete}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, selector *metav1.LabelSelector, namespace string) {
						Expect(selector).To(Equal(deploymentData.Spec.DeviceSelector))
					}).
					Return([]v1alpha1.EdgeDevice{*device}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("testdevice"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("todelete"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
						Expect(edgeDevice.Status.Deployments[0].Name).To(Equal("otherWorkload"))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Patch(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, old, new *v1alpha1.EdgeDevice) {
						Expect(new.Labels).To(Equal(map[string]string{"workload/test": "true"}))
					}).
					Return(nil).
					Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})
		})

		Context("using device selector", func() {

			var (
				deploymentData *v1alpha1.EdgeDeployment
				device         *v1alpha1.EdgeDevice
			)

			BeforeEach(func() {
				deploymentData = &v1alpha1.EdgeDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Namespace:  "test",
						Finalizers: []string{controllers.YggdrasilDeviceReferenceFinalizer},
						Labels:     map[string]string{labels.CreateSelectorLabel(labels.DeviceNameLabel): "test"},
					},
					Spec: v1alpha1.EdgeDeploymentSpec{
						Device: "test",
						Type:   "test",
						Pod:    v1alpha1.Pod{},
						Data:   &v1alpha1.DataConfiguration{},
					}}

				deployRepoMock.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(deploymentData, nil).Times(1)

				device = getDevice("testdevice")
			})

			It("Cannot get edgedevices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("err")).
					Times(1)
				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})

			It("edgedevices return 404", func() {
				// given
				ReturnErr := errors.NewNotFound(
					schema.GroupResource{Group: "", Resource: "notfound"},
					"notfound")

				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{}, ReturnErr).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), req.Name, req.Namespace).
					Return(&v1alpha1.EdgeDevice{}, ReturnErr).
					Times(1)

					// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

			It("cannot retrieve edgedevices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{}, fmt.Errorf("Invalid")).
					Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})

			It("Add deployments to devices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*device}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), req.Name, req.Namespace).
					Return(device, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Patch(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, old, new *v1alpha1.EdgeDevice) {
						Expect(new.Labels).To(Equal(map[string]string{"workload/test": "true"}))
					}).
					Return(nil).
					Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

		})

		Context("Remove", func() {

			var (
				deploymentData *v1alpha1.EdgeDeployment
				fooDevice      *v1alpha1.EdgeDevice
				barDevice      *v1alpha1.EdgeDevice
			)

			BeforeEach(func() {
				deploymentData = &v1alpha1.EdgeDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name:              "test",
						Namespace:         "test",
						Finalizers:        []string{controllers.YggdrasilDeviceReferenceFinalizer},
						DeletionTimestamp: &v1.Time{Time: time.Now()},
						Labels:            map[string]string{labels.CreateSelectorLabel("test"): "true"},
					},
					Spec: v1alpha1.EdgeDeploymentSpec{
						DeviceSelector: &v1.LabelSelector{
							MatchLabels: map[string]string{"test": "test"},
						},
						Type: "test",
						Pod:  v1alpha1.Pod{},
						Data: &v1alpha1.DataConfiguration{},
					}}

				deployRepoMock.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(deploymentData, nil).Times(1)
				deployRepoMock.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				fooDevice = getDevice("foo")
				fooDevice.Status.Deployments = []v1alpha1.Deployment{
					{Name: "test"},
					{Name: "otherWorkload"},
				}

				barDevice = getDevice("bar")
				barDevice.Status.Deployments = []v1alpha1.Deployment{
					{Name: "test"},
				}

			})

			It("works as expected", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*fooDevice}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*barDevice}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("bar"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(0))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("foo"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
						Expect(edgeDevice.Status.Deployments[0].Name).To(Equal("otherWorkload"))
					}).
					Return(nil).
					Times(1)

				deployRepoMock.EXPECT().
					RemoveFinalizer(gomock.Any(), gomock.Any(), gomock.Eq("yggdrasil-device-reference-finalizer")).
					Return(nil).Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

			It("Failed to remove workload label", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*fooDevice}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*barDevice}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("bar"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(0))
					}).
					Return(fmt.Errorf("FAILED")).
					Times(1)

					// this should be removed even if the first one failed
				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("foo"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
						Expect(edgeDevice.Status.Deployments[0].Name).To(Equal("otherWorkload"))
					}).
					Return(nil).
					Times(1)

				deployRepoMock.EXPECT().
					RemoveFinalizer(gomock.Any(), gomock.Any(), gomock.Eq("yggdrasil-device-reference-finalizer")).
					Return(nil).Times(0)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})

			It("Failed to remove finalizer label", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*fooDevice}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{*barDevice}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("bar"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(0))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal("foo"))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
						Expect(edgeDevice.Status.Deployments[0].Name).To(Equal("otherWorkload"))
					}).
					Return(nil).
					Times(1)

				deployRepoMock.EXPECT().
					RemoveFinalizer(gomock.Any(), gomock.Any(), gomock.Eq("yggdrasil-device-reference-finalizer")).
					Return(fmt.Errorf("Failed")).Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: 0}))
			})
		})
		Context("Concurrency", func() {
			var (
				deploymentData *v1alpha1.EdgeDeployment
				devices        []v1alpha1.EdgeDevice
				numDevices     = 100
				concurrency    = 7
				expectedSplit  = map[int]int{15: 2, 14: 5}
				actualSplit    map[int]int
				syncMap        = sync.Mutex{}
			)
			executeConcurrent := func(concurrency uint, f controllers.ConcurrentFunc, devices []v1alpha1.EdgeDevice) []error {
				if len(devices) == 0 {
					return nil
				}
				testF := func(devices []v1alpha1.EdgeDevice) []error {
					defer GinkgoRecover()
					errs := f(devices)
					lenErrs := len(errs)
					syncMap.Lock()
					val, ok := actualSplit[lenErrs]
					if ok {
						val += 1
					} else {
						val = 1
					}
					actualSplit[lenErrs] = val
					syncMap.Unlock()
					return errs
				}
				_ = controllers.ExecuteConcurrent(concurrency, testF, devices)
				return nil
			}

			BeforeEach(func() {
				deploymentData = &v1alpha1.EdgeDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Namespace:  "test",
						Labels:     map[string]string{labels.CreateSelectorLabel("test"): "true"},
						Finalizers: []string{controllers.YggdrasilDeviceReferenceFinalizer},
					},
					Spec: v1alpha1.EdgeDeploymentSpec{
						DeviceSelector: &v1.LabelSelector{
							MatchLabels: map[string]string{"test": "test"},
						},
						Type: "test",
						Pod:  v1alpha1.Pod{},
						Data: &v1alpha1.DataConfiguration{},
					}}

				deployRepoMock.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(deploymentData, nil).Times(1)
				deployRepoMock.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				devices = nil
				for i := 0; i < numDevices; i++ {
					devices = append(devices, *getDevice(fmt.Sprintf("testdevice%d", i)))
				}

				actualSplit = map[int]int{}
				edgeDeploymentReconciler.Concurrency = uint(concurrency)
				edgeDeploymentReconciler.ExecuteConcurrent = executeConcurrent
			})

			It("Add deployment to devices", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{}, nil).
					Times(1)
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(devices, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed")).
					Times(numDevices)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
				Expect(actualSplit).To(Equal(expectedSplit))
			})

			It("Delete deployment", func() {
				// given
				deploymentData.ObjectMeta.DeletionTimestamp = &v1.Time{Time: time.Now()}
				for _, d := range devices {
					d.Status.Deployments = []v1alpha1.Deployment{
						{Name: "test"},
					}
				}
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{}, nil).
					Times(1)
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(devices, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed")).
					Times(numDevices)

				deployRepoMock.EXPECT().
					RemoveFinalizer(gomock.Any(), gomock.Any(), gomock.Eq("yggdrasil-device-reference-finalizer")).
					Return(nil).Times(1)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
				Expect(actualSplit).To(Equal(expectedSplit))
			})

			It("Remove deployment from non matching devices", func() {
				// given
				for _, d := range devices {
					d.Status.Deployments = []v1alpha1.Deployment{
						{Name: "test"},
					}
				}
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(devices, nil).
					Times(1)
				edgeDeviceRepoMock.EXPECT().
					ListForSelector(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]v1alpha1.EdgeDevice{}, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed")).
					Times(numDevices)

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
				Expect(actualSplit).To(Equal(expectedSplit))
			})
		})
		Context("Selector labels", func() {
			var (
				deploymentData         *v1alpha1.EdgeDeployment
				expectedSelectorLabels map[string]string
			)

			BeforeEach(func() {
				expectedSelectorLabels = map[string]string{
					labels.CreateSelectorLabel("matchlabel1"):            "true",
					labels.CreateSelectorLabel("matchexp1"):              "true",
					labels.CreateSelectorLabel("matchexp2"):              "true",
					labels.CreateSelectorLabel(labels.DoesNotExistLabel): "true",
				}
				deploymentData = &v1alpha1.EdgeDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Namespace:  "test",
						Finalizers: []string{controllers.YggdrasilDeviceReferenceFinalizer},
					},
					Spec: v1alpha1.EdgeDeploymentSpec{
						DeviceSelector: &v1.LabelSelector{
							MatchLabels: map[string]string{"matchlabel1": "matchlabel1"},
							MatchExpressions: []v1.LabelSelectorRequirement{
								{
									Key: "matchexp1", Operator: metav1.LabelSelectorOpIn, Values: []string{"matchexp1"},
								},
								{
									Key: "matchexp2", Operator: metav1.LabelSelectorOpExists, Values: nil,
								},
								{
									Key: "matchexp3", Operator: metav1.LabelSelectorOpDoesNotExist, Values: nil,
								},
								{
									Key: "matchexp4", Operator: metav1.LabelSelectorOpDoesNotExist, Values: []string{},
								},
							},
						},
						Type: "test",
						Pod:  v1alpha1.Pod{},
						Data: &v1alpha1.DataConfiguration{},
					}}

				deployRepoMock.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(deploymentData, nil).Times(1)

				deployRepoMock.EXPECT().
					Patch(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, old, new *v1alpha1.EdgeDeployment) {
						Expect(new.Labels).To(Equal(expectedSelectorLabels))
					}).Times(1)
			})
			It("New EdgeDeployment", func() {
				// given

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

			It("Updated EdgeDeployment", func() {
				// given
				deploymentData.Labels = map[string]string{
					labels.CreateSelectorLabel("todelete1"): "true",
					labels.CreateSelectorLabel("todelete2"): "true",
				}

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})

			It("Device name label", func() {
				// given
				deploymentData.Spec.Device = "test"
				expectedSelectorLabels = map[string]string{
					labels.CreateSelectorLabel(labels.DeviceNameLabel): "test",
				}

				// when
				res, err := edgeDeploymentReconciler.Reconcile(context.TODO(), req)

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false, RequeueAfter: 0}))
			})
		})
	})
})
