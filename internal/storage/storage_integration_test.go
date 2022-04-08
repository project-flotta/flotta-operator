package storage_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/storage"
	"github.com/project-flotta/flotta-operator/models"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var _ = Describe("Storage", func() {

	var (
		k8sClient client.Client
		testEnv   *envtest.Environment
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

		err = managementv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = obv1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())

		// Add custom schemes
		err = routev1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	createCM := func() {
		cm := corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Data: map[string]string{
				"BUCKET_NAME": "test_bucket",
			},
		}
		err := k8sClient.Create(context.TODO(), &cm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	createRoute := func() {
		nsSpec := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "openshift-storage"}}
		erro := k8sClient.Create(context.TODO(), &nsSpec)
		ExpectWithOffset(1, erro).ToNot(HaveOccurred())

		s3route := routev1.Route{
			ObjectMeta: v1.ObjectMeta{Name: "s3", Namespace: "openshift-storage"},
			Spec:       routev1.RouteSpec{Host: "test.com"},
		}
		err := k8sClient.Create(context.TODO(), &s3route)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	createSecret := func(meta v1.ObjectMeta, data map[string][]byte) {
		secret := corev1.Secret{
			ObjectMeta: meta,
			Data:       data,
		}
		err := k8sClient.Create(context.TODO(), &secret)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	getDevice := func() *v1alpha1.EdgeDevice {
		dataOBC := "test"
		return &v1alpha1.EdgeDevice{
			TypeMeta:   v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: v1alpha1.EdgeDeviceSpec{
				RequestTime: &v1.Time{},
				Heartbeat:   &v1alpha1.HeartbeatConfiguration{},
			},
			Status: managementv1alpha1.EdgeDeviceStatus{
				DataOBC: &dataOBC,
			},
		}
	}

	Context("GetStorageConfiguration", func() {
		It("Cannot get device", func() {
			// given
			claimer := storage.NewClaimer(k8sClient)
			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), nil)
			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Device without Status", func() {
			// given
			claimer := storage.NewClaimer(k8sClient)
			device := &v1alpha1.EdgeDevice{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: v1alpha1.EdgeDeviceSpec{
					RequestTime: &v1.Time{},
					Heartbeat:   &v1alpha1.HeartbeatConfiguration{},
				},
			}

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)
			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Claim is correct but without configMap", func() {

			// given
			claimer := storage.NewClaimer(k8sClient)
			device := getDevice()

			_, err := claimer.CreateClaim(context.TODO(), device)
			Expect(err).NotTo(HaveOccurred())

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)
			apiError, ok := err.(errors.APIStatus)

			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(apiError.Status().Code).To(Equal(int32(http.StatusNotFound)))
			Expect(apiError.Status().Details.Kind).To(Equal("configmaps"))
		})

		It("Claim is correct but cannot get Openshift route", func() {

			// given
			claimer := storage.NewClaimer(k8sClient)
			device := getDevice()
			createCM()

			_, err := claimer.CreateClaim(context.TODO(), device)
			Expect(err).NotTo(HaveOccurred())

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)
			apiError, _ := err.(errors.APIStatus)

			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())

			Expect(apiError.Status().Code).To(Equal(int32(http.StatusNotFound)))
			Expect(apiError.Status().Details.Kind).To(Equal("routes"))
		})

		It("Cannot get secret", func() {

			// given
			claimer := storage.NewClaimer(k8sClient)
			device := getDevice()

			createCM()
			createRoute()

			_, err := claimer.CreateClaim(context.TODO(), device)
			Expect(err).NotTo(HaveOccurred())

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)
			apiError, _ := err.(errors.APIStatus)

			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apiError.Status().Code).To(Equal(int32(http.StatusNotFound)))
			Expect(apiError.Status().Details.Kind).To(Equal("secrets"))
		})

		It("Got only half of secrets", func() {

			// given
			claimer := storage.NewClaimer(k8sClient)
			device := getDevice()

			createCM()
			createRoute()
			createSecret(v1.ObjectMeta{Name: "test", Namespace: "default"},
				map[string][]byte{
					"AWS_ACCESS_KEY_ID":     []byte("foo"),
					"AWS_SECRET_ACCESS_KEY": []byte("foo"),
				})

			_, err := claimer.CreateClaim(context.TODO(), device)
			Expect(err).NotTo(HaveOccurred())

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)
			apiError, _ := err.(errors.APIStatus)

			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apiError.Status().Code).To(Equal(int32(http.StatusNotFound)))
			Expect(apiError.Status().Details.Kind).To(Equal("secrets"))
			Expect(apiError.Status().Details.Name).To(Equal("router-ca"))
		})

		It("Cannot retrieve AWS secrets", func() {

			// given
			claimer := storage.NewClaimer(k8sClient)
			device := getDevice()

			createCM()
			createRoute()
			createSecret(v1.ObjectMeta{Name: "test", Namespace: "default"},
				map[string][]byte{
					"AWS_SECRET_ACCESS_KEY": []byte("foo"),
				})

			_, err := claimer.CreateClaim(context.TODO(), device)
			Expect(err).NotTo(HaveOccurred())

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)

			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Cannot get AWS_ACCESS_KEY_ID"))
		})

		It("Cannot retrieve AWS access key secret", func() {

			// given
			claimer := storage.NewClaimer(k8sClient)
			device := getDevice()

			createCM()
			createRoute()
			createSecret(v1.ObjectMeta{Name: "test", Namespace: "default"},
				map[string][]byte{
					"AWS_ACCESS_KEY_ID": []byte("foo"),
				})

			_, err := claimer.CreateClaim(context.TODO(), device)
			Expect(err).NotTo(HaveOccurred())

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)

			// then
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Cannot get AWS_SECRET_ACCESS_KEY_ID"))
		})

		It("work as expected", func() {

			// given
			claimer := storage.NewClaimer(k8sClient)
			device := getDevice()

			createCM()
			createRoute()
			createSecret(v1.ObjectMeta{Name: "test", Namespace: "default"},
				map[string][]byte{
					"AWS_ACCESS_KEY_ID":     []byte("foo"),
					"AWS_SECRET_ACCESS_KEY": []byte("foo"),
				})

			nsSpec := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "openshift-ingress-operator"}}
			erro := k8sClient.Create(context.TODO(), &nsSpec)
			ExpectWithOffset(1, erro).ToNot(HaveOccurred())

			createSecret(v1.ObjectMeta{Name: "router-ca", Namespace: "openshift-ingress-operator"},
				map[string][]byte{
					"tls.crt": []byte("foo"),
				})

			_, err := claimer.CreateClaim(context.TODO(), device)
			Expect(err).NotTo(HaveOccurred())

			// when
			result, err := claimer.GetStorageConfiguration(context.TODO(), device)

			// then
			cfg := models.S3StorageConfiguration{
				AwsAccessKeyID:     "Zm9v",
				AwsCaBundle:        "Zm9v",
				AwsSecretAccessKey: "Zm9v",
				BucketHost:         "test.com",
				BucketName:         "test_bucket",
				BucketPort:         443,
				BucketRegion:       "ignore",
			}
			Expect(result).To(Equal(&cfg))

			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("GetExternalStorageConfig", func() {

		const namespace = "default"
		var (
			claimer    *storage.Claimer
			storageObj *v1alpha1.Storage
		)

		getCmData := func() map[string]string {
			return map[string]string{
				"BUCKET_HOST":   "host",
				"BUCKET_PORT":   "443",
				"BUCKET_NAME":   "bucket",
				"BUCKET_REGION": "region",
			}
		}

		getSecretData := func() map[string][]byte {
			return map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("AWS_ACCESS_KEY_ID"),
				"AWS_SECRET_ACCESS_KEY": []byte("AWS_SECRET_ACCESS_KEY"),
				"tls.crt":               []byte("cert"),
			}
		}

		createS3CM := func(data map[string]string) {
			cm := corev1.ConfigMap{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test",
					Namespace: namespace,
				},
				Data: data,
			}
			err := k8sClient.Create(context.TODO(), &cm)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		}

		createS3Secret := func(data map[string][]byte) {
			secret := corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test",
					Namespace: namespace,
				},
				Data: data,
			}
			err := k8sClient.Create(context.TODO(), &secret)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		}

		BeforeEach(func() {
			claimer = storage.NewClaimer(k8sClient)
			storageObj = &v1alpha1.Storage{
				S3: &v1alpha1.S3Storage{
					SecretName:    "test",
					ConfigMapName: "test",
				},
			}
		})

		It("missing S3 configuration", func() {
			// given
			storageObj = &v1alpha1.Storage{}

			// when
			result, err := claimer.GetExternalStorageConfig(context.TODO(), namespace, storageObj)

			// then
			Expect(err).ToNot(BeNil())
			Expect(result).To(BeNil())
		})

		It("missing config map", func() {
			// given
			storageObj.S3.ConfigMapName = "missing"
			createS3Secret(nil)

			// when
			result, err := claimer.GetExternalStorageConfig(context.TODO(), namespace, storageObj)

			// then
			Expect(err).ToNot(BeNil())
			Expect(result).To(BeNil())
		})

		It("missing secret", func() {
			// given
			storageObj.S3.SecretName = "missing"
			createS3CM(nil)

			// when
			result, err := claimer.GetExternalStorageConfig(context.TODO(), namespace, storageObj)

			// then
			Expect(err).ToNot(BeNil())
			Expect(result).To(BeNil())
		})

		DescribeTable("missing ConfigMap field",
			func(fieldName string) {
				//given
				cmData := getCmData()
				delete(cmData, fieldName)
				createS3CM(cmData)
				createS3Secret(getSecretData())

				// when
				result, err := claimer.GetExternalStorageConfig(context.TODO(), namespace, storageObj)

				// then
				Expect(err).ToNot(BeNil())
				Expect(result).To(BeNil())
			},
			Entry("BUCKET_HOST", "BUCKET_HOST"),
			Entry("BUCKET_PORT", "BUCKET_PORT"),
			Entry("BUCKET_NAME", "BUCKET_NAME"),
			Entry("BUCKET_REGION", "BUCKET_REGION"),
		)

		DescribeTable("missing Secret field",
			func(fieldName string, optional bool) {
				//given
				secretData := getSecretData()
				delete(secretData, fieldName)
				createS3CM(getCmData())
				createS3Secret(secretData)

				// when
				result, err := claimer.GetExternalStorageConfig(context.TODO(), namespace, storageObj)

				// then
				if optional {
					Expect(err).To(BeNil())
					Expect(result).ToNot(BeNil())
				} else {
					Expect(err).ToNot(BeNil())
					Expect(result).To(BeNil())
				}
			},
			Entry("AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID", false),
			Entry("AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY", false),
			Entry("tls.crt", "tls.crt", true),
		)

		It("success", func() {
			// given
			cmData := getCmData()
			createS3CM(cmData)
			secretData := getSecretData()
			createS3Secret(secretData)
			expectedPort, _ := strconv.Atoi(cmData["BUCKET_PORT"])
			expected := models.S3StorageConfiguration{
				AwsAccessKeyID:     base64.StdEncoding.EncodeToString(secretData["AWS_ACCESS_KEY_ID"]),
				AwsSecretAccessKey: base64.StdEncoding.EncodeToString(secretData["AWS_SECRET_ACCESS_KEY"]),
				AwsCaBundle:        base64.StdEncoding.EncodeToString(secretData["tls.crt"]),
				BucketHost:         cmData["BUCKET_HOST"],
				BucketPort:         int32(expectedPort),
				BucketName:         cmData["BUCKET_NAME"],
				BucketRegion:       cmData["BUCKET_REGION"],
			}

			// when
			result, err := claimer.GetExternalStorageConfig(context.TODO(), namespace, storageObj)

			// then
			Expect(err).To(BeNil())
			Expect(result).ToNot(BeNil())
			Expect(*result).To(Equal(expected))
		})
	})

	Context("ShouldUseExternalConfig", func() {

		It("storage configuration is nil", func() {
			// given
			// when
			result := storage.ShouldUseExternalConfig(nil)
			// then
			Expect(result).To(BeFalse())
		})
		It("s3 configuration does not exist", func() {
			// given
			storageCfg := &managementv1alpha1.Storage{}
			// when
			result := storage.ShouldUseExternalConfig(storageCfg)
			// then
			Expect(result).To(BeFalse())
		})
		It("s3 configuration is empty", func() {
			// given
			storageCfg := &managementv1alpha1.Storage{
				S3: &managementv1alpha1.S3Storage{},
			}
			// when
			result := storage.ShouldUseExternalConfig(storageCfg)
			// then
			Expect(result).To(BeFalse())
		})
		It("s3 configuration should be used", func() {
			// given
			storageCfg := &managementv1alpha1.Storage{
				S3: &managementv1alpha1.S3Storage{
					SecretName: "s3secret",
				},
			}
			// when
			result := storage.ShouldUseExternalConfig(storageCfg)
			// then
			Expect(result).To(BeTrue())
		})
	})

	Context("ShouldCreateOBC", func() {
		It("storage configuration is nil", func() {
			// given
			// when
			result := storage.ShouldCreateOBC(nil)
			// then
			Expect(result).To(BeFalse())
		})
		It("s3 configuration does not exist", func() {
			// given
			storageCfg := &managementv1alpha1.Storage{}
			// when
			result := storage.ShouldCreateOBC(storageCfg)
			// then
			Expect(result).To(BeFalse())
		})
		It("s3 external configuration not empty", func() {
			// given
			storageCfg := &managementv1alpha1.Storage{
				S3: &managementv1alpha1.S3Storage{
					SecretName: "s3secret",
					CreateOBC:  true,
				},
			}
			// when
			result := storage.ShouldCreateOBC(storageCfg)
			// then
			Expect(result).To(BeFalse())
		})
		It("should not create OBC", func() {
			// given
			storageCfg := &managementv1alpha1.Storage{
				S3: &managementv1alpha1.S3Storage{
					CreateOBC: false,
				},
			}
			// when
			result := storage.ShouldCreateOBC(storageCfg)
			// then
			Expect(result).To(BeFalse())
		})
		It("should create OBC", func() {
			// given
			storageCfg := &managementv1alpha1.Storage{
				S3: &managementv1alpha1.S3Storage{
					CreateOBC: true,
				},
			}
			// when
			result := storage.ShouldCreateOBC(storageCfg)
			// then
			Expect(result).To(BeTrue())
		})
	})
})
