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

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
)

const (
	testImageName = "test-image-for-kubeflow-tf-operator:latest"
	testTFJobName = "test-tfjob"
	labelWorker   = "worker"
	labelPS       = "ps"
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

func newTFReplicaSpecTemplate() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Image: testImageName,
				},
			},
		},
	}
}

func newTFJob(worker, ps int) *tfv1alpha2.TFJob {
	tfJob := &tfv1alpha2.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testTFJobName,
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
			Template: newTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker] = workerReplicaSpec
	}

	if ps > 0 {
		ps := int32(ps)
		psReplicaSpec := &tfv1alpha2.TFReplicaSpec{
			Replicas: &ps,
			Template: newTFReplicaSpecTemplate(),
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

func getCondition(tfJob *tfv1alpha2.TFJob, condition tfv1alpha2.TFJobConditionType, reason string) bool {
	for _, v := range tfJob.Status.Conditions {
		if v.Type == condition && v.Status == v1.ConditionTrue && v.Reason == reason {
			return true
		}
	}
	return false
}

func TestNormalPath(t *testing.T) {
	testCases := map[string]struct {
		worker int
		ps     int

		// pod setup
		ControllerError error
		jobKeyForget    bool

		pendingWorkerPods   int32
		activeWorkerPods    int32
		succeededWorkerPods int32
		failedWorkerPods    int32

		pendingPSPods   int32
		activePSPods    int32
		succeededPSPods int32
		failedPSPods    int32

		// TODO(gaocegege): Add service setup.

		// expectations
		expectedCreations int32
		expectedDeletions int32

		expectedActiveWorkerPods    int32
		expectedSucceededWorkerPods int32
		expectedFailedWorkerPods    int32

		expectedActivePSPods    int32
		expectedSucceededPSPods int32
		expectedFailedPSPods    int32

		// TODO(gaocegege): Add condition check.
		expectedCondition       *tfv1alpha2.TFJobConditionType
		expectedConditionReason string
	}{
		"Local TFJob is created": {
			1, 0,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			1, 0,
			1, 0, 0,
			0, 0, 0,
			nil, "",
		},
		"Distributed TFJob (4 workers, 2 PS) is created": {
			4, 2,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			6, 0,
			4, 0, 0,
			2, 0, 0,
			nil, "",
		},
		"Distributed TFJob (4 workers, 2 PS) is created and all replicas are pending": {
			4, 2,
			nil, true,
			4, 0, 0, 0,
			2, 0, 0, 0,
			0, 0,
			4, 0, 0,
			2, 0, 0,
			nil, "",
		},
		"Distributed TFJob (4 workers, 2 PS) is created and all replicas are running": {
			4, 2,
			nil, true,
			0, 4, 0, 0,
			0, 2, 0, 0,
			0, 0,
			4, 0, 0,
			2, 0, 0,
			nil, "",
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending": {
			4, 2,
			nil, true,
			2, 0, 0, 0,
			1, 0, 0, 0,
			3, 0,
			4, 0, 0,
			2, 0, 0,
			nil, "",
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is running": {
			4, 2,
			nil, true,
			2, 1, 0, 0,
			1, 0, 0, 0,
			2, 0,
			4, 0, 0,
			2, 0, 0,
			nil, "",
		},
		"Distributed TFJob (4 workers, 2 PS) is succeeded": {
			4, 2,
			nil, true,
			0, 0, 4, 0,
			0, 0, 2, 0,
			0, 0,
			0, 4, 0,
			0, 2, 0,
			nil, "",
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
		var actual *tfv1alpha2.TFJob
		controller.updateStatusHandler = func(tfJob *tfv1alpha2.TFJob) error {
			actual = tfJob
			return nil
		}

		// Run the test logic.
		tfJob := newTFJob(tc.worker, tc.ps)
		tfJobInformerFactory.Kubeflow().V1alpha2().TFJobs().Informer().GetIndexer().Add(tfJob)
		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		setPodsStatuses(podIndexer, tfJob, labelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, t)
		setPodsStatuses(podIndexer, tfJob, labelPS, tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods, t)
		log.Info(podIndexer.List())

		forget, err := controller.syncTFJob(getKey(tfJob, t))
		// We need requeue syncJob task if podController error
		if tc.ControllerError != nil {
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

		fakePodControl := controller.podControl.(*FakePodControl)
		if int32(len(fakePodControl.Templates)) != tc.expectedCreations {
			t.Errorf("%s: unexpected number of creates.  Expected %d, saw %d\n", name, tc.expectedCreations, len(fakePodControl.Templates))
		}
		if int32(len(fakePodControl.DeletePodName)) != tc.expectedDeletions {
			t.Errorf("%s: unexpected number of deletes.  Expected %d, saw %d\n", name, tc.expectedDeletions, len(fakePodControl.DeletePodName))
		}
		// Each create should have an accompanying ControllerRef.
		if len(fakePodControl.ControllerRefs) != int(tc.expectedCreations) {
			t.Errorf("%s: unexpected number of ControllerRefs.  Expected %d, saw %d\n", name, tc.expectedCreations, len(fakePodControl.ControllerRefs))
		}
		// Make sure the ControllerRefs are correct.
		for _, controllerRef := range fakePodControl.ControllerRefs {
			if got, want := controllerRef.APIVersion, tfv1alpha2.SchemeGroupVersion.String(); got != want {
				t.Errorf("controllerRef.APIVersion = %q, want %q", got, want)
			}
			if got, want := controllerRef.Kind, tfv1alpha2.TFJobResourceKind; got != want {
				t.Errorf("controllerRef.Kind = %q, want %q", got, want)
			}
			if got, want := controllerRef.Name, tfJob.Name; got != want {
				t.Errorf("controllerRef.Name = %q, want %q", got, want)
			}
			if got, want := controllerRef.UID, tfJob.UID; got != want {
				t.Errorf("controllerRef.UID = %q, want %q", got, want)
			}
			if controllerRef.Controller == nil || !*controllerRef.Controller {
				t.Errorf("controllerRef.Controller is not set to true")
			}
		}
		// Validate worker status.
		if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker] != nil {
			if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker].Active != tc.expectedActiveWorkerPods {
				t.Errorf("%s: unexpected number of active pods.  Expected %d, saw %d\n", name, tc.expectedActiveWorkerPods, actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker].Active)
			}
			if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker].Succeeded != tc.expectedSucceededWorkerPods {
				t.Errorf("%s: unexpected number of succeeded pods.  Expected %d, saw %d\n", name, tc.expectedSucceededWorkerPods, actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker].Succeeded)
			}
			if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker].Failed != tc.expectedFailedWorkerPods {
				t.Errorf("%s: unexpected number of failed pods.  Expected %d, saw %d\n", name, tc.expectedFailedWorkerPods, actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypeWorker].Failed)
			}
		}
		// Validate PS status.
		if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypePS] != nil {
			if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypePS].Active != tc.expectedActivePSPods {
				t.Errorf("%s: unexpected number of active pods.  Expected %d, saw %d\n", name, tc.expectedActivePSPods, actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypePS].Active)
			}
			if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypePS].Succeeded != tc.expectedSucceededPSPods {
				t.Errorf("%s: unexpected number of succeeded pods.  Expected %d, saw %d\n", name, tc.expectedSucceededPSPods, actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypePS].Succeeded)
			}
			if actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypePS].Failed != tc.expectedFailedPSPods {
				t.Errorf("%s: unexpected number of failed pods.  Expected %d, saw %d\n", name, tc.expectedFailedPSPods, actual.Status.TFReplicaStatuses[tfv1alpha2.TFReplicaTypePS].Failed)
			}
		}
		// TODO(gaocegege): Set StartTime for the status.
		// Validate StartTime.
		// if actual.Status.StartTime == nil {
		// 	t.Errorf("%s: .status.startTime was not set", name)
		// }
		// Validate conditions.
		if tc.expectedCondition != nil && !getCondition(actual, *tc.expectedCondition, tc.expectedConditionReason) {
			t.Errorf("%s: expected completion condition.  Got %#v", name, actual.Status.Conditions)
		}
	}
}
