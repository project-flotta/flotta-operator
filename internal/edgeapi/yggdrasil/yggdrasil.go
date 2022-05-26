package yggdrasil

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	"github.com/project-flotta/flotta-operator/internal/common/metrics"
	"github.com/project-flotta/flotta-operator/internal/common/storage"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/k8s"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/configmaps"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/devicemetrics"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/heartbeat"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/images"
	"github.com/project-flotta/flotta-operator/models"
	"github.com/project-flotta/flotta-operator/pkg/mtls"
	apioperations "github.com/project-flotta/flotta-operator/restapi/operations"
	"github.com/project-flotta/flotta-operator/restapi/operations/yggdrasil"
	operations "github.com/project-flotta/flotta-operator/restapi/operations/yggdrasil"
)

const (
	YggdrasilRegisterAuth                                = 1
	YggdrasilCompleteAuth                                = 0
	AuthzKey                         mtls.RequestAuthKey = "APIAuthzkey"
	YggrasilAPIRegistrationOperation                     = "PostDataMessageForDevice"
)

type Handler struct {
	backend          backend.Backend
	initialNamespace string
	metrics          metrics.Metrics
	heartbeatHandler heartbeat.Handler
	mtlsConfig       *mtls.TLSConfig
	logger           *zap.SugaredLogger
}

func NewYggdrasilHandler(claimer *storage.Claimer, initialNamespace string,
	recorder record.EventRecorder, registryAuth images.RegistryAuthAPI, metrics metrics.Metrics,
	allowLists devicemetrics.AllowListGenerator, configMaps configmaps.ConfigMap,
	mtlsConfig *mtls.TLSConfig, logger *zap.SugaredLogger, repository k8s.RepositoryFacade) *Handler {
	assembler := k8s.NewConfigurationAssembler(
		allowLists,
		claimer,
		configMaps,
		recorder,
		registryAuth,
		repository,
	)
	return &Handler{
		initialNamespace: initialNamespace,
		metrics:          metrics,
		heartbeatHandler: heartbeat.NewSynchronousHandler(repository, recorder, metrics, logger),
		mtlsConfig:       mtlsConfig,
		logger:           logger,
		backend:          k8s.NewBackend(repository, assembler, logger, initialNamespace),
	}
}

func IsOwnDevice(ctx context.Context, deviceID string) bool {
	if deviceID == "" {
		return false
	}

	val, ok := ctx.Value(AuthzKey).(mtls.RequestAuthVal)
	if !ok {
		return false
	}
	return val.CommonName == strings.ToLower(deviceID)
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

func (h *Handler) getNamespace(ctx context.Context) string {
	ns := h.initialNamespace

	val, ok := ctx.Value(AuthzKey).(mtls.RequestAuthVal)
	if !ok {
		return ns
	}

	if val.Namespace != "" {
		return val.Namespace
	}
	return ns
}

func (h *Handler) GetControlMessageForDevice(ctx context.Context, params yggdrasil.GetControlMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	if !IsOwnDevice(ctx, deviceID) {
		h.metrics.IncEdgeDeviceInvalidOwnerCounter()
		return operations.NewGetControlMessageForDeviceForbidden()
	}
	logger := h.logger.With("DeviceID", deviceID)

	deregister, err := h.backend.ShouldEdgeDeviceBeUnregistered(ctx, deviceID, h.getNamespace(ctx))
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("edge device is not found")
			return operations.NewGetControlMessageForDeviceNotFound()
		}
		logger.Error(err, "failed to get edge device")
		return operations.NewGetControlMessageForDeviceInternalServerError()
	}

	if deregister {
		h.metrics.IncEdgeDeviceUnregistration()
		message := h.createDisconnectCommand()
		return operations.NewGetControlMessageForDeviceOK().WithPayload(message)
	}

	return operations.NewGetControlMessageForDeviceOK()
}

