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
package tensorflow

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"

	batchv1beta1 "volcano.sh/volcano/pkg/apis/scheduling/v1beta1"
	volcanoclient "volcano.sh/volcano/pkg/client/clientset/versioned"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/tf-operator/cmd/tf-operator.v1/app/options"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/common/util/v1/testutil"
)

func TestFailed(t *testing.T) {
	// Prepare the clientset and controller for the test.
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)

	// Prepare the volcano clientset and controller for the test.
	volcanoClientSet := volcanoclient.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &batchv1beta1.SchemeGroupVersion,
		},
	},
	)

	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &tfv1.SchemeGroupVersion,
		},
	}
	tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newTFController(config, kubeClientSet, volcanoClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	ctr.tfJobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady

	tfJob := testutil.NewTFJob(3, 0)
	initializeReplicaStatuses(&tfJob.Status, tfv1.TFReplicaTypeWorker)
	pod := testutil.NewBasePod("pod", tfJob)
	pod.Status.Phase = v1.PodFailed

	updateJobReplicaStatuses(&tfJob.Status, tfv1.TFReplicaTypeWorker, pod)
	if tfJob.Status.ReplicaStatuses[commonv1.ReplicaType(tfv1.TFReplicaTypeWorker)].Failed != 1 {
		t.Errorf("Failed to set the failed to 1")
	}

	err := ctr.UpdateJobStatus(tfJob, tfJob.Spec.TFReplicaSpecs, &tfJob.Status)
	if err != nil {
		t.Errorf("Expected error %v to be nil", err)
	}
	found := false
	for _, condition := range tfJob.Status.Conditions {
		if condition.Type == commonv1.JobFailed {
			found = true
		}
	}
	if !found {
		t.Errorf("Failed condition is not found")
	}
}

