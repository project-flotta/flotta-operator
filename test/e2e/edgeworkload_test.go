package e2e_test

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	managementv1alpha1 "github.com/project-flotta/flotta-operator/generated/clientset/versioned/typed/v1alpha1"
)

type EdgeWorkload interface {
	Create(*v1alpha1.EdgeWorkload) (*v1alpha1.EdgeWorkload, error)
	Get(string) (*v1alpha1.EdgeWorkload, error)
	Remove(string) error
	RemoveAll() error
}

type edgeWorkload struct {
	workload managementv1alpha1.ManagementV1alpha1Interface
}

func NewEdgeWorkload(client managementv1alpha1.ManagementV1alpha1Interface) (*edgeWorkload, error) {
	return &edgeWorkload{workload: client}, nil
}

func (e *edgeWorkload) Get(name string) (*v1alpha1.EdgeWorkload, error) {
	return e.workload.EdgeWorkloads(Namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (e *edgeWorkload) Create(ew *v1alpha1.EdgeWorkload) (*v1alpha1.EdgeWorkload, error) {
	return e.workload.EdgeWorkloads(Namespace).Create(context.TODO(), ew, metav1.CreateOptions{})
}

func (e *edgeWorkload) RemoveAll() error {
	return e.workload.EdgeWorkloads(Namespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
}

func (e *edgeWorkload) Remove(name string) error {
	err := e.workload.EdgeWorkloads(Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return e.waitForWorkload(func() bool {
		if _, err := e.Get(name); err != nil {
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

	return fmt.Errorf("error waiting for edgeworkload")
}
