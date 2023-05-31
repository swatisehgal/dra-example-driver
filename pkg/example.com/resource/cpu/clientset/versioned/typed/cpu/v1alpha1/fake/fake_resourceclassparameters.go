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

	v1alpha1 "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeResourceClassParameterses implements ResourceClassParametersInterface
type FakeResourceClassParameterses struct {
	Fake *FakeCpuV1alpha1
}

var resourceclassparametersesResource = schema.GroupVersionResource{Group: "cpu.resource.example.com", Version: "v1alpha1", Resource: "resourceclassparameterses"}

var resourceclassparametersesKind = schema.GroupVersionKind{Group: "cpu.resource.example.com", Version: "v1alpha1", Kind: "ResourceClassParameters"}

// Get takes name of the resourceClassParameters, and returns the corresponding resourceClassParameters object, and an error if there is any.
func (c *FakeResourceClassParameterses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ResourceClassParameters, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(resourceclassparametersesResource, name), &v1alpha1.ResourceClassParameters{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ResourceClassParameters), err
}

// List takes label and field selectors, and returns the list of ResourceClassParameterses that match those selectors.
func (c *FakeResourceClassParameterses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ResourceClassParametersList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(resourceclassparametersesResource, resourceclassparametersesKind, opts), &v1alpha1.ResourceClassParametersList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ResourceClassParametersList{ListMeta: obj.(*v1alpha1.ResourceClassParametersList).ListMeta}
	for _, item := range obj.(*v1alpha1.ResourceClassParametersList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested resourceClassParameterses.
func (c *FakeResourceClassParameterses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(resourceclassparametersesResource, opts))
}

// Create takes the representation of a resourceClassParameters and creates it.  Returns the server's representation of the resourceClassParameters, and an error, if there is any.
func (c *FakeResourceClassParameterses) Create(ctx context.Context, resourceClassParameters *v1alpha1.ResourceClassParameters, opts v1.CreateOptions) (result *v1alpha1.ResourceClassParameters, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(resourceclassparametersesResource, resourceClassParameters), &v1alpha1.ResourceClassParameters{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ResourceClassParameters), err
}

// Update takes the representation of a resourceClassParameters and updates it. Returns the server's representation of the resourceClassParameters, and an error, if there is any.
func (c *FakeResourceClassParameterses) Update(ctx context.Context, resourceClassParameters *v1alpha1.ResourceClassParameters, opts v1.UpdateOptions) (result *v1alpha1.ResourceClassParameters, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(resourceclassparametersesResource, resourceClassParameters), &v1alpha1.ResourceClassParameters{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ResourceClassParameters), err
}

// Delete takes name of the resourceClassParameters and deletes it. Returns an error if one occurs.
func (c *FakeResourceClassParameterses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(resourceclassparametersesResource, name), &v1alpha1.ResourceClassParameters{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeResourceClassParameterses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(resourceclassparametersesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ResourceClassParametersList{})
	return err
}

// Patch applies the patch and returns the patched resourceClassParameters.
func (c *FakeResourceClassParameterses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ResourceClassParameters, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(resourceclassparametersesResource, name, pt, data, subresources...), &v1alpha1.ResourceClassParameters{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ResourceClassParameters), err
}