func TestStatus(t *testing.T) {
	type testCase struct {
		description string
		tfJob       *tfv1.TFJob

		expectedFailedPS    int32
		expectedSucceededPS int32
		expectedActivePS    int32

		expectedFailedWorker    int32
		expectedSucceededWorker int32
		expectedActiveWorker    int32

		expectedFailedChief    int32
		expectedSucceededChief int32
		expectedActiveChief    int32

		restart          bool
		worker0Completed bool

		expectedType commonv1.JobConditionType
	}

	testCases := []testCase{
		testCase{
			description:             "Chief worker is succeeded",
			tfJob:                   testutil.NewTFJobWithChief(1, 0),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  1,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobSucceeded,
		},
		testCase{
			description:             "Chief worker is running",
			tfJob:                   testutil.NewTFJobWithChief(1, 0),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     1,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "Chief worker is failed",
			tfJob:                   testutil.NewTFJobWithChief(1, 0),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     1,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "(No chief worker) Worker is failed",
			tfJob:                   testutil.NewTFJob(1, 0),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    1,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "(No chief worker) Worker is succeeded",
			tfJob:                   testutil.NewTFJob(1, 0),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobSucceeded,
		},
		testCase{
			description:             "(No chief worker) Worker is running",
			tfJob:                   testutil.NewTFJob(1, 0),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    1,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "(No chief worker) 2 workers are succeeded, 2 workers are active",
			tfJob:                   testutil.NewTFJob(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 2,
			expectedActiveWorker:    2,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "(No chief worker) 2 workers are running, 2 workers are failed",
			tfJob:                   testutil.NewTFJob(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    2,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    2,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "(No chief worker) 2 workers are succeeded, 2 workers are failed",
			tfJob:                   testutil.NewTFJob(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    2,
			expectedSucceededWorker: 2,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "(No chief worker) worker-0 are succeeded, 3 workers are active",
			tfJob:                   testutil.NewTFJob(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    3,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        true,
			expectedType:            commonv1.JobSucceeded,
		},
		testCase{
			description:             "(No chief worker, successPolicy: AllWorkers) worker-0 are succeeded, 3 workers are active",
			tfJob:                   testutil.NewTFJobWithSuccessPolicy(4, 0, tfv1.SuccessPolicyAllWorkers),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    3,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        true,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "(No chief worker, successPolicy: AllWorkers) 4 workers are succeeded",
			tfJob:                   testutil.NewTFJobWithSuccessPolicy(4, 0, tfv1.SuccessPolicyAllWorkers),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 4,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        true,
			expectedType:            commonv1.JobSucceeded,
		},
		testCase{
			description:             "(No chief worker, successPolicy: AllWorkers) worker-0 is succeeded, 2 workers are running, 1 worker is failed",
			tfJob:                   testutil.NewTFJobWithSuccessPolicy(4, 0, tfv1.SuccessPolicyAllWorkers),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    1,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    2,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        true,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "(No chief worker, failurePolicy: AllWorkers) worker-0 are failed, 3 workers are active",
			tfJob:                   testutil.NewTFJobWithFailurePolicy(4, 0, tfv1.FailurePolicyAllWorkers),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    3,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "(No chief worker, failurePolicy: AllWorkers) 4 workers are failed",
			tfJob:                   testutil.NewTFJobWithFailurePolicy(4, 0, tfv1.FailurePolicyAllWorkers),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "(No chief worker, failurePolicy: AllWorkers) worker-0 is failed, 2 workers are running, 1 worker is succeeded",
			tfJob:                   testutil.NewTFJobWithFailurePolicy(4, 0, tfv1.FailurePolicyAllWorkers),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    1,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    2,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "(failurePolicy: Chief) chief is failed, 4 workers are running",
			tfJob:                   testutil.NewTFJobWithChiefAndFailurePolicy(4, 0, tfv1.FailurePolicyChief),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    4,
			expectedFailedChief:     1,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "(failurePolicy: Chief) chief is running, 4 workers are failed",
			tfJob:                   testutil.NewTFJobWithChiefAndFailurePolicy(4, 0, tfv1.FailurePolicyChief),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        0,
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     1,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "Chief is running, workers are failed",
			tfJob:                   testutil.NewTFJobWithChief(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     1,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "Chief is running, workers are succeeded",
			tfJob:                   testutil.NewTFJobWithChief(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 4,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     1,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobRunning,
		},
		testCase{
			description:             "Chief is running, a PS is failed",
			tfJob:                   testutil.NewTFJobWithChief(4, 2),
			expectedFailedPS:        1,
			expectedSucceededPS:     0,
			expectedActivePS:        1,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 4,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  0,
			expectedActiveChief:     1,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "Chief is failed, workers are succeeded",
			tfJob:                   testutil.NewTFJobWithChief(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    0,
			expectedSucceededWorker: 4,
			expectedActiveWorker:    0,
			expectedFailedChief:     1,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobFailed,
		},
		testCase{
			description:             "Chief is succeeded, workers are failed",
			tfJob:                   testutil.NewTFJobWithChief(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     0,
			expectedSucceededChief:  1,
			expectedActiveChief:     0,
			restart:                 false,
			worker0Completed:        false,
			expectedType:            commonv1.JobSucceeded,
		},
		testCase{
			description:             "Chief is failed and restarting",
			tfJob:                   testutil.NewTFJobWithChief(4, 2),
			expectedFailedPS:        0,
			expectedSucceededPS:     0,
			expectedActivePS:        2,
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedChief:     1,
			expectedSucceededChief:  0,
			expectedActiveChief:     0,
			restart:                 true,
			worker0Completed:        false,
			expectedType:            commonv1.JobRestarting,
		},
	}

	for i, c := range testCases {
		// Prepare the clientset and controller for the test.
		kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &v1.SchemeGroupVersion,
			},
		},
		)

		// Prepare the volcano clientset and controller for the test.
		volcanoClientSet := volcanoclient.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &batchv1beta1.SchemeGroupVersion,
			},
		},
		)

		config := &rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &tfv1.SchemeGroupVersion,
			},
		}
		tfJobClientSet := tfjobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newTFController(config, kubeClientSet, volcanoClientSet, tfJobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
		fakePodControl := &controller.FakePodControl{}
		ctr.PodControl = fakePodControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.tfJobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		tfJobIndexer := ctr.tfJobInformer.GetIndexer()
		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()

		stopCh := make(chan struct{})
		run := func(<-chan struct{}) {
			if err := ctr.Run(testutil.ThreadCount, stopCh); err != nil {
				t.Errorf("Failed to run the controller: %v", err)
			}
		}
		go run(stopCh)

		unstructured, err := testutil.ConvertTFJobToUnstructured(c.tfJob)
		if err != nil {
			t.Errorf("Failed to convert the TFJob to Unstructured: %v", err)
		}

		if err := tfJobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add tfjob to tfJobIndexer: %v", err)
		}

		initializeReplicaStatuses(&c.tfJob.Status, tfv1.TFReplicaTypeWorker)
		initializeReplicaStatuses(&c.tfJob.Status, tfv1.TFReplicaTypeChief)
		initializeReplicaStatuses(&c.tfJob.Status, tfv1.TFReplicaTypePS)

		setStatusForTest(c.tfJob, tfv1.TFReplicaTypePS, c.expectedFailedPS, c.expectedSucceededPS, c.expectedActivePS, c.restart, c.worker0Completed, podIndexer, t)
		setStatusForTest(c.tfJob, tfv1.TFReplicaTypeWorker, c.expectedFailedWorker, c.expectedSucceededWorker, c.expectedActiveWorker, c.restart, c.worker0Completed, podIndexer, t)
		setStatusForTest(c.tfJob, tfv1.TFReplicaTypeChief, c.expectedFailedChief, c.expectedSucceededChief, c.expectedActiveChief, c.restart, c.worker0Completed, podIndexer, t)

		// err = ctr.UpdateJobStatus(c.tfJob, c.tfJob.Spec.TFReplicaSpecs, &c.tfJob.Status)
		// if err != nil {
		// 	t.Errorf("%s: Expected error %v to be nil", c.description, err)
		// }
		_ = ctr.ReconcileJobs(c.tfJob, c.tfJob.Spec.TFReplicaSpecs, c.tfJob.Status, &c.tfJob.Spec.RunPolicy)

		// Test filterOutCondition
		filterOutConditionTest(c.tfJob.Status, t)

		found := false
		for _, condition := range c.tfJob.Status.Conditions {
			if condition.Type == c.expectedType {
				found = true
			}
		}
		if !found {
			t.Errorf("Case[%d]%s: Condition %s is not found", i, c.description, c.expectedType)
		}
	}
}

