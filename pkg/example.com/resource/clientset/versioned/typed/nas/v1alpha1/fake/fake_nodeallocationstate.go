/*
 * Copyright 2023 The Kubernetes Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/gpu/nas/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNodeAllocationStates implements NodeAllocationStateInterface
type FakeNodeAllocationStates struct {
	Fake *FakeNasV1alpha1
	ns   string
}

var nodeallocationstatesResource = v1alpha1.SchemeGroupVersion.WithResource("nodeallocationstates")

var nodeallocationstatesKind = v1alpha1.SchemeGroupVersion.WithKind("NodeAllocationState")

// Get takes name of the nodeAllocationState, and returns the corresponding nodeAllocationState object, and an error if there is any.
func (c *FakeNodeAllocationStates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NodeAllocationState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(nodeallocationstatesResource, c.ns, name), &v1alpha1.NodeAllocationState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeAllocationState), err
}

// List takes label and field selectors, and returns the list of NodeAllocationStates that match those selectors.
func (c *FakeNodeAllocationStates) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NodeAllocationStateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(nodeallocationstatesResource, nodeallocationstatesKind, c.ns, opts), &v1alpha1.NodeAllocationStateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.NodeAllocationStateList{ListMeta: obj.(*v1alpha1.NodeAllocationStateList).ListMeta}
	for _, item := range obj.(*v1alpha1.NodeAllocationStateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested nodeAllocationStates.
func (c *FakeNodeAllocationStates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(nodeallocationstatesResource, c.ns, opts))

}

// Create takes the representation of a nodeAllocationState and creates it.  Returns the server's representation of the nodeAllocationState, and an error, if there is any.
func (c *FakeNodeAllocationStates) Create(ctx context.Context, nodeAllocationState *v1alpha1.NodeAllocationState, opts v1.CreateOptions) (result *v1alpha1.NodeAllocationState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(nodeallocationstatesResource, c.ns, nodeAllocationState), &v1alpha1.NodeAllocationState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeAllocationState), err
}

// Update takes the representation of a nodeAllocationState and updates it. Returns the server's representation of the nodeAllocationState, and an error, if there is any.
func (c *FakeNodeAllocationStates) Update(ctx context.Context, nodeAllocationState *v1alpha1.NodeAllocationState, opts v1.UpdateOptions) (result *v1alpha1.NodeAllocationState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(nodeallocationstatesResource, c.ns, nodeAllocationState), &v1alpha1.NodeAllocationState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeAllocationState), err
}

// Delete takes name of the nodeAllocationState and deletes it. Returns an error if one occurs.
func (c *FakeNodeAllocationStates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(nodeallocationstatesResource, c.ns, name, opts), &v1alpha1.NodeAllocationState{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNodeAllocationStates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(nodeallocationstatesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.NodeAllocationStateList{})
	return err
}

// Patch applies the patch and returns the patched nodeAllocationState.
func (c *FakeNodeAllocationStates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NodeAllocationState, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(nodeallocationstatesResource, c.ns, name, pt, data, subresources...), &v1alpha1.NodeAllocationState{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeAllocationState), err
}
