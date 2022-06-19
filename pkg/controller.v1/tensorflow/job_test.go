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
	"github.com/kubeflow/common/pkg/controller.v1/common"
	commonutil "github.com/kubeflow/common/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/pointer"
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

	Context("Test Add TFJob", func() {
		It("should get the exact TFJob", func() {
			By("submitting an TFJob")

			testJobName := "test-case-12"
			testNamespace := metav1.NamespaceDefault

			decoyJobName := "decoy-case-34"

			ctx := context.Background()

			tfJob := testutil.NewTFJob(1, 0)
			tfJob.SetName(testJobName)
			tfJob.SetNamespace(testNamespace)

			decoyJob := testutil.NewTFJob(2, 3)
			decoyJob.SetName(decoyJobName)
			decoyJob.SetNamespace(testNamespace)

			Expect(testK8sClient.Create(ctx, tfJob)).Should(Succeed())
			Expect(testK8sClient.Create(ctx, decoyJob)).Should(Succeed())

			key := types.NamespacedName{
				Namespace: testNamespace,
				Name:      testJobName,
			}
			Eventually(func() error {
				job := &kubeflowv1.TFJob{}
				return reconciler.Get(ctx, key, job)
			}, timeout, interval).Should(BeNil())

			Expect(testK8sClient.Delete(ctx, tfJob)).Should(Succeed())
			Expect(testK8sClient.Delete(ctx, decoyJob)).Should(Succeed())
		})
	})

	Context("Test Copy Labels and Annotation", func() {
		It("should copy labels and annotation from the spec to generated Pods", func() {
			ctx := context.Background()
			testAnnotationKey := "annotation1"
			testAnnotationVal := "1"
			testLabelKey := "label1"
			testLabelVal := "1"

			testJobName := "test-copy-labels-anno"
			tfjob := testutil.NewTFJob(1, 0)
			tfjob.SetName(testJobName)
			annotations := map[string]string{
				testAnnotationKey: testAnnotationVal,
			}
			labels := map[string]string{
				testLabelKey: testLabelVal,
			}
			tfjob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].Template.Labels = labels
			tfjob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].Template.Annotations = annotations

			By("submitting an TFJob with specific labels and annotations")
			Expect(testK8sClient.Create(ctx, tfjob)).Should(Succeed())

			Eventually(func() error {
				pod := &corev1.Pod{}
				key := types.NamespacedName{
					Namespace: metav1.NamespaceDefault,
					Name:      common.GenGeneralName(tfjob.Name, "worker", "0"),
				}
				err := testK8sClient.Get(ctx, key, pod)
				if err != nil {
					return err
				}

				if pod.Annotations == nil {
					return fmt.Errorf("annotation of %s/%s is nil", pod.GetNamespace(), pod.GetName())
				}
				if val, exist := pod.Annotations[testAnnotationKey]; exist {
					if val != testAnnotationVal {
						return fmt.Errorf("annotation of %s not match with %s", testAnnotationKey, testAnnotationVal)
					}
				} else {
					return fmt.Errorf("annotation %s not found", testAnnotationKey)
				}

				if pod.Labels == nil {
					return fmt.Errorf("label of %s/%s is nil", pod.GetNamespace(), pod.GetName())
				}
				if val, exist := pod.Labels[testLabelKey]; exist {
					if val != testLabelVal {
						return fmt.Errorf("annotation of %s not match with %s", testLabelKey, testLabelVal)
					}
				} else {
					return fmt.Errorf("label %s not found", testLabelKey)
				}

				return nil
			}, timeout, interval).Should(BeNil())
		})
	})

	Context("Test Delete Pods and Services", func() {
		It("it should clean associated Pods and Services according to clean policy", func() {
			type testCase struct {
				description string
				tfJob       *kubeflowv1.TFJob

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

				expectedPodRemaining int
			}

			testCases := []testCase{
				{
					description: "4 workers and 2 ps is running, policy is all",
					tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, commonv1.CleanPodPolicyAll),

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

					expectedPodRemaining: 0,
				},
				{
					description: "4 workers and 2 ps is running, policy is running",
					tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, commonv1.CleanPodPolicyRunning),

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

					expectedPodRemaining: 0,
				},
				{
					description: "4 workers and 2 ps is succeeded, policy is running",
					tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, commonv1.CleanPodPolicyRunning),

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

					expectedPodRemaining: 6,
				},
				{
					description: "4 workers and 2 ps is succeeded, policy is None",
					tfJob:       testutil.NewTFJobWithCleanPolicy(0, 4, 2, commonv1.CleanPodPolicyNone),

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

					expectedPodRemaining: 6,
				},
			}

			jobNameTemplate := "test-del-pod-svc-%d"
			for idx, tc := range testCases {
				By(fmt.Sprintf("preparing cases %s", tc.description))
				ctx := context.Background()
				tc.tfJob.SetName(fmt.Sprintf(jobNameTemplate, idx))
				tc.tfJob.SetUID(uuid.NewUUID())
				Expect(commonutil.UpdateJobConditions(&tc.tfJob.Status, commonv1.JobSucceeded, tfJobSucceededReason, "")).Should(Succeed())

				refs := []metav1.OwnerReference{
					*reconciler.GenOwnerReference(tc.tfJob),
				}

				basicLabels := reconciler.GenLabels(tc.tfJob.GetName())
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: basicLabels,
				})
				Expect(err).Should(BeNil())
				listOpt := client.MatchingLabelsSelector{
					Selector: selector,
				}

				By("creating Services and Pods with designed phases")
				testutil.SetPodsStatuses(testK8sClient, tc.tfJob, testutil.LabelWorker,
					tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods,
					nil, refs, basicLabels)
				testutil.SetPodsStatuses(testK8sClient, tc.tfJob, testutil.LabelPS,
					tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods,
					nil, refs, basicLabels)

				testutil.SetServices(testK8sClient, tc.tfJob, testutil.LabelWorker, tc.activeWorkerServices, refs, basicLabels)
				testutil.SetServices(testK8sClient, tc.tfJob, testutil.LabelPS, tc.activePSServices, refs, basicLabels)

				podList := &corev1.PodList{}
				Expect(testK8sClient.List(ctx, podList, listOpt)).Should(Succeed())
				Expect(len(podList.Items)).To(Equal(
					int(tc.pendingPSPods + tc.activePSPods + tc.failedPSPods + tc.succeededPSPods +
						tc.pendingWorkerPods + tc.activeWorkerPods + tc.failedWorkerPods + tc.succeededWorkerPods)))

				By("calling ReconcileJob")
				_ = reconciler.ReconcileJobs(tc.tfJob, tc.tfJob.Spec.TFReplicaSpecs, tc.tfJob.Status, &tc.tfJob.Spec.RunPolicy)

				podList = &corev1.PodList{}
				Expect(testK8sClient.List(ctx, podList, listOpt, client.InNamespace(tc.tfJob.GetNamespace()))).Should(Succeed())
				podRemainingCount := len(podList.Items)
				Expect(podRemainingCount).To(Equal(tc.expectedPodRemaining))

				svcList := &corev1.ServiceList{}
				Expect(testK8sClient.List(ctx, svcList, listOpt)).Should(Succeed())
				svcRemainingCount := len(svcList.Items)
				Expect(svcRemainingCount).To(Equal(tc.expectedPodRemaining))
			}
		})
	})

	Context("Test Active Deadline Seconds", func() {
		It("clean desired Pods and Services according to TFJob config", func() {
			type testCase struct {
				description string
				tfJob       *kubeflowv1.TFJob

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

				expectedPodRemaining int
			}

			ads2 := int64(2)
			adsTest2 := &ads2
			testCases := []testCase{
				{
					description: "4 workers and 2 ps is running, ActiveDeadlineSeconds unset",
					tfJob:       testutil.NewTFJobWithActiveDeadlineSeconds(0, 4, 2, nil),

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

					expectedPodRemaining: 6,
				},
				{
					description: "4 workers and 2 ps is running, ActiveDeadlineSeconds is 2",
					tfJob:       testutil.NewTFJobWithActiveDeadlineSeconds(0, 4, 2, adsTest2),

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

					expectedPodRemaining: 0,
				},
			}
			jobNameTemplate := "test-ads-%d"
			for idx, tc := range testCases {
				By(fmt.Sprintf("preparing cases %s", tc.description))
				ctx := context.Background()
				tc.tfJob.SetName(fmt.Sprintf(jobNameTemplate, idx))
				tc.tfJob.SetUID(uuid.NewUUID())

				refs := []metav1.OwnerReference{
					*reconciler.GenOwnerReference(tc.tfJob),
				}

				basicLabels := reconciler.GenLabels(tc.tfJob.GetName())
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: basicLabels,
				})
				Expect(err).Should(BeNil())
				listOpt := client.MatchingLabelsSelector{
					Selector: selector,
				}

				By("creating Services and Pods with designed phases")
				testutil.SetPodsStatuses(testK8sClient, tc.tfJob, testutil.LabelWorker,
					tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods,
					nil, refs, basicLabels)
				testutil.SetPodsStatuses(testK8sClient, tc.tfJob, testutil.LabelPS,
					tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods,
					nil, refs, basicLabels)

				testutil.SetServices(testK8sClient, tc.tfJob, testutil.LabelWorker, tc.activeWorkerServices, refs, basicLabels)
				testutil.SetServices(testK8sClient, tc.tfJob, testutil.LabelPS, tc.activePSServices, refs, basicLabels)

				podList := &corev1.PodList{}
				Expect(testK8sClient.List(ctx, podList, listOpt)).Should(Succeed())
				Expect(len(podList.Items)).To(Equal(
					int(tc.pendingPSPods + tc.activePSPods + tc.failedPSPods + tc.succeededPSPods +
						tc.pendingWorkerPods + tc.activeWorkerPods + tc.failedWorkerPods + tc.succeededWorkerPods)))

				By("waiting enough time")
				now := metav1.Now()
				tc.tfJob.Status.StartTime = &now
				ads := tc.tfJob.Spec.RunPolicy.ActiveDeadlineSeconds
				if ads != nil {
					dur := time.Second * time.Duration(*ads)
					time.Sleep(dur)
				}

				By("calling ReconcileJob")
				_ = reconciler.ReconcileJobs(tc.tfJob, tc.tfJob.Spec.TFReplicaSpecs, tc.tfJob.Status, &tc.tfJob.Spec.RunPolicy)

				podList = &corev1.PodList{}
				Expect(testK8sClient.List(ctx, podList, listOpt, client.InNamespace(tc.tfJob.GetNamespace()))).Should(Succeed())
				podRemainingCount := len(podList.Items)
				Expect(podRemainingCount).To(Equal(tc.expectedPodRemaining))

				svcList := &corev1.ServiceList{}
				Expect(testK8sClient.List(ctx, svcList, listOpt)).Should(Succeed())
				svcRemainingCount := len(svcList.Items)
				Expect(svcRemainingCount).To(Equal(tc.expectedPodRemaining))
			}
		})
	})

	Context("Test Backoff For On Failure(", func() {
		It("clean desired Pods and Services according to TFJob config", func() {
			type testCase struct {
				description string
				tfJob       *kubeflowv1.TFJob

				pendingWorkerPods   int32
				activeWorkerPods    int32
				succeededWorkerPods int32
				failedWorkerPods    int32

				restartCounts []int32

				pendingPSPods   int32
				activePSPods    int32
				succeededPSPods int32
				failedPSPods    int32

				activeWorkerServices int32
				activePSServices     int32

				expectedPodRemaining int
			}

			backoffLimit4 := int32(4)
			backoffLimitTest4 := &backoffLimit4
			testCases := []testCase{
				{
					description: "4 workers each having 1 restartCount and 2 ps is running, backoffLimit 4 ",
					tfJob:       testutil.NewTFJobWithBackoffLimit(0, 4, 2, backoffLimitTest4),

					pendingWorkerPods:   0,
					activeWorkerPods:    4,
					succeededWorkerPods: 0,
					failedWorkerPods:    0,

					restartCounts: []int32{1, 1, 1, 1},

					pendingPSPods:   0,
					activePSPods:    2,
					succeededPSPods: 0,
					failedPSPods:    0,

					activeWorkerServices: 4,
					activePSServices:     2,

					expectedPodRemaining: 0,
				},
			}

			jobNameTemplate := "test-bof-%d"
			for idx, tc := range testCases {
				By(fmt.Sprintf("preparing cases %s", tc.description))
				ctx := context.Background()
				tc.tfJob.SetName(fmt.Sprintf(jobNameTemplate, idx))
				tc.tfJob.SetUID(uuid.NewUUID())

				refs := []metav1.OwnerReference{
					*reconciler.GenOwnerReference(tc.tfJob),
				}

				basicLabels := reconciler.GenLabels(tc.tfJob.GetName())
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: basicLabels,
				})
				Expect(err).Should(BeNil())
				listOpt := client.MatchingLabelsSelector{
					Selector: selector,
				}

				By("creating Services and Pods with designed phases")
				testutil.SetPodsStatuses(testK8sClient, tc.tfJob, testutil.LabelWorker,
					tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods,
					tc.restartCounts, refs, basicLabels)
				testutil.SetPodsStatuses(testK8sClient, tc.tfJob, testutil.LabelPS,
					tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods,
					tc.restartCounts, refs, basicLabels)

				testutil.SetServices(testK8sClient, tc.tfJob, testutil.LabelWorker, tc.activeWorkerServices, refs, basicLabels)
				testutil.SetServices(testK8sClient, tc.tfJob, testutil.LabelPS, tc.activePSServices, refs, basicLabels)

				podList := &corev1.PodList{}
				Expect(testK8sClient.List(ctx, podList, listOpt)).Should(Succeed())
				Expect(len(podList.Items)).To(Equal(
					int(tc.pendingPSPods + tc.activePSPods + tc.failedPSPods + tc.succeededPSPods +
						tc.pendingWorkerPods + tc.activeWorkerPods + tc.failedWorkerPods + tc.succeededWorkerPods)))

				By("calling ReconcileJob")
				_ = reconciler.ReconcileJobs(tc.tfJob, tc.tfJob.Spec.TFReplicaSpecs, tc.tfJob.Status, &tc.tfJob.Spec.RunPolicy)

				podList = &corev1.PodList{}
				Expect(testK8sClient.List(ctx, podList, listOpt, client.InNamespace(tc.tfJob.GetNamespace()))).Should(Succeed())
				podRemainingCount := len(podList.Items)
				Expect(podRemainingCount).To(Equal(tc.expectedPodRemaining))

				svcList := &corev1.ServiceList{}
				Expect(testK8sClient.List(ctx, svcList, listOpt)).Should(Succeed())
				svcRemainingCount := len(svcList.Items)
				Expect(svcRemainingCount).To(Equal(tc.expectedPodRemaining))
			}
		})
	})

	Context("Test TTL Seconds After Finished", func() {
		It("should delete job when expired time is up", func() {
			type testCase struct {
				description string
				tfJob       *kubeflowv1.TFJob
				phase       corev1.PodPhase
			}
			testCases := []testCase{
				{
					description: "succeeded job with TTL 3s",
					tfJob:       testutil.NewTFJobWithCleanupJobDelay(0, 1, 0, pointer.Int32(3)),
					phase:       corev1.PodSucceeded,
				},
				{
					description: "failed job with TTL 3s",
					tfJob:       testutil.NewTFJobWithCleanupJobDelay(0, 1, 0, pointer.Int32(3)),
					phase:       corev1.PodFailed,
				},
			}
			jobNameTemplate := "test-bof-%d"
			for idx, tc := range testCases {
				By(fmt.Sprintf("preparing cases %s", tc.description))
				ctx := context.Background()
				name := fmt.Sprintf(jobNameTemplate, idx)
				tc.tfJob.SetName(name)

				By("creating a TFJob")
				Expect(reconciler.Create(ctx, tc.tfJob)).Should(Succeed())

				initializeReplicaStatuses(&tc.tfJob.Status, kubeflowv1.TFJobReplicaTypeWorker)

				By("prepare pod")
				refs := []metav1.OwnerReference{
					*reconciler.GenOwnerReference(tc.tfJob),
				}
				pod := testutil.NewBasePod("pod", tc.tfJob, refs)
				pod.Status.Phase = tc.phase

				By("update job replica statuses")
				updateJobReplicaStatuses(&tc.tfJob.Status, kubeflowv1.TFJobReplicaTypeWorker, pod)

				By("update job status")
				Expect(reconciler.UpdateJobStatus(tc.tfJob, tc.tfJob.Spec.TFReplicaSpecs, &tc.tfJob.Status)).To(Succeed())
				Expect(reconciler.Status().Update(ctx, tc.tfJob)).Should(Succeed())

				ttl := tc.tfJob.Spec.RunPolicy.TTLSecondsAfterFinished
				if ttl != nil {
					dur := time.Second * time.Duration(*ttl)
					time.Sleep(dur)
				}

				Eventually(func() error {
					tfJob := &kubeflowv1.TFJob{}
					key := types.NamespacedName{
						Namespace: metav1.NamespaceDefault,
						Name:      name,
					}
					if err := reconciler.Get(ctx, key, tfJob); err != nil {
						if errors.IsNotFound(err) {
							return nil
						}
						return err
					}
					return fmt.Errorf("job %s still remains", name)
				}, timeout, interval).Should(BeNil())
			}
		})
	})

})
