package configmaps_test

import (
	"context"

	gomock "github.com/golang/mock/gomock"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/configmaps"
	"github.com/jakub-dzon/k4e-operator/internal/k8sclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ConfigMaps", func() {

	var (
		mockCtrl         *gomock.Controller
		k8sClient        *k8sclient.MockK8sClient
		configMapManager configmaps.ConfigMap
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		k8sClient = k8sclient.NewMockK8sClient(mockCtrl)
		configMapManager = configmaps.NewConfigMap(k8sClient)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Fetch", func() {
		It("expect configmap to be properly fetched if returned by k8sclient for EnvFrom", func() {
			// given
			k8sClient.EXPECT().Get(
				gomock.AssignableToTypeOf(context.TODO()),
				gomock.Eq(client.ObjectKey{Name: "mycm1", Namespace: "default"}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				DoAndReturn(configMapGenerator("mycm1", "default", map[string]string{"key": "value"}))
			podData := &v1alpha1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test",
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "mycm1",
										},
									},
								},
							},
						},
					},
				},
			}
			deployment := getDeployment(podData)

			// when
			cm, err := configMapManager.Fetch(context.TODO(), *deployment, "default")

			// then]
			Expect(err).To(BeNil())
			Expect(len(cm)).To(Equal(1))
		})
	})

	It("no configmaps for no sources", func() {
		// given
		podData := &v1alpha1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "test",
					},
				},
			},
		}
		deployment := getDeployment(podData)

		// when
		cm, err := configMapManager.Fetch(context.TODO(), *deployment, "default")

		// then]
		Expect(err).To(BeNil())
		Expect(len(cm)).To(Equal(0))
	})

	It("expect configmap to be properly fetched if returned by k8sclient for Env", func() {
		// given
		k8sClient.EXPECT().Get(
			gomock.AssignableToTypeOf(context.TODO()),
			gomock.Eq(client.ObjectKey{Name: "mycm1", Namespace: "default"}),
			gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			DoAndReturn(configMapGenerator("mycm1", "default", map[string]string{"key": "value"}))
		podData := &v1alpha1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "test",
						Env: []corev1.EnvVar{
							{
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &v1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "mycm1",
										},
										Key: "key",
									},
								},
							},
						},
					},
				},
			},
		}
		deployment := getDeployment(podData)

		// when
		cm, err := configMapManager.Fetch(context.TODO(), *deployment, "default")

		// then]
		Expect(err).To(BeNil())
		Expect(len(cm)).To(Equal(1))
	})

	It("expect configmap to be properly fetched if returned by k8sclient for Volume", func() {
		// given
		k8sClient.EXPECT().Get(
			gomock.AssignableToTypeOf(context.TODO()),
			gomock.Eq(client.ObjectKey{Name: "mycm1", Namespace: "default"}),
			gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			DoAndReturn(configMapGenerator("mycm1", "default", map[string]string{"key": "value"}))
		podData := &v1alpha1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "test",
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "vol1",
								MountPath: "/path",
							},
						},
					},
				},
				Volumes: []v1.Volume{
					{
						Name: "vol1",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "mycm1",
								},
							},
						},
					},
				},
			},
		}
		deployment := getDeployment(podData)

		// when
		cm, err := configMapManager.Fetch(context.TODO(), *deployment, "default")

		// then]
		Expect(err).To(BeNil())
		Expect(len(cm)).To(Equal(1))
	})

})

func getDeployment(podData *v1alpha1.Pod) *v1alpha1.EdgeDeployment {
	return &v1alpha1.EdgeDeployment{
		Spec: v1alpha1.EdgeDeploymentSpec{
			Type: "pod",
			Pod:  *podData,
		},
	}
}

func configMapGenerator(name, namespace string, data map[string]string) func(_ context.Context, _ client.ObjectKey, obj client.Object) error {
	return func(_ context.Context, _ client.ObjectKey, obj client.Object) error {
		cm := obj.(*corev1.ConfigMap)
		cm.SetName(name)
		cm.SetNamespace(namespace)
		cm.Data = data
		return nil
	}
}
