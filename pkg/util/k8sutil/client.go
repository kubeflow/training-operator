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
	"fmt"
	"net/http"

	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// CRDRestClient defines an interface for working with CRDs using the REST client.
// In most cases we want to use the auto-generated clientset for specific CRDs.
// The only exception is when the CRD spec is invalid and we can't parse the type into the corresponding
// go struct.
type CRDClient interface {
	// Update a TfJob.
	Update(obj *metav1unstructured.Unstructured) error
}

// CRDRestClient uses the Kubernetes rest interface to talk to the CRD.
type CRDRestClient struct {
	restcli *rest.RESTClient
}

func NewCRDRestClient(version *schema.GroupVersion) (*CRDRestClient, error) {
	config, err := GetClusterConfig()
	if err != nil {
		return nil, err
	}
	config.GroupVersion = version
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	restcli, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	cli := &CRDRestClient{
		restcli: restcli,
	}
	return cli, nil
}

// HttpClient returns the http client used.
func (c *CRDRestClient) Client() *http.Client {
	return c.restcli.Client
}

func (c *CRDRestClient) Update(obj *metav1unstructured.Unstructured, plural string) error {
	logger := tflogger.LoggerForUnstructured(obj, obj.GetKind())
	// TODO(jlewi): Can we just call obj.GetKind() to get the kind? I think that will return the singular
	// not plural will that work?
	if plural == "" {
		logger.Errorf("Could not issue update because plural not set.")
		return fmt.Errorf("plural must be set")
	}
	r := c.restcli.Put().Resource(plural).Namespace(obj.GetNamespace()).Name(obj.GetName()).Body(obj)
	_, err := r.DoRaw()
	if err != nil {
		logger.Errorf("Could not issue update using URL: %v; error; %v", r.URL().String(), err)
	}
	return err
}

func (c *CRDRestClient) UpdateStatus(obj *metav1unstructured.Unstructured, plural string) error {
	logger := tflogger.LoggerForUnstructured(obj, obj.GetKind())
	if plural == "" {
		logger.Errorf("Could not issue update because plural not set.")
		return fmt.Errorf("plural must be set")
	}
	r := c.restcli.Put().Resource(plural).Namespace(obj.GetNamespace()).Name(obj.GetName()).SubResource("status").Body(obj)
	_, err := r.DoRaw()
	if err != nil {
		logger.Errorf("Could not issue update using URL: %v; error; %v", r.URL().String(), err)
	}
	return err
}
