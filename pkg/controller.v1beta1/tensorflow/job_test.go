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

package tensorflow

import (
	"testing"
	"time"

	"k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/kubeflow/tf-operator/cmd/tf-operator.v1beta1/app/options"
	common "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta1"
	tfv1beta1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta1"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/common/util/testutil"
	"github.com/kubeflow/tf-operator/pkg/control"
)

func TestAddTFJob(t *testing.T) {
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
			GroupVersion: &tfv1beta1.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	var key string
	syncChan := make(chan string)
	ctr.syncHandler = func(tfJobKey string) (bool, error) {
		key = tfJobKey
		<-syncChan
		return true, nil
	}
	ctr.updateStatusHandler = func(tfjob *tfv1beta1.TFJob) error {
		return nil
	}
	ctr.deleteTFJobHandler = func(tfjob *tfv1beta1.TFJob) error {
		return nil
	}

	tfJob := testutil.NewTFJob(1, 0)
	unstructured, err := testutil.ConvertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}
	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}
	ctr.addTFJob(unstructured)

	syncChan <- "sync"
	if key != testutil.GetKey(tfJob, t) {
		t.Errorf("Failed to enqueue the TFJob %s: expected %s, got %s", tfJob.Name, testutil.GetKey(tfJob, t), key)
	}
	close(stopCh)
}

func TestCopyLabelsAndAnnotation(t *testing.T) {
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
			GroupVersion: &tfv1beta1.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	fakePodControl := &controller.FakePodControl{}
	ctr.PodControl = fakePodControl
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	ctr.updateStatusHandler = func(tfJob *tfv1beta1.TFJob) error {
		return nil
	}

	tfJob := testutil.NewTFJob(1, 0)
	annotations := map[string]string{
		"annotation1": "1",
	}
	labels := map[string]string{
		"label1": "1",
	}
	tfJob.Spec.TFReplicaSpecs[tfv1beta1.TFReplicaTypeWorker].Template.Labels = labels
	tfJob.Spec.TFReplicaSpecs[tfv1beta1.TFReplicaTypeWorker].Template.Annotations = annotations
	unstructured, err := testutil.ConvertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
	}

	if err := tfJobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
	}

	_, err = ctr.syncTFJob(testutil.GetKey(tfJob, t))
	if err != nil {
		t.Errorf("%s: unexpected error when syncing jobs %v", tfJob.Name, err)
	}

	if len(fakePodControl.Templates) != 1 {
		t.Errorf("Expected to create 1 pod while got %d", len(fakePodControl.Templates))
	}
	actual := fakePodControl.Templates[0]
	v, exist := actual.Labels["label1"]
	if !exist {
		t.Errorf("Labels does not exist")
	}
	if v != "1" {
		t.Errorf("Labels value do not equal")
	}

	v, exist = actual.Annotations["annotation1"]
	if !exist {
		t.Errorf("Annotations does not exist")
	}
	if v != "1" {
		t.Errorf("Annotations value does not equal")
	}

	close(stopCh)
}

func TestDeletePodsAndServices(t *testing.T) {
	type testCase struct {
		description string
		tfJob       *tfv1beta1.TFJob

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

		expectedPodDeletions int
	}

	testCases := []testCase{
		testCase{
			description: "4 workers and 2 ps is running, policy is all",
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, common.CleanPodPolicyAll),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingPSPods:   0,
			activePSPods:    2,
			succeededPSPods: 0,
			failedPSPods:    0,

			activeWorkerServices: 4,
			activePSServices:     2,

			expectedPodDeletions: 6,
		},
		testCase{
			description: "4 workers and 2 ps is running, policy is running",
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, common.CleanPodPolicyRunning),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingPSPods:   0,
			activePSPods:    2,
			succeededPSPods: 0,
			failedPSPods:    0,

			activeWorkerServices: 4,
			activePSServices:     2,

			expectedPodDeletions: 6,
		},
		testCase{
			description: "4 workers and 2 ps is succeeded, policy is running",
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, common.CleanPodPolicyRunning),

			pendingWorkerPods:   0,
			activeWorkerPods:    0,
			succeededWorkerPods: 4,
			failedWorkerPods:    0,

			pendingPSPods:   0,
			activePSPods:    0,
			succeededPSPods: 2,
			failedPSPods:    0,

			activeWorkerServices: 4,
			activePSServices:     2,

			expectedPodDeletions: 0,
		},
		testCase{
			description: "4 workers and 2 ps is succeeded, policy is None",
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, common.CleanPodPolicyNone),

			pendingWorkerPods:   0,
			activeWorkerPods:    0,
			succeededWorkerPods: 4,
			failedWorkerPods:    0,

			pendingPSPods:   0,
			activePSPods:    0,
			succeededPSPods: 2,
			failedPSPods:    0,

			activeWorkerServices: 4,
			activePSServices:     2,

			expectedPodDeletions: 0,
		},
	}
	for _, tc := range testCases {
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
				GroupVersion: &tfv1beta1.SchemeGroupVersion,
			},
		}
		tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
		fakePodControl := &controller.FakePodControl{}
		ctr.PodControl = fakePodControl
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.tfJobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		tfJobIndexer := ctr.tfJobInformer.GetIndexer()
		ctr.updateStatusHandler = func(tfJob *tfv1beta1.TFJob) error {
			return nil
		}

		// Set succeeded to run the logic about deleting.
		err := updateTFJobConditions(tc.tfJob, common.JobSucceeded, tfJobSucceededReason, "")
		if err != nil {
			t.Errorf("Append tfjob condition error: %v", err)
		}

		unstructured, err := testutil.ConvertTFJobToUnstructured(tc.tfJob)
		if err != nil {
			t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
		}

		if err := tfJobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, tc.tfJob, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, t)
		testutil.SetPodsStatuses(podIndexer, tc.tfJob, testutil.LabelPS, tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
		testutil.SetServices(serviceIndexer, tc.tfJob, testutil.LabelWorker, tc.activeWorkerServices, t)
		testutil.SetServices(serviceIndexer, tc.tfJob, testutil.LabelPS, tc.activePSServices, t)

		forget, err := ctr.syncTFJob(testutil.GetKey(tc.tfJob, t))
		if err != nil {
			t.Errorf("%s: unexpected error when syncing jobs %v", tc.description, err)
		}
		if !forget {
			t.Errorf("%s: unexpected forget value. Expected true, saw %v\n", tc.description, forget)
		}

		if len(fakePodControl.DeletePodName) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", tc.description, tc.expectedPodDeletions, len(fakePodControl.DeletePodName))
		}
		if len(fakeServiceControl.DeleteServiceName) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of service deletes.  Expected %d, saw %d\n", tc.description, tc.expectedPodDeletions, len(fakeServiceControl.DeleteServiceName))
		}
	}
}

