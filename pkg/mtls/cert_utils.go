package mtls

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
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

const (
	ECPrivateKeyBlockType  = "EC PRIVATE KEY"
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
)

// CertificateGroup a bunch of methods to help to work with certificates.
type CertificateGroup struct {
	cert       *x509.Certificate
	signedCert *x509.Certificate
	privKey    crypto.PrivateKey
	certBytes  []byte
	CertPEM    *bytes.Buffer
	PrivKeyPEM *bytes.Buffer
}

func NewCACertificateGroupFromSecret(secretData map[string][]byte) (*CertificateGroup, error) {
	certGroup := &CertificateGroup{
		CertPEM:    bytes.NewBuffer(secretData[caCertCertKey]),
		PrivKeyPEM: bytes.NewBuffer(secretData[caCertSecretKey]),
	}
	err := certGroup.ImportFromPem()
	if err != nil {
		return nil, err
	}
	return certGroup, nil
}

func (c *CertificateGroup) ImportFromPem() error {
	block, _ := pem.Decode(c.CertPEM.Bytes())
	if block == nil {
		return fmt.Errorf("Cannot get CA certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("Failing parsing cert: %w", err)
	}
	err = c.decodePrivKeyFromPEM()
	if err != nil {
		return err
	}

	c.cert = cert // Not real at all, because this is already signed.
	c.signedCert = cert
	return nil
}

func (c *CertificateGroup) decodePrivKeyFromPEM() error {
	block, _ := pem.Decode(c.PrivKeyPEM.Bytes())
	if block == nil {
		return fmt.Errorf("Cannot get Certificate key")
	}
	switch block.Type {
	case ECPrivateKeyBlockType:
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return err
		}
		c.privKey = key
	case RSAPrivateKeyBlockType:
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failing parsing key: %w", err)
		}
		c.privKey = key
	default:
		return fmt.Errorf("Cannot decode PEM cert key")
	}
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
	privKeyPEM, err := c.MarshalKeyToPem(c.privKey)
	if err != nil {
		return fmt.Errorf("Cannot marshal to PEM: %w", err)
	}
	c.CertPEM = caPEM
	c.PrivKeyPEM = privKeyPEM
	return nil
}

func (c *CertificateGroup) MarshalKeyToPem(privKey crypto.PrivateKey) (*bytes.Buffer, error) {
	privKeyPEM := new(bytes.Buffer)
	switch t := privKey.(type) {
	case *ecdsa.PrivateKey:
		res, err := x509.MarshalECPrivateKey(t)
		if err != nil {
			return nil, err
		}
		err = pem.Encode(privKeyPEM, &pem.Block{
			Type:  ECPrivateKeyBlockType,
			Bytes: res,
		})
		if err != nil {
			return nil, err
		}

	case *rsa.PrivateKey:
		err := pem.Encode(privKeyPEM, &pem.Block{
			Type:  RSAPrivateKeyBlockType,
			Bytes: x509.MarshalPKCS1PrivateKey(t),
		})

		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Provider key not supported")
	}
	return privKeyPEM, nil
}

func (c *CertificateGroup) GetNewKey() (crypto.Signer, error) {
	switch c.privKey.(type) {
	case *ecdsa.PrivateKey:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case *rsa.PrivateKey:
		return rsa.GenerateKey(rand.Reader, 4096)
	default:
		return nil, fmt.Errorf("unknown algorithm to create the key")
	}
}

func (c *CertificateGroup) parseSignedCertificate() error {
	var err error
	c.signedCert, err = x509.ParseCertificate(c.certBytes)
	return err
}

// GetCertificate returns the certificate Group in tls.Certificate format.
func (c *CertificateGroup) GetCertificate() (tls.Certificate, error) {
	return tls.X509KeyPair(c.CertPEM.Bytes(), c.PrivKeyPEM.Bytes())
}

func (c *CertificateGroup) GetCert() *x509.Certificate {
	return c.cert
}

func (c *CertificateGroup) GetKey() crypto.PrivateKey {
	return c.privKey
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
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("Cannot generate CA Key")
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, caPrivKey.Public(), caPrivKey)
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
		return nil, fmt.Errorf("Cannot encode certificate: %w", err)
	}
	err = certificateBundle.parseSignedCertificate()
	if err != nil {
		return nil, fmt.Errorf("Cannot parse PEM certificate: %w", err)
	}

	// Just pushed the signed cert, so PubkeyAlgo is present on the cert
	certificateBundle.cert = certificateBundle.signedCert
	return &certificateBundle, nil
}

func createKeyAndCSR(cert *x509.Certificate, caCert *CertificateGroup) (*CertificateGroup, error) {
	certKey, err := caCert.GetNewKey()
	if err != nil {
		return nil, fmt.Errorf("Cannot generate cert Key: %w", err)
	}

	// sign the cert by the CA
	certBytes, err := x509.CreateCertificate(
		rand.Reader, cert, caCert.cert, certKey.Public(), caCert.privKey)
	if err != nil {
		return nil, fmt.Errorf("cannot sign certificate: %w", err)
	}

	certificateBundle := CertificateGroup{
		cert:      cert,
		privKey:   certKey,
		certBytes: certBytes,
	}
	err = certificateBundle.CreatePem()
	if err != nil {
		return nil, fmt.Errorf("Cannot encode certificate: %w", err)
	}

	err = certificateBundle.parseSignedCertificate()
	if err != nil {
		return nil, fmt.Errorf("Cannot parse PEM certificate: %w", err)
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
