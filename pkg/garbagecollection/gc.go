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

package garbagecollection

import (
	"github.com/jlewi/mlkube.io/pkg/spec"
	"github.com/jlewi/mlkube.io/pkg/util/k8sutil"

	log "github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	NullUID = ""
)

type GC struct {
	kubecli     kubernetes.Interface
	tfJobClient k8sutil.TfJobClient
	ns          string
}

func New(kubecli kubernetes.Interface, tfJobClient k8sutil.TfJobClient, ns string) *GC {
	return &GC{
		kubecli:     kubecli,
		tfJobClient: tfJobClient,
		ns:          ns,
	}
}

// CollectJob collects resources that matches job label, but
// does not belong to the job with given jobUID
func (gc *GC) CollectJob(job string, jobUID types.UID) {
	gc.collectResources(k8sutil.JobListOpt(job), map[types.UID]bool{jobUID: true})
}

// FullyCollect collects resources that were created before,
// but does not belong to any current running clusters.
func (gc *GC) FullyCollect() error {
	jobs, err := gc.tfJobClient.List(gc.ns)
	if err != nil {
		return err
	}

	clusterUIDSet := make(map[types.UID]bool)
	for _, c := range jobs.Items {
		clusterUIDSet[c.Metadata.UID] = true
	}

	option := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app": spec.AppLabel,
		}).String(),
	}

	gc.collectResources(option, clusterUIDSet)
	return nil
}

func (gc *GC) collectResources(option metav1.ListOptions, runningSet map[types.UID]bool) {
	if err := gc.collectPods(option, runningSet); err != nil {
		log.Errorf("gc pods failed: %v", err)
	}
	if err := gc.collectServices(option, runningSet); err != nil {
		log.Errorf("gc services failed: %v", err)
	}
	if err := gc.collectDeployment(option, runningSet); err != nil {
		log.Errorf("gc deployments failed: %v", err)
	}
}

func (gc *GC) collectPods(option metav1.ListOptions, runningSet map[types.UID]bool) error {
	pods, err := gc.kubecli.CoreV1().Pods(gc.ns).List(option)
	if err != nil {
		return err
	}

	for _, p := range pods.Items {
		if len(p.OwnerReferences) == 0 {
			log.Warningf("failed to check pod %s: no owner", p.GetName())
			continue
		}
		// Pods failed due to liveness probe are also collected
		if !runningSet[p.OwnerReferences[0].UID] || p.Status.Phase == v1.PodFailed {
			// kill bad pods without grace period to kill it immediately
			err = gc.kubecli.CoreV1().Pods(gc.ns).Delete(p.GetName(), metav1.NewDeleteOptions(0))
			if err != nil && !k8sutil.IsKubernetesResourceNotFoundError(err) {
				return err
			}
			log.Infof("deleted pod (%v)", p.GetName())
		}
	}
	return nil
}

func (gc *GC) collectServices(option metav1.ListOptions, runningSet map[types.UID]bool) error {
	srvs, err := gc.kubecli.CoreV1().Services(gc.ns).List(option)
	if err != nil {
		return err
	}

	for _, srv := range srvs.Items {
		if len(srv.OwnerReferences) == 0 {
			log.Warningf("failed to check service %s: no owner", srv.GetName())
			continue
		}
		if !runningSet[srv.OwnerReferences[0].UID] {
			err = gc.kubecli.CoreV1().Services(gc.ns).Delete(srv.GetName(), nil)
			if err != nil && !k8sutil.IsKubernetesResourceNotFoundError(err) {
				return err
			}
			log.Infof("deleted service (%v)", srv.GetName())
		}
	}

	return nil
}

func (gc *GC) collectDeployment(option metav1.ListOptions, runningSet map[types.UID]bool) error {
	ds, err := gc.kubecli.AppsV1beta1().Deployments(gc.ns).List(option)
	if err != nil {
		return err
	}

	for _, d := range ds.Items {
		if len(d.OwnerReferences) == 0 {
			log.Warningf("failed to GC deployment (%s): no owner", d.GetName())
			continue
		}
		if !runningSet[d.OwnerReferences[0].UID] {
			err = gc.kubecli.AppsV1beta1().Deployments(gc.ns).Delete(d.GetName(), k8sutil.CascadeDeleteOptions(0))
			if err != nil {
				if !k8sutil.IsKubernetesResourceNotFoundError(err) {
					return err
				}
			}
			log.Infof("deleted deployment (%s)", d.GetName())
		}
	}

	return nil
}
