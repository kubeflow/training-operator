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

// Package controller provides a Kubernetes controller for a TensorFlow job resource.
package controller

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	_ "k8s.io/kubernetes/pkg/apis/core/install"
	"k8s.io/kubernetes/pkg/controller"
	"github.com/cloudflare/cfssl/log"

	tfjobclient "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/trainer"
)

var alwaysReady = func() bool { return true }

// func newTFJob() *tfv1alpha1.TFJob {
// 	j := &tfv1alpha1.TFJob{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "foobar",
// 			UID:       uuid.NewUUID(),
// 			Namespace: metav1.NamespaceDefault,
// 		},
// 		Spec: batch.JobSpec{
// 			Selector: &metav1.LabelSelector{
// 				MatchLabels: map[string]string{"foo": "bar"},
// 			},
// 			Template: v1.PodTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Labels: map[string]string{
// 						"foo": "bar",
// 					},
// 				},
// 				Spec: v1.PodSpec{
// 					Containers: []v1.Container{
// 						{Image: "foo/bar"},
// 					},
// 				},
// 			},
// 		},
// 	}
// 	// Special case: -1 for either completions or parallelism means leave nil (negative is not allowed
// 	// in practice by validation.
// 	if completions >= 0 {
// 		j.Spec.Completions = &completions
// 	} else {
// 		j.Spec.Completions = nil
// 	}
// 	if parallelism >= 0 {
// 		j.Spec.Parallelism = &parallelism
// 	} else {
// 		j.Spec.Parallelism = nil
// 	}
// 	j.Spec.BackoffLimit = &backoffLimit

// 	return j
// }

// func getKey(job *tfv1alpha1.TFJob, t *testing.T) string {
// 	if key, err := controller.KeyFunc(job); err != nil {
// 		t.Errorf("Unexpected error getting key for job %v: %v", job.Name, err)
// 		return ""
// 	} else {
// 		return key
// 	}
// }


// // create count pods with the given phase for the given job
// func newPodList(count int32, status v1.PodPhase, job *batch.Job) []v1.Pod {
// 	pods := []v1.Pod{}
// 	for i := int32(0); i < count; i++ {
// 		newPod := v1.Pod{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:            fmt.Sprintf("pod-%v", rand.String(10)),
// 				Labels:          job.Spec.Selector.MatchLabels,
// 				Namespace:       job.Namespace,
// 				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(job, controllerKind)},
// 			},
// 			Status: v1.PodStatus{Phase: status},
// 		}
// 		pods = append(pods, newPod)
// 	}
// 	return pods
// }

