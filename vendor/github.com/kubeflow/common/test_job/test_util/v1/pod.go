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

package v1

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	testjobv1 "github.com/kubeflow/common/test_job/apis/test_job/v1"
)

const (
	// labels for pods and servers.
	testReplicaTypeLabel  = "test-replica-type"
	testReplicaIndexLabel = "test-replica-index"
)

var (
	controllerKind = testjobv1.SchemeGroupVersionKind
)

func NewBasePod(name string, testJob *testjobv1.TestJob, t *testing.T) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          GenLabels(testJob.Name),
			Namespace:       testJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(testJob, controllerKind)},
		},
	}
}

func NewPod(testJob *testjobv1.TestJob, typ string, index int, t *testing.T) *v1.Pod {
	pod := NewBasePod(fmt.Sprintf("%s-%d", typ, index), testJob, t)
	pod.Labels[testReplicaTypeLabel] = typ
	pod.Labels[testReplicaIndexLabel] = fmt.Sprintf("%d", index)
	return pod
}

// create count pods with the given phase for the given testjob
func NewPodList(count int32, status v1.PodPhase, testJob *testjobv1.TestJob, typ string, start int32, t *testing.T) []*v1.Pod {
	pods := []*v1.Pod{}
	for i := int32(0); i < count; i++ {
		newPod := NewPod(testJob, typ, int(start+i), t)
		newPod.Status = v1.PodStatus{Phase: status}
		pods = append(pods, newPod)
	}
	return pods
}

func SetPodsStatuses(podIndexer cache.Indexer, testJob *testjobv1.TestJob, typ string, pendingPods, activePods, succeededPods, failedPods int32, restartCounts []int32, t *testing.T) {
	var index int32
	for _, pod := range NewPodList(pendingPods, v1.PodPending, testJob, typ, index, t) {
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", testJob.Name, err)
		}
	}
	index += pendingPods
	for i, pod := range NewPodList(activePods, v1.PodRunning, testJob, typ, index, t) {
		if restartCounts != nil {
			pod.Status.ContainerStatuses = []v1.ContainerStatus{{RestartCount: restartCounts[i]}}
		}
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", testJob.Name, err)
		}
	}
	index += activePods
	for _, pod := range NewPodList(succeededPods, v1.PodSucceeded, testJob, typ, index, t) {
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", testJob.Name, err)
		}
	}
	index += succeededPods
	for _, pod := range NewPodList(failedPods, v1.PodFailed, testJob, typ, index, t) {
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", testJob.Name, err)
		}
	}
}
