/*
Copyright 2018 The Kubernetes Authors.

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

package fake

import (
	v1alpha1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTfJobs implements TfJobInterface
type FakeTfJobs struct {
	Fake *FakeTensorflowV1alpha1
	ns   string
}

var tfjobsResource = schema.GroupVersionResource{Group: "tensorflow.org", Version: "v1alpha1", Resource: "tfjobs"}

var tfjobsKind = schema.GroupVersionKind{Group: "tensorflow.org", Version: "v1alpha1", Kind: "TfJob"}

// Get takes name of the tfJob, and returns the corresponding tfJob object, and an error if there is any.
func (c *FakeTfJobs) Get(name string, options v1.GetOptions) (result *v1alpha1.TfJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(tfjobsResource, c.ns, name), &v1alpha1.TfJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TfJob), err
}

// List takes label and field selectors, and returns the list of TfJobs that match those selectors.
func (c *FakeTfJobs) List(opts v1.ListOptions) (result *v1alpha1.TfJobList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(tfjobsResource, tfjobsKind, c.ns, opts), &v1alpha1.TfJobList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.TfJobList{}
	for _, item := range obj.(*v1alpha1.TfJobList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tfJobs.
func (c *FakeTfJobs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(tfjobsResource, c.ns, opts))

}

// Create takes the representation of a tfJob and creates it.  Returns the server's representation of the tfJob, and an error, if there is any.
func (c *FakeTfJobs) Create(tfJob *v1alpha1.TfJob) (result *v1alpha1.TfJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(tfjobsResource, c.ns, tfJob), &v1alpha1.TfJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TfJob), err
}

// Update takes the representation of a tfJob and updates it. Returns the server's representation of the tfJob, and an error, if there is any.
func (c *FakeTfJobs) Update(tfJob *v1alpha1.TfJob) (result *v1alpha1.TfJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(tfjobsResource, c.ns, tfJob), &v1alpha1.TfJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TfJob), err
}

// Delete takes name of the tfJob and deletes it. Returns an error if one occurs.
func (c *FakeTfJobs) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(tfjobsResource, c.ns, name), &v1alpha1.TfJob{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTfJobs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(tfjobsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.TfJobList{})
	return err
}

// Patch applies the patch and returns the patched tfJob.
func (c *FakeTfJobs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.TfJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(tfjobsResource, c.ns, name, data, subresources...), &v1alpha1.TfJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TfJob), err
}
