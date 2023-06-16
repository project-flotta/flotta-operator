package edgeapi

import "time"

type Config struct {

	// The port of the HTTPs server
	HttpsPort uint16 `envconfig:"HTTPS_PORT" default:"8043"`

	// Domain where TLS certificate listen.
	// FIXME check default here
	Domain string `envconfig:"DOMAIN" default:"project-flotta.io"`

	// If TLS server certificates should work on 127.0.0.1
	TLSLocalhostEnabled bool `envconfig:"TLS_LOCALHOST_ENABLED" default:"false"`

	// The address the metric endpoint binds to.
	MetricsAddr string `envconfig:"METRICS_ADDR" default:":8080"`

	// Verbosity of the logger.
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Client Certificate expiration time
	ClientCertExpirationTime uint `envconfig:"CLIENT_CERT_EXPIRATION_DAYS" default:"30"`

	// Kubeconfig specifies path to a kubeconfig file if the server is run outside of a cluster
	Kubeconfig string `envconfig:"KUBECONFIG" default:""`

	// Backend specifies which backend storage should be used. Allowed values: "crd" and "remote".
	Backend string `envconfig:"BACKEND" default:"crd"`

	// RemoteBackendURL contains URL to a remote data store that should be used instead of the default CRD-based one.
	// For HTTPS mTLS connections server cert and CA will be used.
	RemoteBackendURL string `envconfig:"REMOTE_BACKEND_URL" default:""`

	// RemoteBackendTimeout specifies timeout. Has to be parsable to time.Duration
	RemoteBackendTimeout time.Duration `envconfig:"REMOTE_BACKEND_TIMEOUT" default:"5s"`
}
