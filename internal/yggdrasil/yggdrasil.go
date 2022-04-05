package yggdrasil

import (
	"context"
	"encoding/json"
	"github.com/project-flotta/flotta-operator/internal/configmaps"
	"github.com/project-flotta/flotta-operator/internal/devicemetrics"
	"github.com/project-flotta/flotta-operator/internal/heartbeat"
	"github.com/project-flotta/flotta-operator/pkg/mtls"

	"net/http"
	"strings"

	"time"

	"github.com/project-flotta/flotta-operator/internal/k8sclient"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/hardware"
	"github.com/project-flotta/flotta-operator/internal/images"
	"github.com/project-flotta/flotta-operator/internal/metrics"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedeployment"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/storage"
	"github.com/project-flotta/flotta-operator/internal/utils"
	"github.com/project-flotta/flotta-operator/models"
	apioperations "github.com/project-flotta/flotta-operator/restapi/operations"
	"github.com/project-flotta/flotta-operator/restapi/operations/yggdrasil"
	operations "github.com/project-flotta/flotta-operator/restapi/operations/yggdrasil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	YggdrasilConnectionFinalizer                         = "yggdrasil-connection-finalizer"
	YggdrasilWorkloadFinalizer                           = "yggdrasil-workload-finalizer"
	YggdrasilRegisterAuth                                = 1
	YggdrasilCompleteAuth                                = 0
	AuthzKey                         mtls.RequestAuthKey = "APIAuthzkey"
	YggrasilAPIRegistrationOperation                     = "PostDataMessageForDevice"
)

var (
	defaultHeartbeatConfiguration = models.HeartbeatConfiguration{
		HardwareProfile: &models.HardwareProfileConfiguration{},
		PeriodSeconds:   60,
	}
)

type Handler struct {
	deviceRepository       edgedevice.Repository
	deploymentRepository   edgedeployment.Repository
	initialNamespace       string
	metrics                metrics.Metrics
	heartbeatHandler       heartbeat.Handler
	mtlsConfig             *mtls.TLSConfig
	configurationAssembler configurationAssembler
}

type keyMapType = map[string]interface{}
type secretMapType = map[string]keyMapType

func NewYggdrasilHandler(deviceRepository edgedevice.Repository, deploymentRepository edgedeployment.Repository,
	claimer *storage.Claimer, k8sClient k8sclient.K8sClient, initialNamespace string, recorder record.EventRecorder,
	registryAuth images.RegistryAuthAPI, metrics metrics.Metrics, allowLists devicemetrics.AllowListGenerator,
	configMaps configmaps.ConfigMap, mtlsConfig *mtls.TLSConfig) *Handler {
	return &Handler{
		deviceRepository:     deviceRepository,
		deploymentRepository: deploymentRepository,
		initialNamespace:     initialNamespace,
		metrics:              metrics,
		heartbeatHandler:     heartbeat.NewSynchronousHandler(deviceRepository, recorder, metrics),
		mtlsConfig:           mtlsConfig,
		configurationAssembler: configurationAssembler{
			allowLists:             allowLists,
			claimer:                claimer,
			client:                 k8sClient,
			configMaps:             configMaps,
			deploymentRepository:   deploymentRepository,
			recorder:               recorder,
			registryAuthRepository: registryAuth},
	}
}

func IsOwnDevice(ctx context.Context, deviceID string) bool {
	if deviceID == "" {
		return false
	}

	val, ok := ctx.Value(AuthzKey).(string)
	if !ok {
		return false
	}
	return val == strings.ToLower(deviceID)
}

// GetAuthType returns the kind of the authz that need to happen on the API call, the options are:
// YggdrasilCompleteAuth: need to be a valid client certificate and not expired.
// YggdrasilRegisterAuth: it is only valid for registering action.
func (h *Handler) GetAuthType(r *http.Request, api *apioperations.FlottaManagementAPI) int {
	res := YggdrasilCompleteAuth
	if api == nil {
		return res
	}

	route, _, matches := api.Context().RouteInfo(r)
	if !matches {
		return res
	}

	if route != nil && route.Operation != nil {
		if route.Operation.ID == YggrasilAPIRegistrationOperation {
			return YggdrasilRegisterAuth
		}
	}
	return res
}

