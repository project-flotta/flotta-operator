package e2e_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"

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
		client    dynamic.Interface
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
		device.DumpLogs()
		isValid, err := device.ValidateNoDataRaceInLogs()
		Expect(err).NotTo(HaveOccurred())
		Expect(isValid).To(BeTrue(), "Found data race in logs")
	})

	AfterFailed(func() {
		device.DumpLogs()
	})

	Context("Sanity", func() {
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
			stdout, err = device.Exec("podman ps --noheading | wc -l")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("0"))

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
			stdout, err := device.Exec("systemctl is-failed pod-*.service")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("failed"))
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
			stdout, err := device.Exec("podman exec nginx_pod-nginx env | grep key1")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("key1=config1"))

			err = clientset.CoreV1().Secrets(Namespace).Delete(context.TODO(), "mysecret", metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		})

		FIt("Create edgeworkload with env configmap", func() {
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
			stdout, err := device.Exec("podman exec nginx_pod-nginx env | grep key1")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("key1=config1"))

			err = clientset.CoreV1().ConfigMaps(Namespace).Delete(context.TODO(), "myconfigmap", metav1.DeleteOptions{})
			Expect(err).ToNot(BeNil())

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

func edgeworkloadWithSecretFromEnv(name string, device string, secretName string) map[string]interface{} {
	workload := edgeworkload(name, hostPort, nginxPort, &secretName, nil)
	workload["spec"].(map[string]interface{})["device"] = device
	return workload
}

func edgeworkloadWithConfigMapFromEnv(name string, device string, configMap string) map[string]interface{} {
	workload := edgeworkload(name, hostPort, nginxPort, nil, &configMap)
	workload["spec"].(map[string]interface{})["device"] = device
	return workload
}

func edgeworkloadDeviceIdContainers(name string, device string, hostport int, containerport int, ctrCount int) map[string]interface{} {
	workload := edgeworkloadContainers(name, hostport, containerport, nil, nil, ctrCount)
	workload["spec"].(map[string]interface{})["device"] = device
	return workload
}

func edgeworkloadDeviceId(name string, device string, hostport int, containerport int) map[string]interface{} {
	workload := edgeworkload(name, hostport, containerport, nil, nil)
	workload["spec"].(map[string]interface{})["device"] = device
	return workload
}

func edgeworkloadDeviceLabel(name string, labels map[string]string, hostport int, containerport int) map[string]interface{} {
	workload := edgeworkload(name, hostport, containerport, nil, nil)
	spec := workload["spec"].(map[string]interface{})
	spec["deviceSelector"] = map[string]interface{}{}
	spec["deviceSelector"].(map[string]interface{})["matchLabels"] = labels
	return workload
}

func edgeworkload(name string, hostport int, containerport int, secretRef *string, configRef *string) map[string]interface{} {
	return edgeworkloadContainers(name, hostport, containerport, secretRef, configRef, 1)
}

func edgeworkloadContainers(name string, hostport int, containerport int, secretRef *string, configRef *string, ctrCount int) map[string]interface{} {
	workload := map[string]interface{}{}
	workload["apiVersion"] = "management.project-flotta.io/v1alpha1"
	workload["kind"] = "EdgeWorkload"
	workload["metadata"] = map[string]interface{}{
		"name": name,
	}

	var containers []map[string]interface{}
	for i := 0; i < ctrCount; i++ {
		// Let's use same name as workload for container, and different name
		// if multiple containers, so we use both cases.
		ctrName := name
		if i > 0 {
			ctrName = fmt.Sprintf("%s_%d", name, i)
		}
		containers = append(containers, map[string]interface{}{
			"name":  ctrName,
			"image": "quay.io/project-flotta/nginx:1.21.6",
			"ports": []map[string]int{{
				"hostPort":      hostport + i,
				"containerPort": containerport,
			}},
		})
		containers[i]["envFrom"] = []map[string]interface{}{}
		envFrom := containers[i]["envFrom"].([]map[string]interface{})
		if secretRef != nil {
			containers[i]["envFrom"] = append(envFrom, map[string]interface{}{"secretRef": map[string]string{"name": *secretRef}})
		}
		if configRef != nil {
			containers[i]["envFrom"] = append(envFrom, map[string]interface{}{"configMapRef": map[string]string{"name": *configRef}})
		}
	}

	workload["spec"] = map[string]interface{}{
		"type": "pod",
		"pod": map[string]interface{}{
			"spec": map[string]interface{}{
				"containers": containers,
			},
		},
	}
	return workload
}
