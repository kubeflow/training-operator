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
	"strconv"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/controller.v1/common"
	tftestutil "github.com/kubeflow/training-operator/pkg/controller.v1/tensorflow/testutil"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

var _ = Describe("TFJob controller", func() {
	Context("Test Add TFJob", func() {
		It("should get the exact TFJob", func() {
			By("submitting an TFJob")

			testJobName := "test-case-12"
			testNamespace := metav1.NamespaceDefault

			decoyJobName := "decoy-case-34"

			ctx := context.Background()

			tfJob := tftestutil.NewTFJob(1, 0)
			tfJob.SetName(testJobName)
			tfJob.SetNamespace(testNamespace)

			decoyJob := tftestutil.NewTFJob(2, 3)
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
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

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
			tfjob := tftestutil.NewTFJob(1, 0)
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
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
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
					tfJob:       tftestutil.NewTFJobWithCleanPolicy(0, 4, 2, kubeflowv1.CleanPodPolicyAll),

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
					tfJob:       tftestutil.NewTFJobWithCleanPolicy(0, 4, 2, kubeflowv1.CleanPodPolicyRunning),

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
					tfJob:       tftestutil.NewTFJobWithCleanPolicy(0, 4, 2, kubeflowv1.CleanPodPolicyRunning),

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
					tfJob:       tftestutil.NewTFJobWithCleanPolicy(0, 4, 2, kubeflowv1.CleanPodPolicyNone),

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
				commonutil.UpdateJobConditions(&tc.tfJob.Status, kubeflowv1.JobSucceeded, corev1.ConditionTrue, commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobSucceededReason), "")

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
				tftestutil.SetPodsStatuses(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypeWorker,
					tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods,
					nil, refs, basicLabels)
				tftestutil.SetPodsStatuses(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypePS,
					tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods,
					nil, refs, basicLabels)

				tftestutil.SetServices(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypeWorker, tc.activeWorkerServices, refs, basicLabels)
				tftestutil.SetServices(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypePS, tc.activePSServices, refs, basicLabels)

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
					tfJob:       tftestutil.NewTFJobWithActiveDeadlineSeconds(0, 4, 2, nil),

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
					tfJob:       tftestutil.NewTFJobWithActiveDeadlineSeconds(0, 4, 2, adsTest2),

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
				tftestutil.SetPodsStatuses(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypeWorker,
					tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods,
					nil, refs, basicLabels)
				tftestutil.SetPodsStatuses(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypePS,
					tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods,
					nil, refs, basicLabels)

				tftestutil.SetServices(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypeWorker, tc.activeWorkerServices, refs, basicLabels)
				tftestutil.SetServices(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypePS, tc.activePSServices, refs, basicLabels)

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
					tfJob:       tftestutil.NewTFJobWithBackoffLimit(0, 4, 2, backoffLimitTest4),

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
				tftestutil.SetPodsStatuses(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypeWorker,
					tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods,
					tc.restartCounts, refs, basicLabels)
				tftestutil.SetPodsStatuses(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypePS,
					tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods,
					tc.restartCounts, refs, basicLabels)

				tftestutil.SetServices(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypeWorker, tc.activeWorkerServices, refs, basicLabels)
				tftestutil.SetServices(testK8sClient, tc.tfJob, kubeflowv1.TFJobReplicaTypePS, tc.activePSServices, refs, basicLabels)

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
					tfJob:       tftestutil.NewTFJobWithCleanupJobDelay(0, 1, 0, ptr.To[int32](3)),
					phase:       corev1.PodSucceeded,
				},
				{
					description: "failed job with TTL 3s",
					tfJob:       tftestutil.NewTFJobWithCleanupJobDelay(0, 1, 0, ptr.To[int32](3)),
					phase:       corev1.PodFailed,
				},
			}
			jobNameTemplate := "test-bof-%d"
			for idx, tc := range testCases {
				By(fmt.Sprintf("preparing cases %s", tc.description))
				ctx := context.Background()
				name := fmt.Sprintf(jobNameTemplate, idx)
				tc.tfJob.SetName(name)
				tc.tfJob.CreationTimestamp = metav1.Now()

				By("creating a TFJob")
				Expect(reconciler.Create(ctx, tc.tfJob)).Should(Succeed())

				// We need to wait for synchronizing cache.
				By("getting a created TFJob")
				var updatedTFJob kubeflowv1.TFJob
				Eventually(func() error {
					return reconciler.Get(ctx, client.ObjectKeyFromObject(tc.tfJob), &updatedTFJob)
				}, testutil.Timeout, testutil.Interval).Should(BeNil())

				initializeReplicaStatuses(&updatedTFJob.Status, kubeflowv1.TFJobReplicaTypeWorker)

				By("prepare pod")
				refs := []metav1.OwnerReference{
					*reconciler.GenOwnerReference(tc.tfJob),
				}
				pod := tftestutil.NewBasePod("pod", tc.tfJob, refs)
				pod.Status.Phase = tc.phase

				By("update job replica statuses")
				updateJobReplicaStatuses(&updatedTFJob.Status, kubeflowv1.TFJobReplicaTypeWorker, pod)

				By("update job status")
				Expect(reconciler.UpdateJobStatus(&updatedTFJob, updatedTFJob.Spec.TFReplicaSpecs, &updatedTFJob.Status)).To(Succeed())
				By("updating job status...")
				Expect(reconciler.Status().Update(ctx, &updatedTFJob)).To(Succeed())

				By("waiting for updating replicaStatus for workers")
				Eventually(func() *kubeflowv1.ReplicaStatus {
					var getTFJob kubeflowv1.TFJob
					Expect(reconciler.Get(ctx, client.ObjectKeyFromObject(tc.tfJob), &getTFJob)).Should(Succeed())
					return getTFJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker]
				}, testutil.Timeout, testutil.Interval).ShouldNot(BeNil())

				ttl := updatedTFJob.Spec.RunPolicy.TTLSecondsAfterFinished
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
				}, testutil.Timeout, testutil.Interval).Should(BeNil())
			}
		})
	})
})

