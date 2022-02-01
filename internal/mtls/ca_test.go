package mtls_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/project-flotta/flotta-operator/internal/mtls"
)

const (
	certRegisterCN = "register" // Important, make a copy here to prevent breaking changes
)

var _ = Describe("CA test", func() {

	Context("TLSConfig", func() {

		var (
			k8sClient client.Client
			namespace = "test"
			testEnv   *envtest.Environment
			dnsNames  = []string{"foo.com"}
			ips       = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
		)

		BeforeEach(func() {
			By("bootstrapping test environment")
			testEnv = &envtest.Environment{
				CRDDirectoryPaths: []string{
					filepath.Join("../..", "config", "crd", "bases"),
					filepath.Join("../..", "config", "test", "crd"),
				},
				ErrorIfCRDPathMissing: true,
			}
			var err error
			cfg, err := testEnv.Start()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())

			k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(err).NotTo(HaveOccurred())

			nsSpec := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: namespace}}
			err = k8sClient.Create(context.TODO(), &nsSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := testEnv.Stop()
			Expect(err).NotTo(HaveOccurred())
		})

		It("No namespace exists", func() {
			// given
			config := mtls.NewMTLSConfig(k8sClient, "falsy", []string{"foo.com"}, true)

			// when
			tlsConfig, caChain, err := config.InitCertificates()

			// then
			Expect(tlsConfig).To(BeNil())
			Expect(caChain).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"1 error occurred:\n\t* cannot get CA certificate for provider secret: namespaces \"falsy\" not found\n\n"))
		})

		It("retrieve correctly", func() {
			// given
			config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, true)

			// when
			tlsConfig, caChain, err := config.InitCertificates()

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(tlsConfig.Certificates).To(HaveLen(1))
			Expect(tlsConfig.ClientAuth).To(Equal(tls.RequireAnyClientCert))
			Expect(tlsConfig.MinVersion).To(Equal(uint16(tls.VersionTLS13)))
			Expect(caChain).To(HaveLen(1))

			cert, err := x509.ParseCertificate(tlsConfig.Certificates[0].Certificate[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(cert).NotTo(BeNil())
			Expect(cert.SerialNumber).To(Equal(caChain[0].SerialNumber))
			Expect(cert.Subject.CommonName).To(Equal("*"))
			Expect(cert.DNSNames).To(Equal(dnsNames))
			Expect(cert.IPAddresses).To(HaveLen(2))
			Expect(cert.IPAddresses[0].To16()).To(Equal(ips[0].To16())) // IPV4 reflect issues.
			Expect(cert.IPAddresses[1]).To(Equal(ips[1]))
		})

		It("Server cert without localhost IPS", func() {
			// given
			config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)

			// when
			tlsConfig, caChain, err := config.InitCertificates()

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(tlsConfig.Certificates).To(HaveLen(1))
			Expect(tlsConfig.ClientAuth).To(Equal(tls.RequireAnyClientCert))
			Expect(tlsConfig.MinVersion).To(Equal(uint16(tls.VersionTLS13)))
			Expect(caChain).To(HaveLen(1))

			cert, err := x509.ParseCertificate(tlsConfig.Certificates[0].Certificate[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(cert).NotTo(BeNil())
			Expect(cert.SerialNumber).To(Equal(caChain[0].SerialNumber))
			Expect(cert.Subject.CommonName).To(Equal("*"))
			Expect(cert.DNSNames).To(Equal(dnsNames))
			Expect(cert.IPAddresses).To(HaveLen(0))
		})

		It("No CaProviders defined", func() {
			// given
			config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
			config.SetCAProvider([]mtls.CAProvider{})

			// when
			tlsConfig, caChain, err := config.InitCertificates()

			// then
			Expect(err).To(HaveOccurred())
			Expect(tlsConfig).To(BeNil())
			Expect(caChain).To(BeNil())
		})

		Context("Registration client", func() {

			checkingOneSecret := func() {
				options := client.ListOptions{
					Namespace:     namespace,
					LabelSelector: labels.Set{"reg-client-ca": "true"}.AsSelector(),
				}
				secrets := corev1.SecretList{}
				err := k8sClient.List(context.TODO(), &secrets, &options)
				ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Cannot get list secrets")

				Expect(secrets.Items).To(HaveLen(1))
				Expect(secrets.Items[0].Data).To(HaveKey("client.crt"))
				Expect(secrets.Items[0].Data).To(HaveKey("client.key"))
			}

			It("Create cert", func() {
				// given
				config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
				_, _, err := config.InitCertificates()
				Expect(err).NotTo(HaveOccurred())

				// when
				err = config.CreateRegistrationClientCerts()

				// then
				Expect(err).NotTo(HaveOccurred())
				checkingOneSecret()
			})

			It("Not valid CA set ", func() {
				// given
				config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
				config.SetCAProvider([]mtls.CAProvider{})

				// when
				err := config.CreateRegistrationClientCerts()

				// then
				Expect(err).To(HaveOccurred())
			})

			It("If ca not started return new one", func() {
				// given
				config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)

				// when
				err := config.CreateRegistrationClientCerts()

				// then
				Expect(err).NotTo(HaveOccurred())
				checkingOneSecret()
			})
		})

		Context("SetClientExpiration", func() {

			var config *mtls.TLSConfig

			BeforeEach(func() {
				config = mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
				_, _, err := config.InitCertificates()
				Expect(err).NotTo(HaveOccurred())
			})

			It("Zero is not allowed", func() {
				err := config.SetClientExpiration(0)

				Expect(err).To(HaveOccurred())
			})

			It("Negative is not allowed", func() {
				err := config.SetClientExpiration(-3)

				Expect(err).To(HaveOccurred())
			})

			It("Positive is allowed", func() {
				err := config.SetClientExpiration(10)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Sign CSR", func() {
			var config *mtls.TLSConfig

			BeforeEach(func() {
				config = mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
				_, _, err := config.InitCertificates()
				Expect(err).NotTo(HaveOccurred())
			})

			It("Sign valid pem", func() {
				// this csr is created by openssl command, just to make sure that
				// works.
				// given
				csr := `
-----BEGIN CERTIFICATE REQUEST-----
MIIBhTCB7wIBADAdMQwwCgYDVQQKEwNrNGUxDTALBgNVBAMTBHRlc3QwgZ8wDQYJ
KoZIhvcNAQEBBQADgY0AMIGJAoGBAMLnQ2J7NfJzp+v6VLXjPi7EHKhlYSepgcMb
K1N//FszeHjMhRlhJLYCC3gpKm5xjujA8l191iMJFGGh4PZEKhCi2fV8bQ0QAFjJ
VSIBJRxN2GOUteGTxXndM5x2pVjz7qYgYJ/PopbP0PylYv4EGDx5x1ElHQuQ8tiL
rIgoITfVAgMBAAGgKTAnBgkqhkiG9w0BCQ4xGjAYMBYGAyoDBAQPZXh0cmEgZXh0
ZW5zaW9uMA0GCSqGSIb3DQEBDQUAA4GBAGf6yNp3Cl+74qlNNfhMqiQSrcfMOM4l
rPQVtIYx6ZBA9q85sqNbUZAGnNzQw6pUj7YEVHwtvj8QBsIau+gkr2dl0nqhfTOV
uduLP2w/1jLbouiuyjOUFJuSIUjW2Os/7PD+cWbcxE8IrhW5FnR9c1H8JkIfRB0D
KVwIKwl1tEGP
-----END CERTIFICATE REQUEST-----
`
				// when
				pemCert, err := config.SignCSR(csr, "test")

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(pemCert).NotTo(BeNil())

				block, _ := pem.Decode(pemCert)
				Expect(block).NotTo(BeNil())

				cert, err := x509.ParseCertificate(block.Bytes)
				Expect(err).NotTo(HaveOccurred())
				Expect(cert.Subject.CommonName).To(Equal("test"), "CommonName was not updated")
			})

			It("Invalid CSR failed", func() {
				csr := `
-----BEGIN CERTIFICATE REQUEST-----
MIIBhTCB7wIBADAdMQwwCgYDVQQKEwNrNGUxDTALBgNVBAMTBHRlc3QwgZ8wDQYJ
KoZIhvcNAQEBBQA-----END CERTIFICATE REQUEST-----
`
				// when
				pemCert, err := config.SignCSR(csr, "test")

				//  then
				Expect(err).To(HaveOccurred())
				Expect(pemCert).To(BeNil())
			})

			It("Sending a Cert failed", func() {
				// given
				givenCert := pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: createCACert().certBytes,
				})

				// when
				pemCert, err := config.SignCSR(string(givenCert), "test")

				// then
				Expect(err).To(HaveOccurred())
				Expect(pemCert).To(BeNil())
			})

			It("No valid CA is set", func() {
				// given
				config = mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
				givenCert := pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: createCSR(),
				})

				// when
				pemCert, err := config.SignCSR(string(givenCert), "test")

				// then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Cannot get CA certificate"))
				Expect(pemCert).To(BeNil())
			})

			It("It's getting the correct certification expire", func() {
				// given
				givenCert := pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE REQUEST",
					Bytes: createCSR(),
				})

				err := config.SetClientExpiration(1)
				Expect(err).NotTo(HaveOccurred())

				date := time.Now().AddDate(0, 0, 1)
				// when

				pemCert, err := config.SignCSR(string(givenCert), "test")
				// then

				Expect(err).NotTo(HaveOccurred())
				Expect(pemCert).NotTo(BeNil())

				block, _ := pem.Decode(pemCert)
				Expect(block).NotTo(BeNil())

				cert, err := x509.ParseCertificate(block.Bytes)
				Expect(err).NotTo(HaveOccurred())
				Expect(cert.NotAfter.Year()).To(Equal(date.Year()))
				Expect(cert.NotAfter.Month()).To(Equal(date.Month()))
				Expect(cert.NotAfter.Day()).To(Equal(date.Day()))
			})
		})
	})

	Context("VerifyRequest", func() {
		var (
			ca         []*certificate
			CACertPool *x509.CertPool
			CAChain    []*x509.Certificate
			opts       x509.VerifyOptions
		)

		BeforeEach(func() {
			ca = []*certificate{createCACert(), createCACert()}

			CACertPool = x509.NewCertPool()
			CAChain = []*x509.Certificate{}

			for _, cert := range ca {
				CACertPool.AddCert(cert.signedCert)
				CAChain = append(CAChain, cert.signedCert)
			}

			opts = x509.VerifyOptions{
				Roots:         CACertPool,
				Intermediates: x509.NewCertPool(),
			}
		})

		It("No peer certificates are present", func() {
			// given
			r := &http.Request{
				TLS: &tls.ConnectionState{
					PeerCertificates: []*x509.Certificate{},
				},
			}

			// when
			res := mtls.VerifyRequest(r, 0, opts, CAChain)

			// then
			Expect(res).To(BeFalse())
		})

		Context("Registration Auth", func() {
			const (
				AuthType = 1 // Equals to YggdrasilRegisterAuth, but it's important, so keep a copy here.
			)

			It("Peer certificate is valid", func() {
				// given
				cert := createRegistrationClientCert(ca[0])
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeTrue())
			})

			It("Peer certificate is invalid", func() {
				// given
				cert := createRegistrationClientCert(createCACert())
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeFalse())
			})

			It("Last CA certificate is valid", func() {
				// given
				cert := createRegistrationClientCert(ca[1])
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeTrue())
			})

			It("Expired certificate is valid", func() {
				// given
				c := &x509.Certificate{
					SerialNumber: big.NewInt(time.Now().Unix()),
					Subject: pkix.Name{
						Organization: []string{"Flotta-operator"},
					},
					NotBefore:             time.Now(),
					NotAfter:              time.Now().AddDate(0, 0, 0),
					ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
					KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
					BasicConstraintsValid: true,
				}

				cert := createGivenClientCert(c, ca[1])
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeTrue())
			})

		})

		Context("Normal device Auth", func() {
			const (
				AuthType = 0 // Equals to YggdrasilCompleteAuth, but it's important, so keep a copy here.
			)

			It("Register certificate is invalid", func() {
				// given
				cert := createRegistrationClientCert(ca[0])
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeFalse())
			})

			It("Certificate is correct", func() {
				// given
				cert := createClientCert(ca[0])
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeTrue())
			})

			It("Invalid certificate is rejected", func() {

				// given
				cert := createClientCert(createCACert())
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeFalse())
			})

			It("Certificate valid with any CA position on the store.", func() {
				// given
				cert := createClientCert(ca[1])
				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeTrue())
			})

			It("Expired certificate is not working", func() {
				// given
				c := &x509.Certificate{
					SerialNumber: big.NewInt(time.Now().Unix()),
					Subject: pkix.Name{
						Organization: []string{"Flotta-operator"},
						CommonName:   "test-device",
					},
					NotBefore:             time.Now(),
					NotAfter:              time.Now().AddDate(0, 0, 0),
					ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
					KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
					BasicConstraintsValid: true,
				}
				cert := createGivenClientCert(c, ca[0])

				r := &http.Request{
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert.signedCert},
					},
				}

				// when
				res := mtls.VerifyRequest(r, AuthType, opts, CAChain)

				// then
				Expect(res).To(BeFalse())
			})

		})

	})
})