func (h *Handler) GetDataMessageForDevice(ctx context.Context, params yggdrasil.GetDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	if !IsOwnDevice(ctx, deviceID) {
		h.metrics.IncEdgeDeviceInvalidOwnerCounter()
		return operations.NewGetDataMessageForDeviceForbidden()
	}
	logger := h.logger.With("DeviceID", deviceID)

	dc, err := h.backend.GetDeviceConfiguration(ctx, deviceID, h.getNamespace(ctx))
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("edge device is not found")
			return operations.NewGetDataMessageForDeviceNotFound()
		}
		logger.Error(err, "failed to get edge device configuration")
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
		h.metrics.IncEdgeDeviceInvalidOwnerCounter()
		return operations.NewPostDataMessageForDeviceForbidden()
	}
	return operations.NewPostControlMessageForDeviceOK()
}

func (h *Handler) PostDataMessageForDevice(ctx context.Context, params yggdrasil.PostDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	logger := h.logger.With("DeviceID", deviceID)
	msg := params.Message
	switch msg.Directive {
	case "registration", "enrolment":
		break
	default:
		if !IsOwnDevice(ctx, deviceID) {
			h.metrics.IncEdgeDeviceInvalidOwnerCounter()
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
			Namespace: h.getNamespace(ctx),
			Heartbeat: &hb,
		})
		if err != nil {
			if errors.IsNotFound(err) {
				logger.Debug("Device not found")
				return operations.NewPostDataMessageForDeviceNotFound()
			}
			logger.Error(err, "Device not found")
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
	case "enrolment":
		contentJson, _ := json.Marshal(msg.Content)
		enrolmentInfo := models.EnrolmentInfo{}
		err := json.Unmarshal(contentJson, &enrolmentInfo)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}
		logger.Debug("received enrolment info", "content", enrolmentInfo)

		alreadyCreated, err := h.backend.EnrolEdgeDevice(ctx, deviceID, &enrolmentInfo)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}

		if alreadyCreated {
			return operations.NewPostDataMessageForDeviceAlreadyReported()
		}

		return operations.NewPostDataMessageForDeviceOK()
	case "registration":
		// register new edge device
		contentJson, _ := json.Marshal(msg.Content)
		registrationInfo := models.RegistrationInfo{}
		err := json.Unmarshal(contentJson, &registrationInfo)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}
		logger.Debug("received registration info", "content", registrationInfo)
		res := models.MessageResponse{
			Directive: msg.Directive,
			MessageID: msg.MessageID,
		}
		content := models.RegistrationResponse{}
		ns := h.getNamespace(ctx)

		var isInit bool
		isInit, ns, err = h.backend.InitializeEdgeDeviceRegistration(ctx, deviceID, ns, IsOwnDevice(ctx, deviceID))
		if err != nil {
			logger.Error(err, "can't initialize edge device registration")
			if !errors.IsNotFound(err) {
				h.metrics.IncEdgeDeviceFailedRegistration()
				return operations.NewPostDataMessageForDeviceInternalServerError()
			}

			if _, ok := err.(*backend.NotApproved); !ok {
				h.metrics.IncEdgeDeviceFailedRegistration()
			}
			return operations.NewPostDataMessageForDeviceNotFound()
		}
		cert, err := h.mtlsConfig.SignCSR(registrationInfo.CertificateRequest, deviceID, ns)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}
		content.Certificate = string(cert)

		res.Content = content
		err = h.backend.FinalizeEdgeDeviceRegistration(ctx, deviceID, ns, &registrationInfo)

		if err != nil {
			logger.Error(err, "cannot finalize device registration")
			h.metrics.IncEdgeDeviceFailedRegistration()
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		if isInit {
			logger.Info("EdgeDevice registered correctly for first time")
		} else {
			logger.Info("EdgeDevice renew registration correctly")
		}

		h.metrics.IncEdgeDeviceSuccessfulRegistration()
		return operations.NewPostDataMessageForDeviceOK().WithPayload(&res)
	default:
		logger.Info("received unknown message", "message", msg)
		return operations.NewPostDataMessageForDeviceBadRequest()
	}
	return operations.NewPostDataMessageForDeviceOK()
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