var _ = Describe("Test for controller.v1/common", func() {
	var (
		ctx = context.Background()
		ns  *corev1.Namespace
		now metav1.Time
	)
	BeforeEach(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "tfjob-ns-",
			},
		}
		now = metav1.Now()
		Expect(testK8sClient.Create(ctx, ns)).Should(Succeed())
	})
	AfterEach(func() {
		Expect(testK8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	type cleanUpCases struct {
		tfJob              *kubeflowv1.TFJob
		runPolicy          *kubeflowv1.RunPolicy
		jobStatus          kubeflowv1.JobStatus
		wantTFJobIsRemoved bool
		wantErr            bool
	}
	DescribeTable("TFJob is created and is cleaned up",
		func(tc *cleanUpCases) {
			tc.tfJob.SetNamespace(ns.Name)
			Expect(testK8sClient.Create(ctx, tc.tfJob)).Should(Succeed())

			if tc.wantErr {
				Expect(reconciler.CleanupJob(tc.runPolicy, tc.jobStatus, tc.tfJob)).ShouldNot(Succeed())
			} else {
				Expect(reconciler.CleanupJob(tc.runPolicy, tc.jobStatus, tc.tfJob)).Should(Succeed())
			}
			if tc.wantTFJobIsRemoved {
				Eventually(func() bool {
					gotErr := testK8sClient.Get(ctx, client.ObjectKeyFromObject(tc.tfJob), &kubeflowv1.TFJob{})
					return errors.IsNotFound(gotErr)
				}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			} else {
				Eventually(func() error {
					return testK8sClient.Get(ctx, client.ObjectKeyFromObject(tc.tfJob), &kubeflowv1.TFJob{})
				}, testutil.Timeout, testutil.Interval).Should(BeNil())
			}
		},
		Entry("TFJob shouldn't be removed since TTL is nil", &cleanUpCases{
			tfJob: tftestutil.NewTFJobWithCleanupJobDelay(1, 2, 0, nil),
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: nil,
			},
			jobStatus:          kubeflowv1.JobStatus{},
			wantTFJobIsRemoved: false,
			wantErr:            false,
		}),
		Entry("No error with completionTime is nil if suspended", &cleanUpCases{
			tfJob: tftestutil.NewTFJobWithCleanupJobDelay(1, 2, 0, nil),
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: nil,
				Suspend: ptr.To(true),
			},
			jobStatus: kubeflowv1.JobStatus{
				CompletionTime: nil,
			},
			wantTFJobIsRemoved: false,
			wantErr:            false,
		}),
		Entry("Error is occurred since completionTime is nil", &cleanUpCases{
			tfJob: tftestutil.NewTFJobWithCleanupJobDelay(1, 2, 0, ptr.To[int32](10)),
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: ptr.To[int32](10),
			},
			jobStatus: kubeflowv1.JobStatus{
				CompletionTime: nil,
			},
			wantTFJobIsRemoved: false,
			wantErr:            true,
		}),
		Entry("TFJob is removed since exceeded TTL (TTL is 180s)", &cleanUpCases{
			tfJob: tftestutil.NewTFJobWithCleanupJobDelay(1, 2, 0, ptr.To[int32](180)),
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: ptr.To[int32](180),
			},
			jobStatus: kubeflowv1.JobStatus{
				CompletionTime: &metav1.Time{
					Time: now.AddDate(0, 0, -1),
				},
			},
			wantTFJobIsRemoved: true,
			wantErr:            false,
		}),
		Entry("TFJob is removed since (TTL is 0s)", &cleanUpCases{
			tfJob: tftestutil.NewTFJobWithCleanupJobDelay(1, 2, 0, ptr.To[int32](0)),
			runPolicy: &kubeflowv1.RunPolicy{
				TTLSecondsAfterFinished: ptr.To[int32](0),
			},
			jobStatus: kubeflowv1.JobStatus{
				CompletionTime: &now,
			},
			wantTFJobIsRemoved: true,
			wantErr:            false,
		}),
	)

	type createServiceCases struct {
		tfJob   *kubeflowv1.TFJob
		rType   kubeflowv1.ReplicaType
		spec    *kubeflowv1.ReplicaSpec
		uid     types.UID
		index   int
		wantErr bool
	}
	DescribeTable("CreateNewService",
		func(tc *createServiceCases) {
			tc.tfJob.SetUID(tc.uid)
			tc.tfJob.SetNamespace(ns.Name)

			gotErr := reconciler.CreateNewService(tc.tfJob, tc.rType, tc.spec, strconv.Itoa(tc.index))
			if tc.wantErr {
				Expect(gotErr).ShouldNot(Succeed())
			} else {
				Expect(gotErr).Should(Succeed())

				svcInternalTPC := corev1.ServiceInternalTrafficPolicyCluster
				svcSingleStack := corev1.IPFamilyPolicySingleStack
				wantSvc := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s-%d", tc.tfJob.Name, tc.rType, tc.index),
						Namespace: ns.Name,
						OwnerReferences: []metav1.OwnerReference{
							*reconciler.GenOwnerReference(tc.tfJob),
						},
						Labels: map[string]string{
							kubeflowv1.JobNameLabel:      tc.tfJob.Name,
							kubeflowv1.OperatorNameLabel: controllerName,
							kubeflowv1.ReplicaIndexLabel: strconv.Itoa(tc.index),
							kubeflowv1.ReplicaTypeLabel:  "",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:     kubeflowv1.TFJobDefaultPortName,
								Protocol: corev1.ProtocolTCP,
								Port:     kubeflowv1.TFJobDefaultPort,
								TargetPort: intstr.IntOrString{
									IntVal: kubeflowv1.TFJobDefaultPort,
								},
							},
						},
						Selector: map[string]string{
							kubeflowv1.JobNameLabel:      tc.tfJob.Name,
							kubeflowv1.OperatorNameLabel: controllerName,
							kubeflowv1.ReplicaIndexLabel: strconv.Itoa(tc.index),
							kubeflowv1.ReplicaTypeLabel:  "",
						},
						ClusterIP:             corev1.ClusterIPNone,
						Type:                  corev1.ServiceTypeClusterIP,
						ClusterIPs:            []string{corev1.ClusterIPNone},
						SessionAffinity:       corev1.ClusterIPNone,
						IPFamilies:            []corev1.IPFamily{corev1.IPv4Protocol},
						IPFamilyPolicy:        &svcSingleStack,
						InternalTrafficPolicy: &svcInternalTPC,
					},
				}
				Eventually(func() *corev1.Service {
					svc := &corev1.Service{}
					Expect(testK8sClient.Get(ctx, client.ObjectKeyFromObject(wantSvc), svc)).Should(Succeed())
					return svc
				}, testutil.Timeout, testutil.Interval).Should(BeComparableTo(wantSvc,
					cmpopts.IgnoreFields(metav1.ObjectMeta{}, "UID", "ResourceVersion", "Generation", "CreationTimestamp", "ManagedFields")))
			}
		},
		Entry("Failed to create service since containerPort is missing", &createServiceCases{
			tfJob: tftestutil.NewTFJobV2(2, 0, 0, 1, 0),
			spec: &kubeflowv1.ReplicaSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: kubeflowv1.TFJobDefaultContainerName,
							},
						},
					},
				},
			},
			index:   0,
			wantErr: true,
		}),
		Entry("Failed to create service since Job's ownerReference is invalid", &createServiceCases{
			tfJob:   tftestutil.NewTFJobV2(2, 0, 0, 1, 0),
			spec:    &kubeflowv1.ReplicaSpec{Template: tftestutil.NewTFReplicaSpecTemplate()},
			index:   1,
			wantErr: true,
		}),
		Entry("Succeeded to create service", &createServiceCases{
			tfJob:   tftestutil.NewTFJobV2(2, 0, 0, 1, 0),
			spec:    &kubeflowv1.ReplicaSpec{Template: tftestutil.NewTFReplicaSpecTemplate()},
			index:   0,
			wantErr: false,
			uid:     uuid.NewUUID(),
		}),
	)
})
