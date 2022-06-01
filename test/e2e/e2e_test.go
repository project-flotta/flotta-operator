package e2e_test

import (
	"context"
	"fmt"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	managementv1alpha1 "github.com/project-flotta/flotta-operator/generated/clientset/versioned/typed/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	hostPort  = 8885
	nginxPort = 8080
)

var _ = Describe("e2e", func() {

	var (
		clientset *kubernetes.Clientset
		client    managementv1alpha1.ManagementV1alpha1Interface
		workload  EdgeWorkload
		device    EdgeDevice
		err       error
	)

	BeforeEach(func() {
		clientset, err = newClientset()
		Expect(err).To(BeNil())
		client, err = newClient()
		Expect(err).To(BeNil())
		device, err = NewEdgeDevice(client, "edgedevice1")
		Expect(err).To(BeNil())
		workload, err = NewEdgeWorkload(client)
		Expect(err).To(BeNil())

	})

	AfterEach(func() {
		_ = workload.RemoveAll()
		_ = device.Unregister()
		_ = device.Remove()
	})

	JustAfterEach(func() {
		isValid, err := device.ValidateNoDataRaceInLogs()
		Expect(err).NotTo(HaveOccurred())
		Expect(isValid).To(BeTrue(), "Found data race in logs")
	})

	AfterFailed(func() {
		device.DumpLogs()
	})

	Context("Sanity", func() {
		It("Check services are running after installation", func() {
			// when
			err := device.Register()
			Expect(err).To(BeNil())

			// Check the node_exporter service is running:
			stdout, err := device.Exec("systemctl is-active node_exporter.service")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("active"))

			// Check the podman.socket is running:
			stdout, err = device.Exec("systemctl --machine flotta@.host is-active --user podman.socket")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("active"))

			// Check the nftables.service is running:
			stdout, err = device.Exec("systemctl is-active nftables.service")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("active"))

			// Check lingering of the flotta user:
			stdout, err = device.Exec("loginctl show-user flotta --property=Linger")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("Linger=yes"))
		})

		It("Deploy valid edgeworkload to registered device", func() {
			// given
			err := device.Register("dnf install ansible -y")
			Expect(err).To(BeNil())

			// when
			_, err = workload.Create(edgeworkloadDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			// Check the edgedevice report proper state of workload:
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil(), "cannot get workload status for nginx workload")

			// Check the nginx is serving content:
			stdout, err := device.Exec(fmt.Sprintf("curl http://localhost:%d", hostPort))
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("Welcome to nginx!"))
		})

		It("Deploy valid edgeworkload to registered device where pod and container name is different", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			_, err = workload.Create(edgeworkloadDeviceIdCtrName("nginx", "nginx-ctr", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			// Check the edgedevice report proper state of workload:
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil(), "cannot get workload status for nginx workload")

			// Check the nginx is serving content:
			stdout, err := device.Exec(fmt.Sprintf("curl http://localhost:%d", hostPort))
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("Welcome to nginx!"))
		})

		It("Deploy multiple containers part of workload", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			_, err = workload.Create(edgeworkloadDeviceIdContainers("nginx", device.GetId(), hostPort, nginxPort, 2))
			Expect(err).To(BeNil())

			// then
			// Check the edgedevice report proper state of workload:
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil(), "cannot get workload status for nginx workload")

			// Check the nginx1 is serving content:
			stdout, err := device.Exec(fmt.Sprintf("curl http://localhost:%d", hostPort))
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("Welcome to nginx!"))

			// Check the nginx2 is serving content:
			stdout, err = device.Exec(fmt.Sprintf("curl http://localhost:%d", hostPort+1))
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("Welcome to nginx!"))
		})

		It("Unregister device without any workloads", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			err = device.Unregister()
			Expect(err).To(BeNil())

			// then
			stdout, err := device.Exec("ls /etc/yggdrasil/device/ | wc -l")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("0"))
		})

		It("Unregister device with running workloads", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())
			_, err = workload.Create(edgeworkloadDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil())

			// when
			err = device.Unregister()
			Expect(err).To(BeNil())

			// then
			// properly cleaned ygg dir
			stdout, err := device.Exec("ls /etc/yggdrasil/device/ | wc -l")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("0"))

			// no pods running
			stdout, err = device.Exec("machinectl shell -q flotta@.host /usr/bin/podman ps --noheading | wc -l")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("1")) // machinectl print one empty new line

			// EdgeWorkload CR still exists
			depCr, err := workload.Get("nginx")
			Expect(err).To(BeNil())
			Expect(depCr).ToNot(BeNil())
		})

		It("Deploy edgeworkload with incorrect label", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			labels := map[string]string{"label": "yxz"}
			_, err = workload.Create(edgeworkloadDeviceLabel("nginx", labels, hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			depCr, err := workload.Get("nginx")
			Expect(err).To(BeNil())
			Expect(depCr).ToNot(BeNil())

			// no pods running
			stdout, err := device.Exec("podman ps --noheading | wc -l")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("0"))
		})

		It("Expose reserved container port", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			_, err = device.Exec(fmt.Sprintf("nc -l 127.0.0.1 %d &", hostPort))
			Expect(err).To(BeNil())
			_, err = workload.Create(edgeworkloadDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForWorkloadState("nginx", "Created")
			Expect(err).To(BeNil())

			// then
			stdout, err := device.Exec("systemctl --machine flotta@.host is-failed --user pod-nginx_pod.service")
			Expect(err).To(BeNil())
			Expect(stdout).To(BeElementOf([]string{"activating", "deactivating", "inactive"}))
		})

		It("Re-create the edgeworkload", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			_, err = workload.Create(edgeworkloadDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil())

			// then
			// remove
			err = workload.Remove("nginx")
			Expect(err).To(BeNil())

			// re-create the same
			_, err = workload.Create(edgeworkloadDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil())
		})

		It("Create edgeworkload with env secret", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			// create secret
			secret := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: Namespace,
				},
				Data: map[string][]byte{"key1": []byte("config1")},
			}
			_, err = clientset.CoreV1().Secrets(Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// create workload
			_, err = workload.Create(edgeworkloadWithSecretFromEnv("nginx", device.GetId(), "mysecret"))
			Expect(err).To(BeNil())
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil())

			// then
			stdout, err := device.Exec("machinectl shell -q flotta@.host /usr/bin/podman exec nginx_pod-nginx env | grep key1")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("key1=config1"))

			err = clientset.CoreV1().Secrets(Namespace).Delete(context.TODO(), "mysecret", metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		})

		It("Create edgeworkload with env configmap", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			configmap := &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myconfigmap",
					Namespace: Namespace,
				},
				Data: map[string]string{"key1": "config1"},
			}
			_, err = clientset.CoreV1().ConfigMaps(Namespace).Create(context.TODO(), configmap, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			_, err = workload.Create(edgeworkloadWithConfigMapFromEnv("nginx", device.GetId(), "myconfigmap"))
			Expect(err).To(BeNil())
			err = device.WaitForWorkloadState("nginx", "Running")
			Expect(err).To(BeNil())

			// then
			stdout, err := device.Exec("machinectl shell -q flotta@.host /usr/bin/podman exec nginx_pod-nginx env | grep key1")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("key1=config1"))

			err = clientset.CoreV1().ConfigMaps(Namespace).Delete(context.TODO(), "myconfigmap", metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		})
	})

	Context("Network issues", func() {
		conformanceTest := func() {
			// when
			_, err = workload.Create(edgeworkloadDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			// Check the edgedevice report proper state of workload:
			err = device.WaitForWorkloadState("nginx", "Running")
			device.DumpLogs()
			Expect(err).To(BeNil(), "cannot get workload status for nginx workload")

			// Check the nginx is serving content:
			stdout, err := device.Exec(fmt.Sprintf("curl http://localhost:%d", hostPort))
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("Welcome to nginx!"))
		}

		It("Packets delayed 200ms", func() {
			// given
			err := device.Register(
				"dnf install iproute -y",
				"tc qdisc add dev eth0 root netem delay 200ms")
			Expect(err).To(BeNil())

			conformanceTest()
		})

		It("Some packets drops", func() {
			// given
			err := device.Register(
				"dnf install iproute -y",
				"tc qdisc add dev eth0 root netem loss 10%")
			Expect(err).To(BeNil())

			conformanceTest()
		})
	})
})

