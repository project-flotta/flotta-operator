package mtls

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
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

type CASecretProvider struct {
	client    client.Client
	namespace string
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

	if err == nil {
		// Certificate is already created, parse it as *certificateGroup and return
		// it
		certGroup, err := NewCertificateGroupFromCACM(secret.Data)
		return certGroup, err
	}

	if !errors.IsNotFound(err) {
		return nil, err
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
			// Country:      []string{"US"},
			SerialNumber: name,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(certDefaultExpiration, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certGroup, err := getKeyAndCSR(cert, CACert)
	if err != nil {
		return nil, fmt.Errorf("Cannot sign certificate request")
	}
	certGroup.CreatePem()

	res := map[string][]byte{
		clientCertCertKey:   certGroup.certPEM.Bytes(),
		clientCertSecretKey: certGroup.PrivKeyPEM.Bytes(),
	}
	return res, nil
}
