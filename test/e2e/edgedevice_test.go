package e2e_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var edgeDeviceResource = schema.GroupVersionResource{Group: "management.project-flotta.io", Version: "v1alpha1", Resource: "edgedevices"}

const (
	EdgeDeviceImage string = "quay.io/project-flotta/edgedevice"
	Namespace       string = "default" // the namespace where flotta operator is running
	waitTimeout     int    = 120
	sleepInterval   int    = 2
)

type EdgeDevice interface {
	GetId() string
	Register() error
	Unregister() error
	Get() (*unstructured.Unstructured, error)
	Remove() error
	Exec([]string) (string, error)
	WaitForDeploymentState(string, string) error
}

type edgeDeviceDocker struct {
	device    dynamic.NamespaceableResourceInterface
	cli       *client.Client
	name      string
	machineId string
}

func NewEdgeDevice(deviceName string) (EdgeDevice, error) {
	k8sclient, err := newClient()
	if err != nil {
		return nil, err
	}
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

func (e *edgeDeviceDocker) WaitForDeploymentState(deploymentName string, deploymentPhase string) error {
	return e.waitForDevice(func() bool {
		device, err := e.Get()
		if device == nil || err != nil {
			return false
		}
		status := device.Object["status"].(map[string]interface{})
		if status["deployments"] == nil {
			return false
		}
		deployments := status["deployments"].([]interface{})
		for _, deployment := range deployments {
			deployment := deployment.(map[string]interface{})
			if deployment["name"].(string) == deploymentName && deployment["phase"].(string) == deploymentPhase {
				return true
			}
		}

		return false
	})
}

func (e *edgeDeviceDocker) Exec(command []string) (string, error) {
	resp, err := e.cli.ContainerExecCreate(context.TODO(), e.name, types.ExecConfig{AttachStdout: true, AttachStderr: true, Cmd: command})
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
	return string(data), nil
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

func (e *edgeDeviceDocker) Register() error {
	ctx := context.Background()
	resp, err := e.cli.ContainerCreate(ctx, &container.Config{
		Image: EdgeDeviceImage, Cmd: []string{fmt.Sprintf("--machine-id=%s", e.machineId)},
	}, &container.HostConfig{Privileged: true}, nil, nil, e.name)
	if err != nil {
		return err
	}

	if err := e.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return e.waitForDevice(func() bool {
		if eCr, _ := e.Get(); eCr != nil && err == nil {
			return true
		}
		return false
	})
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
