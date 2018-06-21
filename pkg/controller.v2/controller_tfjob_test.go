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

package controller

import (
	"testing"

	"k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/control"
	"github.com/kubeflow/tf-operator/pkg/generator"
	"github.com/kubeflow/tf-operator/pkg/util/testutil"
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
			GroupVersion: &tfv1alpha2.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.podInformerSynced = testutil.AlwaysReady
	ctr.serviceInformerSynced = testutil.AlwaysReady
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
	ctr.updateStatusHandler = func(tfjob *tfv1alpha2.TFJob) error {
		return nil
	}

	tfJob := testutil.NewTFJob(1, 0)
	unstructured, err := generator.ConvertTFJobToUnstructured(tfJob)
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
			GroupVersion: &tfv1alpha2.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
	fakePodControl := &controller.FakePodControl{}
	ctr.podControl = fakePodControl
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.podInformerSynced = testutil.AlwaysReady
	ctr.serviceInformerSynced = testutil.AlwaysReady
	tfJobIndexer := ctr.tfJobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	ctr.updateStatusHandler = func(tfJob *tfv1alpha2.TFJob) error {
		return nil
	}

	tfJob := testutil.NewTFJob(1, 0)
	annotations := map[string]string{
		"annotation1": "1",
	}
	labels := map[string]string{
		"label1": "1",
	}
	tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].Template.Labels = labels
	tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker].Template.Annotations = annotations
	unstructured, err := generator.ConvertTFJobToUnstructured(tfJob)
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
		tfJob       *tfv1alpha2.TFJob

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
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, tfv1alpha2.CleanPodPolicyAll),

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
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, tfv1alpha2.CleanPodPolicyRunning),

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
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, tfv1alpha2.CleanPodPolicyRunning),

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
			tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, tfv1alpha2.CleanPodPolicyNone),

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
				GroupVersion: &tfv1alpha2.SchemeGroupVersion,
			},
		}
		tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newTFJobController(config, kubeClientSet, tfJobClientSet, controller.NoResyncPeriodFunc)
		fakePodControl := &controller.FakePodControl{}
		ctr.podControl = fakePodControl
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.serviceControl = fakeServiceControl
		ctr.recorder = &record.FakeRecorder{}
		ctr.tfJobInformerSynced = testutil.AlwaysReady
		ctr.podInformerSynced = testutil.AlwaysReady
		ctr.serviceInformerSynced = testutil.AlwaysReady
		tfJobIndexer := ctr.tfJobInformer.GetIndexer()
		ctr.updateStatusHandler = func(tfJob *tfv1alpha2.TFJob) error {
			return nil
		}

		// Set succeeded to run the logic about deleting.
		err := updateTFJobConditions(tc.tfJob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, "")
		if err != nil {
			t.Errorf("Append tfjob condition error: %v", err)
		}

		unstructured, err := generator.ConvertTFJobToUnstructured(tc.tfJob)
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
