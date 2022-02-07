package mtls

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// CertificateGroup a bunch of methods to help to work with certificates.
type CertificateGroup struct {
	cert       *x509.Certificate
	signedCert *x509.Certificate
	privKey    *rsa.PrivateKey
	certBytes  []byte
	certPEM    *bytes.Buffer
	PrivKeyPEM *bytes.Buffer
}

func NewCACertificateGroupFromSecret(secretData map[string][]byte) (*CertificateGroup, error) {
	certGroup := &CertificateGroup{
		certPEM:    bytes.NewBuffer(secretData[caCertCertKey]),
		PrivKeyPEM: bytes.NewBuffer(secretData[caCertSecretKey]),
	}
	err := certGroup.ImportFromPem()
	if err != nil {
		return nil, err
	}
	return certGroup, nil
}

func (c *CertificateGroup) ImportFromPem() error {
	block, _ := pem.Decode(c.certPEM.Bytes())
	if block == nil {
		return fmt.Errorf("Cannot get CA certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("Failing parsing cert: %v", err)
	}
	block, _ = pem.Decode(c.PrivKeyPEM.Bytes())
	if block == nil {
		return fmt.Errorf("Cannot get CA certificate key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failing parsing key: %v", err)
	}

	c.cert = cert // Not real at all, because this is already signed.
	c.signedCert = cert
	c.privKey = key
	return nil
}

// CreatePem from the load certificates create the PEM file and stores in local
func (c *CertificateGroup) CreatePem() error {

	caPEM := new(bytes.Buffer)
	err := pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.certBytes,
	})
	if err != nil {
		return err
	}

	privKeyPEM := new(bytes.Buffer)
	err = pem.Encode(privKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(c.privKey),
	})

	if err != nil {
		return err
	}
	c.certPEM = caPEM
	c.PrivKeyPEM = privKeyPEM
	return nil
}

func (c *CertificateGroup) parseSignedCertificate() error {
	var err error
	c.signedCert, err = x509.ParseCertificate(c.certBytes)
	return err
}

// GetCertificate returns the certificate Group in tls.Certificate format.
func (c *CertificateGroup) GetCertificate() (tls.Certificate, error) {
	return tls.X509KeyPair(c.certPEM.Bytes(), c.PrivKeyPEM.Bytes())
}

func (c *CertificateGroup) GetCert() *x509.Certificate {
	return c.cert
}

func getCACertificate() (*CertificateGroup, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{serverCertOrganization},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("Cannot generate CA Key")
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	certificateBundle := CertificateGroup{
		cert:      ca,
		privKey:   caPrivKey,
		certBytes: caBytes,
	}
	err = certificateBundle.CreatePem()
	if err != nil {
		return nil, fmt.Errorf("Cannot encode certificate: %v", err)
	}
	err = certificateBundle.parseSignedCertificate()
	if err != nil {
		return nil, fmt.Errorf("Cannot parse PEM certificate: %v", err)
	}
	return &certificateBundle, nil
}

func createKeyAndCSR(cert *x509.Certificate, caCert *CertificateGroup) (*CertificateGroup, error) {

	certKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("Cannot generate cert Key")
	}

	// sign the cert by the CA
	certBytes, err := x509.CreateCertificate(
		rand.Reader, cert, caCert.cert, &certKey.PublicKey, caCert.privKey)
	if err != nil {
		return nil, err
	}

	certificateBundle := CertificateGroup{
		cert:      cert,
		privKey:   certKey,
		certBytes: certBytes,
	}
	err = certificateBundle.CreatePem()
	if err != nil {
		return nil, fmt.Errorf("Cannot encode certificate: %v", err)
	}

	err = certificateBundle.parseSignedCertificate()
	if err != nil {
		return nil, fmt.Errorf("Cannot parse PEM certificate: %v", err)
	}
	return &certificateBundle, nil
}

func getServerCertificate(dnsNames []string, localhostEnabled bool, CACert *CertificateGroup) (*CertificateGroup, error) {

	ips := []net.IP{}
	if localhostEnabled {
		ips = append(ips, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))
	}

	cert := &x509.Certificate{
		SerialNumber: CACert.cert.SerialNumber,
		Subject: pkix.Name{
			CommonName:   "*", // CommonName match all, and using ASN names
			Organization: []string{serverCertOrganization},
		},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(certDefaultExpiration, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	return createKeyAndCSR(cert, CACert)
}
