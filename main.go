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
	"fmt"
	"github.com/jakub-dzon/k4e-operator/internal/images"
	"log"
	"net/http"
	"os"

	"github.com/jakub-dzon/k4e-operator/internal/storage"
	routev1 "github.com/openshift/api/route/v1"

	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/yggdrasil"
	"github.com/jakub-dzon/k4e-operator/restapi"
	"github.com/kelseyhightower/envconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/controllers"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

const (
	initialDeviceNamespace = "default"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var Config struct {
	// The port of the HTTP server
	HttpPort uint16 `envconfig:"HTTP_PORT" default:"8888"`

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
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(managementv1alpha1.AddToScheme(scheme))
	utilruntime.Must(obv1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := envconfig.Process("", &Config)
	if err != nil {
		setupLog.Error(err, "unable to process configuration values")
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog.Info("Started with configuration", "configuration", Config)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     Config.MetricsAddr,
		Port:                   Config.WebhookPort,
		HealthProbeBindAddress: Config.ProbeAddr,
		LeaderElection:         Config.EnableLeaderElection,
		LeaderElectionID:       "b9eebab3.k4e.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cache := mgr.GetCache()

	indexFunc := func(obj client.Object) []string {
		return []string{obj.(*managementv1alpha1.EdgeDeployment).Spec.Device}
	}

	if err := cache.IndexField(context.Background(), &managementv1alpha1.EdgeDeployment{}, "spec.device", indexFunc); err != nil {
		panic(err)
	}

	edgeDeviceRepository := edgedevice.NewEdgeDeviceRepository(mgr.GetClient())
	edgeDeploymentRepository := edgedeployment.NewEdgeDeploymentRepository(mgr.GetClient())
	claimer := storage.NewClaimer(mgr.GetClient())

	if err = (&controllers.EdgeDeviceReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		EdgeDeviceRepository: edgeDeviceRepository,
		Claimer:              claimer,
		ObcAutoCreate:        Config.EnableObcAutoCreation,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeDevice")
		os.Exit(1)
	}
	if err = (&controllers.EdgeDeploymentReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		EdgeDeviceRepository:     edgeDeviceRepository,
		EdgeDeploymentRepository: edgeDeploymentRepository,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeDeployment")
		os.Exit(1)
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

	go func() {
		registryAuth := images.NewRegistryAuth(mgr.GetClient())
		h, err := restapi.Handler(restapi.Config{
			YggdrasilAPI: yggdrasil.NewYggdrasilHandler(edgeDeviceRepository, edgeDeploymentRepository, claimer,
				initialDeviceNamespace, mgr.GetEventRecorderFor("edgedeployment-controller"), registryAuth),
		})
		if err != nil {
			setupLog.Error(err, "cannot start http server")
		}
		address := fmt.Sprintf(":%v", Config.HttpPort)
		setupLog.Info("starting http server", "address", address)
		log.Fatal(http.ListenAndServe(address, h))
	}()
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}
