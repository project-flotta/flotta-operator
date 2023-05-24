/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	routev1 "github.com/openshift/api/route/v1"
	"go.uber.org/zap/zapcore"

	"github.com/project-flotta/flotta-operator/internal/common/indexer"
	"github.com/project-flotta/flotta-operator/internal/common/metrics"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeautoconfig"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeconfig"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevicesignedrequest"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeworkload"
	"github.com/project-flotta/flotta-operator/internal/common/repository/playbookexecution"
	"github.com/project-flotta/flotta-operator/internal/common/storage"
	"github.com/project-flotta/flotta-operator/internal/operator/informers"
	watchers "github.com/project-flotta/flotta-operator/internal/operator/watchers"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/util/flowcontrol"

	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/controllers"
	//+kubebuilder:scaffold:imports
)

const (
	initialDeviceNamespace   = "default"
	defaultOperatorNamespace = "flotta"
	defaultConfigMapName     = "flotta-manager-config"
	logLevelLabel            = "LOG_LEVEL"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	// @TODO read /var/run/secrets/kubernetes.io/serviceaccount/namespace to get
	// the correct namespace if it's installed in k8s
	operatorNamespace = "flotta"
)

var Config struct {

	// The address the metric endpoint binds to.
	MetricsAddr string `envconfig:"METRICS_ADDR" default:":8080"`

	// The address the probe endpoint binds to.
	ProbeAddr string `envconfig:"PROBE_ADDR" default:":8081"`

	// Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
	EnableLeaderElection bool `envconfig:"LEADER_ELECT" default:"false"`

	// WebhookPort is the port that the webhook server serves at.
	WebhookPort int `envconfig:"WEBHOOK_PORT" default:"9443"`

	// Enable OBC auto creation when EdgeDevice is registered
	EnableObcAutoCreation bool `envconfig:"OBC_AUTO_CREATE" default:"false"`

	// Verbosity of the logger.
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Number of concurrent goroutines to create for handling EdgeWorkload reconcile
	EdgeWorkloadConcurrency uint `envconfig:"EDGEWORKLOAD_CONCURRENCY" default:"5"`

	// Number of concurrent goroutines to create for handling EdgeConfig reconcile
	EdgeConfigConcurrency uint `envconfig:"EDGECONFIG_CONCURRENCY" default:"5"`

	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run
	MaxConcurrentReconciles uint `envconfig:"MAX_CONCURRENT_RECONCILES" default:"3"`

	// AutoApprovalprocess enable auto approval on devices
	AutoApproval bool `envconfig:"AUTO_APPROVAL_PROCESS" default:"true"`

	// If Webhooks are enabled, an admission webhook is created and checked when
	// any user submits any change to any project-flotta.io CRD.
	EnableWebhooks bool `envconfig:"ENABLE_WEBHOOKS" default:"true"`
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(managementv1alpha1.AddToScheme(scheme))
	utilruntime.Must(obv1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))

	ns, err := getOperatorNamespace()
	if err != nil {
		log.Fatalf("Cannot get running namespace: %v", err)
	}
	operatorNamespace = ns
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := envconfig.Process("", &Config)
	setupLog = ctrl.Log.WithName("setup")

	if err != nil {
		setupLog.Error(err, "unable to process configuration values")
		os.Exit(1)
	}
	if Config.EdgeWorkloadConcurrency == 0 {
		setupLog.Error(err, "config field EDGEWORKLOAD_CONCURRENCY must be greater than 0")
		os.Exit(1)
	}

	var level zapcore.Level
	err = level.UnmarshalText([]byte(Config.LogLevel))
	if err != nil {
		setupLog.Error(err, "unable to unmarshal log level", "log level", Config.LogLevel)
		os.Exit(1)
	}
	opts := zap.Options{}
	opts.Level = level
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	setupLog = ctrl.Log
	setupLog.Info("Started with configuration", "configuration", Config)

	r, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "Unable to retrieve config")
		os.Exit(1)
	}
	r.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
	mgr, err := ctrl.NewManager(r, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     Config.MetricsAddr,
		Port:                   Config.WebhookPort,
		HealthProbeBindAddress: Config.ProbeAddr,
		LeaderElection:         Config.EnableLeaderElection,
		LeaderElectionID:       "b9eebab3.project-flotta.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	addIndexersToCache(mgr)

	edgeDeviceSignedRequestRepository := edgedevicesignedrequest.NewEdgedeviceSignedRequestRepository(mgr.GetClient())
	edgeDeviceRepository := edgedevice.NewEdgeDeviceRepository(mgr.GetClient())
	edgeWorkloadRepository := edgeworkload.NewEdgeWorkloadRepository(mgr.GetClient())
	edgeAutoConfigRepository := edgeautoconfig.NewEdgeAutoConfigRepository(mgr.GetClient())
	claimer := storage.NewClaimer(mgr.GetClient())
	metricsObj := metrics.New()

	if err = (&controllers.EdgeDeviceSignedRequestReconciler{
		Client:                            mgr.GetClient(),
		Scheme:                            mgr.GetScheme(),
		EdgedeviceSignedRequestRepository: edgeDeviceSignedRequestRepository,
		EdgeDeviceRepository:              edgeDeviceRepository,
		EdgeAutoConfigRepository:          edgeAutoConfigRepository,
		EdgeWorkloadRepository:            edgeWorkloadRepository,
		MaxConcurrentReconciles:           int(Config.MaxConcurrentReconciles),
		AutoApproval:                      Config.AutoApproval,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "edgedeviceSignedRequest")
		os.Exit(1)
	}

	if err = (&controllers.EdgeDeviceReconciler{
		Client:                            mgr.GetClient(),
		Scheme:                            mgr.GetScheme(),
		EdgeDeviceRepository:              edgeDeviceRepository,
		EdgeDeviceSignedRequestRepository: edgeDeviceSignedRequestRepository,
		InitialDeviceNamespace:            initialDeviceNamespace,
		Claimer:                           claimer,
		ObcAutoCreate:                     Config.EnableObcAutoCreation,
		MaxConcurrentReconciles:           int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeDevice")
		os.Exit(1)
	}

	if err = (&controllers.EdgeDeviceLabelsReconciler{
		EdgeDeviceRepository:    edgeDeviceRepository,
		EdgeWorkloadRepository:  edgeWorkloadRepository,
		MaxConcurrentReconciles: int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeDeviceLabels")
		os.Exit(1)
	}
	if err = (&controllers.EdgeWorkloadReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		EdgeDeviceRepository:    edgeDeviceRepository,
		EdgeWorkloadRepository:  edgeWorkloadRepository,
		Concurrency:             Config.EdgeWorkloadConcurrency,
		ExecuteConcurrent:       controllers.ExecuteConcurrent,
		Metrics:                 metricsObj,
		MaxConcurrentReconciles: int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeWorkload")
		os.Exit(1)
	}

	if err = (&controllers.EdgeAutoConfigReconciler{
		Client:                            mgr.GetClient(),
		Scheme:                            mgr.GetScheme(),
		EdgeAutoConfigRepository:          edgeAutoConfigRepository,
		EdgeDeviceSignedRequestRepository: edgeDeviceSignedRequestRepository,
		EdgeDeviceRepository:              edgeDeviceRepository,
		EdgeWorkloadRepository:            edgeWorkloadRepository,

		MaxConcurrentReconciles: int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeAutoConfig")
		os.Exit(1)
	}

	edgeConfigRepository := edgeconfig.NewEdgeConfigRepository(mgr.GetClient())
	playbookExecutionRepository := playbookexecution.NewPlaybookExecutionRepository(mgr.GetClient())

	if err = (&controllers.EdgeConfigReconciler{
		Client:                      mgr.GetClient(),
		Scheme:                      mgr.GetScheme(),
		EdgeConfigRepository:        edgeConfigRepository,
		EdgeDeviceRepository:        edgeDeviceRepository,
		PlaybookExecutionRepository: playbookExecutionRepository,
		Concurrency:                 Config.EdgeConfigConcurrency,
		ExecuteConcurrent:           controllers.ExecuteConcurrent,
		MaxConcurrentReconciles:     int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeConfig")
		os.Exit(1)
	}
	// webhooks
	if Config.EnableWebhooks {
		if err = (&managementv1alpha1.EdgeWorkload{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "EdgeWorkload")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	informer, err := mgr.GetCache().GetInformer(context.Background(), &managementv1alpha1.EdgeDevice{})
	if err != nil {
		setupLog.Error(err, "unable to get EdgeDevice Informer")
		os.Exit(1)
	}
	informer.AddEventHandler(informers.NewEdgeDeviceEventHandler(metricsObj))

	if isInCluster() {
		k8sClient, err := kubernetes.NewForConfig(mgr.GetConfig())
		if err != nil {
			setupLog.Error(err, "cannot get the k8s client set")
			os.Exit(1)
		}
		setupLog.V(1).Info("operator namespace found", "operatorNamespace", operatorNamespace)

		currentConfigMap, err := k8sClient.CoreV1().ConfigMaps(operatorNamespace).Get(context.TODO(), defaultConfigMapName, metav1.GetOptions{})
		if err != nil {
			setupLog.Error(err, "cannot get ConfigMap", "namespace", operatorNamespace)
			os.Exit(1)
		}
		setupLog.V(1).Info("operator configmap found", "operatorNamespace", operatorNamespace, "configmap name", defaultConfigMapName)

		// get the configmap to be watched
		configMapGetter := k8sClient.CoreV1().ConfigMaps(operatorNamespace)
		go watchers.WatchForChanges(configMapGetter, defaultConfigMapName, logLevelLabel, currentConfigMap.Data[logLevelLabel], setupLog, currentConfigMap.ObjectMeta.ResourceVersion, func() { os.Exit(1) })
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func addIndexersToCache(mgr manager.Manager) {
	ctx := context.Background()
	err := mgr.GetFieldIndexer().IndexField(ctx, &managementv1alpha1.EdgeDevice{}, indexer.DeviceByWorkloadIndexKey, indexer.DeviceByWorkloadIndexFunc)
	if err != nil {
		setupLog.Error(err, "Failed to create indexer for EdgeDevice")
		os.Exit(1)
	}
	err = mgr.GetFieldIndexer().IndexField(ctx, &managementv1alpha1.EdgeWorkload{}, indexer.WorkloadByDeviceIndexKey, indexer.WorkloadByDeviceIndexFunc)
	if err != nil {
		setupLog.Error(err, "Failed to create indexer for EdgeWorkload")
		os.Exit(1)
	}
}

func getOperatorNamespace() (operatorNamespace string, err error) {
	if !isInCluster() {
		return defaultOperatorNamespace, nil
	}

	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	ns := strings.TrimSpace(string(nsBytes))
	return ns, nil
}

func isInCluster() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount")
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
