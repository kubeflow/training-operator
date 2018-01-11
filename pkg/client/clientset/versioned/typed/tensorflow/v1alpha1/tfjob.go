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

package v1alpha1

import (
	v1alpha1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	scheme "github.com/tensorflow/k8s/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// TfJobsGetter has a method to return a TfJobInterface.
// A group's client should implement this interface.
type TfJobsGetter interface {
	TfJobs(namespace string) TfJobInterface
}

// TfJobInterface has methods to work with TfJob resources.
type TfJobInterface interface {
	Create(*v1alpha1.TfJob) (*v1alpha1.TfJob, error)
	Update(*v1alpha1.TfJob) (*v1alpha1.TfJob, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.TfJob, error)
	List(opts v1.ListOptions) (*v1alpha1.TfJobList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.TfJob, err error)
	TfJobExpansion
}

// tfJobs implements TfJobInterface
type tfJobs struct {
	client rest.Interface
	ns     string
}

// newTfJobs returns a TfJobs
func newTfJobs(c *TensorflowV1alpha1Client, namespace string) *tfJobs {
	return &tfJobs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the tfJob, and returns the corresponding tfJob object, and an error if there is any.
func (c *tfJobs) Get(name string, options v1.GetOptions) (result *v1alpha1.TfJob, err error) {
	result = &v1alpha1.TfJob{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tfjobs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of TfJobs that match those selectors.
func (c *tfJobs) List(opts v1.ListOptions) (result *v1alpha1.TfJobList, err error) {
	result = &v1alpha1.TfJobList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tfjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested tfJobs.
func (c *tfJobs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("tfjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a tfJob and creates it.  Returns the server's representation of the tfJob, and an error, if there is any.
func (c *tfJobs) Create(tfJob *v1alpha1.TfJob) (result *v1alpha1.TfJob, err error) {
	result = &v1alpha1.TfJob{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("tfjobs").
		Body(tfJob).
		Do().
		Into(result)
	return
}

// Update takes the representation of a tfJob and updates it. Returns the server's representation of the tfJob, and an error, if there is any.
func (c *tfJobs) Update(tfJob *v1alpha1.TfJob) (result *v1alpha1.TfJob, err error) {
	result = &v1alpha1.TfJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("tfjobs").
		Name(tfJob.Name).
		Body(tfJob).
		Do().
		Into(result)
	return
}

// Delete takes name of the tfJob and deletes it. Returns an error if one occurs.
func (c *tfJobs) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tfjobs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *tfJobs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tfjobs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched tfJob.
func (c *tfJobs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.TfJob, err error) {
	result = &v1alpha1.TfJob{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("tfjobs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
