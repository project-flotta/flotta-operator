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
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"github.com/project-flotta/flotta-operator/internal/configmaps"
	"github.com/project-flotta/flotta-operator/internal/devicemetrics"

	"github.com/kelseyhightower/envconfig"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/project-flotta/flotta-operator/internal/images"
	"github.com/project-flotta/flotta-operator/internal/k8sclient"
	"github.com/project-flotta/flotta-operator/internal/metrics"
	"github.com/project-flotta/flotta-operator/internal/mtls"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedeployment"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/storage"
	"github.com/project-flotta/flotta-operator/internal/yggdrasil"
	"github.com/project-flotta/flotta-operator/restapi"
	watchers "github.com/project-flotta/flotta-operator/watchers"

	"go.uber.org/zap/zapcore"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/util/flowcontrol"

	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
	_ "net/http/pprof"
)

var (
	profileAddress = "localhost:6060"
)

const (
	initialDeviceNamespace   = "default"
	defaultOperatorNamespace = "flotta"
	defaultConfigMapName     = "flotta-operator-manager-config"
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

	// NOTE DEPRECATE
	// The port of the HTTP server
	HttpPort uint16 `envconfig:"HTTP_PORT" default:"8888"`

	// The port of the HTTPs server
	HttpsPort uint16 `envconfig:"HTTPS_PORT" default:"8043"`

	// Domain where TLS certificate listen.
	// FIXME check default here
	Domain string `envconfig:"DOMAIN" default:"project-flotta.io"`

	// If TLS server certificates should work on 127.0.0.1
	TLSLocalhostEnabled bool `envconfig:"TLS_LOCALHOST_ENABLED" default:"true"`

	// The address the metric endpoint binds to.
	MetricsAddr string `envconfig:"METRICS_ADDR" default:":8080"`

	// The address the probe endpoint binds to.
	ProbeAddr string `envconfig:"PROBE_ADDR" default:":8081"`

	// Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
	EnableLeaderElection bool `envconfig:"LEADER_ELECT" default:"false"`

	// WebhookPort is the port that the webhook server serves at.
	WebhookPort int `envconfig:"WEBHOOK_PORT" default:"9443"`

	// Enable OBC auto creation when EdgeDevice is registered
	EnableObcAutoCreation bool `envconfig:"OBC_AUTO_CREATE" default:"true"`

	// Verbosity of the logger.
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Number of concurrent goroutines to create for handling EdgeDeployment reconcile
	EdgeDeploymentConcurrency uint `envconfig:"EDGEDEPLOYMENT_CONCURRENCY" default:"5"`

	// Client Certificate expiration time
	ClientCertExpirationTime uint `envconfig:"CLIENT_CERT_EXPIRATION_DAYS" default:"30"`

	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run
	MaxConcurrentReconciles uint `envconfig:"MAX_CONCURRENT_RECONCILES" default:"3"`

	// EnableProfiling enables profiling
	EnableProfiling bool `envconfig:"ENABLE_PROFILING" default:"false"`
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

	if Config.EnableProfiling {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		go func() {
			log.Println(http.ListenAndServe(":6060", mux))
		}()
	}

	if Config.EdgeDeploymentConcurrency == 0 {
		setupLog.Error(err, "config field EDGEDEPLOYMENT_CONCURRENCY must be greater than 0")
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

	edgeDeviceRepository := edgedevice.NewEdgeDeviceRepository(mgr.GetClient())
	edgeDeploymentRepository := edgedeployment.NewEdgeDeploymentRepository(mgr.GetClient())
	claimer := storage.NewClaimer(mgr.GetClient())
	metricsObj := metrics.New()

	if err = (&controllers.EdgeDeviceReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		EdgeDeviceRepository:    edgeDeviceRepository,
		Claimer:                 claimer,
		ObcAutoCreate:           Config.EnableObcAutoCreation,
		MaxConcurrentReconciles: int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeDevice")
		os.Exit(1)
	}
	if err = (&controllers.EdgeDeviceLabelsReconciler{
		EdgeDeviceRepository:     edgeDeviceRepository,
		EdgeDeploymentRepository: edgeDeploymentRepository,
		MaxConcurrentReconciles:  int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeDeviceLabels")
		os.Exit(1)
	}
	if err = (&controllers.EdgeDeploymentReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		EdgeDeviceRepository:     edgeDeviceRepository,
		EdgeDeploymentRepository: edgeDeploymentRepository,
		Concurrency:              Config.EdgeDeploymentConcurrency,
		ExecuteConcurrent:        controllers.ExecuteConcurrent,
		Metrics:                  metricsObj,
		MaxConcurrentReconciles:  int(Config.MaxConcurrentReconciles),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeDeployment")
		os.Exit(1)
	}

	// webhooks
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&v1alpha1.EdgeDeployment{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "EdgeDeployment")
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
	registryAuth := images.NewRegistryAuth(mgr.GetClient())
	go func() {

		if !mgr.GetCache().WaitForCacheSync(context.TODO()) {
			setupLog.Error(err, "Cache cannot start")
			os.Exit(1)
		}

		mtlsConfig := mtls.NewMTLSConfig(mgr.GetClient(), operatorNamespace,
			[]string{Config.Domain}, Config.TLSLocalhostEnabled)

		err = mtlsConfig.SetClientExpiration(int(Config.ClientCertExpirationTime))
		if err != nil {
			setupLog.Error(err, "Cannot set MTLS client certificate expiration time")
		}

		tlsConfig, CACertChain, err := mtlsConfig.InitCertificates()
		if err != nil {
			setupLog.Error(err, "Cannot retrieve any MTLS configuration")
			os.Exit(1)
		}

		// @TODO check here what to do with leftovers or if a new one is need to be created
		err = mtlsConfig.CreateRegistrationClientCerts()
		if err != nil {
			setupLog.Error(err, "Cannot create registration client certificate")
			os.Exit(1)
		}

		opts := x509.VerifyOptions{
			Roots:         tlsConfig.ClientCAs,
			Intermediates: x509.NewCertPool(),
		}

		k8sClient := k8sclient.NewK8sClient(mgr.GetClient())

		yggdrasilAPIHandler := yggdrasil.NewYggdrasilHandler(
			edgeDeviceRepository,
			edgeDeploymentRepository,
			claimer,
			k8sClient,
			initialDeviceNamespace,
			mgr.GetEventRecorderFor("edgedeployment-controller"),
			registryAuth,
			metricsObj,
			devicemetrics.NewAllowListGenerator(k8sClient),
			configmaps.NewConfigMap(k8sClient),
			mtlsConfig,
		)

		h, err := restapi.Handler(restapi.Config{
			YggdrasilAPI: yggdrasilAPIHandler,
			InnerMiddleware: func(h http.Handler) http.Handler {
				// This is needed for one reason. Registration endpoint can be
				// triggered with a certificate signed by the CA, but can be expired
				// The main reason to allow expired certificates in this endpoint, it's
				// to renew client certificates, and because some devices can be
				// disconnected for days and does not have the option to renew it.
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.TLS != nil {
						authType := yggdrasilAPIHandler.GetAuthType(r)
						if !mtls.VerifyRequest(r, authType, opts, CACertChain) {
							w.WriteHeader(http.StatusUnauthorized)
							return
						}
					}
					h.ServeHTTP(w, r)
				})
			},
		})

		if err != nil {
			setupLog.Error(err, "cannot start http server")
		}

		//@TODO This is a hack to keep compatibility now, to be deleted.
		go func() {
			_ = http.ListenAndServe(fmt.Sprintf(":%v", Config.HttpPort), h)
		}()

		server := &http.Server{
			Addr:      fmt.Sprintf(":%v", Config.HttpsPort),
			TLSConfig: tlsConfig,
			Handler:   h,
		}
		log.Fatal(server.ListenAndServeTLS("", ""))
	}()

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
		go watchers.WatchForChanges(k8sClient, operatorNamespace, defaultConfigMapName, logLevelLabel, currentConfigMap.Data[logLevelLabel], setupLog)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getOperatorNamespace() (operatorNamespace string, err error) {
	if !isInCluster() {
		return defaultOperatorNamespace, nil
	}

	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
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
