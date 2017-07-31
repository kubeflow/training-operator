// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mlkube.io/pkg/spec"
	"mlkube.io/pkg/util/retryutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
)

// TFJobClient defines an interface for working with TfJob TPRs.
type TfJobClient interface {
	// Get returns a TfJob
	Get(ns string, name string) (*spec.TfJob, error)

	// List returns a list of TfJobs
	List(ns string) (*spec.TfJobList, error)

	// Update a TfJob.
	Update(ns string, c *spec.TfJob) (*spec.TfJob, error)

	// Watch TfJobs.
	Watch(host, ns string, httpClient *http.Client, resourceVersion string) (*http.Response, error)

	// WaitTPRReady blocks until the TfJob TPR is ready.
	WaitTPRReady(interval, timeout time.Duration, ns string) error
}

// TfJobRestClient uses the Kubernetes rest interface to talk to the TPR.
type TfJobRestClient struct {
	restcli *rest.RESTClient
}

func NewTfJobClient() (*TfJobRestClient, error) {
	config, err := InClusterConfig()
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &schema.GroupVersion{
		Group:   spec.TPRGroup,
		Version: spec.TPRVersion,
	}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	restcli, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	cli := &TfJobRestClient{
		restcli: restcli,
	}
	return cli, nil
}

// HttpClient returns the http client used.
func (c *TfJobRestClient) Client() *http.Client {
	return c.restcli.Client
}

func (c *TfJobRestClient) Watch(host, ns string, httpClient *http.Client, resourceVersion string) (*http.Response, error) {
	return c.restcli.Client.Get(fmt.Sprintf("%s/apis/%s/%s/namespaces/%s/%s?watch=true&resourceVersion=%s",
		host, spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural, resourceVersion))
}

func (c *TfJobRestClient) List(ns string) (*spec.TfJobList, error) {
	b, err := c.restcli.Get().RequestURI(listTfJobsURI(ns)).DoRaw()
	if err != nil {
		return nil, err
	}

	jobs := &spec.TfJobList{}
	if err := json.Unmarshal(b, jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

// WaitTPRReady blocks until the TPR is ready.
// Readiness is determined based on when we can list the resources.
func (c *TfJobRestClient) WaitTPRReady(interval, timeout time.Duration, ns string) error {
	return retryutil.Retry(interval, int(timeout/interval), func() (bool, error) {
		_, err := c.restcli.Get().RequestURI(listTfJobsURI(ns)).DoRaw()
		if err != nil {
			if apierrors.IsNotFound(err) { // not set up yet. wait more.
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func listTfJobsURI(ns string) string {
	return fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s", spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural)
}

func (c *TfJobRestClient) Create(ns string, j *spec.TfJob) (*spec.TfJob, error) {
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/", spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural)
	b, err := c.restcli.Post().RequestURI(uri).Body(j).DoRaw()
	if err != nil {
		return nil, err
	}
	return readOutTfJob(b)
}

func (c *TfJobRestClient) Get(ns, name string) (*spec.TfJob, error) {
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s", spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural, name)
	b, err := c.restcli.Get().RequestURI(uri).DoRaw()
	if err != nil {
		return nil, err
	}
	return readOutTfJob(b)
}

func (c *TfJobRestClient) Update(ns string, j *spec.TfJob) (*spec.TfJob, error) {
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s", spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural, j.Metadata.Name)
	b, err := c.restcli.Put().RequestURI(uri).Body(j).DoRaw()
	if err != nil {
		return nil, err
	}
	return readOutTfJob(b)
}

func (c *TfJobRestClient) Delete(ns, name string) error {
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s", spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural, name)
	_, err := c.restcli.Delete().RequestURI(uri).DoRaw()
	return err
}

func readOutTfJob(b []byte) (*spec.TfJob, error) {
	cluster := &spec.TfJob{}
	if err := json.Unmarshal(b, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}
