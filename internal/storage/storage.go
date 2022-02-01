package storage

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Claimer struct {
	client client.Client
}

func NewClaimer(client client.Client) *Claimer {
	return &Claimer{client: client}
}

func (c *Claimer) GetClaim(ctx context.Context, name string, namespace string) (*obv1.ObjectBucketClaim, error) {
	obc := obv1.ObjectBucketClaim{}
	err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &obc)
	return &obc, err
}

func (c *Claimer) CreateClaim(ctx context.Context, device *v1alpha1.EdgeDevice) (*obv1.ObjectBucketClaim, error) {
	obc := obv1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      device.Name,
			Namespace: device.Namespace,
		},
		Spec: obv1.ObjectBucketClaimSpec{
			GenerateBucketName: device.Name,
			StorageClassName:   "openshift-storage.noobaa.io",
		},
	}
	err := c.client.Create(ctx, &obc)
	return &obc, err
}

func (c *Claimer) GetStorageConfiguration(ctx context.Context, device *v1alpha1.EdgeDevice) (*models.S3StorageConfiguration, error) {

	if device == nil {
		return nil, fmt.Errorf("Cannot get device")
	}

	if device.Status.DataOBC == nil {
		return nil, fmt.Errorf("Cannot get device OBC config")
	}

	obc, err := c.GetClaim(ctx, *device.Status.DataOBC, device.Namespace)
	if err != nil {
		return nil, err
	}

	// get bucket name for the device
	cm := corev1.ConfigMap{}
	err = c.client.Get(ctx, client.ObjectKey{Namespace: device.Namespace, Name: obc.Name}, &cm)
	if err != nil {
		return nil, err
	}
	conf := &models.S3StorageConfiguration{
		BucketName: cm.Data["BUCKET_NAME"],
		BucketPort: 443,
	}

	// get routable s3 endpoint
	s3route := routev1.Route{}
	err = c.client.Get(ctx, client.ObjectKey{Namespace: "openshift-storage", Name: "s3"}, &s3route)
	if err != nil {
		return nil, err
	}
	conf.BucketHost = s3route.Spec.Host

	// get s3 credentials
	secret := corev1.Secret{}
	err = c.client.Get(ctx, client.ObjectKey{Namespace: device.Namespace, Name: obc.Name}, &secret)
	if err != nil {
		return nil, err
	}
	awsAccessKeyID, exist := secret.Data["AWS_ACCESS_KEY_ID"]
	if !exist {
		return nil, fmt.Errorf("Cannot get AWS_ACCESS_KEY_ID")
	}
	conf.AwsAccessKeyID = base64.StdEncoding.EncodeToString(awsAccessKeyID)
	awsSecretAccessKey, exist := secret.Data["AWS_SECRET_ACCESS_KEY"]
	if !exist {
		return nil, fmt.Errorf("Cannot get AWS_SECRET_ACCESS_KEY_ID")
	}
	conf.AwsSecretAccessKey = base64.StdEncoding.EncodeToString(awsSecretAccessKey)

	// get ca for SSL endpoint
	secret = corev1.Secret{}
	err = c.client.Get(ctx, client.ObjectKey{Namespace: "openshift-ingress-operator", Name: "router-ca"}, &secret)
	if err != nil {
		return nil, err
	}
	caBundle, exist := secret.Data["tls.crt"]
	if exist {
		conf.AwsCaBundle = base64.StdEncoding.EncodeToString(caBundle)
	}
	conf.BucketRegion = "ignore"
	return conf, nil
}

func (c *Claimer) GetExternalStorageConfig(ctx context.Context, device *v1alpha1.EdgeDevice) (*models.S3StorageConfiguration, error) {
	config := device.Spec.Storage.S3
	if config == nil {
		return nil, fmt.Errorf("missing storage in device configuration. Device name: %s", device.Name)
	}
	cm := corev1.ConfigMap{}
	err := c.client.Get(ctx, client.ObjectKey{Namespace: config.ConfigMapNamespace, Name: config.ConfigMapName}, &cm)
	if err != nil {
		return nil, err
	}
	secret := corev1.Secret{}
	err = c.client.Get(ctx, client.ObjectKey{Namespace: config.SecretNamespace, Name: config.SecretName}, &secret)
	if err != nil {
		return nil, err
	}
	configMapFullName := types.NamespacedName{
		Namespace: config.ConfigMapNamespace,
		Name:      config.ConfigMapName,
	}.String()
	secretFullName := types.NamespacedName{
		Namespace: config.SecretNamespace,
		Name:      config.SecretName,
	}.String()
	missingFieldMessage := "Missing field %s in resource %s"
	bucketName, exists := cm.Data["BUCKET_NAME"]
	if !exists {
		return nil, fmt.Errorf(missingFieldMessage, "BUCKET_NAME", configMapFullName)
	}
	bucketHost, exists := cm.Data["BUCKET_HOST"]
	if !exists {
		return nil, fmt.Errorf(missingFieldMessage, "BUCKET_HOST", configMapFullName)
	}
	bucketPort, exists := cm.Data["BUCKET_PORT"]
	if !exists {
		return nil, fmt.Errorf(missingFieldMessage, "BUCKET_PORT", configMapFullName)
	}
	bucketRegion, exists := cm.Data["BUCKET_REGION"]
	if !exists {
		return nil, fmt.Errorf(missingFieldMessage, "BUCKET_REGION", configMapFullName)
	}
	accessKeyID, exists := secret.Data["AWS_ACCESS_KEY_ID"]
	if !exists {
		return nil, fmt.Errorf(missingFieldMessage, "AWS_ACCESS_KEY_ID", secretFullName)
	}
	secretAccessKey, exists := secret.Data["AWS_SECRET_ACCESS_KEY"]
	if !exists {
		return nil, fmt.Errorf(missingFieldMessage, "AWS_SECRET_ACCESS_KEY", secretFullName)
	}
	caBundle := secret.Data["tls.crt"]
	bucketPortNumeric, err := strconv.Atoi(bucketPort)
	if err != nil {
		return nil, err
	}
	return &models.S3StorageConfiguration{
		BucketName:         bucketName,
		BucketPort:         int32(bucketPortNumeric),
		BucketHost:         bucketHost,
		BucketRegion:       bucketRegion,
		AwsAccessKeyID:     base64.StdEncoding.EncodeToString(accessKeyID),
		AwsSecretAccessKey: base64.StdEncoding.EncodeToString(secretAccessKey),
		AwsCaBundle:        base64.StdEncoding.EncodeToString(caBundle),
	}, nil
}

func ShouldUseExternalConfig(device *v1alpha1.EdgeDevice) bool {
	s3Obj := getS3(device)
	if s3Obj != nil {
		if s3Obj.ConfigMapName != "" ||
			s3Obj.ConfigMapNamespace != "" ||
			s3Obj.SecretName != "" ||
			s3Obj.SecretNamespace != "" {
			return true
		}
	}
	return false
}

func ShouldCreateOBC(device *v1alpha1.EdgeDevice) bool {
	s3Obj := getS3(device)
	if s3Obj != nil {
		if !ShouldUseExternalConfig(device) {
			return s3Obj.CreateOBC
		}
	}
	return false
}

func getS3(device *v1alpha1.EdgeDevice) *v1alpha1.S3Storage {
	if device != nil {
		storageObj := device.Spec.Storage
		if storageObj != nil {
			return storageObj.S3
		}
	}

	return nil
}
