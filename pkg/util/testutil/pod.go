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

package testutil

import (
	"fmt"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/generator"
)

const (
	// labels for pods and servers.
	tfReplicaTypeLabel  = "tf-replica-type"
	tfReplicaIndexLabel = "tf-replica-index"
)

var (
	controllerKind = tfv1alpha2.SchemeGroupVersionKind
)

func NewBasePod(name string, tfJob *tfv1alpha2.TFJob, t *testing.T) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          generator.GenLabels(tfJob.Name),
			Namespace:       tfJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(tfJob, controllerKind)},
		},
	}
}

func NewPod(tfJob *tfv1alpha2.TFJob, typ string, index int, t *testing.T) *v1.Pod {
	pod := NewBasePod(fmt.Sprintf("%s-%d", typ, index), tfJob, t)
	pod.Labels[tfReplicaTypeLabel] = typ
	pod.Labels[tfReplicaIndexLabel] = fmt.Sprintf("%d", index)
	return pod
}

// create count pods with the given phase for the given tfJob
func NewPodList(count int32, status v1.PodPhase, tfJob *tfv1alpha2.TFJob, typ string, start int32, t *testing.T) []*v1.Pod {
	pods := []*v1.Pod{}
	for i := int32(0); i < count; i++ {
		newPod := NewPod(tfJob, typ, int(start+i), t)
		newPod.Status = v1.PodStatus{Phase: status}
		pods = append(pods, newPod)
	}
	return pods
}

func SetPodsStatuses(podIndexer cache.Indexer, tfJob *tfv1alpha2.TFJob, typ string, pendingPods, activePods, succeededPods, failedPods int32, t *testing.T) {
	var index int32
	for _, pod := range NewPodList(pendingPods, v1.PodPending, tfJob, typ, index, t) {
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
		}
	}
	index += pendingPods
	for _, pod := range NewPodList(activePods, v1.PodRunning, tfJob, typ, index, t) {
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
		}
	}
	index += activePods
	for _, pod := range NewPodList(succeededPods, v1.PodSucceeded, tfJob, typ, index, t) {
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
		}
	}
	index += succeededPods
	for _, pod := range NewPodList(failedPods, v1.PodFailed, tfJob, typ, index, t) {
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
		}
	}
}
