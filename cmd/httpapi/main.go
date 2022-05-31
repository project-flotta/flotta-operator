package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/metrics"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/factory"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/yggdrasil"
	"github.com/project-flotta/flotta-operator/pkg/mtls"
	"github.com/project-flotta/flotta-operator/restapi"
	"github.com/project-flotta/flotta-operator/restapi/operations"
)

const (
	initialDeviceNamespace = "default"
)

var (
	operatorNamespace = "flotta"
	scheme            = runtime.NewScheme()
)

var Config struct {

	// The port of the HTTPs server
	HttpsPort uint16 `envconfig:"HTTPS_PORT" default:"8043"`

	// Domain where TLS certificate listen.
	// FIXME check default here
	Domain string `envconfig:"DOMAIN" default:"project-flotta.io"`

	// If TLS server certificates should work on 127.0.0.1
	TLSLocalhostEnabled bool `envconfig:"TLS_LOCALHOST_ENABLED" default:"true"`

	// The address the metric endpoint binds to.
	MetricsAddr string `envconfig:"METRICS_ADDR" default:":8080"`

	// Verbosity of the logger.
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Client Certificate expiration time
	ClientCertExpirationTime uint `envconfig:"CLIENT_CERT_EXPIRATION_DAYS" default:"30"`

	// Kubeconfig specifies path to a kubeconfig file if the server is run outside of a cluster
	Kubeconfig string `envconfig:"KUBECONFIG" default:""`
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(managementv1alpha1.AddToScheme(scheme))
	utilruntime.Must(obv1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
}

func main() {
	err := envconfig.Process("", &Config)
	if err != nil {
		panic(err.Error())
	}
	err, logger := logger(Config.LogLevel)
	if err != nil {
		panic(err.Error())
	}

	clientConfig, err := getRestConfig(Config.Kubeconfig)
	if err != nil {
		logger.Errorf("Cannot prepare k8s client config: %v. Kubeconfig was: %s", err, Config.Kubeconfig)
		panic(err.Error())
	}

	c, err := getClient(clientConfig, client.Options{Scheme: scheme})
	if err != nil {
		logger.Errorf("Cannot create k8s client: %v", err)
		panic(err.Error())
	}

	mtlsConfig := mtls.NewMTLSConfig(c, operatorNamespace, []string{Config.Domain}, Config.TLSLocalhostEnabled)

	err = mtlsConfig.SetClientExpiration(int(Config.ClientCertExpirationTime))
	if err != nil {
		logger.Errorf("Cannot set MTLS client certificate expiration time: %w", err)
	}

	tlsConfig, CACertChain, err := mtlsConfig.InitCertificates()
	if err != nil {
		logger.Errorf("Cannot retrieve any MTLS configuration: %w", err)
		os.Exit(1)
	}

	// @TODO check here what to do with leftovers or if a new one is need to be created
	err = mtlsConfig.CreateRegistrationClientCerts()
	if err != nil {
		logger.Errorf("Cannot create registration client certificate: %w", err)
		os.Exit(1)
	}

	opts := x509.VerifyOptions{
		Roots:         tlsConfig.ClientCAs,
		Intermediates: x509.NewCertPool(),
	}

	metricsObj := metrics.New()

	corev1Client, err := corev1client.NewForConfig(clientConfig)
	if err != nil {
		panic(err)
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: corev1Client.Events("")})
	defer func() {
		broadcaster.Shutdown()
	}()
	eventRecorder := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "flotta-edge-api"})

	backend := factory.Create(initialDeviceNamespace, c, logger, eventRecorder)

	yggdrasilAPIHandler := yggdrasil.NewYggdrasilHandler(
		initialDeviceNamespace,
		metricsObj,
		mtlsConfig,
		logger,
		backend,
	)

	var api *operations.FlottaManagementAPI
	var handler http.Handler

	APIConfig := restapi.Config{
		YggdrasilAPI: yggdrasilAPIHandler,
		InnerMiddleware: func(h http.Handler) http.Handler {
			// This is needed for one reason. Registration endpoint can be
			// triggered with a certificate signed by the CA, but can be expired
			// The main reason to allow expired certificates in this endpoint, it's
			// to renew client certificates, and because some devices can be
			// disconnected for days and does not have the option to renew it.
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.TLS == nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				authType := yggdrasilAPIHandler.GetAuthType(r, api)
				if ok, err := mtls.VerifyRequest(r, authType, opts, CACertChain, yggdrasil.AuthzKey, logger); !ok {
					metricsObj.IncEdgeDeviceFailedAuthenticationCounter()
					logger.Info("cannot verify request:", "authType", authType, "method", r.Method, "url", r.URL, "err", err)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				h.ServeHTTP(w, r)
			})
		},
	}
	handler, api, err = restapi.HandlerAPI(APIConfig)
	if err != nil {
		logger.Errorf("cannot start http server: %w", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", Config.HttpsPort),
		TLSConfig: tlsConfig,
		Handler:   handler,
	}
	go func() {
		logger.Fatal(server.ListenAndServeTLS("", ""))
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(crmetrics.Registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", httpOK)
	mux.HandleFunc("/readyz", httpOK)
	logger.Fatal(http.ListenAndServe(Config.MetricsAddr, mux))
}

func logger(logLevel string) (error, *zap.SugaredLogger) {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return err, nil
	}
	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level.SetLevel(level)
	log, err := logConfig.Build()
	if err != nil {
		return err, nil
	}
	return nil, log.Sugar()
}

func httpOK(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func getRestConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}

func getClient(config *rest.Config, options client.Options) (client.Client, error) {
	c, err := client.New(config, options)
	if err != nil {
		return nil, err
	}

	cacheOpts := cache.Options{
		Scheme: options.Scheme,
		Mapper: options.Mapper,
	}
	objCache, err := cache.New(config, cacheOpts)
	if err != nil {
		return nil, err
	}
	background := context.Background()
	go func() {
		err = objCache.Start(background)
	}()
	if err != nil {
		return nil, err
	}
	if !objCache.WaitForCacheSync(background) {
		return nil, errors.New("cannot sync cache")
	}
	return client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader:     objCache,
		Client:          c,
		UncachedObjects: []client.Object{},
	})
}
