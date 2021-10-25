package storage_test

import (
	"context"
	"net/http"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/storage"
	"github.com/jakub-dzon/k4e-operator/models"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
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
		routev1.AddToScheme(scheme.Scheme)
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
				OsImageId:   "test",
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
					OsImageId:   "test",
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
			Expect(err.Error()).To(Equal("Cannot get AWS_ACCESS_KEY_ID for device 'test'"))
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
			Expect(err.Error()).To(Equal("Cannot get AWS_SECRET_ACCESS_KEY_ID for device 'test'"))
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
			}
			Expect(result).To(Equal(&cfg))

			Expect(err).NotTo(HaveOccurred())
		})
	})

})
