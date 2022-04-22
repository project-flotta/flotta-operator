package e2e_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/client-go/kubernetes"
)

var certsPath = "/etc/pki/consumer"
var CAcertsPath = filepath.Join(certsPath, "ca.pem")
var ClientCertPath = filepath.Join(certsPath, "cert.pem")
var ClientKeyPath = filepath.Join(certsPath, "key.pem")
var testCertificates = []string{CAcertsPath, ClientKeyPath, ClientCertPath}
var localTestCertificates = []string{
	"/tmp/ca.pem",
	"/tmp/cert.pem",
	"/tmp/key.pem",
}

var edgeDeviceResource = schema.GroupVersionResource{Group: "management.project-flotta.io", Version: "v1alpha1", Resource: "edgedevices"}

const (
	EdgeDeviceImage string = "quay.io/project-flotta/edgedevice"
	Namespace       string = "default" // the namespace where flotta operator is running
	waitTimeout     int    = 120
	sleepInterval   int    = 2
)

type EdgeDevice interface {
	GetId() string
	Register(cmds ...string) error
	Unregister() error
	Get() (*unstructured.Unstructured, error)
	Remove() error
	DumpLogs(extraCommands ...string)
	Exec(string) (string, error)
	WaitForWorkloadState(string, string) error
	ValidateNoDataRaceInLogs(extraCommands ...string) (bool, error)
}

type edgeDeviceDocker struct {
	device    dynamic.NamespaceableResourceInterface
	cli       *client.Client
	name      string
	machineId string
}

func NewEdgeDevice(k8sclient dynamic.Interface, deviceName string) (EdgeDevice, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	machineId := uuid.NewString()
	resource := k8sclient.Resource(edgeDeviceResource)
	return &edgeDeviceDocker{device: resource, cli: cli, name: deviceName, machineId: machineId}, nil
}

func (e *edgeDeviceDocker) GetId() string {
	return e.machineId
}

func (e *edgeDeviceDocker) WaitForWorkloadState(workloadName string, workloadPhase string) error {
	return e.waitForDevice(func() bool {
		device, err := e.Get()
		if device == nil || err != nil {
			if device == nil {
				ginkgo.GinkgoT().Logf("WaitForWorkloadState failed since the Get() returned empty device\n")
			}
			if err != nil {
				ginkgo.GinkgoT().Logf("WaitForWorkloadState failed since Get() failed. Error: %v\n", err)
			}
			return false
		}

		if device.Object == nil {
			return false
		}

		status, ok := device.Object["status"].(map[string]interface{})
		if !ok {
			return false
		}
		if status["workloads"] == nil {
			ginkgo.GinkgoT().Logf("WaitForWorkloadState failed since status contained no workloads\n")
			return false
		}
		workloads := status["workloads"].([]interface{})
		for _, workload := range workloads {
			workload := workload.(map[string]interface{})
			if workload["name"].(string) == workloadName && workload["phase"].(string) == workloadPhase {
				return true
			}
		}
		ginkgo.GinkgoT().Logf("WaitForWorkloadState failed since workloadName didn't match any workload\n")
		return false
	})
}

func (e *edgeDeviceDocker) CopyCerts() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, certificatePath := range localTestCertificates {
		fp, err := archive.Tar(certificatePath, archive.Gzip)
		if err != nil {
			return err
		}
		err = e.cli.CopyToContainer(ctx, e.name, certsPath, fp, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
		if err != nil {
			return err
		}
	}

	for _, certificatePath := range testCertificates {
		if _, err := e.Exec(fmt.Sprintf("chmod 660 %s", certificatePath)); err != nil {
			return err
		}
	}

	if _, err := e.Exec(fmt.Sprintf("echo 'ca-root = [\"%v\"]' >> /etc/yggdrasil/config.toml", CAcertsPath)); err != nil {
		return err
	}

	return nil
}

func (e *edgeDeviceDocker) Exec(command string) (string, error) {
	resp, err := e.cli.ContainerExecCreate(context.TODO(), e.name, types.ExecConfig{AttachStdout: true, AttachStderr: true, Cmd: []string{"/bin/bash", "-c", command}})
	if err != nil {
		return "", err
	}
	response, err := e.cli.ContainerExecAttach(context.Background(), resp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", err
	}
	defer response.Close()

	data, err := ioutil.ReadAll(response.Reader)
	if err != nil {
		return "", err
	}

	return strings.TrimFunc(string(data), func(r rune) bool {
		return !unicode.IsGraphic(r)
	}), nil
}

