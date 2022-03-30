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
		clientset  *kubernetes.Clientset
		client     dynamic.Interface
		deployment EdgeDeployment
		device     EdgeDevice
		err        error
	)

	BeforeEach(func() {
		clientset, err = newClientset()
		Expect(err).To(BeNil())
		client, err = newClient()
		Expect(err).To(BeNil())
		device, err = NewEdgeDevice(client, "edgedevice1")
		Expect(err).To(BeNil())
		deployment, err = NewEdgeDeployment(client)
		Expect(err).To(BeNil())

	})

	AfterFailed(func() {
		device.DumpLogs()
	})

	AfterEach(func() {
		_ = deployment.RemoveAll()
		_ = device.Unregister()
		_ = device.Remove()
	})
	Context("Sanity", func() {
		It("Deploy valid edgedeployment to registered device", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			_, err = deployment.Create(edgedeployemntDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			// Check the edgedevice report proper state of workload:
			err = device.WaitForDeploymentState("nginx", "Running")
			Expect(err).To(BeNil(), "cannot get deployment status for nginx workload")

			// Check the nginx is serving content:
			stdout, err := device.Exec(fmt.Sprintf("curl http://localhost:%d", hostPort))
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("Welcome to nginx!"))
		})

		It("Unregister device without any deployments", func() {
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

		It("Unregister device with running deployments", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())
			_, err = deployment.Create(edgedeployemntDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForDeploymentState("nginx", "Running")
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

			// EdgeDeployment CR still exists
			depCr, err := deployment.Get("nginx")
			Expect(err).To(BeNil())
			Expect(depCr).ToNot(BeNil())
		})

		It("Deploy edgedeployment with incorrect label", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			labels := map[string]string{"label": "yxz"}
			_, err = deployment.Create(edgedeployemntDeviceLabel("nginx", labels, hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			depCr, err := deployment.Get("nginx")
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
			_, err = deployment.Create(edgedeployemntDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForDeploymentState("nginx", "Created")
			Expect(err).To(BeNil())

			// then
			stdout, err := device.Exec("systemctl is-failed pod-*.service")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("failed"))
		})

		It("Re-create the edgedeployment", func() {
			// given
			err := device.Register()
			Expect(err).To(BeNil())

			// when
			_, err = deployment.Create(edgedeployemntDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForDeploymentState("nginx", "Running")
			Expect(err).To(BeNil())

			// then
			// remove
			err = deployment.Remove("nginx")
			Expect(err).To(BeNil())

			// re-create the same
			_, err = deployment.Create(edgedeployemntDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())
			err = device.WaitForDeploymentState("nginx", "Running")
			Expect(err).To(BeNil())
		})

		It("Create edgedeployemnt with env secret", func() {
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

			// create deployemnt
			_, err = deployment.Create(edgedeployemntWithSecretFromEnv("nginx", device.GetId(), "mysecret"))
			Expect(err).To(BeNil())
			err = device.WaitForDeploymentState("nginx", "Running")
			Expect(err).To(BeNil())

			// then
			stdout, err := device.Exec("podman exec nginx_pod-nginx env | grep key1")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("key1=config1"))

			err = clientset.CoreV1().Secrets(Namespace).Delete(context.TODO(), "mysecret", metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		})

		It("Create edgedeployemnt with env configmap", func() {
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

			_, err = deployment.Create(edgedeployemntWithConfigMapFromEnv("nginx", device.GetId(), "myconfigmap"))
			Expect(err).To(BeNil())
			err = device.WaitForDeploymentState("nginx", "Running")
			Expect(err).To(BeNil())

			// then
			stdout, err := device.Exec("podman exec nginx_pod-nginx env | grep key1")
			Expect(err).To(BeNil())
			Expect(stdout).To(Equal("key1=config1"))

			err = clientset.CoreV1().ConfigMaps(Namespace).Delete(context.TODO(), "myconfigmap", metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		})
	})

	Context("Network issues", func() {
		conformanceTest := func() {
			// when
			_, err = deployment.Create(edgedeployemntDeviceId("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			// Check the edgedevice report proper state of workload:
			err = device.WaitForDeploymentState("nginx", "Running")
			Expect(err).To(BeNil(), "cannot get deployment status for nginx workload")

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

func edgedeployemntWithSecretFromEnv(name string, device string, secretName string) map[string]interface{} {
	deployment := edgedeployemnt(name, hostPort, nginxPort, &secretName, nil)
	deployment["spec"].(map[string]interface{})["device"] = device
	return deployment
}

func edgedeployemntWithConfigMapFromEnv(name string, device string, configMap string) map[string]interface{} {
	deployment := edgedeployemnt(name, hostPort, nginxPort, nil, &configMap)
	deployment["spec"].(map[string]interface{})["device"] = device
	return deployment
}

func edgedeployemntDeviceId(name string, device string, hostport int, containerport int) map[string]interface{} {
	deployment := edgedeployemnt(name, hostport, containerport, nil, nil)
	deployment["spec"].(map[string]interface{})["device"] = device
	return deployment
}

func edgedeployemntDeviceLabel(name string, labels map[string]string, hostport int, containerport int) map[string]interface{} {
	deployment := edgedeployemnt(name, hostport, containerport, nil, nil)
	spec := deployment["spec"].(map[string]interface{})
	spec["deviceSelector"] = map[string]interface{}{}
	spec["deviceSelector"].(map[string]interface{})["matchLabels"] = labels
	return deployment
}

func edgedeployemnt(name string, hostport int, containerport int, secretRef *string, configRef *string) map[string]interface{} {
	deployment := map[string]interface{}{}
	deployment["apiVersion"] = "management.project-flotta.io/v1alpha1"
	deployment["kind"] = "EdgeDeployment"
	deployment["metadata"] = map[string]interface{}{
		"name": name,
	}
	containers := []map[string]interface{}{{
		"name":  name,
		"image": "quay.io/bitnami/nginx:latest",
		"ports": []map[string]int{{
			"hostPort":      hostport,
			"containerPort": containerport,
		}},
	}}
	containers[0]["envFrom"] = []map[string]interface{}{}
	envFrom := containers[0]["envFrom"].([]map[string]interface{})
	if secretRef != nil {
		containers[0]["envFrom"] = append(envFrom, map[string]interface{}{"secretRef": map[string]string{"name": *secretRef}})
	}
	if configRef != nil {
		containers[0]["envFrom"] = append(envFrom, map[string]interface{}{"configMapRef": map[string]string{"name": *configRef}})
	}

	deployment["spec"] = map[string]interface{}{
		"type": "pod",
		"pod": map[string]interface{}{
			"spec": map[string]interface{}{
				"containers": containers,
			},
		},
	}
	return deployment
}