func setStatusForTest(tfJob *tfv1.TFJob, rtype commonv1.ReplicaType, failed, succeeded, active int32, restart bool, worker0Completed bool, podIndexer cache.Indexer, t *testing.T) {
	if restart == true {
		tfJob.Spec.TFReplicaSpecs[rtype].RestartPolicy = commonv1.RestartPolicyExitCode
	}

	var typ string
	switch rtype {
	case tfv1.TFReplicaTypeWorker:
		typ = testutil.LabelWorker
	case tfv1.TFReplicaTypePS:
		typ = testutil.LabelPS
	case tfv1.TFReplicaTypeChief:
		typ = testutil.LabelChief
	default:
		fmt.Println("wrong type")
	}

	var i int32
	index := 0
	for i = 0; i < succeeded; i++ {
		pod := testutil.NewPod(tfJob, typ, index)
		pod.Status.Phase = v1.PodSucceeded
		if worker0Completed == true && rtype == tfv1.TFReplicaTypeWorker && index == 0 {
			pod.Status.ContainerStatuses = []v1.ContainerStatus{
				{
					Name: tfv1.DefaultContainerName,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: int32(0), // exit with 0
						},
					},
				},
			}
		}
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
		}
		updateJobReplicaStatuses(&tfJob.Status, rtype, pod)

		index++
	}
	for i = 0; i < failed; i++ {
		pod := testutil.NewPod(tfJob, typ, index)
		pod.Status.Phase = v1.PodFailed
		if restart == true {
			if pod.Status.ContainerStatuses == nil {
				pod.Status.ContainerStatuses = []v1.ContainerStatus{
					{
						Name: tfv1.DefaultContainerName,
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: int32(130), // 130 is a retryable code
							},
						},
					},
				}
			}
		}
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
		}
		updateJobReplicaStatuses(&tfJob.Status, rtype, pod)
		index++
	}
	for i = 0; i < active; i++ {
		pod := testutil.NewPod(tfJob, typ, index)
		pod.Status.Phase = v1.PodRunning
		if err := podIndexer.Add(pod); err != nil {
			t.Errorf("%s: unexpected error when adding pod %v", tfJob.Name, err)
		}
		updateJobReplicaStatuses(&tfJob.Status, rtype, pod)
		index++
	}
}

func filterOutConditionTest(status commonv1.JobStatus, t *testing.T) {
	flag := isFailed(status) || isSucceeded(status)
	for _, condition := range status.Conditions {
		if flag && condition.Type == commonv1.JobRunning && condition.Status == v1.ConditionTrue {
			t.Error("Error condition status when succeeded or failed")
		}
	}
}