func (h *Handler) GetControlMessageForDevice(ctx context.Context, params yggdrasil.GetControlMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	if !IsOwnDevice(ctx, deviceID) {
		return operations.NewGetControlMessageForDeviceForbidden()
	}
	logger := log.FromContext(ctx, "DeviceID", deviceID)
	edgeDevice, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("edge device is not found")
			return operations.NewGetControlMessageForDeviceNotFound()
		}
		logger.Error(err, "failed to get edge device")
		return operations.NewGetControlMessageForDeviceInternalServerError()
	}

	if edgeDevice.DeletionTimestamp != nil && !utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilWorkloadFinalizer) {
		if utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilConnectionFinalizer) {
			err = h.deviceRepository.RemoveFinalizer(ctx, edgeDevice, YggdrasilConnectionFinalizer)
			if err != nil {
				return operations.NewGetControlMessageForDeviceInternalServerError()
			}
			h.metrics.IncEdgeDeviceUnregistration()
		}
		message := h.createDisconnectCommand()
		return operations.NewGetControlMessageForDeviceOK().WithPayload(message)
	}
	return operations.NewGetControlMessageForDeviceOK()
}

func (h *Handler) GetDataMessageForDevice(ctx context.Context, params yggdrasil.GetDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	if !IsOwnDevice(ctx, deviceID) {
		return operations.NewGetDataMessageForDeviceForbidden()
	}
	logger := log.FromContext(ctx, "DeviceID", deviceID)
	edgeDevice, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("edge device is not found")
			return operations.NewGetDataMessageForDeviceNotFound()
		}
		logger.Error(err, "failed to get edge device")
		return operations.NewGetDataMessageForDeviceInternalServerError()
	}

	if edgeDevice.DeletionTimestamp != nil {
		if utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilWorkloadFinalizer) {
			err := h.deviceRepository.RemoveFinalizer(ctx, edgeDevice, YggdrasilWorkloadFinalizer)
			if err != nil {
				return operations.NewGetDataMessageForDeviceInternalServerError()
			}
		}
	}
	dc, err := h.configurationAssembler.getDeviceConfiguration(ctx, edgeDevice, logger)
	if err != nil {
		logger.Error(err, "failed to assemble edge device configuration")
		return operations.NewGetDataMessageForDeviceInternalServerError()
	}

	// TODO: Network optimization: Decide whether there is a need to return any payload based on difference between last applied configuration and current state in the cluster.
	message := models.Message{
		Type:      models.MessageTypeData,
		Directive: "device",
		MessageID: uuid.New().String(),
		Version:   1,
		Sent:      strfmt.DateTime(time.Now()),
		Content:   *dc,
	}
	return operations.NewGetDataMessageForDeviceOK().WithPayload(&message)
}

func (h *Handler) PostControlMessageForDevice(ctx context.Context, params yggdrasil.PostControlMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	if !IsOwnDevice(ctx, deviceID) {
		return operations.NewPostDataMessageForDeviceForbidden()
	}
	return operations.NewPostControlMessageForDeviceOK()
}

