package factory

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/tools/record"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	backendclient "github.com/project-flotta/flotta-operator/backend/client"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedeviceset"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevicesignedrequest"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeworkload"
	"github.com/project-flotta/flotta-operator/internal/common/storage"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/k8s"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/remote"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/configmaps"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/devicemetrics"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/images"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/k8sclient"
)

type Factory struct {
	InitialDeviceNamespace string
	Logger                 *zap.SugaredLogger
	Client                 kubeclient.Client
	EventRecorder          record.EventRecorder
	TLSConfig              *tls.Config
}

func (f *Factory) Create(remoteBackendURL string, remoteBackendTimeout time.Duration) (backend.EdgeDeviceBackend, error) {
	if remoteBackendURL == "" {
		f.Logger.Infof("Using Kubernetes, CRD-based backend")
		return f.createK8sBackend(), nil
	}

	f.Logger.Infof("Using remote, HTTP-based backend")
	return f.createRemoteBackend(remoteBackendURL, remoteBackendTimeout)
}

func (f *Factory) createRemoteBackend(remoteBackendURL string, remoteBackendTimeout time.Duration) (backend.EdgeDeviceBackend, error) {
	backendURL, err := url.Parse(remoteBackendURL)
	if err != nil {
		return nil, err
	}
	var roundTripper http.RoundTripper
	if backendURL.Scheme == "https" {
		roundTripper = &http.Transport{
			TLSClientConfig: f.TLSConfig,
		}
	}
	config := backendclient.Config{
		URL:       backendURL,
		Transport: roundTripper,
	}
	backendApi := backendclient.New(config)
	return remote.NewBackend(f.InitialDeviceNamespace, backendApi, remoteBackendTimeout, f.Logger), nil
}

func (f *Factory) createK8sBackend() backend.EdgeDeviceBackend {
	// For now just one implementation is supported
	k8sClient := k8sclient.NewK8sClient(f.Client)

	edgeDeviceSignedRequestRepository := edgedevicesignedrequest.NewEdgedeviceSignedRequestRepository(f.Client)
	edgeDeviceRepository := edgedevice.NewEdgeDeviceRepository(f.Client)
	edgeWorkloadRepository := edgeworkload.NewEdgeWorkloadRepository(f.Client)
	edgeDeviceSetRepository := edgedeviceset.NewEdgeDeviceSetRepository(f.Client)
	k8sRepository := k8s.NewRepository(edgeDeviceSignedRequestRepository, edgeDeviceRepository, edgeWorkloadRepository,
		edgeDeviceSetRepository, k8sClient)

	claimer := storage.NewClaimer(f.Client)
	registryAuth := images.NewRegistryAuth(f.Client)

	assembler := k8s.NewConfigurationAssembler(
		devicemetrics.NewAllowListGenerator(k8sClient),
		claimer,
		configmaps.NewConfigMap(k8sClient),
		f.EventRecorder,
		registryAuth,
		k8sRepository,
	)
	return k8s.NewBackend(k8sRepository, assembler, f.Logger, f.InitialDeviceNamespace, f.EventRecorder)
}
