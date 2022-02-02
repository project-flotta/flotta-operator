package e2e_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	hostPort  = 8885
	nginxPort = 8080
)

var _ = Describe("e2e", func() {

	var (
		deployment EdgeDeployment
		device     EdgeDevice
		err        error
	)

	BeforeEach(func() {
		device, err = NewEdgeDevice("edgedevice1")
		Expect(err).To(BeNil())
		deployment, err = NewEdgeDeployment()
		Expect(err).To(BeNil())

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
			Expect(err).To(BeNil())
			_, err = deployment.Create(edgedeployemnt("nginx", device.GetId(), hostPort, nginxPort))
			Expect(err).To(BeNil())

			// then
			// Check the edgedevice report proper state of workload:
			err = device.WaitForDeploymentState("nginx", "Running")
			Expect(err).To(BeNil())

			// Check the nginx is serving content:
			stdout, err := device.Exec([]string{"curl", fmt.Sprintf("http://localhost:%d", hostPort)})
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("Welcome to nginx!"))
		})
	})
})

func edgedeployemnt(name string, device string, hostport int, containerport int) map[string]interface{} {
	deployment := map[string]interface{}{}
	deployment["apiVersion"] = "management.project-flotta.io/v1alpha1"
	deployment["kind"] = "EdgeDeployment"
	deployment["metadata"] = map[string]interface{}{
		"name": name,
	}
	deployment["spec"] = map[string]interface{}{
		"type":   "pod",
		"device": device,
		"pod": map[string]interface{}{
			"spec": map[string]interface{}{
				"containers": []map[string]interface{}{{
					"name":  name,
					"image": "quay.io/bitnami/nginx:latest",
					"ports": []map[string]int{{
						"hostPort":      hostport,
						"containerPort": containerport,
					}},
				}},
			},
		},
	}
	return deployment
}
