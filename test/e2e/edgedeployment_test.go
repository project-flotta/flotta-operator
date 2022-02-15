package e2e_test

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var edgeDeploymentResource = schema.GroupVersionResource{Group: "management.project-flotta.io", Version: "v1alpha1", Resource: "edgedeployments"}

type EdgeDeployment interface {
	Create(map[string]interface{}) (*unstructured.Unstructured, error)
	Get(string) (*unstructured.Unstructured, error)
	Remove(string) error
	RemoveAll() error
}

type edgeDeployment struct {
	deployment dynamic.NamespaceableResourceInterface
}

func NewEdgeDeployment(k8sclient dynamic.Interface) (EdgeDeployment, error) {
	resource := k8sclient.Resource(edgeDeploymentResource)
	return &edgeDeployment{deployment: resource}, nil
}

func (e *edgeDeployment) Get(name string) (*unstructured.Unstructured, error) {
	return e.deployment.Namespace(Namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (e *edgeDeployment) Create(data map[string]interface{}) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{
		Object: data,
	}

	return e.deployment.Namespace(Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})

}

func (e *edgeDeployment) RemoveAll() error {
	return e.deployment.Namespace(Namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
}

func (e *edgeDeployment) Remove(name string) error {
	err := e.deployment.Namespace(Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return e.waitForDeployment(func() bool {
		if eCr, err := e.Get(name); eCr == nil && err != nil {
			return true
		}
		return false
	})
}

func (e *edgeDeployment) waitForDeployment(cond func() bool) error {
	for i := 0; i <= waitTimeout; i += sleepInterval {
		if cond() {
			return nil
		} else {
			time.Sleep(time.Duration(sleepInterval) * time.Second)
		}
	}

	return fmt.Errorf("Error waiting for edgedeployment")
}
