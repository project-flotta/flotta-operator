package yggdrasil_test

import (
	"context"
	"fmt"
	"github.com/jakub-dzon/k4e-operator/internal/images"
	"strings"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/yggdrasil"
	"github.com/jakub-dzon/k4e-operator/models"
	api "github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	operations "github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
)

const (
	testNamespace = "testNS"

	YggdrasilWorkloadFinalizer   = "yggdrasil-workload-finalizer"
	YggdrasilConnectionFinalizer = "yggdrasil-connection-finalizer"

	MessageTypeConnectionStatus string = "connection-status"
	MessageTypeCommand          string = "command"
	MessageTypeEvent            string = "event"
	MessageTypeData             string = "data"
)

var _ = Describe("Yggdrasil", func() {
	var (
		mockCtrl           *gomock.Controller
		deployRepoMock     *edgedeployment.MockRepository
		edgeDeviceRepoMock *edgedevice.MockRepository
		registryAuth       *images.MockRegistryAuthAPI
		handler            *yggdrasil.Handler
		eventsRecorder     *record.FakeRecorder

		errorNotFound = errors.NewNotFound(schema.GroupResource{Group: "", Resource: "notfound"}, "notfound")
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		deployRepoMock = edgedeployment.NewMockRepository(mockCtrl)
		edgeDeviceRepoMock = edgedevice.NewMockRepository(mockCtrl)
		registryAuth = images.NewMockRegistryAuthAPI(mockCtrl)
		eventsRecorder = record.NewFakeRecorder(1)
		handler = yggdrasil.NewYggdrasilHandler(edgeDeviceRepoMock, deployRepoMock, nil, testNamespace, eventsRecorder, registryAuth)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	getDevice := func(name string) *v1alpha1.EdgeDevice {
		return &v1alpha1.EdgeDevice{
			TypeMeta:   v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{Name: name, Namespace: testNamespace},
			Spec: v1alpha1.EdgeDeviceSpec{
				OsImageId:   "test",
				RequestTime: &v1.Time{},
				Heartbeat:   &v1alpha1.HeartbeatConfiguration{},
			},
		}
	}

	Context("GetControlMessageForDevice", func() {
		var (
			params = api.GetControlMessageForDeviceParams{
				DeviceID: "foo",
			}
		)

		It("Can retrieve message correctly", func() {
			// given
			device := getDevice("foo")
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			// when
			res := handler.GetControlMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetControlMessageForDeviceOK()))
		})

		It("Device does not exists", func() {
			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(nil, errorNotFound).
				Times(1)

			// when
			res := handler.GetControlMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetControlMessageForDeviceNotFound()))
		})

		It("Cannot retrieve device", func() {
			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(nil, fmt.Errorf("Failed")).
				Times(1)

			// when
			res := handler.GetControlMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetControlMessageForDeviceInternalServerError()))
		})

		It("Delete without finalizer return ok", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				RemoveFinalizer(gomock.Any(), device, YggdrasilConnectionFinalizer).
				Return(nil).
				Times(1)

			// when
			res := handler.GetControlMessageForDevice(context.TODO(), params)
			data := res.(*api.GetControlMessageForDeviceOK)

			// then
			Expect(data.Payload.Type).To(Equal("command"))
		})

		It("Has the finalizer, will not be deleted", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}
			device.Finalizers = []string{YggdrasilWorkloadFinalizer, YggdrasilConnectionFinalizer}
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			// when
			res := handler.GetControlMessageForDevice(context.TODO(), params)
			// then
			Expect(res).To(Equal(operations.NewGetControlMessageForDeviceOK()))
		})

		It("With other finalizers, will be deleted", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}
			device.Finalizers = []string{YggdrasilConnectionFinalizer}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				RemoveFinalizer(gomock.Any(), device, YggdrasilConnectionFinalizer).
				Return(nil).
				Times(1)

			// when
			res := handler.GetControlMessageForDevice(context.TODO(), params)
			data := res.(*api.GetControlMessageForDeviceOK)

			// then
			Expect(data.Payload.Type).To(Equal("command"))
		})

		It("Remove finalizer failed", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}
			device.Finalizers = []string{YggdrasilConnectionFinalizer}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				RemoveFinalizer(gomock.Any(), device, YggdrasilConnectionFinalizer).
				Return(fmt.Errorf("Failed")).
				Times(1)

			// when
			res := handler.GetControlMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetControlMessageForDeviceInternalServerError()))
		})

	})

	Context("GetDataMessageForDevice", func() {

		var (
			params = api.GetDataMessageForDeviceParams{
				DeviceID: "foo",
			}
		)

		validateAndGetDeviceConfig := func(res middleware.Responder) models.DeviceConfigurationMessage {

			data, ok := res.(*operations.GetDataMessageForDeviceOK)
			ExpectWithOffset(1, ok).To(BeTrue())
			ExpectWithOffset(1, data.Payload.Type).To(Equal(MessageTypeData))

			content, ok := data.Payload.Content.(models.DeviceConfigurationMessage)

			ExpectWithOffset(1, ok).To(BeTrue())
			return content
		}

		It("Device is not in repo", func() {
			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(nil, errorNotFound).
				Times(1)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetDataMessageForDeviceNotFound()))
		})

		It("Device repo failed", func() {
			// given
			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(nil, fmt.Errorf("failed")).
				Times(1)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetDataMessageForDeviceInternalServerError()))
		})

		It("Delete without finalizer", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceOK{}))
			config := validateAndGetDeviceConfig(res)
			Expect(config.DeviceID).To(Equal("foo"))
			Expect(config.Workloads).To(HaveLen(0))
		})

		It("Delete with invalid finalizer", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}
			device.Finalizers = []string{"foo"}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceOK{}))
			config := validateAndGetDeviceConfig(res)
			Expect(config.DeviceID).To(Equal("foo"))
			Expect(config.Workloads).To(HaveLen(0))
		})

		It("Delete finalizer failed", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}
			device.Finalizers = []string{YggdrasilWorkloadFinalizer}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				RemoveFinalizer(gomock.Any(), device, YggdrasilWorkloadFinalizer).
				Return(fmt.Errorf("Failed to remove")).
				Times(1)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetDataMessageForDeviceInternalServerError()))
		})

		It("Delete with finalizer works as expected", func() {
			// given
			device := getDevice("foo")
			device.DeletionTimestamp = &v1.Time{Time: time.Now()}
			device.Finalizers = []string{YggdrasilWorkloadFinalizer}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			edgeDeviceRepoMock.EXPECT().
				RemoveFinalizer(gomock.Any(), device, YggdrasilWorkloadFinalizer).
				Return(nil).
				Times(1)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceOK{}))
			config := validateAndGetDeviceConfig(res)
			Expect(config.DeviceID).To(Equal("foo"))
			Expect(config.Workloads).To(HaveLen(0))
		})

		It("Retrival of deployment failed", func() {
			// given
			device := getDevice("foo")
			device.Status.Deployments = []v1alpha1.Deployment{{Name: "workload1"}}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), "foo", testNamespace).
				Return(device, nil).
				Times(1)

			deployRepoMock.EXPECT().
				Read(gomock.Any(), "workload1", testNamespace).
				Return(nil, fmt.Errorf("Failed"))

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(Equal(operations.NewGetDataMessageForDeviceInternalServerError()))
		})

		It("Cannot find deployment for device status", func() {
			// given
			deviceName := "foo"
			device := getDevice(deviceName)
			device.Status.Deployments = []v1alpha1.Deployment{{Name: "workload1"}}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), deviceName, testNamespace).
				Return(device, nil).
				Times(1)

			deployRepoMock.EXPECT().
				Read(gomock.Any(), "workload1", testNamespace).
				Return(nil, errorNotFound)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceOK{}))
			config := validateAndGetDeviceConfig(res)
			Expect(config.DeviceID).To(Equal(deviceName))
			Expect(config.Workloads).To(HaveLen(0))
		})

		It("Deployment status reported correctly on device status", func() {
			// given
			deviceName := "foo"
			device := getDevice(deviceName)
			device.Status.Deployments = []v1alpha1.Deployment{{Name: "workload1"}}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), deviceName, testNamespace).
				Return(device, nil).
				Times(1)

			deploymentData := &v1alpha1.EdgeDeployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      "workload1",
					Namespace: "default",
				},
				Spec: v1alpha1.EdgeDeploymentSpec{
					DeviceSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{"test": "test"},
					},
					Type: "pod",
					Pod:  v1alpha1.Pod{},
					Data: &v1alpha1.DataConfiguration{},
				}}
			deployRepoMock.EXPECT().
				Read(gomock.Any(), "workload1", testNamespace).
				Return(deploymentData, nil)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceOK{}))
			config := validateAndGetDeviceConfig(res)

			Expect(config.DeviceID).To(Equal(deviceName))
			Expect(config.Workloads).To(HaveLen(1))
			workload := config.Workloads[0]
			Expect(workload.Name).To(Equal("workload1"))
			Expect(workload.ImageRegistries).To(BeNil())
		})

		It("Image registry authfile is included", func() {
			// given
			deviceName := "foo"
			device := getDevice(deviceName)
			device.Status.Deployments = []v1alpha1.Deployment{{Name: "workload1"}}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), deviceName, testNamespace).
				Return(device, nil).
				Times(1)

			deploymentData := &v1alpha1.EdgeDeployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      "workload1",
					Namespace: "default",
				},
				Spec: v1alpha1.EdgeDeploymentSpec{
					Type: "pod",
					Pod:  v1alpha1.Pod{},
					ImageRegistries: &v1alpha1.ImageRegistriesConfiguration{
						AuthFileSecret: &v1alpha1.ObjectRef{
							Name:      "fooSecret",
							Namespace: "fooNamespace",
						},
					},
				}}
			deployRepoMock.EXPECT().
				Read(gomock.Any(), "workload1", testNamespace).
				Return(deploymentData, nil)

			authFileContent := "authfile-content"
			registryAuth.EXPECT().
				GetAuthFileFromSecret(gomock.Any(), gomock.Eq("fooNamespace"), gomock.Eq("fooSecret")).
				Return(authFileContent, nil)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceOK{}))
			config := validateAndGetDeviceConfig(res)

			Expect(config.DeviceID).To(Equal(deviceName))
			Expect(config.Workloads).To(HaveLen(1))
			workload := config.Workloads[0]
			Expect(workload.Name).To(Equal("workload1"))
			Expect(workload.ImageRegistries).To(Not(BeNil()))
			Expect(workload.ImageRegistries.AuthFile).To(Equal(authFileContent))

			Expect(eventsRecorder.Events).ToNot(Receive())
		})

		It("Image registry authfile is included when secret namespace is missing", func() {
			// given
			deviceName := "foo"
			device := getDevice(deviceName)
			device.Status.Deployments = []v1alpha1.Deployment{{Name: "workload1"}}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), deviceName, testNamespace).
				Return(device, nil).
				Times(1)

			deploymentData := &v1alpha1.EdgeDeployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      "workload1",
					Namespace: "default",
				},
				Spec: v1alpha1.EdgeDeploymentSpec{
					Type: "pod",
					Pod:  v1alpha1.Pod{},
					ImageRegistries: &v1alpha1.ImageRegistriesConfiguration{
						AuthFileSecret: &v1alpha1.ObjectRef{
							Name: "fooSecret",
						},
					},
				}}
			deployRepoMock.EXPECT().
				Read(gomock.Any(), "workload1", testNamespace).
				Return(deploymentData, nil)

			authFileContent := "authfile-content"
			registryAuth.EXPECT().
				GetAuthFileFromSecret(gomock.Any(), gomock.Eq(deploymentData.Namespace), gomock.Eq("fooSecret")).
				Return(authFileContent, nil)

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceOK{}))
			config := validateAndGetDeviceConfig(res)

			Expect(config.DeviceID).To(Equal(deviceName))
			Expect(config.Workloads).To(HaveLen(1))
			workload := config.Workloads[0]
			Expect(workload.Name).To(Equal("workload1"))
			Expect(workload.ImageRegistries).To(Not(BeNil()))
			Expect(workload.ImageRegistries.AuthFile).To(Equal(authFileContent))

			Expect(eventsRecorder.Events).ToNot(Receive())
		})

		It("Image registry authfile retrieval error", func() {
			// given
			deviceName := "foo"
			device := getDevice(deviceName)
			device.Status.Deployments = []v1alpha1.Deployment{{Name: "workload1"}}

			edgeDeviceRepoMock.EXPECT().
				Read(gomock.Any(), deviceName, testNamespace).
				Return(device, nil).
				Times(1)

			deploymentData := &v1alpha1.EdgeDeployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      "workload1",
					Namespace: "default",
				},
				Spec: v1alpha1.EdgeDeploymentSpec{
					Type: "pod",
					Pod:  v1alpha1.Pod{},
					ImageRegistries: &v1alpha1.ImageRegistriesConfiguration{
						AuthFileSecret: &v1alpha1.ObjectRef{
							Name:      "fooSecret",
							Namespace: "fooNamespace",
						},
					},
				}}
			deployRepoMock.EXPECT().
				Read(gomock.Any(), "workload1", testNamespace).
				Return(deploymentData, nil)

			registryAuth.EXPECT().
				GetAuthFileFromSecret(gomock.Any(), gomock.Eq("fooNamespace"), gomock.Eq("fooSecret")).
				Return("", fmt.Errorf("failure"))

			// when
			res := handler.GetDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&operations.GetDataMessageForDeviceInternalServerError{}))

			Expect(eventsRecorder.Events).To(HaveLen(1))
			Expect(eventsRecorder.Events).To(Receive(ContainSubstring("Auth file secret")))
		})

	})

	Context("PostDataMessageForDevice", func() {

		var (
			deviceName string
			device     *v1alpha1.EdgeDevice
		)

		BeforeEach(func() {
			deviceName = "foo"
			device = getDevice(deviceName)
		})

		It("Invalid params", func() {
			// given
			params := api.PostDataMessageForDeviceParams{
				DeviceID: deviceName,
				Message: &models.Message{
					Directive: "NOT VALID ONE",
				},
			}

			// when
			res := handler.PostDataMessageForDevice(context.TODO(), params)

			// then
			Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceBadRequest{}))
		})

		Context("Heartbeat", func() {
			var directiveName = "heartbeat"

			It("Device not found", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, errorNotFound).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceNotFound{}))
			})

			It("Device cannot be retrieved", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, fmt.Errorf("failed")).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceInternalServerError{}))
			})

			It("Work without content", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(device, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), device, gomock.Any()).
					Return(nil).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceOK{}))
			})

			It("Work with content", func() {
				// given
				content := models.Heartbeat{
					Status:  "running",
					Version: "1",
					Workloads: []*models.WorkloadStatus{
						{Name: "workload-1", Status: "running"}},
					Hardware: &models.HardwareInfo{
						Hostname: "test-hostname",
					},
				}

				device.Status.Deployments = []v1alpha1.Deployment{{
					Name:  "workload-1",
					Phase: "failing",
				}}

				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(device, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), device, gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
						Expect(edgeDevice.Status.Deployments[0].Phase).To(
							Equal(v1alpha1.EdgeDeploymentPhase("running")))
						Expect(edgeDevice.Status.Deployments[0].Name).To(Equal("workload-1"))
					}).
					Return(nil).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
						Content:   content,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceOK{}))
			})

			It("Work with content and events", func() {
				// given
				content := models.Heartbeat{
					Status:  "running",
					Version: "1",
					Workloads: []*models.WorkloadStatus{
						{Name: "workload-1", Status: "created"}},
					Hardware: &models.HardwareInfo{
						Hostname: "test-hostname",
					},
					Events: []*models.EventInfo{{
						Message: "failed to start container",
						Reason:  "Started",
						Type:    models.EventInfoTypeWarn,
					}},
				}

				device.Status.Deployments = []v1alpha1.Deployment{{
					Name:  "workload-1",
					Phase: "failing",
				}}

				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(device, nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), device, gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Status.Deployments).To(HaveLen(1))
						Expect(edgeDevice.Status.Deployments[0].Phase).To(
							Equal(v1alpha1.EdgeDeploymentPhase("created")))
						Expect(edgeDevice.Status.Deployments[0].Name).To(Equal("workload-1"))
					}).
					Return(nil).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
						Content:   content,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// test emmiting the events:
				close(eventsRecorder.Events)
				found := false
				for event := range eventsRecorder.Events {
					if strings.Contains(event, "failed to start container") {
						found = true
					}
				}
				Expect(found).To(BeTrue())

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceOK{}))
			})

			It("Fail on invalid content", func() {
				// given
				content := "invalid"

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
						Content:   content,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceBadRequest{}))
			})

			It("Fail on update device status", func() {
				// given
				// updateDeviceStatus try to patch the status 4 times, and Read the
				// device from repo too.
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(device, nil).
					Times(4)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), device, gomock.Any()).
					Return(fmt.Errorf("Failed")).
					Times(4)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceInternalServerError{}))
			})

			It("Update device status retries if any error", func() {
				// given
				// updateDeviceStatus try to patch the status 4 times, and Read the
				// device from repo too, in this case will retry 2 times.

				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(device, nil).
					Times(4)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), device, gomock.Any()).
					Return(fmt.Errorf("Failed")).
					Times(3)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), device, gomock.Any()).
					Return(nil).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceOK{}))
			})
		})

		Context("Registration", func() {
			var directiveName = "registration"

			It("Device is already registered", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, nil).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceOK{}))
			})

			It("Read device from repository failed", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, fmt.Errorf("Failed")).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceInternalServerError{}))
			})

			It("Create device without any content", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, errorNotFound).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) {
						Expect(edgeDevice.Name).To(Equal(deviceName))
						Expect(edgeDevice.Namespace).To(Equal(testNamespace))
						Expect(edgeDevice.Finalizers).To(HaveLen(2))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal(deviceName))
						Expect(edgeDevice.Namespace).To(Equal(testNamespace))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(0))
					}).
					Return(nil).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceOK{}))
			})

			It("Create device with valid content", func() {
				// given
				content := models.RegistrationInfo{
					Hardware:  &models.HardwareInfo{Hostname: "fooHostname"},
					OsImageID: "TestOsImageID",
				}

				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, errorNotFound).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) {
						Expect(edgeDevice.Name).To(Equal(deviceName))
						Expect(edgeDevice.Namespace).To(Equal(testNamespace))
						Expect(edgeDevice.Finalizers).To(HaveLen(2))
						Expect(edgeDevice.Spec.OsImageId).To(Equal("TestOsImageID"))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal(deviceName))
						Expect(edgeDevice.Namespace).To(Equal(testNamespace))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(0))
						Expect(edgeDevice.Status.Hardware.Hostname).To(Equal("fooHostname"))
					}).
					Return(nil).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
						Content:   content,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceOK{}))
			})

			It("Create device with invalid content", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, errorNotFound).
					Times(1)

				content := "Invalid--"
				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
						Content:   &content,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceBadRequest{}))
			})

			It("Cannot create device on repo", func() {
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, errorNotFound).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) {
						Expect(edgeDevice.Name).To(Equal(deviceName))
						Expect(edgeDevice.Namespace).To(Equal(testNamespace))
						Expect(edgeDevice.Finalizers).To(HaveLen(2))
					}).
					Return(fmt.Errorf("Failed")).
					Times(1)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceInternalServerError{}))
			})

			It("Update device status failed", func() {
				// retry on status is already tested on heartbeat section
				// given
				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, errorNotFound).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) {
						Expect(edgeDevice.Name).To(Equal(deviceName))
						Expect(edgeDevice.Namespace).To(Equal(testNamespace))
						Expect(edgeDevice.Finalizers).To(HaveLen(2))
					}).
					Return(nil).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, patch *client.Patch) {
						Expect(edgeDevice.Name).To(Equal(deviceName))
						Expect(edgeDevice.Namespace).To(Equal(testNamespace))
						Expect(edgeDevice.Status.Deployments).To(HaveLen(0))
					}).
					Return(fmt.Errorf("Failed")).
					Times(1)

				edgeDeviceRepoMock.EXPECT().
					Read(gomock.Any(), deviceName, testNamespace).
					Return(nil, fmt.Errorf("Failed")).
					Times(3)

				params := api.PostDataMessageForDeviceParams{
					DeviceID: deviceName,
					Message: &models.Message{
						Directive: directiveName,
					},
				}

				// when
				res := handler.PostDataMessageForDevice(context.TODO(), params)

				// then
				Expect(res).To(BeAssignableToTypeOf(&api.PostDataMessageForDeviceInternalServerError{}))
			})

		})

	})
})
