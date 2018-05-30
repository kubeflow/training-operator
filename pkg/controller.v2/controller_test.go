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
	"testing"
	"time"

	"k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/controller"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
)

const (
	testImageName = "test-image-for-kubeflow-tf-operator:latest"
	testTFJobName = "test-tfjob"
	labelWorker   = "worker"
	labelPS       = "ps"

	sleepInterval = 500 * time.Millisecond
	threadCount   = 1
)

var (
	alwaysReady = func() bool { return true }

	tfJobRunning   = tfv1alpha2.TFJobRunning
	tfJobSucceeded = tfv1alpha2.TFJobSucceeded
)

func newTFJobController(
	config *rest.Config,
	kubeClientSet kubeclientset.Interface,
	tfJobClientSet tfjobclientset.Interface,
	resyncPeriod controller.ResyncPeriodFunc,
) (
	*TFJobController,
	kubeinformers.SharedInformerFactory, tfjobinformers.SharedInformerFactory,
) {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientSet, resyncPeriod())
	tfJobInformerFactory := tfjobinformers.NewSharedInformerFactory(tfJobClientSet, resyncPeriod())

	tfJobInformer := NewUnstructuredTFJobInformer(config)

	ctr := NewTFJobController(tfJobInformer, kubeClientSet, tfJobClientSet, kubeInformerFactory, tfJobInformerFactory)
	ctr.podControl = &controller.FakePodControl{}
	ctr.serviceControl = &FakeServiceControl{}
	return ctr, kubeInformerFactory, tfJobInformerFactory
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

		activeWorkerServices int32
		activePSServices     int32

		// expectations
		expectedPodCreations     int32
		expectedPodDeletions     int32
		expectedServiceCreations int32

		expectedActiveWorkerPods    int32
		expectedSucceededWorkerPods int32
		expectedFailedWorkerPods    int32

		expectedActivePSPods    int32
		expectedSucceededPSPods int32
		expectedFailedPSPods    int32

		expectedCondition       *tfv1alpha2.TFJobConditionType
		expectedConditionReason string

		// There are some cases that should not check start time since the field should be set in the previous sync loop.
		needCheckStartTime bool
	}{
		"Local TFJob is created": {
			1, 0,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0,
			1, 0, 1,
			0, 0, 0,
			0, 0, 0,
			// We can not check if it is created since the condition is set in addTFJob.
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created": {
			4, 2,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0,
			6, 0, 6,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created and all replicas are pending": {
			4, 2,
			nil, true,
			4, 0, 0, 0,
			2, 0, 0, 0,
			4, 2,
			0, 0, 0,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created and all replicas are running": {
			4, 2,
			nil, true,
			0, 4, 0, 0,
			0, 2, 0, 0,
			4, 2,
			0, 0, 0,
			4, 0, 0,
			2, 0, 0,
			&tfJobRunning, tfJobRunningReason,
			true,
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending": {
			4, 2,
			nil, true,
			2, 0, 0, 0,
			1, 0, 0, 0,
			2, 1,
			3, 0, 3,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is running": {
			4, 2,
			nil, true,
			2, 1, 0, 0,
			1, 0, 0, 0,
			3, 1,
			2, 0, 2,
			1, 0, 0,
			0, 0, 0,
			&tfJobRunning, tfJobRunningReason,
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is succeeded": {
			4, 2,
			nil, true,
			2, 0, 1, 0,
			1, 0, 0, 0,
			3, 1,
			2, 0, 2,
			0, 1, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed TFJob (4 workers, 2 PS) is succeeded": {
			4, 2,
			nil, true,
			0, 0, 4, 0,
			0, 0, 2, 0,
			4, 2,
			0, 0, 0,
			0, 4, 0,
			0, 2, 0,
			&tfJobSucceeded, tfJobSucceededReason,
			false,
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
		config := &rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &tfv1alpha2.SchemeGroupVersion,
			},
		}
		tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
		ctr.tfJobInformerSynced = alwaysReady
		ctr.podInformerSynced = alwaysReady
		ctr.serviceInformerSynced = alwaysReady
		tfJobIndexer := ctr.tfJobInformer.GetIndexer()

		var actual *tfv1alpha2.TFJob
		ctr.updateStatusHandler = func(tfJob *tfv1alpha2.TFJob) error {
			actual = tfJob
			return nil
		}

		// Run the test logic.
		tfJob := newTFJob(tc.worker, tc.ps)
		unstructured, err := convertTFJobToUnstructured(tfJob)
		if err != nil {
			t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
		}

		if err := tfJobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		setPodsStatuses(podIndexer, tfJob, labelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, t)
		setPodsStatuses(podIndexer, tfJob, labelPS, tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
		setServices(serviceIndexer, tfJob, labelWorker, tc.activeWorkerServices, t)
		setServices(serviceIndexer, tfJob, labelPS, tc.activePSServices, t)

		forget, err := ctr.syncTFJob(getKey(tfJob, t))
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

		fakePodControl := ctr.podControl.(*controller.FakePodControl)
		fakeServiceControl := ctr.serviceControl.(*FakeServiceControl)
		if int32(len(fakePodControl.Templates)) != tc.expectedPodCreations {
			t.Errorf("%s: unexpected number of pod creates.  Expected %d, saw %d\n", name, tc.expectedPodCreations, len(fakePodControl.Templates))
		}
		if int32(len(fakeServiceControl.Templates)) != tc.expectedServiceCreations {
			t.Errorf("%s: unexpected number of service creates.  Expected %d, saw %d\n", name, tc.expectedServiceCreations, len(fakeServiceControl.Templates))
		}
		if int32(len(fakePodControl.DeletePodName)) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", name, tc.expectedPodDeletions, len(fakePodControl.DeletePodName))
		}
		// Each create should have an accompanying ControllerRef.
		if len(fakePodControl.ControllerRefs) != int(tc.expectedPodCreations) {
			t.Errorf("%s: unexpected number of ControllerRefs.  Expected %d, saw %d\n", name, tc.expectedPodCreations, len(fakePodControl.ControllerRefs))
		}
		// Make sure the ControllerRefs are correct.
		for _, controllerRef := range fakePodControl.ControllerRefs {
			if got, want := controllerRef.APIVersion, tfv1alpha2.SchemeGroupVersion.String(); got != want {
				t.Errorf("controllerRef.APIVersion = %q, want %q", got, want)
			}
			if got, want := controllerRef.Kind, tfv1alpha2.Kind; got != want {
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
		// Validate StartTime.
		if tc.needCheckStartTime && actual.Status.StartTime == nil {
			t.Errorf("%s: StartTime was not set", name)
		}
		// Validate conditions.
		if tc.expectedCondition != nil && !checkCondition(actual, *tc.expectedCondition, tc.expectedConditionReason) {
			t.Errorf("%s: expected condition %#v, got %#v", name, *tc.expectedCondition, actual.Status.Conditions)
		}
	}
}

func TestRun(t *testing.T) {
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

	stopCh := make(chan struct{})
	go func() {
		// It is a hack to let the controller stop to run without errors.
		// We can not just send a struct to stopCh because there are multiple
		// receivers in controller.Run.
		time.Sleep(sleepInterval)
		stopCh <- struct{}{}
	}()
	err := ctr.Run(threadCount, stopCh)
	if err != nil {
		t.Errorf("Failed to run: %v", err)
	}
}
