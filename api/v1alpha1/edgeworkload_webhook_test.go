package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
)

var _ = Describe("EdgeWorkload Webhook", func() {
	var (
		edgeWorkload v1alpha1.EdgeWorkload
		podSpec      *corev1.PodSpec
	)
	BeforeEach(func() {
		edgeWorkload = v1alpha1.EdgeWorkload{
			Spec: v1alpha1.EdgeWorkloadSpec{
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
		podSpec = &edgeWorkload.Spec.Pod.Spec
	})

	Context("EdgeWorkload validating webhook", func() {
		It("delete should always succeed", func() {
			// given
			podSpec.Volumes = append(edgeWorkload.Spec.Pod.Spec.Volumes,
				corev1.Volume{
					Name: "volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "secret",
						},
					},
				})

			// when
			err := edgeWorkload.ValidateDelete()

			// then
			Expect(err).NotTo(HaveOccurred())
		})

		It("create valid EdgeWorkload", func() {
			// given

			// when
			err := edgeWorkload.ValidateCreate()

			// then
			Expect(err).NotTo(HaveOccurred())
		})

		It("update valid EdgeWorkload", func() {
			// given

			// when
			err := edgeWorkload.ValidateUpdate(nil)

			// then
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("test all invalid fields", func(editEdgeWorkload func()) {

			// given
			editEdgeWorkload()

			// when
			errCreate := edgeWorkload.ValidateCreate()
			errUpdate := edgeWorkload.ValidateUpdate(nil)

			// then
			Expect(errCreate).To(HaveOccurred())
			Expect(errUpdate).To(HaveOccurred())
		},
			Entry("container.lifecycle", func() {
				podSpec.Containers[0].Lifecycle = &corev1.Lifecycle{}
			}),
			Entry("container.livenessProbe", func() {
				podSpec.Containers[0].LivenessProbe = &corev1.Probe{}
			}),
			Entry("container.readinessProbe", func() {
				podSpec.Containers[0].ReadinessProbe = &corev1.Probe{}
			}),
			Entry("container.startupProbe", func() {
				podSpec.Containers[0].StartupProbe = &corev1.Probe{}
			}),
			Entry("container.volumeDevices", func() {
				podSpec.Containers[0].VolumeDevices = []corev1.VolumeDevice{{}}
			}),
			Entry("container.resources.limits", func() {
				podSpec.Containers[0].Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(0, resource.BinarySI),
				}
			}),
			Entry("container.resources.requests", func() {
				podSpec.Containers[0].Resources.Requests = corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(0, resource.BinarySI),
				}
			}),
			Entry("container.env.valueFrom.fieldRef", func() {
				podSpec.Containers[0].Env = []corev1.EnvVar{
					{
						Name: "var",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{},
						},
					},
				}
			}),
			Entry("container.env.valueFrom.resourceFieldRef", func() {
				podSpec.Containers[0].Env = []corev1.EnvVar{
					{
						Name: "var",
						ValueFrom: &corev1.EnvVarSource{
							ResourceFieldRef: &corev1.ResourceFieldSelector{},
						},
					},
				}
			}),
			Entry("volumes", func() {
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

		It("reuse container name", func() {
			// given
			podSpec.Containers = append(edgeWorkload.Spec.Pod.Spec.Containers,
				corev1.Container{
					Name:            "container",
					Image:           "stam",
					ImagePullPolicy: corev1.PullAlways,
				})

			// when
			err := edgeWorkload.ValidateCreate()

			// then
			Expect(err).Should(MatchError("name collisions for containers within the same pod spec are not supported.\n" +
				"container name: 'container' has been reused"))
		})

		It("reuse init container name", func() {
			// given
			podSpec.InitContainers = append(edgeWorkload.Spec.Pod.Spec.Containers,
				corev1.Container{
					Name:            "container",
					Image:           "stam",
					ImagePullPolicy: corev1.PullAlways,
				})

			// when
			err := edgeWorkload.ValidateCreate()

			// then
			Expect(err).Should(MatchError("name collisions for containers within the same pod spec are not supported.\n" +
				"container name: 'container' has been reused"))
		})

		It("set port 9100 for edge workload", func() {
			// given
			podSpec.InitContainers = append(edgeWorkload.Spec.Pod.Spec.Containers,
				corev1.Container{
					Ports: []corev1.ContainerPort{
						{
							HostPort: 9100,
						},
					},
				})

			// when
			err := edgeWorkload.ValidateCreate()

			// then
			Expect(err).Should(MatchError("HostPort 9100 is reserved for internal use on the device and cannot be set for user workloads"))
		})
	})
})
