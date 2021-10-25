package storage

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/models"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return nil, fmt.Errorf("Cannot get device OBC config, name: %v", device.Name)
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
		return nil, fmt.Errorf("Cannot get AWS_ACCESS_KEY_ID for device '%v'", device.Name)
	}
	conf.AwsAccessKeyID = base64.StdEncoding.EncodeToString(awsAccessKeyID)
	awsSecretAccessKey, exist := secret.Data["AWS_SECRET_ACCESS_KEY"]
	if !exist {
		return nil, fmt.Errorf("Cannot get AWS_SECRET_ACCESS_KEY_ID for device '%v'", device.Name)
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
	return conf, nil
}
