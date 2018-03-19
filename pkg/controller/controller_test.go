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
	"k8s.io/apimachinery/pkg/util/rand"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
)

var alwaysReady = func() bool { return true }

func newTFJobControllerFromClient(kubeClientSet kubeclientset.Interface, tfJobClientSet tfjobclientset.Interface, resyncPeriod ResyncPeriodFunc) (*TFJobController, kubeinformers.SharedInformerFactory, tfjobinformers.SharedInformerFactory) {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientSet, resyncPeriod())
	tfJobInformerFactory := tfjobinformers.NewSharedInformerFactory(tfJobClientSet, resyncPeriod())

	controller := NewTFJobController(kubeClientSet, tfJobClientSet, kubeInformerFactory, tfJobInformerFactory)
	controller.podControl = &FakePodControl{}
	// TODO(gaocegege): Add FakeServiceControl.
	controller.serviceControl = &FakeServiceControl{}
	return controller, kubeInformerFactory, tfJobInformerFactory
}

func newTFJob(worker, ps int) *tfv1alpha2.TFJob {
	tfJob := &tfv1alpha2.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foobar",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: tfv1alpha2.TFJobSpec{
			TFReplicaSpecs: make(map[tfv1alpha2.TFReplicaType]*tfv1alpha2.TFReplicaSpec),
		},
	}

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &tfv1alpha2.TFReplicaSpec{
			Replicas: &worker,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Image: "foo/bar",
						},
					},
				},
			},
		}
		tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker] = workerReplicaSpec
	}

	if ps > 0 {
		ps := int32(ps)
		psReplicaSpec := &tfv1alpha2.TFReplicaSpec{
			Replicas: &ps,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Image: "foo/bar",
						},
					},
				},
			},
		}
		tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypePS] = psReplicaSpec
	}
	return tfJob
}

func getKey(tfJob *tfv1alpha2.TFJob, t *testing.T) string {
	if key, err := KeyFunc(tfJob); err != nil {
		t.Errorf("Unexpected error getting key for job %v: %v", tfJob.Name, err)
		return ""
	} else {
		return key
	}
}

func newPod(name string, tfJob *tfv1alpha2.TFJob) *v1.Pod {
	tfjobKey, err := KeyFunc(tfJob)
	if err != nil {
		fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfJob, err)
	}

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          genLabels(tfjobKey),
			Namespace:       tfJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(tfJob, controllerKind)},
		},
	}
}

// create count pods with the given phase for the given tfJob
func newPodList(count int32, status v1.PodPhase, tfJob *tfv1alpha2.TFJob) []v1.Pod {
	pods := []v1.Pod{}
	for i := int32(0); i < count; i++ {
		newPod := newPod(fmt.Sprintf("pod-%v", rand.String(10)), tfJob)
		newPod.Status = v1.PodStatus{Phase: status}
		pods = append(pods, *newPod)
	}
	return pods
}

func setPodsStatuses(podIndexer cache.Indexer, tfJob *tfv1alpha2.TFJob, pendingPods, activePods, succeededPods, failedPods int32) {
	for _, pod := range newPodList(pendingPods, v1.PodPending, tfJob) {
		podIndexer.Add(&pod)
	}
	for _, pod := range newPodList(activePods, v1.PodRunning, tfJob) {
		podIndexer.Add(&pod)
	}
	for _, pod := range newPodList(succeededPods, v1.PodSucceeded, tfJob) {
		podIndexer.Add(&pod)
	}
	for _, pod := range newPodList(failedPods, v1.PodFailed, tfJob) {
		podIndexer.Add(&pod)
	}
}

func TestNormalPath(t *testing.T) {
	testCases := map[string]struct {
		worker int
		ps     int

		// pod setup
		podControllerError error
		jobKeyForget       bool
		pendingPods        int32
		activePods         int32
		succeededPods      int32
		failedPods         int32

		// TODO(gaocegege): Add service setup.

		// expectations
		expectedCreations int32
		expectedDeletions int32
		expectedActive    int32
		expectedSucceeded int32
		expectedFailed    int32
		// TODO(gaocegege): Add condition check.
		// expectedCondition       *tfv1alpha2.TFJobConditionType
		// expectedConditionReason string
	}{
		"Local TFJob created": {
			1, 0,
			nil, true, 0, 0, 0, 0,
			1, 0, 1, 0, 0,
		},
	}

	for name, tc := range testCases {
		// Prepare the clientset and controller for the test.
		kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &v1.SchemeGroupVersion,
			},
		},
		)
		tfJobClientSet := tfjobclientset.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &tfv1alpha2.SchemeGroupVersion,
			},
		},
		)
		controller, kubeInformerFactory, tfJobInformerFactory := newTFJobControllerFromClient(kubeClientSet, tfJobClientSet, NoResyncPeriodFunc)
		controller.tfJobListerSynced = alwaysReady
		controller.podListerSynced = alwaysReady
		controller.serviceListerSynced = alwaysReady

		// Run the test logic.
		tfJob := newTFJob(tc.worker, tc.ps)
		tfJobInformerFactory.Kubeflow().V1alpha2().TFJobs().Informer().GetIndexer().Add(tfJob)
		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		setPodsStatuses(podIndexer, tfJob, tc.pendingPods, tc.activePods, tc.succeededPods, tc.failedPods)

		forget, err := controller.syncTFJob(getKey(tfJob, t))
		// We need requeue syncJob task if podController error
		if tc.podControllerError != nil {
			if err == nil {
				t.Errorf("%s: Syncing jobs would return error when podController exception", name)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected error when syncing jobs %v", name, err)
			}
		}
		if forget != tc.jobKeyForget {
			t.Errorf("%s: unexpected forget value. Expected %v, saw %v\n", name, tc.jobKeyForget, forget)
		}
		if int32(len(controller.podControl.(*FakePodControl).Templates)) != tc.expectedCreations {
			t.Errorf("%s: unexpected number of creates.  Expected %d, saw %d\n", name, tc.expectedCreations, len(controller.podControl.(*FakePodControl).Templates))
		}
	}
}