func edgeworkloadWithSecretFromEnv(name string, device string, secretName string) *v1alpha1.EdgeWorkload {
	workload := edgeworkload(name, name, hostPort, nginxPort, &secretName, nil)
	workload.Spec.Device = device
	return workload
}

func edgeworkloadWithConfigMapFromEnv(name string, device string, configMap string) *v1alpha1.EdgeWorkload {
	workload := edgeworkload(name, name, hostPort, nginxPort, nil, &configMap)
	workload.Spec.Device = device
	return workload
}

func edgeworkloadDeviceIdContainers(name string, device string, hostport int, containerport int, ctrCount int) *v1alpha1.EdgeWorkload {
	workload := edgeworkloadContainers(name, name, hostport, containerport, nil, nil, ctrCount)
	workload.Spec.Device = device
	return workload
}

func edgeworkloadDeviceIdCtrName(name string, ctrName string, device string, hostport int, containerport int) *v1alpha1.EdgeWorkload {
	workload := edgeworkload(name, ctrName, hostport, containerport, nil, nil)
	workload.Spec.Device = device
	return workload
}

func edgeworkloadDeviceId(name string, device string, hostport int, containerport int) *v1alpha1.EdgeWorkload {
	return edgeworkloadDeviceIdCtrName(name, name, device, hostport, containerport)
}

func edgeworkloadDeviceLabel(name string, labels map[string]string, hostport int, containerport int) *v1alpha1.EdgeWorkload {
	workload := edgeworkload(name, name, hostport, containerport, nil, nil)
	workload.Spec.DeviceSelector = &metav1.LabelSelector{
		MatchLabels: labels,
	}
	return workload
}

func edgeworkload(name string, ctrName string, hostport int, containerport int, secretRef *string, configRef *string) *v1alpha1.EdgeWorkload {
	return edgeworkloadContainers(name, ctrName, hostport, containerport, secretRef, configRef, 1)
}

func edgeworkloadContainers(name string, ctrName string, hostport int, containerport int, secretRef *string, configRef *string, ctrCount int) *v1alpha1.EdgeWorkload {
	envFrom := make([]corev1.EnvFromSource, 0)
	if secretRef != nil {
		envFrom = append(envFrom, corev1.EnvFromSource{SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: *secretRef},
		}})
	}
	if configRef != nil {
		envFrom = append(envFrom, corev1.EnvFromSource{ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: *configRef},
		}})
	}

	var containers = make([]corev1.Container, 0)
	for i := 0; i < ctrCount; i++ {
		// Let's use same name as workload for container, and different name
		// if multiple containers, so we use both cases.
		if i > 0 {
			ctrName = fmt.Sprintf("%s_%d", name, i)
		}
		containers = append(containers, corev1.Container{
			Name:  ctrName,
			Image: "quay.io/project-flotta/nginx:1.21.6",
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: int32(containerport),
					HostPort:      int32(hostport + i),
				},
			},
			EnvFrom: envFrom,
		})
	}

	workload := &v1alpha1.EdgeWorkload{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.EdgeWorkloadSpec{
			Type: "pod",
			Pod: v1alpha1.Pod{
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}

	return workload
}
