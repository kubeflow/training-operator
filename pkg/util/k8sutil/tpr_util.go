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
	"k8s.io/client-go/rest"
)

// TODO: replace this package with Operator client

// TfJobTPRUpdateFunc is a function to be used when atomically
// updating a Cluster TPR.
type TfJobTPRUpdateFunc func(*spec.TfJob)

func WatchClusters(host, ns string, httpClient *http.Client, resourceVersion string) (*http.Response, error) {
	return httpClient.Get(fmt.Sprintf("%s/apis/%s/%s/namespaces/%s/%s?watch=true&resourceVersion=%s",
		host, spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural, resourceVersion))
}

func GetTfJobsList(restcli rest.Interface, ns string) (*spec.TfJobList, error) {
	b, err := restcli.Get().RequestURI(listTfJobsURI(ns)).DoRaw()
	if err != nil {
		return nil, err
	}

	clusters := &spec.TfJobList{}
	if err := json.Unmarshal(b, clusters); err != nil {
		return nil, err
	}
	return clusters, nil
}

// WaitTfJobTPRReady blocks until the TPR is ready.
// Readiness is determined based on when we can list the resources.
func WaitTfJobTPRReady(restcli rest.Interface, interval, timeout time.Duration, ns string) error {
	return retryutil.Retry(interval, int(timeout/interval), func() (bool, error) {
		_, err := restcli.Get().RequestURI(listTfJobsURI(ns)).DoRaw()
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

func GetClusterTPRObject(restcli rest.Interface, ns, name string) (*spec.TfJob, error) {
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s", spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural, name)
	b, err := restcli.Get().RequestURI(uri).DoRaw()
	if err != nil {
		return nil, err
	}
	return readOutTfJob(b)
}

// AtomicUpdateClusterTPRObject will get the latest result of a TfJob,
// let user modify it, and update the cluster with modified result
// The entire process would be retried if there is a conflict of resource version
//
// TODO(jlewi): This function is from the etcd-operator. Is it still useful?
func AtomicUpdateClusterTPRObject(restcli rest.Interface, name, namespace string, maxRetries int, updateFunc TfJobTPRUpdateFunc) (*spec.TfJob, error) {
	var updatedCluster *spec.TfJob
	err := retryutil.Retry(1*time.Second, maxRetries, func() (done bool, err error) {
		currCluster, err := GetClusterTPRObject(restcli, namespace, name)
		if err != nil {
			return false, err
		}

		updateFunc(currCluster)

		updatedCluster, err = UpdateTfJobTPRObject(restcli, namespace, currCluster)
		if err != nil {
			if apierrors.IsConflict(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	return updatedCluster, err
}

func UpdateTfJobTPRObject(restcli rest.Interface, ns string, c *spec.TfJob) (*spec.TfJob, error) {
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s", spec.TPRGroup, spec.TPRVersion, ns, spec.TPRKindPlural, c.Metadata.Name)
	b, err := restcli.Put().RequestURI(uri).Body(c).DoRaw()
	if err != nil {
		return nil, err
	}
	return readOutTfJob(b)
}

func readOutTfJob(b []byte) (*spec.TfJob, error) {
	cluster := &spec.TfJob{}
	if err := json.Unmarshal(b, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}
