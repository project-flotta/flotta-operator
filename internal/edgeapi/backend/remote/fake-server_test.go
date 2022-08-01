package remote_test

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	"github.com/project-flotta/flotta-operator/backend/models"
	"github.com/project-flotta/flotta-operator/backend/restapi"
	"github.com/project-flotta/flotta-operator/backend/restapi/operations/backend"
	models2 "github.com/project-flotta/flotta-operator/models"
)

var (
	deviceConfiguration = models2.DeviceConfigurationMessage{DeviceID: deviceID}
)

const existingDeviceID = "existing"

type server struct {
	httpServer *http.Server
	address    string
	recorder
}

type happyHandler struct {
	commonRecorder
}

type failingHandler struct {
	commonRecorder
}

type recorder interface {
	GetHeartbeats() []messageDescriptor
	GetEnrolments() []messageDescriptor
	GetRegistrations() []messageDescriptor
}

type commonRecorder struct {
	heartbeats    []messageDescriptor
	enrolments    []messageDescriptor
	registrations []messageDescriptor
}
type messageDescriptor struct {
	deviceID  string
	namespace string
	data      interface{}
}

func NewHappyServer() (*server, error) {
	return NewFakeServer(&happyHandler{})
}

func NewFailingServer() (*server, error) {
	return NewFakeServer(&failingHandler{})
}

func NewFakeServer(targetHandler restapi.BackendAPI) (*server, error) {
	APIConfig := restapi.Config{
		BackendAPI: targetHandler,
	}
	handler, _, err := restapi.HandlerAPI(APIConfig)
	if err != nil {
		return nil, err
	}
	svr := &http.Server{
		Handler: handler,
	}
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	fmt.Println("Using port:", listener.Addr().(*net.TCPAddr).Port)
	go func() {
		_ = svr.Serve(listener)
	}()

	return &server{
		httpServer: svr,
		address:    listener.Addr().String(),
		recorder:   targetHandler.(recorder),
	}, nil
}

func (s *server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (h *happyHandler) EnrolDevice(_ context.Context, params backend.EnrolDeviceParams) middleware.Responder {
	h.enrolments = append(h.enrolments, messageDescriptor{
		deviceID:  params.DeviceID,
		namespace: params.Namespace,
		data:      params.EnrolmentInfo,
	})
	if params.DeviceID == existingDeviceID {
		return backend.NewEnrolDeviceOK()

	}
	return backend.NewEnrolDeviceCreated()
}

func (h *happyHandler) GetDeviceConfiguration(_ context.Context, _ backend.GetDeviceConfigurationParams) middleware.Responder {
	return backend.NewGetDeviceConfigurationOK().WithPayload(&models.DeviceConfigurationResponse{DeviceConfiguration: deviceConfiguration})
}

func (h *happyHandler) GetRegistrationStatus(_ context.Context, _ backend.GetRegistrationStatusParams) middleware.Responder {
	return backend.NewGetRegistrationStatusOK().WithPayload(
		&models.DeviceRegistrationStatusResponse{
			Status:    "registered",
			Namespace: namespace},
	)
}

func (h *happyHandler) RegisterDevice(_ context.Context, params backend.RegisterDeviceParams) middleware.Responder {
	h.registrations = append(h.registrations, messageDescriptor{
		deviceID:  params.DeviceID,
		namespace: params.Namespace,
		data:      params.RegistrationInfo,
	})
	return backend.NewRegisterDeviceOK()
}

func (h *happyHandler) UpdateHeartBeat(_ context.Context, params backend.UpdateHeartBeatParams) middleware.Responder {
	h.heartbeats = append(h.heartbeats, messageDescriptor{
		deviceID:  params.DeviceID,
		namespace: params.Namespace,
		data:      params.Heartbeat,
	})
	return backend.NewUpdateHeartBeatOK()
}

func (h *happyHandler) GetPlaybookExecutions(_ context.Context, _ backend.GetPlaybookExecutionsParams) middleware.Responder {
	return backend.NewGetPlaybookExecutionsOK().
		WithPayload(models2.PlaybookExecutionsResponse{
			&models2.PlaybookExecution{
				AnsiblePlaybookString: "test-playbook",
			},
		})
}

func (h *failingHandler) EnrolDevice(_ context.Context, params backend.EnrolDeviceParams) middleware.Responder {
	h.enrolments = append(h.enrolments, messageDescriptor{
		deviceID:  params.DeviceID,
		namespace: params.Namespace,
		data:      params.EnrolmentInfo,
	})
	return backend.NewEnrolDeviceDefault(500)
}

func (h *failingHandler) GetDeviceConfiguration(_ context.Context, _ backend.GetDeviceConfigurationParams) middleware.Responder {
	return backend.NewGetDeviceConfigurationDefault(500)
}

func (h *failingHandler) GetRegistrationStatus(_ context.Context, _ backend.GetRegistrationStatusParams) middleware.Responder {
	return backend.NewGetRegistrationStatusDefault(500)
}

func (h *failingHandler) RegisterDevice(_ context.Context, params backend.RegisterDeviceParams) middleware.Responder {
	h.registrations = append(h.registrations, messageDescriptor{
		deviceID:  params.DeviceID,
		namespace: params.Namespace,
		data:      params.RegistrationInfo,
	})
	return backend.NewRegisterDeviceDefault(500)
}

func (h *failingHandler) UpdateHeartBeat(_ context.Context, params backend.UpdateHeartBeatParams) middleware.Responder {
	h.heartbeats = append(h.heartbeats, messageDescriptor{
		deviceID:  params.DeviceID,
		namespace: params.Namespace,
		data:      params.Heartbeat,
	})
	return backend.NewUpdateHeartBeatDefault(500)
}

func (h *failingHandler) GetPlaybookExecutions(_ context.Context, _ backend.GetPlaybookExecutionsParams) middleware.Responder {
	return backend.NewGetPlaybookExecutionsDefault(500)
}
func (h *commonRecorder) GetHeartbeats() []messageDescriptor {
	return h.heartbeats
}
func (h *commonRecorder) GetEnrolments() []messageDescriptor {
	return h.enrolments
}
func (h *commonRecorder) GetRegistrations() []messageDescriptor {
	return h.registrations
}