func (h *Handler) PostDataMessageForDevice(ctx context.Context, params yggdrasil.PostDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	logger := log.FromContext(ctx, "DeviceID", deviceID)
	msg := params.Message
	if msg.Directive != "registration" {
		if !IsOwnDevice(ctx, deviceID) {
			return operations.NewPostDataMessageForDeviceForbidden()
		}
	}
	switch msg.Directive {
	case "heartbeat":
		hb := models.Heartbeat{}
		contentJson, _ := json.Marshal(msg.Content)
		err := json.Unmarshal(contentJson, &hb)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}
		err = h.heartbeatHandler.Process(ctx, heartbeat.Notification{
			DeviceID:  deviceID,
			Namespace: h.initialNamespace,
			Heartbeat: &hb,
		})
		if err != nil {
			if errors.IsNotFound(err) {
				logger.V(1).Info("Device not found")
				return operations.NewPostDataMessageForDeviceNotFound()
			}
			logger.Error(err, "Device not found")
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
	case "registration":
		// register new edge device
		contentJson, _ := json.Marshal(msg.Content)
		registrationInfo := models.RegistrationInfo{}
		err := json.Unmarshal(contentJson, &registrationInfo)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}
		logger.V(1).Info("received registration info", "content", registrationInfo)

		res := models.MessageResponse{
			Directive: msg.Directive,
			MessageID: msg.MessageID,
		}
		content := models.RegistrationResponse{}

		_, err = h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
		if err == nil {

			if !IsOwnDevice(ctx, deviceID) {
				authKeyVal, _ := ctx.Value(AuthzKey).(string)
				logger.V(0).Info("Device tries to re-register with an invalid certificate", "certcn", authKeyVal)
				// At this moment, the registration certificate it's no longer valid,
				// because the CR is already created, and need to be a device
				// certificate.
				return operations.NewPostDataMessageForDeviceForbidden()
			}

			// @TODO remove this IF when MTLS is finished
			if registrationInfo.CertificateRequest != "" {
				cert, err := h.mtlsConfig.SignCSR(registrationInfo.CertificateRequest, deviceID)
				if err != nil {
					return operations.NewPostDataMessageForDeviceBadRequest()
				}
				content.Certificate = string(cert)
			}
			res.Content = content
			return operations.NewPostDataMessageForDeviceOK().WithPayload(&res)
		}

		if !errors.IsNotFound(err) {
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		// @TODO here the base certificate should be the same CN as DeviceID and only expired.
		// @TODO remove this IF when MTLS is finished
		// @TODO remove this lines on ECOPROJECT-402
		if registrationInfo.CertificateRequest != "" {
			cert, err := h.mtlsConfig.SignCSR(registrationInfo.CertificateRequest, deviceID)
			if err != nil {
				return operations.NewPostDataMessageForDeviceBadRequest()
			}
			content.Certificate = string(cert)
			res.Content = content
		}

		now := metav1.Now()
		device := v1alpha1.EdgeDevice{
			Spec: v1alpha1.EdgeDeviceSpec{
				RequestTime: &now,
			},
		}
		device.Name = deviceID
		device.Namespace = h.initialNamespace
		device.Finalizers = []string{YggdrasilConnectionFinalizer, YggdrasilWorkloadFinalizer}
		err = h.deviceRepository.Create(ctx, &device)
		if err != nil {
			logger.Error(err, "cannot save EdgeDevice")
			h.metrics.IncEdgeDeviceFailedRegistration()
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		err = h.updateDeviceStatus(ctx, &device, func(device *v1alpha1.EdgeDevice) {
			device.Status = v1alpha1.EdgeDeviceStatus{
				Hardware: hardware.MapHardware(registrationInfo.Hardware),
			}
		})

		if err != nil {
			logger.Error(err, "cannot update EdgeDevice status")
			h.metrics.IncEdgeDeviceFailedRegistration()
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		err = h.deviceRepository.UpdateLabels(ctx, &device, hardware.MapLabels(registrationInfo.Hardware))
		if err != nil {
			logger.Error(err, "cannot update EdgeDevice labels")
			h.metrics.IncEdgeDeviceFailedRegistration()
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		logger.Info("EdgeDevice created")
		h.metrics.IncEdgeDeviceSuccessfulRegistration()

		return operations.NewPostDataMessageForDeviceOK().WithPayload(&res)
	default:
		logger.Info("received unknown message", "message", msg)
		return operations.NewPostDataMessageForDeviceBadRequest()
	}
	return operations.NewPostDataMessageForDeviceOK()
}

func (h *Handler) updateDeviceStatus(ctx context.Context, device *v1alpha1.EdgeDevice, updateFunc func(d *v1alpha1.EdgeDevice)) error {
	patch := client.MergeFrom(device.DeepCopy())
	updateFunc(device)
	err := h.deviceRepository.PatchStatus(ctx, device, &patch)
	if err == nil {
		return nil
	}

	// retry patching the edge device status
	for i := 1; i < 4; i++ {
		time.Sleep(time.Duration(i*50) * time.Millisecond)
		device2, err := h.deviceRepository.Read(ctx, device.Name, device.Namespace)
		if err != nil {
			continue
		}
		patch = client.MergeFrom(device2.DeepCopy())
		updateFunc(device2)
		err = h.deviceRepository.PatchStatus(ctx, device2, &patch)
		if err == nil {
			return nil
		}
	}
	return err
}

func (h *Handler) createDisconnectCommand() *models.Message {
	command := struct {
		Command   string            `json:"command"`
		Arguments map[string]string `json:"arguments"`
	}{
		Command: "disconnect",
	}

	return &models.Message{
		Type:      models.MessageTypeCommand,
		MessageID: uuid.New().String(),
		Version:   1,
		Sent:      strfmt.DateTime(time.Now()),
		Content:   command,
	}
}