func TestControllerSyncJob(t *testing.T) {
	jobConditionComplete := batch.JobComplete
	jobConditionFailed := batch.JobFailed

	testCases := map[string]struct {
		// job setup
		parallelism  int32
		completions  int32
		backoffLimit int32
		deleting     bool
		podLimit     int

		// pod setup
		podControllerError error
		jobKeyForget       bool
		pendingPods        int32
		activePods         int32
		succeededPods      int32
		failedPods         int32

		// expectations
		expectedCreations       int32
		expectedDeletions       int32
		expectedActive          int32
		expectedSucceeded       int32
		expectedFailed          int32
		expectedCondition       *batch.JobConditionType
		expectedConditionReason string
	}{
		"job start": {
			2, 5, 6, false, 0,
			nil, true, 0, 0, 0, 0,
			2, 0, 2, 0, 0, nil, "",
		},
		"WQ job start": {
			2, -1, 6, false, 0,
			nil, true, 0, 0, 0, 0,
			2, 0, 2, 0, 0, nil, "",
		},
		"pending pods": {
			2, 5, 6, false, 0,
			nil, true, 2, 0, 0, 0,
			0, 0, 2, 0, 0, nil, "",
		},
		"correct # of pods": {
			2, 5, 6, false, 0,
			nil, true, 0, 2, 0, 0,
			0, 0, 2, 0, 0, nil, "",
		},
		"WQ job: correct # of pods": {
			2, -1, 6, false, 0,
			nil, true, 0, 2, 0, 0,
			0, 0, 2, 0, 0, nil, "",
		},
		"too few active pods": {
			2, 5, 6, false, 0,
			nil, true, 0, 1, 1, 0,
			1, 0, 2, 1, 0, nil, "",
		},
		"too few active pods with a dynamic job": {
			2, -1, 6, false, 0,
			nil, true, 0, 1, 0, 0,
			1, 0, 2, 0, 0, nil, "",
		},
		"too few active pods, with controller error": {
			2, 5, 6, false, 0,
			fmt.Errorf("Fake error"), true, 0, 1, 1, 0,
			1, 0, 1, 1, 0, nil, "",
		},
		"too many active pods": {
			2, 5, 6, false, 0,
			nil, true, 0, 3, 0, 0,
			0, 1, 2, 0, 0, nil, "",
		},
		"too many active pods, with controller error": {
			2, 5, 6, false, 0,
			fmt.Errorf("Fake error"), true, 0, 3, 0, 0,
			0, 1, 3, 0, 0, nil, "",
		},
		"failed pod": {
			2, 5, 6, false, 0,
			fmt.Errorf("Fake error"), false, 0, 1, 1, 1,
			1, 0, 1, 1, 1, nil, "",
		},
		"job finish": {
			2, 5, 6, false, 0,
			nil, true, 0, 0, 5, 0,
			0, 0, 0, 5, 0, nil, "",
		},
		"WQ job finishing": {
			2, -1, 6, false, 0,
			nil, true, 0, 1, 1, 0,
			0, 0, 1, 1, 0, nil, "",
		},
		"WQ job all finished": {
			2, -1, 6, false, 0,
			nil, true, 0, 0, 2, 0,
			0, 0, 0, 2, 0, &jobConditionComplete, "",
		},
		"WQ job all finished despite one failure": {
			2, -1, 6, false, 0,
			nil, true, 0, 0, 1, 1,
			0, 0, 0, 1, 1, &jobConditionComplete, "",
		},
		"more active pods than completions": {
			2, 5, 6, false, 0,
			nil, true, 0, 10, 0, 0,
			0, 8, 2, 0, 0, nil, "",
		},
		"status change": {
			2, 5, 6, false, 0,
			nil, true, 0, 2, 2, 0,
			0, 0, 2, 2, 0, nil, "",
		},
		"deleting job": {
			2, 5, 6, true, 0,
			nil, true, 1, 1, 1, 0,
			0, 0, 2, 1, 0, nil, "",
		},
		"limited pods": {
			100, 200, 6, false, 10,
			nil, true, 0, 0, 0, 0,
			10, 0, 10, 0, 0, nil, "",
		},
		"to many job sync failure": {
			2, 5, 0, true, 0,
			nil, true, 0, 0, 0, 1,
			0, 0, 0, 0, 1, &jobConditionFailed, "BackoffLimitExceeded",
		},
	}

	for name, tc := range testCases {
		kubeClient := clientset.NewForConfigOrDie(&restclient.Config{Host: "", ContentConfig: restclient.ContentConfig{GroupVersion: &legacyscheme.Registry.GroupOrDie(v1.GroupName).GroupVersion}})
		sharedInformers := informers.NewSharedInformerFactory(kubeClient, resyncPeriod())

		tfJobClient := tfjobclient.

		controller := New()
		manager.TFJobSynced = alwaysReady
		var actual *batch.Job
		manager.updateHandler = func(job *batch.Job) error {
			actual = job
			return nil
		}

		// job & pods setup
		job := newJob(tc.parallelism, tc.completions, tc.backoffLimit)
		if tc.deleting {
			now := metav1.Now()
			job.DeletionTimestamp = &now
		}
		sharedInformerFactory.Batch().V1().Jobs().Informer().GetIndexer().Add(job)
		podIndexer := sharedInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		for _, pod := range newPodList(tc.pendingPods, v1.PodPending, job) {
			podIndexer.Add(&pod)
		}
		for _, pod := range newPodList(tc.activePods, v1.PodRunning, job) {
			podIndexer.Add(&pod)
		}
		for _, pod := range newPodList(tc.succeededPods, v1.PodSucceeded, job) {
			podIndexer.Add(&pod)
		}
		for _, pod := range newPodList(tc.failedPods, v1.PodFailed, job) {
			podIndexer.Add(&pod)
		}

		// run
		forget, err := manager.syncJob(getKey(job, t))

		// We need requeue syncJob task if podController error
		if tc.podControllerError != nil {
			if err == nil {
				t.Errorf("%s: Syncing jobs would return error when podController exception", name)
			}
		} else {
			if err != nil && (tc.podLimit == 0 || fakePodControl.CreateCallCount < tc.podLimit) {
				t.Errorf("%s: unexpected error when syncing jobs %v", name, err)
			}
		}
		if forget != tc.jobKeyForget {
			t.Errorf("%s: unexpected forget value. Expected %v, saw %v\n", name, tc.jobKeyForget, forget)
		}
		// validate created/deleted pods
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
			if got, want := controllerRef.APIVersion, "batch/v1"; got != want {
				t.Errorf("controllerRef.APIVersion = %q, want %q", got, want)
			}
			if got, want := controllerRef.Kind, "Job"; got != want {
				t.Errorf("controllerRef.Kind = %q, want %q", got, want)
			}
			if got, want := controllerRef.Name, job.Name; got != want {
				t.Errorf("controllerRef.Name = %q, want %q", got, want)
			}
			if got, want := controllerRef.UID, job.UID; got != want {
				t.Errorf("controllerRef.UID = %q, want %q", got, want)
			}
			if controllerRef.Controller == nil || *controllerRef.Controller != true {
				t.Errorf("controllerRef.Controller is not set to true")
			}
		}
		// validate status
		if actual.Status.Active != tc.expectedActive {
			t.Errorf("%s: unexpected number of active pods.  Expected %d, saw %d\n", name, tc.expectedActive, actual.Status.Active)
		}
		if actual.Status.Succeeded != tc.expectedSucceeded {
			t.Errorf("%s: unexpected number of succeeded pods.  Expected %d, saw %d\n", name, tc.expectedSucceeded, actual.Status.Succeeded)
		}
		if actual.Status.Failed != tc.expectedFailed {
			t.Errorf("%s: unexpected number of failed pods.  Expected %d, saw %d\n", name, tc.expectedFailed, actual.Status.Failed)
		}
		if actual.Status.StartTime == nil {
			t.Errorf("%s: .status.startTime was not set", name)
		}
		// validate conditions
		if tc.expectedCondition != nil && !getCondition(actual, *tc.expectedCondition, tc.expectedConditionReason) {
			t.Errorf("%s: expected completion condition.  Got %#v", name, actual.Status.Conditions)
		}
		// validate slow start
		expectedLimit := 0
		for pass := uint8(0); expectedLimit <= tc.podLimit; pass++ {
			expectedLimit += controller.SlowStartInitialBatchSize << pass
		}
		if tc.podLimit > 0 && fakePodControl.CreateCallCount > expectedLimit {
			t.Errorf("%s: Unexpected number of create calls.  Expected <= %d, saw %d\n", name, fakePodControl.CreateLimit*2, fakePodControl.CreateCallCount)
		}
	}
}

// func getCondition(job *batch.Job, condition batch.JobConditionType, reason string) bool {
// 	for _, v := range job.Status.Conditions {
// 		if v.Type == condition && v.Status == v1.ConditionTrue && v.Reason == reason {
// 			return true
// 		}
// 	}
// 	return false
// }
