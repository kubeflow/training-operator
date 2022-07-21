// Copyright 2021 The Kubeflow Authors
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
	"context"
	"fmt"
	"time"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/common/util/v1/testutil"
)

var _ = Describe("TFJob controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = 10 * time.Second
		interval = 1000 * time.Millisecond
	)

	Context("Test Failed", func() {
		It("should update TFJob with failed status", func() {
			By("creating a TFJob with replicaStatues initialized")
			tfJob := testutil.NewTFJob(3, 0)
			initializeReplicaStatuses(&tfJob.Status, kubeflowv1.TFJobReplicaTypeWorker)

			By("prepare pod")
			refs := []metav1.OwnerReference{
				*reconciler.GenOwnerReference(tfJob),
			}
			pod := testutil.NewBasePod("pod", tfJob, refs)
			pod.Status.Phase = v1.PodFailed

			By("update job replica statuses")
			updateJobReplicaStatuses(&tfJob.Status, kubeflowv1.TFJobReplicaTypeWorker, pod)
			Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Failed).Should(Equal(int32(1)))

			By("update job status")
			Expect(reconciler.UpdateJobStatus(tfJob, tfJob.Spec.TFReplicaSpecs, &tfJob.Status)).To(Succeed())

			By("finding failed job status")
			found := false
			for _, condition := range tfJob.Status.Conditions {
				if condition.Type == commonv1.JobFailed {
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Context("Test Status", func() {
		It("should update TFJob with desired status", func() {
			type testCase struct {
				description string
				tfJob       *kubeflowv1.TFJob

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
				{
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
				{
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
				{
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
				{
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
				{
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
				{
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
				{
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
				{
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
				{
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
				{
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
				{
					description:             "(No chief worker, successPolicy: AllWorkers) worker-0 are succeeded, 3 workers are active",
					tfJob:                   testutil.NewTFJobWithSuccessPolicy(4, 0, kubeflowv1.SuccessPolicyAllWorkers),
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
				{
					description:             "(No chief worker, successPolicy: AllWorkers) 4 workers are succeeded",
					tfJob:                   testutil.NewTFJobWithSuccessPolicy(4, 0, kubeflowv1.SuccessPolicyAllWorkers),
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
				{
					description:             "(No chief worker, successPolicy: AllWorkers) worker-0 is succeeded, 2 workers are running, 1 worker is failed",
					tfJob:                   testutil.NewTFJobWithSuccessPolicy(4, 0, kubeflowv1.SuccessPolicyAllWorkers),
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
				{
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
				{
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
				{
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
				{
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
				{
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
				{
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

			jobNameTemplate := "test-status-%d"
			for i, c := range testCases {
				reconciler.Log.Info("testing case", "description", c.description)
				c.tfJob.SetName(fmt.Sprintf(jobNameTemplate, i))
				c.tfJob.SetUID(uuid.NewUUID())

				initializeReplicaStatuses(&c.tfJob.Status, kubeflowv1.TFJobReplicaTypeWorker)
				initializeReplicaStatuses(&c.tfJob.Status, kubeflowv1.TFJobReplicaTypeChief)
				initializeReplicaStatuses(&c.tfJob.Status, kubeflowv1.TFJobReplicaTypePS)

				setStatusForTest(c.tfJob, kubeflowv1.TFJobReplicaTypePS, c.expectedFailedPS, c.expectedSucceededPS, c.expectedActivePS, c.restart, c.worker0Completed, testK8sClient)
				setStatusForTest(c.tfJob, kubeflowv1.TFJobReplicaTypeWorker, c.expectedFailedWorker, c.expectedSucceededWorker, c.expectedActiveWorker, c.restart, c.worker0Completed, testK8sClient)
				setStatusForTest(c.tfJob, kubeflowv1.TFJobReplicaTypeChief, c.expectedFailedChief, c.expectedSucceededChief, c.expectedActiveChief, c.restart, c.worker0Completed, testK8sClient)

				// Adding this section to make sure all pods are created and cached
				Eventually(func() error {
					podList := &corev1.PodList{}
					basicLabels := reconciler.GenLabels(c.tfJob.GetName())
					selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
						MatchLabels: basicLabels,
					})
					if err != nil {
						return err
					}
					listOpt := client.MatchingLabelsSelector{
						Selector: selector,
					}
					err = testK8sClient.List(context.Background(), podList, listOpt)
					if err != nil {
						return nil
					}
					totalExpectedPodCount := c.expectedFailedPS + c.expectedSucceededPS + c.expectedActivePS +
						c.expectedFailedWorker + c.expectedSucceededWorker + c.expectedActiveWorker +
						c.expectedFailedChief + c.expectedSucceededChief + c.expectedActiveChief
					if len(podList.Items) != int(totalExpectedPodCount) {
						return fmt.Errorf("pod number (%d) for %s not match for expected pod number %d",
							len(podList.Items), c.tfJob.GetName(), totalExpectedPodCount)
					}
					return nil
				}, timeout, interval).Should(BeNil())

				_ = reconciler.ReconcileJobs(c.tfJob, c.tfJob.Spec.TFReplicaSpecs, c.tfJob.Status, &c.tfJob.Spec.RunPolicy)

				Expect(filterOutConditionTest(c.tfJob.Status)).Should(Succeed())

				reconciler.Log.Info("checking status", "tfJob.Status", c.tfJob.Status)
				found := false
				for _, condition := range c.tfJob.Status.Conditions {
					if condition.Type == c.expectedType {
						found = true
					}
				}
				Expect(found).To(BeTrue())
				reconciler.Log.Info("passed!",
					"job name", c.tfJob.GetName(), "job description", c.description)
			}
		})
	})
})

func setStatusForTest(tfJob *kubeflowv1.TFJob, rtype commonv1.ReplicaType, failed, succeeded, active int32, restart bool, worker0Completed bool, client client.Client) {
	if restart == true {
		tfJob.Spec.TFReplicaSpecs[rtype].RestartPolicy = commonv1.RestartPolicyExitCode
	}

	basicLabels := reconciler.GenLabels(tfJob.GetName())

	const (
		timeout  = 10 * time.Second
		interval = 1000 * time.Millisecond
	)

	ctx := context.Background()

	var typ string
	switch rtype {
	case kubeflowv1.TFJobReplicaTypeWorker:
		typ = testutil.LabelWorker
	case kubeflowv1.TFJobReplicaTypePS:
		typ = testutil.LabelPS
	case kubeflowv1.TFJobReplicaTypeChief:
		typ = testutil.LabelChief
	default:
		fmt.Println("wrong type")
	}
	Expect(typ).ShouldNot(Equal(""))

	refs := []metav1.OwnerReference{
		*reconciler.GenOwnerReference(tfJob),
	}

	var i int32
	index := 0
	for i = 0; i < succeeded; i++ {
		pod := testutil.NewPod(tfJob, typ, index, refs)
		for k, v := range basicLabels {
			pod.Labels[k] = v
		}
		po := &corev1.Pod{}
		Expect(client.Create(ctx, pod)).Should(Succeed())

		key := genKeyFromJob(pod)
		Eventually(func() error {
			po = &corev1.Pod{}
			if err := client.Get(ctx, key, po); err != nil {
				return err
			}

			po.Status.Phase = corev1.PodSucceeded
			if worker0Completed == true && rtype == kubeflowv1.TFJobReplicaTypeWorker && index == 0 {
				po.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: reconciler.GetDefaultContainerName(),
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								ExitCode: int32(0), // exit with 0
							},
						},
					},
				}
			}

			return client.Status().Update(ctx, po)
		}, timeout, interval).Should(BeNil())

		updateJobReplicaStatuses(&tfJob.Status, rtype, po)

		index++
	}

	for i = 0; i < failed; i++ {
		pod := testutil.NewPod(tfJob, typ, index, refs)
		for k, v := range basicLabels {
			pod.Labels[k] = v
		}
		po := &corev1.Pod{}
		Expect(client.Create(ctx, pod)).Should(Succeed())

		key := genKeyFromJob(pod)
		Eventually(func() error {
			po = &corev1.Pod{}
			if err := client.Get(ctx, key, po); err != nil {
				return err
			}

			po.Status.Phase = corev1.PodFailed
			if restart == true {
				if po.Status.ContainerStatuses == nil {
					po.Status.ContainerStatuses = []corev1.ContainerStatus{
						{
							Name: reconciler.GetDefaultContainerName(),
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: int32(130), // 130 is a retryable code
								},
							},
						},
					}
				}
			}

			return client.Status().Update(ctx, po)
		}, timeout, interval).Should(BeNil())

		updateJobReplicaStatuses(&tfJob.Status, rtype, po)
		index++
	}

	for i = 0; i < active; i++ {
		pod := testutil.NewPod(tfJob, typ, index, refs)
		for k, v := range basicLabels {
			pod.Labels[k] = v
		}
		po := &corev1.Pod{}
		Expect(client.Create(ctx, pod)).Should(Succeed())

		key := genKeyFromJob(pod)
		Eventually(func() error {
			po = &corev1.Pod{}
			if err := client.Get(ctx, key, po); err != nil {
				return err
			}

			po.Status.Phase = corev1.PodRunning

			return client.Status().Update(ctx, po)
		}, timeout, interval).Should(BeNil())

		updateJobReplicaStatuses(&tfJob.Status, rtype, po)
		index++
	}
}

func genKeyFromJob(job client.Object) types.NamespacedName {
	ns := metav1.NamespaceDefault
	if job.GetNamespace() != "" {
		ns = job.GetNamespace()
	}
	return types.NamespacedName{
		Namespace: ns,
		Name:      job.GetName(),
	}
}

func filterOutConditionTest(status commonv1.JobStatus) error {
	flag := util.IsFailed(status) || util.IsSucceeded(status)
	for _, condition := range status.Conditions {
		if flag && condition.Type == commonv1.JobRunning && condition.Status == corev1.ConditionTrue {
			return fmt.Errorf("error condition status when succeeded or failed")
		}
	}
	return nil
}
