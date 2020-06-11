// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sutil

import (
	"net"
	"os"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // for gcp auth
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RecommendedConfigPathEnvVar is a environment variable for path configuration
const RecommendedConfigPathEnvVar = "KUBECONFIG"

// MustNewKubeClient returns new kubernetes client for cluster configuration
func MustNewKubeClient() kubernetes.Interface {
	cfg, err := GetClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	return kubernetes.NewForConfigOrDie(cfg)
}

// GetClusterConfig obtain the config from the Kube configuration used by kubeconfig, or from k8s cluster.
func GetClusterConfig() (*rest.Config, error) {
	if len(os.Getenv(RecommendedConfigPathEnvVar)) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		return clientcmd.BuildConfigFromFlags("", os.Getenv(RecommendedConfigPathEnvVar))
	}

	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		if err := os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0]); err != nil {
			return nil, err
		}
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		if err := os.Setenv("KUBERNETES_SERVICE_PORT", "443"); err != nil {
			panic(err)
		}
	}
	return rest.InClusterConfig()
}

// IsKubernetesResourceAlreadyExistError throws error when kubernetes resources already exist.
func IsKubernetesResourceAlreadyExistError(err error) bool {
	return apierrors.IsAlreadyExists(err)
}

// IsKubernetesResourceNotFoundError throws error when there is no kubernetes resource found.
func IsKubernetesResourceNotFoundError(err error) bool {
	return apierrors.IsNotFound(err)
}

// TODO(jlewi): CascadeDeletOptions are part of garbage collection policy.
// CascadeDeleteOptions deletes the workload after the grace period
// Do we want to use this? See
// https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
func CascadeDeleteOptions(gracePeriodSeconds int64) *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		GracePeriodSeconds: func(t int64) *int64 { return &t }(gracePeriodSeconds),
		PropagationPolicy: func() *metav1.DeletionPropagation {
			foreground := metav1.DeletePropagationForeground
			return &foreground
		}(),
	}
}

// FilterActivePods returns pods that have not terminated.
func FilterActivePods(pods []*v1.Pod) []*v1.Pod {
	var result []*v1.Pod
	for _, p := range pods {
		if IsPodActive(p) {
			result = append(result, p)
		} else {
			log.Infof("Ignoring inactive pod %v/%v in state %v, deletion time %v",
				p.Namespace, p.Name, p.Status.Phase, p.DeletionTimestamp)
		}
	}
	return result
}

func IsPodActive(p *v1.Pod) bool {
	return v1.PodSucceeded != p.Status.Phase &&
		v1.PodFailed != p.Status.Phase &&
		p.DeletionTimestamp == nil
}

// filterPodCount returns pods based on their phase.
func FilterPodCount(pods []*v1.Pod, phase v1.PodPhase) int32 {
	var result int32
	for i := range pods {
		if phase == pods[i].Status.Phase {
			result++
		}
	}
	return result
}

func GetTotalReplicas(replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) int32 {
	jobReplicas := int32(0)
	for _, r := range replicas {
		if r.Replicas != nil {
			jobReplicas += *r.Replicas
		} else {
			// If unspecified, defaults to 1.
			jobReplicas += 1
		}
	}
	return jobReplicas
}

func GetTotalFailedReplicas(replicas map[apiv1.ReplicaType]*apiv1.ReplicaStatus) int32 {
	totalFailedReplicas := int32(0)
	for _, status := range replicas {
		totalFailedReplicas += status.Failed
	}
	return totalFailedReplicas
}
