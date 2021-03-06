/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// EdgeDeviceLister helps list EdgeDevices.
// All objects returned here must be treated as read-only.
type EdgeDeviceLister interface {
	// List lists all EdgeDevices in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.EdgeDevice, err error)
	// EdgeDevices returns an object that can list and get EdgeDevices.
	EdgeDevices(namespace string) EdgeDeviceNamespaceLister
	EdgeDeviceListerExpansion
}

// edgeDeviceLister implements the EdgeDeviceLister interface.
type edgeDeviceLister struct {
	indexer cache.Indexer
}

// NewEdgeDeviceLister returns a new EdgeDeviceLister.
func NewEdgeDeviceLister(indexer cache.Indexer) EdgeDeviceLister {
	return &edgeDeviceLister{indexer: indexer}
}

// List lists all EdgeDevices in the indexer.
func (s *edgeDeviceLister) List(selector labels.Selector) (ret []*v1alpha1.EdgeDevice, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.EdgeDevice))
	})
	return ret, err
}

// EdgeDevices returns an object that can list and get EdgeDevices.
func (s *edgeDeviceLister) EdgeDevices(namespace string) EdgeDeviceNamespaceLister {
	return edgeDeviceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// EdgeDeviceNamespaceLister helps list and get EdgeDevices.
// All objects returned here must be treated as read-only.
type EdgeDeviceNamespaceLister interface {
	// List lists all EdgeDevices in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.EdgeDevice, err error)
	// Get retrieves the EdgeDevice from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.EdgeDevice, error)
	EdgeDeviceNamespaceListerExpansion
}

// edgeDeviceNamespaceLister implements the EdgeDeviceNamespaceLister
// interface.
type edgeDeviceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all EdgeDevices in the indexer for a given namespace.
func (s edgeDeviceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.EdgeDevice, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.EdgeDevice))
	})
	return ret, err
}

// Get retrieves the EdgeDevice from the indexer for a given namespace and name.
func (s edgeDeviceNamespaceLister) Get(name string) (*v1alpha1.EdgeDevice, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("edgedevice"), name)
	}
	return obj.(*v1alpha1.EdgeDevice), nil
}
