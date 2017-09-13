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

// TODO(jlewi): We should rename this file to reflect the fact that we are using CRDs and not TPRs.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/deepinsight/mlkube.io/pkg/spec"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
	"github.com/deepinsight/mlkube.io/pkg/util"
	log "github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TFJobClient defines an interface for working with TfJob CRDs.
type TfJobClient interface {
	// Get returns a TfJob
	Get(ns string, name string) (*spec.TfJob, error)

	// List returns a list of TfJobs
	List(ns string) (*spec.TfJobList, error)

	// Update a TfJob.
	Update(ns string, c *spec.TfJob) (*spec.TfJob, error)

	// Watch TfJobs.
	Watch(host, ns string, httpClient *http.Client, resourceVersion string) (*http.Response, error)
}

// TfJobRestClient uses the Kubernetes rest interface to talk to the CRD.
type TfJobRestClient struct {
	restcli *rest.RESTClient
}

func NewTfJobClient() (*TfJobRestClient, error) {
	config, err := InClusterConfig()
	if err != nil {
		return nil, err
	}
	config.GroupVersion = &spec.SchemeGroupVersion
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

// New TFJob client for out-of-cluster
func NewTfJobClientExternal(config *rest.Config) (*TfJobRestClient, error) {

	config.GroupVersion = &schema.GroupVersion{
		Group:   spec.CRDGroup,
		Version: spec.CRDVersion,
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
	return c.restcli.Client.Get(fmt.Sprintf("%s/apis/%s/%s/%s?watch=true&resourceVersion=%s",
		host, spec.CRDGroup, spec.CRDVersion, spec.CRDKindPlural, resourceVersion))
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

func listTfJobsURI(ns string) string {
	return fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s", spec.CRDGroup, spec.CRDVersion, ns, spec.CRDKindPlural)
}

func (c *TfJobRestClient) Create(ns string, j *spec.TfJob) (*spec.TfJob, error) {
	// Set the TypeMeta or we will get a BadRequest
	j.TypeMeta.APIVersion = fmt.Sprintf("%v/%v", spec.CRDGroup, spec.CRDVersion)
	j.TypeMeta.Kind = spec.CRDKind
	b, err := c.restcli.Post().Resource(spec.CRDKindPlural).Namespace(ns).Body(j).DoRaw()
	if err != nil {
		log.Errorf("Creating the TfJob:\n%v\nError:\n%v", util.Pformat(j), util.Pformat(err))
		return nil, err
	}
	return readOutTfJob(b)
}

func (c *TfJobRestClient) Get(ns, name string) (*spec.TfJob, error) {
	b, err := c.restcli.Get().Resource(spec.CRDKindPlural).Namespace(ns).Name(name).DoRaw()
	if err != nil {
		return nil, err
	}
	return readOutTfJob(b)
}

func (c *TfJobRestClient) Update(ns string, j *spec.TfJob) (*spec.TfJob, error) {
	// Set the TypeMeta or we will get a BadRequest
	j.TypeMeta.APIVersion = fmt.Sprintf("%v/%v", spec.CRDGroup, spec.CRDVersion)
	j.TypeMeta.Kind = spec.CRDKind
	b, err := c.restcli.Put().Resource(spec.CRDKindPlural).Namespace(ns).Name(j.Metadata.Name).Body(j).DoRaw()
	if err != nil {
		return nil, err
	}
	return readOutTfJob(b)
}

func (c *TfJobRestClient) Delete(ns, name string) error {
	_, err := c.restcli.Delete().Resource(spec.CRDKindPlural).Namespace(ns).DoRaw()
	return err
}

func readOutTfJob(b []byte) (*spec.TfJob, error) {
	cluster := &spec.TfJob{}
	if err := json.Unmarshal(b, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}