func TestCleanupTFJob(t *testing.T) {
	type testCase struct {
		description string
		tfJob       *tfv1beta1.TFJob

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

		expectedDeleteFinished bool
	}

	ttlaf0 := int32(0)
	ttl0 := &ttlaf0
	ttlaf2s := int32(2)
	ttl2s := &ttlaf2s
	testCases := []testCase{
		testCase{
			description: "4 workers and 2 ps is running, TTLSecondsAfterFinished unset",
			tfJob:       testutil.NewTFJobWithCleanupJobDelay(0, 4, 2, nil),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingPSPods:   0,
			activePSPods:    2,
			succeededPSPods: 0,
			failedPSPods:    0,

			activeWorkerServices: 4,
			activePSServices:     2,

			expectedDeleteFinished: false,
		},
		testCase{
			description: "4 workers and 2 ps is running, TTLSecondsAfterFinished is 0",
			tfJob:       testutil.NewTFJobWithCleanupJobDelay(0, 4, 2, ttl0),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingPSPods:   0,
			activePSPods:    2,
			succeededPSPods: 0,
			failedPSPods:    0,

			activeWorkerServices: 4,
			activePSServices:     2,

			expectedDeleteFinished: true,
		},
		testCase{
			description: "4 workers and 2 ps is succeeded, TTLSecondsAfterFinished is 2",
			tfJob:       testutil.NewTFJobWithCleanupJobDelay(0, 4, 2, ttl2s),

			pendingWorkerPods:   0,
			activeWorkerPods:    0,
			succeededWorkerPods: 4,
			failedWorkerPods:    0,

			pendingPSPods:   0,
			activePSPods:    0,
			succeededPSPods: 2,
			failedPSPods:    0,

			activeWorkerServices: 4,
			activePSServices:     2,

			expectedDeleteFinished: true,
		},
	}
	for _, tc := range testCases {
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
				GroupVersion: &tfv1beta1.SchemeGroupVersion,
			},
		}
		tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newTFController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
		fakePodControl := &controller.FakePodControl{}
		ctr.PodControl = fakePodControl
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.tfJobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		tfJobIndexer := ctr.tfJobInformer.GetIndexer()
		ctr.updateStatusHandler = func(tfJob *tfv1beta1.TFJob) error {
			return nil
		}
		deleteFinished := false
		ctr.deleteTFJobHandler = func(tfJob *tfv1beta1.TFJob) error {
			deleteFinished = true
			return nil
		}

		// Set succeeded to run the logic about deleting.
		testutil.SetTFJobCompletionTime(tc.tfJob)

		err := updateTFJobConditions(tc.tfJob, common.JobSucceeded, tfJobSucceededReason, "")
		if err != nil {
			t.Errorf("Append tfjob condition error: %v", err)
		}

		unstructured, err := testutil.ConvertTFJobToUnstructured(tc.tfJob)
		if err != nil {
			t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
		}

		if err := tfJobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, tc.tfJob, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, t)
		testutil.SetPodsStatuses(podIndexer, tc.tfJob, testutil.LabelPS, tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
		testutil.SetServices(serviceIndexer, tc.tfJob, testutil.LabelWorker, tc.activeWorkerServices, t)
		testutil.SetServices(serviceIndexer, tc.tfJob, testutil.LabelPS, tc.activePSServices, t)

		ttl := tc.tfJob.Spec.TTLSecondsAfterFinished
		if ttl != nil {
			dur := time.Second * time.Duration(*ttl)
			time.Sleep(dur)
		}

		forget, err := ctr.syncTFJob(testutil.GetKey(tc.tfJob, t))
		if err != nil {
			t.Errorf("%s: unexpected error when syncing jobs %v", tc.description, err)
		}
		if !forget {
			t.Errorf("%s: unexpected forget value. Expected true, saw %v\n", tc.description, forget)
		}

		if deleteFinished != tc.expectedDeleteFinished {
			t.Errorf("%s: unexpected status. Expected %v, saw %v", tc.description, tc.expectedDeleteFinished, deleteFinished)
		}
	}
}
