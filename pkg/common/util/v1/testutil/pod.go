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
	"context"
	"fmt"
	"time"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DummyContainerName  = "dummy"
	DummyContainerImage = "dummy/dummy:latest"
)

func NewBasePod(name string, job metav1.Object, refs []metav1.OwnerReference) *corev1.Pod {

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          map[string]string{},
			Namespace:       job.GetNamespace(),
			OwnerReferences: refs,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  DummyContainerName,
					Image: DummyContainerImage,
				},
			},
		},
	}
}

func NewPod(job metav1.Object, typ string, index int, refs []metav1.OwnerReference) *corev1.Pod {
	pod := NewBasePod(fmt.Sprintf("%s-%s-%d", job.GetName(), typ, index), job, refs)
	pod.Labels[commonv1.ReplicaTypeLabelDeprecated] = typ
	pod.Labels[commonv1.ReplicaTypeLabel] = typ
	pod.Labels[commonv1.ReplicaIndexLabelDeprecated] = fmt.Sprintf("%d", index)
	pod.Labels[commonv1.ReplicaIndexLabel] = fmt.Sprintf("%d", index)
	return pod
}

// NewPodList create count pods with the given phase for the given tfJob
func NewPodList(count int32, status corev1.PodPhase, job metav1.Object, typ string, start int32, refs []metav1.OwnerReference) []*corev1.Pod {
	pods := []*corev1.Pod{}
	for i := int32(0); i < count; i++ {
		newPod := NewPod(job, typ, int(start+i), refs)
		newPod.Status = corev1.PodStatus{Phase: status}
		pods = append(pods, newPod)
	}
	return pods
}

func SetPodsStatuses(client client.Client, job metav1.Object, typ string,
	pendingPods, activePods, succeededPods, failedPods int32, restartCounts []int32,
	refs []metav1.OwnerReference, basicLabels map[string]string) {
	timeout := 10 * time.Second
	interval := 1000 * time.Millisecond
	var index int32
	taskMap := map[corev1.PodPhase]int32{
		corev1.PodFailed:    failedPods,
		corev1.PodPending:   pendingPods,
		corev1.PodRunning:   activePods,
		corev1.PodSucceeded: succeededPods,
	}
	ctx := context.Background()

	for podPhase, desiredCount := range taskMap {
		for i, pod := range NewPodList(desiredCount, podPhase, job, typ, index, refs) {
			for k, v := range basicLabels {
				pod.Labels[k] = v
			}
			_ = client.Create(ctx, pod)
			launcherKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      pod.GetName(),
			}
			Eventually(func() error {
				po := &corev1.Pod{}
				if err := client.Get(ctx, launcherKey, po); err != nil {
					return err
				}
				po.Status.Phase = podPhase
				if podPhase == corev1.PodRunning && restartCounts != nil {
					po.Status.ContainerStatuses = []corev1.ContainerStatus{{RestartCount: restartCounts[i]}}
				}
				return client.Status().Update(ctx, po)
			}, timeout, interval).Should(BeNil())
		}
		index += desiredCount
	}
}
