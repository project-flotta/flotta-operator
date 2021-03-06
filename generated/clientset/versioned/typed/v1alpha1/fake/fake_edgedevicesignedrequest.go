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
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeEdgeDeviceSignedRequests implements EdgeDeviceSignedRequestInterface
type FakeEdgeDeviceSignedRequests struct {
	Fake *FakeManagementV1alpha1
	ns   string
}

var edgedevicesignedrequestsResource = schema.GroupVersionResource{Group: "management.project-flotta.io", Version: "v1alpha1", Resource: "edgedevicesignedrequests"}

var edgedevicesignedrequestsKind = schema.GroupVersionKind{Group: "management.project-flotta.io", Version: "v1alpha1", Kind: "EdgeDeviceSignedRequest"}

// Get takes name of the edgeDeviceSignedRequest, and returns the corresponding edgeDeviceSignedRequest object, and an error if there is any.
func (c *FakeEdgeDeviceSignedRequests) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.EdgeDeviceSignedRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(edgedevicesignedrequestsResource, c.ns, name), &v1alpha1.EdgeDeviceSignedRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EdgeDeviceSignedRequest), err
}

// List takes label and field selectors, and returns the list of EdgeDeviceSignedRequests that match those selectors.
func (c *FakeEdgeDeviceSignedRequests) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.EdgeDeviceSignedRequestList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(edgedevicesignedrequestsResource, edgedevicesignedrequestsKind, c.ns, opts), &v1alpha1.EdgeDeviceSignedRequestList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.EdgeDeviceSignedRequestList{ListMeta: obj.(*v1alpha1.EdgeDeviceSignedRequestList).ListMeta}
	for _, item := range obj.(*v1alpha1.EdgeDeviceSignedRequestList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested edgeDeviceSignedRequests.
func (c *FakeEdgeDeviceSignedRequests) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(edgedevicesignedrequestsResource, c.ns, opts))

}

// Create takes the representation of a edgeDeviceSignedRequest and creates it.  Returns the server's representation of the edgeDeviceSignedRequest, and an error, if there is any.
func (c *FakeEdgeDeviceSignedRequests) Create(ctx context.Context, edgeDeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, opts v1.CreateOptions) (result *v1alpha1.EdgeDeviceSignedRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(edgedevicesignedrequestsResource, c.ns, edgeDeviceSignedRequest), &v1alpha1.EdgeDeviceSignedRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EdgeDeviceSignedRequest), err
}

// Update takes the representation of a edgeDeviceSignedRequest and updates it. Returns the server's representation of the edgeDeviceSignedRequest, and an error, if there is any.
func (c *FakeEdgeDeviceSignedRequests) Update(ctx context.Context, edgeDeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, opts v1.UpdateOptions) (result *v1alpha1.EdgeDeviceSignedRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(edgedevicesignedrequestsResource, c.ns, edgeDeviceSignedRequest), &v1alpha1.EdgeDeviceSignedRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EdgeDeviceSignedRequest), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeEdgeDeviceSignedRequests) UpdateStatus(ctx context.Context, edgeDeviceSignedRequest *v1alpha1.EdgeDeviceSignedRequest, opts v1.UpdateOptions) (*v1alpha1.EdgeDeviceSignedRequest, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(edgedevicesignedrequestsResource, "status", c.ns, edgeDeviceSignedRequest), &v1alpha1.EdgeDeviceSignedRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EdgeDeviceSignedRequest), err
}

// Delete takes name of the edgeDeviceSignedRequest and deletes it. Returns an error if one occurs.
func (c *FakeEdgeDeviceSignedRequests) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(edgedevicesignedrequestsResource, c.ns, name, opts), &v1alpha1.EdgeDeviceSignedRequest{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeEdgeDeviceSignedRequests) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(edgedevicesignedrequestsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.EdgeDeviceSignedRequestList{})
	return err
}

// Patch applies the patch and returns the patched edgeDeviceSignedRequest.
func (c *FakeEdgeDeviceSignedRequests) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.EdgeDeviceSignedRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(edgedevicesignedrequestsResource, c.ns, name, pt, data, subresources...), &v1alpha1.EdgeDeviceSignedRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EdgeDeviceSignedRequest), err
}
