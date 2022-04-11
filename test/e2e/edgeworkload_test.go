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

var edgeWorkloadResource = schema.GroupVersionResource{Group: "management.project-flotta.io", Version: "v1alpha1", Resource: "edgeworkloads"}

type EdgeWorkload interface {
	Create(map[string]interface{}) (*unstructured.Unstructured, error)
	Get(string) (*unstructured.Unstructured, error)
	Remove(string) error
	RemoveAll() error
}

type edgeWorkload struct {
	workload dynamic.NamespaceableResourceInterface
}

func NewEdgeWorkload(k8sclient dynamic.Interface) (EdgeWorkload, error) {
	resource := k8sclient.Resource(edgeWorkloadResource)
	return &edgeWorkload{workload: resource}, nil
}

func (e *edgeWorkload) Get(name string) (*unstructured.Unstructured, error) {
	return e.workload.Namespace(Namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (e *edgeWorkload) Create(data map[string]interface{}) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{
		Object: data,
	}

	return e.workload.Namespace(Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})

}

func (e *edgeWorkload) RemoveAll() error {
	return e.workload.Namespace(Namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
}

func (e *edgeWorkload) Remove(name string) error {
	err := e.workload.Namespace(Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return e.waitForWorkload(func() bool {
		if eCr, err := e.Get(name); eCr == nil && err != nil {
			return true
		}
		return false
	})
}

func (e *edgeWorkload) waitForWorkload(cond func() bool) error {
	for i := 0; i <= waitTimeout; i += sleepInterval {
		if cond() {
			return nil
		} else {
			time.Sleep(time.Duration(sleepInterval) * time.Second)
		}
	}

	return fmt.Errorf("Error waiting for edgeworkload")
}