func (e *edgeDeviceDocker) GetLogs(extraCommands ...string) (map[string]string, error) {
	var err error
	logsMap := make(map[string]string)
	commands := []string{
		"journalctl -u podman",
		"journalctl -u yggdrasild",
		"ps aux",
		"podman ps -a",
		"systemctl status podman",
		"systemctl status yggdrasild",
	}
	commands = append(commands, extraCommands...)

	for _, cmd := range commands {
		output, err := e.Exec(cmd)
		if err != nil {
			ginkgo.GinkgoT().Logf("Error: Failed to retrieve logs for command '%s': %v \n", cmd, err)
		}
		logsMap[cmd] = output
	}

	return logsMap, err
}

func (e *edgeDeviceDocker) DumpLogs(extraCommands ...string) {
	logsMap, err := e.GetLogs(extraCommands...)
	if err != nil {
		ginkgo.GinkgoT().Logf("Error: GetLogs failed: %v \n", err)
	}
	for cmd, output := range logsMap {
		ginkgo.GinkgoT().Logf("Command: %s \n Output:\n %s\n", cmd, output)
	}
}

func (e *edgeDeviceDocker) Get() (*unstructured.Unstructured, error) {
	device, err := e.device.Namespace(Namespace).Get(context.TODO(), e.machineId, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (e *edgeDeviceDocker) Remove() error {
	return e.cli.ContainerRemove(context.TODO(), e.name, types.ContainerRemoveOptions{Force: true})
}

func (e *edgeDeviceDocker) Unregister() error {
	err := e.device.Namespace(Namespace).Delete(context.TODO(), e.machineId, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return e.waitForDevice(func() bool {
		if eCr, err := e.Get(); eCr == nil && err != nil {
			return true
		}
		return false
	})
}

func (e *edgeDeviceDocker) waitForDevice(cond func() bool) error {
	for i := 0; i <= waitTimeout; i += sleepInterval {
		if cond() {
			return nil
		} else {
			time.Sleep(time.Duration(sleepInterval) * time.Second)
		}
	}

	return fmt.Errorf("Error waiting for edgedevice %v[%v]", e.name, e.machineId)
}

// Register register the endgedevice with the operator API. A set of commands
// can be used to execute just before the registration happens. The main use
// case is to add something needed for the test, like network-latency.
func (e *edgeDeviceDocker) Register(cmds ...string) error {
	ctx := context.Background()
	resp, err := e.cli.ContainerCreate(ctx, &container.Config{Image: EdgeDeviceImage}, &container.HostConfig{Privileged: true, ExtraHosts: []string{"project-flotta.io:172.17.0.1"}}, nil, nil, e.name)
	if err != nil {
		return err
	}

	if err := e.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	for _, cmd := range cmds {
		if _, err = e.Exec(cmd); err != nil {
			return fmt.Errorf("Cannot execute register command '%s': %v", cmd, err)
		}
	}

	if _, err = e.Exec(fmt.Sprintf("echo 'client-id = \"%v\"' >> /etc/yggdrasil/config.toml", e.machineId)); err != nil {
		return err
	}

	if err := e.CopyCerts(); err != nil {
		return fmt.Errorf("cannot copy certificates to device: %v", err)
	}

	if _, err = e.Exec("systemctl start podman"); err != nil {
		return err
	}

	if _, err = e.Exec("systemctl start yggdrasild.service"); err != nil {
		return err
	}

	return e.waitForDevice(func() bool {
		if eCr, _ := e.Get(); eCr != nil && err == nil {
			return true
		}
		return false
	})
}

func (e *edgeDeviceDocker) ValidateNoDataRaceInLogs(extraCommands ...string) (bool, error) {
	logsMap, err := e.GetLogs(extraCommands...)
	if err != nil {
		ginkgo.GinkgoT().Logf("Error: GetLogs failed: %v \n", err)
		return false, err
	}

	foundDataRace := false
	for _, output := range logsMap {
		if strings.Contains(output, "WARNING: DATA RACE") {
			foundDataRace = true
		}
	}

	return !foundDataRace, nil
}

func newClient() (dynamic.Interface, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.BuildConfigFromFlags("", path.Join(homedir, ".kube/config"))
	if err != nil {
		return nil, err
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynClient, nil
}

func newClientset() (*kubernetes.Clientset, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.BuildConfigFromFlags("", path.Join(homedir, ".kube/config"))
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
