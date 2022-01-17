package v1alpha1_test

import (
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("EdgeDeployment Webhook", func() {
	var (
		edgeDeployment v1alpha1.EdgeDeployment
		podSpec        *corev1.PodSpec
	)
	BeforeEach(func() {
		edgeDeployment = v1alpha1.EdgeDeployment{
			Spec: v1alpha1.EdgeDeploymentSpec{
				Pod: v1alpha1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "container",
								Image:           "stam",
								ImagePullPolicy: corev1.PullAlways,
								Env: []corev1.EnvVar{
									{
										Name: "MY_ENV_VAR",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												Key: "key",
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "secret",
												},
											},
										},
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "volume",
										MountPath: "/var/local/volume",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "volume",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
		}
		podSpec = &edgeDeployment.Spec.Pod.Spec
	})

	Context("EdgeDeployment validating webhook", func() {
		It("delete should always succeed", func() {
			// given
			podSpec.Volumes = append(edgeDeployment.Spec.Pod.Spec.Volumes,
				corev1.Volume{
					Name: "volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "secret",
						},
					},
				})

			// when
			err := edgeDeployment.ValidateDelete()

			// then
			Expect(err).NotTo(HaveOccurred())
		})

		It("create valid EdgeDeployment", func() {
			// given

			// when
			err := edgeDeployment.ValidateCreate()

			// then
			Expect(err).NotTo(HaveOccurred())
		})

		It("update valid EdgeDeployment", func() {
			// given

			// when
			err := edgeDeployment.ValidateUpdate(nil)

			// then
			Expect(err).NotTo(HaveOccurred())
		})

		table.DescribeTable("test all invalid fields", func(editEdgeDeployment func()) {
			// given
			editEdgeDeployment()

			// when
			errCreate := edgeDeployment.ValidateCreate()
			errUpdate := edgeDeployment.ValidateUpdate(nil)

			// then
			Expect(errCreate).To(HaveOccurred())
			Expect(errUpdate).To(HaveOccurred())
		},
			table.Entry("container.lifecycle", func() {
				podSpec.Containers[0].Lifecycle = &corev1.Lifecycle{}
			}),
			table.Entry("container.livenessProbe", func() {
				podSpec.Containers[0].LivenessProbe = &corev1.Probe{}
			}),
			table.Entry("container.readinessProbe", func() {
				podSpec.Containers[0].ReadinessProbe = &corev1.Probe{}
			}),
			table.Entry("container.startupProbe", func() {
				podSpec.Containers[0].StartupProbe = &corev1.Probe{}
			}),
			table.Entry("container.volumeDevices", func() {
				podSpec.Containers[0].VolumeDevices = []corev1.VolumeDevice{{}}
			}),
			table.Entry("container.resources.limits", func() {
				podSpec.Containers[0].Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(0, resource.BinarySI),
				}
			}),
			table.Entry("container.resources.requests", func() {
				podSpec.Containers[0].Resources.Requests = corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(0, resource.BinarySI),
				}
			}),
			table.Entry("container.env.valueFrom.fieldRef", func() {
				podSpec.Containers[0].Env = []corev1.EnvVar{
					{
						Name: "var",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{},
						},
					},
				}
			}),
			table.Entry("container.env.valueFrom.resourceFieldRef", func() {
				podSpec.Containers[0].Env = []corev1.EnvVar{
					{
						Name: "var",
						ValueFrom: &corev1.EnvVarSource{
							ResourceFieldRef: &corev1.ResourceFieldSelector{},
						},
					},
				}
			}),
			table.Entry("volumes", func() {
				podSpec.Volumes = []corev1.Volume{
					{
						Name: "volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				}
			}),
		)
	})
})
