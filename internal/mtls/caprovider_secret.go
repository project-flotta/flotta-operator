package mtls

import (
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
	CASecretName = "k4e-ca"
	providerName = "secret"

	caCertSecretKey     = "ca.key"
	caCertCertKey       = "ca.crt"
	clientCertSecretKey = "client.key"
	clientCertCertKey   = "client.crt"

	certOrganization       = "k4e-operator"
	certRegisterCN         = "register"
	certDefaultExpiration  = 1 // years
	serverCertOrganization = "k4e-operator"
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
		certGroup, err := NewCertificateGroupFromSecret(secret.Data)
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
			caCertCertKey:   certificateGroup.certPEM.Bytes(),
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

func (config *CASecretProvider) CreateRegistrationCertificate(name string) (map[string][]byte, error) {
	CACert, err := config.GetCACertificate()
	if err != nil {
		return nil, fmt.Errorf("Cannot retrieve caCert")
	}

	cert := &x509.Certificate{
		SerialNumber: CACert.cert.SerialNumber,
		Subject: pkix.Name{
			CommonName:   certRegisterCN,
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
	certGroup.CreatePem()

	res := map[string][]byte{
		clientCertCertKey:   certGroup.certPEM.Bytes(),
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
		return nil, fmt.Errorf("Cannot decode CSR certificate")
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
		NotBefore:          time.Now(),
		NotAfter:           expiration,
		KeyUsage:           x509.KeyUsageDigitalSignature,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
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