type certificate struct {
	cert       *x509.Certificate
	key        *rsa.PrivateKey
	certBytes  []byte
	signedCert *x509.Certificate
}

func createRegistrationClientCert(ca *certificate) *certificate {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Flotta-operator"},
			CommonName:   certRegisterCN,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	return createGivenClientCert(cert, ca)
}

func createClientCert(ca *certificate) *certificate {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Flotta-operator"},
			CommonName:   "device-UUID",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	return createGivenClientCert(cert, ca)
}

func createGivenClientCert(cert *x509.Certificate, ca *certificate) *certificate {
	certKey, err := rsa.GenerateKey(rand.Reader, 1024)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on key generation")

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca.cert, &certKey.PublicKey, ca.key)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on sign generation")

	signedCert, err := x509.ParseCertificate(certBytes)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on parsing certificate")

	err = signedCert.CheckSignatureFrom(ca.signedCert)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on check signature")

	return &certificate{cert, certKey, certBytes, signedCert}
}

func createCACert() *certificate {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Flotta-operator"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 1024)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on key generation")

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on sign generation")

	signedCert, err := x509.ParseCertificate(caBytes)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on parsing certificate")

	return &certificate{ca, caPrivKey, caBytes, signedCert}
}

func createCSR() []byte {
	keys, err := rsa.GenerateKey(rand.Reader, 1024)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Cannot create key")
	var csrTemplate = x509.CertificateRequest{
		Version: 0,
		Subject: pkix.Name{
			CommonName:   "test",
			Organization: []string{"k4e"},
		},
		SignatureAlgorithm: x509.SHA512WithRSA,
	}
	// step: generate the csr request
	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, keys)
	Expect(err).NotTo(HaveOccurred())
	return csrCertificate
}
