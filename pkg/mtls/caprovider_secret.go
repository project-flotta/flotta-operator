package mtls

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CASecretName    = "flotta-ca"
	HostTLSCertName = "flotta-host-certificate"
	providerName    = "secret"

	caCertSecretKey = "ca.key"
	caCertCertKey   = "ca.crt"

	serverSecretKey = "server.key"
	serverCert      = "server.crt"

	clientCertSecretKey = "client.key"
	clientCertCertKey   = "client.crt"

	certOrganization       = "flotta-operator"
	CertRegisterCN         = "register"
	certDefaultExpiration  = 1 // years
	serverCertOrganization = "flotta-operator"
)

// @TODO Add a watcher on the secret if it's manually updated to renew the
// latestCA
type CASecretProvider struct {
	client    client.Client
	namespace string
	latestCA  *CertificateGroup
}

func NewCASecretProvider(client client.Client, namespace string) *CASecretProvider {
	return &CASecretProvider{
		client:    client,
		namespace: namespace,
	}
}

func (config *CASecretProvider) GetName() string {
	return providerName
}

func (config *CASecretProvider) GetCACertificate() (*CertificateGroup, error) {
	var secret corev1.Secret

	err := config.client.Get(context.TODO(), client.ObjectKey{
		Namespace: config.namespace,
		Name:      CASecretName,
	}, &secret)

	if !errors.IsNotFound(err) && err != nil {
		return nil, err
	}

	if err == nil {
		// Certificate is already created, parse it as *certificateGroup and return
		// it
		certGroup, err := NewCACertificateGroupFromSecret(secret.Data)
		if err != nil {
			return nil, err
		}
		config.latestCA = certGroup
		return certGroup, err
	}

	certificateGroup, err := getCACertificate()
	if err != nil {
		return nil, fmt.Errorf("cannot create sample certificate")
	}

	secret = corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: config.namespace,
			Name:      CASecretName,
		},
		Data: map[string][]byte{
			caCertCertKey:   certificateGroup.CertPEM.Bytes(),
			caCertSecretKey: certificateGroup.PrivKeyPEM.Bytes(),
		},
	}

	err = config.client.Create(context.TODO(), &secret)
	if err != nil {
		return nil, err
	}

	config.latestCA = certificateGroup
	return certificateGroup, err
}

func (config *CASecretProvider) GetServerCertificate(dnsNames []string, localhostEnabled bool) (*CertificateGroup, error) {

	var secret corev1.Secret
	err := config.client.Get(context.TODO(), client.ObjectKey{
		Namespace: config.namespace,
		Name:      HostTLSCertName,
	}, &secret)

	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("cannot get Host TLS cert:%v", err)
	}

	if err == nil {
		// Certificate is already created, parse it as *certificateGroup and return
		// it
		certGroup := &CertificateGroup{
			CertPEM:    bytes.NewBuffer(secret.Data[serverCert]),
			PrivKeyPEM: bytes.NewBuffer(secret.Data[serverSecretKey]),
		}

		if err := certGroup.ImportFromPem(); err != nil {
			return nil, fmt.Errorf("cannot import server cert from configmap: %v", err)
		}
		return certGroup, err
	}

	CACert, err := config.GetCACertificate()
	if err != nil {
		return nil, fmt.Errorf("cannot get Host CA TLS cert:%v", err)
	}

	cert, err := getServerCertificate(dnsNames, localhostEnabled, CACert)
	if err != nil {
		return nil, fmt.Errorf("cannot create host TLS cert:%v", err)
	}

	secret = corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: config.namespace,
			Name:      HostTLSCertName,
		},
		Data: map[string][]byte{
			serverCert:      cert.CertPEM.Bytes(),
			serverSecretKey: cert.PrivKeyPEM.Bytes(),
		},
	}

	err = config.client.Create(context.TODO(), &secret)
	if err != nil {
		return nil, fmt.Errorf("cannot store server cert: %v", err)
	}
	return cert, nil
}

func (config *CASecretProvider) CreateRegistrationCertificate(name string) (map[string][]byte, error) {
	CACert, err := config.GetCACertificate()
	if err != nil {
		return nil, fmt.Errorf("Cannot retrieve caCert")
	}

	cert := &x509.Certificate{
		SerialNumber: CACert.cert.SerialNumber,
		Subject: pkix.Name{
			CommonName:   CertRegisterCN,
			Organization: []string{certOrganization},
			SerialNumber: name,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(certDefaultExpiration, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certGroup, err := createKeyAndCSR(cert, CACert)
	if err != nil {
		return nil, fmt.Errorf("Cannot sign certificate request: %v", err)
	}

	err = certGroup.CreatePem()
	if err != nil {
		return nil, fmt.Errorf("Cannot encode certs: %v", err)
	}

	res := map[string][]byte{
		clientCertCertKey:   certGroup.CertPEM.Bytes(),
		clientCertSecretKey: certGroup.PrivKeyPEM.Bytes(),
	}
	return res, nil
}

// SignCSR sign a new CertificateRequest and returns the PEM certificate.
// This function is going to be used a lot, so using config.latestCA ensure
// that APIServer is not overloaded with that.
// Because the CM is always managed by this, should be safe to use that one.
func (config *CASecretProvider) SignCSR(CSRPem string, commonName string, expiration time.Time) ([]byte, error) {
	if config.latestCA == nil {
		return nil, fmt.Errorf("Cannot get CA certificate")
	}
	// next blocks to be avoided just because we only sign one CSR. If more than
	// one maybe it's an attack.
	decodecCert, _ := pem.Decode([]byte(CSRPem))
	if decodecCert == nil {
		return nil, fmt.Errorf("cannot decode CSR certificate")
	}

	CSR, err := x509.ParseCertificateRequest(decodecCert.Bytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse CSR: %v", err)
	}

	clientCert := &x509.Certificate{
		Signature:          CSR.Signature,
		SignatureAlgorithm: CSR.SignatureAlgorithm,
		PublicKeyAlgorithm: CSR.PublicKeyAlgorithm,
		PublicKey:          CSR.PublicKey,
		SerialNumber:       big.NewInt(time.Now().Unix()),
		Subject:            CSR.Subject,
		NotBefore:          time.Now().AddDate(0, 0, -1), // 1 day before for time drift issues
		NotAfter:           expiration,
		KeyUsage:           x509.KeyUsageDigitalSignature,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	// We always make sure that commonName is the device one, so noone can try to
	// get access to another device.
	clientCert.Subject.CommonName = commonName
	clientCert.Subject.Organization = []string{certOrganization}

	certBytes, err := x509.CreateCertificate(
		rand.Reader, clientCert, config.latestCA.cert, CSR.PublicKey, config.latestCA.privKey)
	if err != nil {
		return nil, fmt.Errorf("Cannot sign certificate reques: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return certPEM, nil
}
