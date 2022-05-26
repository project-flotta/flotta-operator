package factory

import (
	"go.uber.org/zap"
	"k8s.io/client-go/tools/record"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedeviceset"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevicesignedrequest"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeworkload"
	"github.com/project-flotta/flotta-operator/internal/common/storage"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/k8s"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/configmaps"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/devicemetrics"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/images"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/k8sclient"
)

func CreateBackend(initialDeviceNamespace string, client kubeclient.Client, logger *zap.SugaredLogger, eventRecorder record.EventRecorder) backend.Backend {
	// For now just one implementation is supported
	k8sClient := k8sclient.NewK8sClient(client)

	edgeDeviceSignedRequestRepository := edgedevicesignedrequest.NewEdgedeviceSignedRequestRepository(client)
	edgeDeviceRepository := edgedevice.NewEdgeDeviceRepository(client)
	edgeWorkloadRepository := edgeworkload.NewEdgeWorkloadRepository(client)
	edgeDeviceSetRepository := edgedeviceset.NewEdgeDeviceSetRepository(client)
	k8sRepository := k8s.NewRepository(edgeDeviceSignedRequestRepository, edgeDeviceRepository, edgeWorkloadRepository,
		edgeDeviceSetRepository, k8sClient)

	claimer := storage.NewClaimer(client)
	registryAuth := images.NewRegistryAuth(client)

	assembler := k8s.NewConfigurationAssembler(
		devicemetrics.NewAllowListGenerator(k8sClient),
		claimer,
		configmaps.NewConfigMap(k8sClient),
		eventRecorder,
		registryAuth,
		k8sRepository,
	)
	return k8s.NewBackend(k8sRepository, assembler, logger, initialDeviceNamespace, eventRecorder)
}
