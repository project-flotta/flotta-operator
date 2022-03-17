package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	regClientSecretNameRandomLen = 10
	regClientSecretNamePrefix    = "reg-client-ca" //#nosec
	regClientSecretLabelKey      = "reg-client-ca" //#nosec

	YggdrasilRegisterAuth = 1
	YggdrasilCompleteAuth = 0

	defaultDaysToExpireClientCertificate = 7
)

// RequestAuthKey is a type to be used on request context and to be validated on verify Request
type RequestAuthKey string

// CAProvider The main reason to have an interface here is to be able to extend this to
// future Cert providers, like:
// - Vault
// - Acme protocol
// Keeping as an interface, so in future users can decice.
type CAProvider interface {
	GetName() string
	GetCACertificate() (*CertificateGroup, error)
	CreateRegistrationCertificate(name string) (map[string][]byte, error)
	SignCSR(CSRPem string, commonName string, expiration time.Time) ([]byte, error)
	GetServerCertificate(dnsNames []string, localhostEnabled bool) (*CertificateGroup, error)
}

type TLSConfig struct {
	config               *tls.Config
	client               client.Client
	caProvider           []CAProvider
	Domains              []string
	LocalhostEnabled     bool
	namespace            string
	clientExpirationDays int
}

func NewMTLSConfig(client client.Client, namespace string, domains []string, localhostEnabled bool) *TLSConfig {
	config := &TLSConfig{
		config:               nil,
		client:               client,
		Domains:              domains,
		namespace:            namespace,
		LocalhostEnabled:     localhostEnabled,
		clientExpirationDays: defaultDaysToExpireClientCertificate,
	}

	// Secret providers here
	secretProvider := NewCASecretProvider(client, namespace)
	config.caProvider = append(config.caProvider, secretProvider)

	return config
}

// SetClientExpiration sets the client expiration time in days
func (conf *TLSConfig) SetClientExpiration(days int) error {
	if days <= 0 {
		return fmt.Errorf("Cannot set 1 day expiration time")
	}
	conf.clientExpirationDays = days
	return nil
}

// SignCSR sign the given CSRPem using the first CA provider in use.
func (conf *TLSConfig) SignCSR(CSRPem string, commonName string) ([]byte, error) {
	if len(conf.caProvider) <= 0 {
		return nil, fmt.Errorf("Cannot get caProvider to sign the CSR")
	}
	return conf.caProvider[0].SignCSR(
		CSRPem,
		commonName,
		time.Now().AddDate(0, 0, conf.clientExpirationDays))
}

// @TODO mainly used for testing, maybe not needed at all
func (conf *TLSConfig) SetCAProvider(caProviders []CAProvider) {
	conf.caProvider = caProviders
}

func (conf *TLSConfig) InitCertificates() (*tls.Config, []*x509.Certificate, error) {
	if len(conf.caProvider) == 0 {
		return nil, nil, fmt.Errorf("no CA provider is set")
	}

	var errors error
	caCerts := []*CertificateGroup{}

	CACertChain := []*x509.Certificate{}
	caCertPool := x509.NewCertPool()

	for _, caProvider := range conf.caProvider {
		caCert, err := caProvider.GetCACertificate()
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf(
				"cannot get CA certificate for provider %s: %v",
				caProvider.GetName(), err))
			continue
		}

		caCerts = append(caCerts, caCert)
		CACertChain = append(CACertChain, caCert.GetCert())
		caCertPool.AppendCertsFromPEM(caCert.CertPEM.Bytes())
	}

	if errors != nil {
		return nil, nil, errors
	}

	if len(caCerts) == 0 {
		return nil, nil, fmt.Errorf("cannot get any CA certificate")
	}

	// We always sign the certificates with the first CA server. I guess that it's normal
	serverCert, err := conf.caProvider[0].GetServerCertificate(conf.Domains, conf.LocalhostEnabled)
	if err != nil {
		return nil, nil, err
	}

	certificate, err := serverCert.GetCertificate()
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot create server certfificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAnyClientCert,
		MinVersion:   tls.VersionTLS13,
	}
	return tlsConfig, CACertChain, nil
}

func (conf *TLSConfig) CreateRegistrationClientCerts() error {

	if len(conf.caProvider) == 0 {
		return fmt.Errorf("Cannot get ca provider")
	}

	name := fmt.Sprintf("%s-%s",
		regClientSecretNamePrefix,
		utilrand.String(regClientSecretNameRandomLen))

	certData, err := conf.caProvider[0].CreateRegistrationCertificate(name)
	if err != nil {
		return fmt.Errorf("Cannot create client certificate")
	}

	secret := corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: conf.namespace,
			Name:      name,
			Labels:    map[string]string{regClientSecretLabelKey: "true"},
		},
		Data: certData,
	}

	return conf.client.Create(context.TODO(), &secret)
}

// isClientCertificateSigned is checking that PeerCertificates are signed by at
// least one give CA certificate. The main reason to do this and not
// x509.Certificate.Cert(certStore) is because is checking expiration time, and
// for registration endpoint, we cannot assume that it'll be ok.
func isClientCertificateSigned(PeerCertificates []*x509.Certificate, CAChain []*x509.Certificate) bool {
	for _, cert := range PeerCertificates {
		for _, caCert := range CAChain {
			err := cert.CheckSignatureFrom(caCert)
			// TODO log debug here with the error. Can be too verbose.
			if err == nil {
				return true
			}
		}
	}
	return false
}

// VerifyRequest check certificate based on the scenario needed:
// registration endpoint: Any cert signed, even if it's expired.
// All endpoints: checking that it's valid certificate.
// @TODO check here the list of rejected certificates.
func VerifyRequest(r *http.Request, verifyType int, verifyOpts x509.VerifyOptions, CACertChain []*x509.Certificate, authzKey RequestAuthKey) bool {

	if len(r.TLS.PeerCertificates) == 0 {
		return false
	}

	*r = *r.WithContext(context.WithValue(
		r.Context(), authzKey, strings.ToLower(r.TLS.PeerCertificates[0].Subject.CommonName)))

	if verifyType == YggdrasilRegisterAuth {
		res := isClientCertificateSigned(r.TLS.PeerCertificates, CACertChain)
		return res
	}

	for _, cert := range r.TLS.PeerCertificates {
		if cert.Subject.CommonName == CertRegisterCN {
			return false
		}

		if _, err := cert.Verify(verifyOpts); err != nil {
			log.Log.V(5).Info("Failed to verify client cert: %v", err)
			return false
		}
	}
	return true
}
