// fake provides a fake implementation of TfJobClient suitable for use in testing.
package fake

import (
	"github.com/tensorflow/k8s/pkg/spec"
	"net/http"
	"time"
)

type TfJobClientFake struct{}

func (c *TfJobClientFake) Get(ns string, name string) (*spec.TfJob, error) {
	return &spec.TfJob{}, nil
}

func (c *TfJobClientFake) Create(ns string, j *spec.TfJob) (*spec.TfJob, error) {
	return &spec.TfJob{}, nil
}

func (c *TfJobClientFake) Delete(ns string, name string) error {
	return nil
}

func (c *TfJobClientFake) List(ns string) (*spec.TfJobList, error) {
	return &spec.TfJobList{}, nil
}

// Update a TfJob.
func (c *TfJobClientFake) Update(ns string, j *spec.TfJob) (*spec.TfJob, error) {
	// TODO(jlewi): We should return a deep copy of j.
	result := *j
	return &result, nil
}

// Watch TfJobs.
func (c *TfJobClientFake) Watch(host, ns string, httpClient *http.Client, resourceVersion string) (*http.Response, error) {
	return nil, nil
}

// WaitTPRReady blocks until the TfJob TPR is ready.
func (c *TfJobClientFake) WaitTPRReady(interval, timeout time.Duration, ns string) error {
	return nil
}
