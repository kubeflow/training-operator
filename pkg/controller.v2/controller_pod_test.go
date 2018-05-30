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

// Package controller provides a Kubernetes controller for a TFJob resource.
package controller

import (
	"fmt"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
)

func newBasePod(name string, tfJob *tfv1alpha2.TFJob, t *testing.T) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          genLabels(getKey(tfJob, t)),
			Namespace:       tfJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(tfJob, controllerKind)},
		},
	}
}

func newPod(tfJob *tfv1alpha2.TFJob, typ string, index int, t *testing.T) *v1.Pod {
	pod := newBasePod(fmt.Sprintf("%s-%d", typ, index), tfJob, t)
	pod.Labels[tfReplicaTypeLabel] = typ
	pod.Labels[tfReplicaIndexLabel] = fmt.Sprintf("%d", index)
	return pod
}

// create count pods with the given phase for the given tfJob
func newPodList(count int32, status v1.PodPhase, tfJob *tfv1alpha2.TFJob, typ string, start int32, t *testing.T) []*v1.Pod {
	pods := []*v1.Pod{}
	for i := int32(0); i < count; i++ {
		newPod := newPod(tfJob, typ, int(start+i), t)
		newPod.Status = v1.PodStatus{Phase: status}
		pods = append(pods, newPod)
	}
	return pods
}

func setPodsStatuses(podIndexer cache.Indexer, tfJob *tfv1alpha2.TFJob, typ string, pendingPods, activePods, succeededPods, failedPods int32, t *testing.T) {
	var index int32
	for _, pod := range newPodList(pendingPods, v1.PodPending, tfJob, typ, index, t) {
		podIndexer.Add(pod)
	}
	index += pendingPods
	for _, pod := range newPodList(activePods, v1.PodRunning, tfJob, typ, index, t) {
		podIndexer.Add(pod)
	}
	index += activePods
	for _, pod := range newPodList(succeededPods, v1.PodSucceeded, tfJob, typ, index, t) {
		podIndexer.Add(pod)
	}
	index += succeededPods
	for _, pod := range newPodList(failedPods, v1.PodFailed, tfJob, typ, index, t) {
		podIndexer.Add(pod)
	}
}

func TestAddPod(t *testing.T) {
	// Prepare the clientset and controller for the test.
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)
	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &tfv1alpha2.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
	ctr.tfJobInformerSynced = alwaysReady
	ctr.podInformerSynced = alwaysReady
	ctr.serviceInformerSynced = alwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(threadCount, stopCh)
	}
	go run(stopCh)

	var key string
	syncChan := make(chan string)
	ctr.syncHandler = func(tfJobKey string) (bool, error) {
		key = tfJobKey
		<-syncChan
		return true, nil
	}

	tfJob := newTFJob(1, 0)
	unstructured, err := convertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}

	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}
	pod := newPod(tfJob, labelWorker, 0, t)
	ctr.addPod(pod)

	syncChan <- "sync"
	if key != getKey(tfJob, t) {
		t.Errorf("Failed to enqueue the TFJob %s: expected %s, got %s", tfJob.Name, getKey(tfJob, t), key)
	}
	close(stopCh)
}
