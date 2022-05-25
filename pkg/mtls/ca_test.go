package mtls_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/yggdrasil"
	"github.com/project-flotta/flotta-operator/pkg/mtls"
)

const (
	certRegisterCN = "register" // Important, make a copy here to prevent breaking changes

	serverSecretKey = "server.key"
	serverCert      = "server.crt"

	HostTLSCertName = "flotta-host-certificate"
)

var _ = Describe("CA test", func() {
	var (
		log, _ = zap.NewDevelopment()
		logger = log.Sugar()
	)
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

		It("Server listen certificate is already created", func() {
			// Given
			// commands for the following test input:
			//   openssl genrsa -out server.key 2048
			//   openssl req -subj "/CN=mytestCERT" -batch -new -x509 -nodes -key server.key -sha256 -days 1024 -out server.pem
			certPEM := `
-----BEGIN CERTIFICATE-----
MIIDCzCCAfOgAwIBAgIUXnsvn8PsplroKq4DcfB1brdfyj0wDQYJKoZIhvcNAQEL
BQAwFTETMBEGA1UEAwwKbXl0ZXN0Q0VSVDAeFw0yMjAyMDcxNjI5MzVaFw0yNDEx
MjcxNjI5MzVaMBUxEzARBgNVBAMMCm15dGVzdENFUlQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDPSbYIhTCOTrGmFusNPu9i2Dk0YfKUZTb31e4llKky
4gUIktUX+7wnWGG/7inM7LIOnCKAjTJyKm9OUvH7Kztfv1rXgfShkZ1NXaY6Uz3r
nX4I8sxGD5OHqxYj3wFMpteqftpB1Xj6exQs50qT8FuB1f/xeM4oREle42ElFeD1
3BkRYGJs6jgVBCcc+XcBMaK9Vs+Jvhvr08oNpLiRdftT2vY8FnmDkyDAEb9YQT28
+8i0y5wery3VJKbX/30rqtF8C/ePF8A69IzPOiZG1wvC+rEMIpIWEVatIz5sxt7t
nukwz2Yy8HslZNQq5Ect1GGHNpXR6E/W1oCEQ8trO5JpAgMBAAGjUzBRMB0GA1Ud
DgQWBBRp0FwTAuzJxWhsRbjVNnTvcoLH+DAfBgNVHSMEGDAWgBRp0FwTAuzJxWhs
RbjVNnTvcoLH+DAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCj
SGqGJO2Q8wAX7Ii3yYqmgmav2WQVPRstDg1VEdtJ9O9+VAQ+UvyDyswiVmIBNTwx
CvLM4+kb1XL5nv2VmtRaRtHYovvTrOENabYkyNBpsIgo6Qs/Gs2LOflyWNoZwaMC
4jqwrvbDjiLcQNgt5/CxXW6qp2KZQT7TmxrM//PPpAKI6liDiwDAU7wnJNoIcfUA
5bf/0xs2U+IHbogva4Mi+VX5h3bipZrdNj24bQmKMsM+jhOu6vyxt8SeGBV+J0eZ
ERVAxkBYLwwM5B1nwWmitWtps27Cd4sN0JWftcImUM8+04OQFslg4R0Rrxa2IpQm
HiUnRV8I4zbqhxtNPAs+
-----END CERTIFICATE-----`
			certKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAz0m2CIUwjk6xphbrDT7vYtg5NGHylGU299XuJZSpMuIFCJLV
F/u8J1hhv+4pzOyyDpwigI0ycipvTlLx+ys7X79a14H0oZGdTV2mOlM9651+CPLM
Rg+Th6sWI98BTKbXqn7aQdV4+nsULOdKk/BbgdX/8XjOKERJXuNhJRXg9dwZEWBi
bOo4FQQnHPl3ATGivVbPib4b69PKDaS4kXX7U9r2PBZ5g5MgwBG/WEE9vPvItMuc
Hq8t1SSm1/99K6rRfAv3jxfAOvSMzzomRtcLwvqxDCKSFhFWrSM+bMbe7Z7pMM9m
MvB7JWTUKuRHLdRhhzaV0ehP1taAhEPLazuSaQIDAQABAoIBAQDGP5IobfG1eM/w
sGSXo4RhvbhgP/k4MeEzgNgl+xsjfgUgYQYKzQjzfFToskgqJIpa7LsWxXPkum7/
staZyIwdk663BCRKTjDqqFFt4OUMrfC3cDcsHoOTsm4XWpYskDkdZ/soEZmFviba
l069VJjAAUKq2EYbPswJQ2BKjrU7jU+A4F82UVoa0FBZJBXMbthblxoeM2eQb/tV
e9AxBsGjAnVhslhhPiBacnKTJjfcWTHGQxxuWg1JSmLBj43pebsJq6BT+ppaCXCt
rzqkLA68qxe2J7iTFAv0xpWyGoxs2/9CpRF92eMs5mz4ml+LM8sLsIy2bF2gl8yD
91bSkJwBAoGBAOncfqQTn7fnB8aR60ZllNb3Tm5VFdJhFQgqvFSUOSsOtRsnLUa9
Bq5XlYn8834v6rFq5rnYV/j2f4X+zAEUkig6jB10r3L1XM/3AyxZbl0oHHbiqsCR
gbyvhsFWKPk7/bM6z6wLMtUGlS5lgWr94zGSBWZoF+VOfXWpAfbepMKBAoGBAOLp
OYXkiwKyad9d+ad8z2SnyFKcemvobseMpILD3IotH6VjiO1LjDtqLUmQN3jwdk22
1cw3t4UavBDI/DOmoedkjm3I9FfStMtooL32zuE9Olv9wplZ70OQP28c9nm2Huim
j96sXmtQwGOdBxwXjasPjtYs7uWncadquyrnawvpAoGBANM83JNeOm3F3EsrsOXk
iZ3mwsx8RHrEQFghKf4H6N+QqFv/djEoOumtqSB8AIDhzU82bXQ/C6+RED07mo/7
Qc3enINay8O+B3i9+PrNSRgSTCvCsFPC2vpRXhoytk3yN0X2gHE5qE+tY4EGJPE8
pUQ4TnJi4fq5fC+UWnbgQtiBAoGAfcQOoet+MMx6addIXFCNEpj8Ku2X3N9DJ08I
j4HHZr6D38M/TWamHvhGiZNpa5q7t28zKLFpAllDC3qabnZZHktZtfe/lj2u/17K
WP/GwoiRJBOOHDkAqE33GrrO0b7jesd2zlBzNL/ZIl0SZ7uWRc2luYfGEXuxPr2l
Z65EYqECgYAPiK3NfvtfsCqeYE+jfxwWjad75TIY/RKd5VCYh2Zu+zw9nej9HesK
AJVwxMCA6+jCpY3tswEAgQU/sooTq4KhSIiCbnIPIH+nq0P7VLNKLX+NfmLMHg5z
pNUUZ3cX4jGObCNfSBauHb73kfjGBo+RM2rnjAWqs9FhpnrJPDNT8Q==
-----END RSA PRIVATE KEY-----
`
			secret := corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      HostTLSCertName,
				},
				Data: map[string][]byte{
					serverCert:      []byte(certPEM),
					serverSecretKey: []byte(certKey),
				},
			}

			err := k8sClient.Create(context.TODO(), &secret)
			Expect(err).NotTo(HaveOccurred())

			// when
			config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
			tlsConfig, _, err := config.InitCertificates()

			// then

			Expect(err).NotTo(HaveOccurred())
			Expect(tlsConfig.Certificates).To(HaveLen(1))
			tlsConfigCert, err := x509.ParseCertificate(tlsConfig.Certificates[0].Certificate[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(tlsConfigCert.Issuer.CommonName).To(Equal("mytestCERT"))
		})

		It("Server cert is invalid, init send error", func() {

			// given
			secret := corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      HostTLSCertName,
				},
				Data: map[string][]byte{
					serverCert:      []byte("XXX"),
					serverSecretKey: []byte("Invalid"),
				},
			}

			err := k8sClient.Create(context.TODO(), &secret)
			Expect(err).NotTo(HaveOccurred())
			// when

			config := mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
			tlsConfig, _, err := config.InitCertificates()
			// then

			Expect(tlsConfig).To(BeNil())
			Expect(err).To(HaveOccurred())
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

			It("Sign valid pem with RSA keys", func() {
				// given

				// Due to the CSR is created using RSA SignatureAlgorithm, we need to
				// pre-upload a RSA keys for the CA.
				config = mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)

				caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
				Expect(err).To(BeNil(), "Fail on key generation")
				res := createCACertUsingKey(caPrivKey)

				certPEM := new(bytes.Buffer)
				keyPem := new(bytes.Buffer)
				err = pem.Encode(certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: res.certBytes})
				Expect(err).NotTo(HaveOccurred())

				err = pem.Encode(keyPem, &pem.Block{
					Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(res.key.(*rsa.PrivateKey)),
				})
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "flotta-ca",
					},
					Data: map[string][]byte{
						"ca.crt": certPEM.Bytes(),
						"ca.key": keyPem.Bytes(),
					},
				}

				err = k8sClient.Create(context.TODO(), &secret)
				Expect(err).NotTo(HaveOccurred())

				_, _, err = config.InitCertificates()
				Expect(err).NotTo(HaveOccurred())

				// this csr is created by openssl command, just to make sure that
				// works.
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
-----END CERTIFICATE REQUEST-----`

				// when
				pemCert, err := config.SignCSR(csr, "test", "test")

				// then
				Expect(err).NotTo(HaveOccurred())
				Expect(pemCert).NotTo(BeNil())

				block, _ := pem.Decode(pemCert)
				Expect(block).NotTo(BeNil())

				cert, err := x509.ParseCertificate(block.Bytes)
				Expect(err).NotTo(HaveOccurred())
				Expect(cert.Subject.CommonName).To(Equal("test"), "CommonName was not updated")
				Expect(cert.Subject.OrganizationalUnit).To(HaveLen(1))
				Expect(cert.Subject.OrganizationalUnit).To(ContainElement("test"))
			})

			Context("With initial config", func() {
				BeforeEach(func() {
					config = mtls.NewMTLSConfig(k8sClient, namespace, dnsNames, false)
					_, _, err := config.InitCertificates()
					Expect(err).NotTo(HaveOccurred())
				})

				It("Invalid CSR failed", func() {
					// given
					csr := `
-----BEGIN CERTIFICATE REQUEST-----
MIIBhTCB7wIBADAdMQwwCgYDVQQKEwNrNGUxDTALBgNVBAMTBHRlc3QwgZ8wDQYJ
KoZIhvcNAQEBBQA-----END CERTIFICATE REQUEST-----
`
					// when
					pemCert, err := config.SignCSR(csr, "test", "test")

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
					pemCert, err := config.SignCSR(string(givenCert), "test", "test")

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
					pemCert, err := config.SignCSR(string(givenCert), "test", "test")

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

					pemCert, err := config.SignCSR(string(givenCert), "test", "test")
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
			res, err := mtls.VerifyRequest(r, 0, opts, CAChain, yggdrasil.AuthzKey, logger)

			// then
			Expect(res).To(BeFalse())
			Expect(err).To(BeAssignableToTypeOf(&mtls.NoClientCertSendError{}))
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeFalse())
				Expect(err).To(BeAssignableToTypeOf(&mtls.RegisterClientVerifyError{}))
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeFalse())
				Expect(err).To(BeAssignableToTypeOf(&mtls.InvalidCertificateKindError{}))
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeFalse())
				Expect(err).To(BeAssignableToTypeOf(&mtls.ClientCertificateVerifyError{}))

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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
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
				res, err := mtls.VerifyRequest(r, AuthType, opts, CAChain, yggdrasil.AuthzKey, logger)

				// then
				Expect(res).To(BeFalse())
				Expect(err).To(BeAssignableToTypeOf(&mtls.ClientCertificateVerifyError{}))
			})

		})

	})
})

type certificate struct {
	cert       *x509.Certificate
	key        crypto.Signer
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
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on key generation")
	return createCACertUsingKey(caPrivKey)
}

func createCACertUsingKey(key crypto.Signer) *certificate {

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

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, key.Public(), key)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on sign generation")

	signedCert, err := x509.ParseCertificate(caBytes)
	ExpectWithOffset(1, err).To(BeNil(), "Fail on parsing certificate")

	return &certificate{ca, key, caBytes, signedCert}
}

func createCSR() []byte {
	keys, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Cannot create key")
	var csrTemplate = x509.CertificateRequest{
		Version: 0,
		Subject: pkix.Name{
			CommonName:   "test",
			Organization: []string{"flotta"},
		},
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}
	// step: generate the csr request
	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, keys)
	Expect(err).NotTo(HaveOccurred())
	return csrCertificate
}
