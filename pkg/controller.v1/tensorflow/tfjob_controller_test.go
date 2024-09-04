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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	tftestutil "github.com/kubeflow/training-operator/pkg/controller.v1/tensorflow/testutil"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

var _ = Describe("TFJob controller", func() {
	Context("Test Normal Path", func() {
		It("should create desired Pods and Services", func() {
			var (
				tfJobRunning   = kubeflowv1.JobRunning
				tfJobSucceeded = kubeflowv1.JobSucceeded
			)

			testCases := map[string]struct {
				worker int
				ps     int

				// pod setup
				// ControllerError error
				// jobKeyForget    bool

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

				expectedCondition       *kubeflowv1.JobConditionType
				expectedConditionReason string

				// There are some cases that should not check start time since the field should be set in the previous sync loop.
				needCheckStartTime bool
			}{
				"Local TFJob is created": {
					1, 0,
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
					0, 4, 0, 0,
					0, 2, 0, 0,
					4, 2,
					0, 0, 0,
					4, 0, 0,
					2, 0, 0,
					&tfJobRunning, commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobRunningReason),
					true,
				},
				"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending": {
					4, 2,
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
					2, 1, 0, 0,
					1, 0, 0, 0,
					3, 1,
					2, 0, 2,
					1, 0, 0,
					0, 0, 0,
					&tfJobRunning, commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobRunningReason),
					false,
				},
				"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is succeeded": {
					4, 2,
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
					0, 0, 4, 0,
					0, 0, 2, 0,
					4, 2,
					0, 0, 0,
					0, 4, 0,
					0, 2, 0,
					&tfJobSucceeded, commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobSucceededReason),
					false,
				},
			}

			jobNameTemplate := "test-case-norm-%d"
			caseIdx := 0
			for name, tc := range testCases {
				By(name)
				ctx := context.Background()
				jobName := fmt.Sprintf(jobNameTemplate, caseIdx)
				caseIdx++

				tfJob := tftestutil.NewTFJob(tc.worker, tc.ps)
				tfJob.SetName(jobName)
				tfJob.SetUID(uuid.NewUUID())

				refs := []metav1.OwnerReference{*reconciler.GenOwnerReference(tfJob)}
				basicLabels := reconciler.GenLabels(tfJob.GetName())

				tftestutil.SetPodsStatuses(testK8sClient, tfJob, kubeflowv1.TFJobReplicaTypeWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, nil, refs, basicLabels)
				tftestutil.SetPodsStatuses(testK8sClient, tfJob, kubeflowv1.TFJobReplicaTypePS, tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods, nil, refs, basicLabels)

				tftestutil.SetServices(testK8sClient, tfJob, kubeflowv1.TFJobReplicaTypeWorker, tc.activeWorkerServices, refs, basicLabels)
				tftestutil.SetServices(testK8sClient, tfJob, kubeflowv1.TFJobReplicaTypePS, tc.activePSServices, refs, basicLabels)

				totalPodNumber := int(tc.pendingWorkerPods + tc.activeWorkerPods + tc.succeededWorkerPods + tc.failedWorkerPods + tc.pendingPSPods + tc.activePSPods + tc.succeededPSPods + tc.failedPSPods)
				totalServiceNumber := int(tc.activeWorkerServices + tc.activePSServices)

				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: reconciler.GenLabels(tfJob.GetName())})
				Expect(err).Should(BeNil())
				listOpt := client.MatchingLabelsSelector{Selector: selector}
				Eventually(func() error {
					podList := &corev1.PodList{}
					svcList := &corev1.ServiceList{}

					err = testK8sClient.List(ctx, podList, listOpt)
					if err != nil {
						return err
					}
					if len(podList.Items) != totalPodNumber {
						return fmt.Errorf("expected %d Pods, got %d", totalPodNumber, len(podList.Items))
					}

					err = testK8sClient.List(ctx, svcList, listOpt)
					if err != nil {
						return err
					}
					if len(svcList.Items) != totalServiceNumber {
						return fmt.Errorf("expected %d Services, got %d", totalServiceNumber, len(svcList.Items))
					}
					return nil
				}).Should(BeNil())

				_ = reconciler.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy)

				// Check the number of Pods and Services
				//var pods []*corev1.Pod = nil
				//var svcs []*corev1.Service = nil
				Eventually(func() error {
					podList := &corev1.PodList{}
					svcList := &corev1.ServiceList{}

					err = testK8sClient.List(ctx, podList, listOpt)
					if err != nil {
						return err
					}
					podCreatedNumber := 0
					if len(podList.Items) > totalPodNumber {
						podCreatedNumber = len(podList.Items) - totalPodNumber
					}
					podDeletedNumber := 0
					if len(podList.Items) < totalPodNumber {
						podDeletedNumber = totalPodNumber - len(podList.Items)
					}
					if podCreatedNumber != int(tc.expectedPodCreations) {
						return fmt.Errorf("%s: unexpected number of pod creates.  Expected %d, saw %d\n", name, tc.expectedPodCreations, podCreatedNumber)
					}
					if podDeletedNumber != int(tc.expectedPodDeletions) {
						return fmt.Errorf("%s: unexpected number of service creates.  Expected %d, saw %d\n", name, tc.expectedServiceCreations, podDeletedNumber)
					}
					// check controller references for all pods
					for _, p := range podList.Items {
						for _, ref := range p.GetOwnerReferences() {
							if ref.APIVersion != kubeflowv1.SchemeGroupVersion.String() {
								return fmt.Errorf("controllerRef.APIVersion = %q, want %q", ref.APIVersion, kubeflowv1.SchemeGroupVersion.String())
							}
							if ref.Kind != kubeflowv1.TFJobKind {
								return fmt.Errorf("controllerRef.MPIKind = %q, want %q", ref.Kind, kubeflowv1.TFJobKind)
							}
							if ref.Name != tfJob.GetName() {
								return fmt.Errorf("controllerRef.Name = %q, want %q", ref.Name, tfJob.GetName())
							}
							if ref.UID != tfJob.GetUID() {
								return fmt.Errorf("controllerRef.UID = %q, want %q", ref.UID, tfJob.GetUID())
							}
						}
					}

					err = testK8sClient.List(ctx, svcList, listOpt)
					if err != nil {
						return err
					}
					serviceCreatedNumber := 0
					if len(svcList.Items) > totalServiceNumber {
						serviceCreatedNumber = len(svcList.Items) - totalServiceNumber
					}
					if serviceCreatedNumber != int(tc.expectedServiceCreations) {
						return fmt.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", name, tc.expectedPodDeletions, serviceCreatedNumber)
					}
					// check controller reference for all services
					for _, s := range svcList.Items {
						for _, ref := range s.GetOwnerReferences() {
							if ref.APIVersion != kubeflowv1.SchemeGroupVersion.String() {
								return fmt.Errorf("controllerRef.APIVersion = %q, want %q", ref.APIVersion, kubeflowv1.SchemeGroupVersion.String())
							}
							if ref.Kind != kubeflowv1.TFJobKind {
								return fmt.Errorf("controllerRef.MPIKind = %q, want %q", ref.Kind, kubeflowv1.TFJobKind)
							}
							if ref.Name != tfJob.GetName() {
								return fmt.Errorf("controllerRef.Name = %q, want %q", ref.Name, tfJob.GetName())
							}
							if ref.UID != tfJob.GetUID() {
								return fmt.Errorf("controllerRef.UID = %q, want %q", ref.UID, tfJob.GetUID())
							}
						}
					}
					return nil
				}).Should(BeNil())

				// Validate Worker status
				if tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker] != nil {
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Active).To(Equal(tc.expectedActiveWorkerPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Succeeded).To(Equal(tc.expectedSucceededWorkerPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Failed).To(Equal(tc.expectedFailedWorkerPods))
				}
				// Validate PS status
				if tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS] != nil {
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS].Active).To(Equal(tc.expectedActivePSPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS].Succeeded).To(Equal(tc.expectedSucceededPSPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS].Failed).To(Equal(tc.expectedFailedPSPods))
				}

				// Validate StartTime
				if tc.needCheckStartTime {
					Expect(tfJob.Status.StartTime).NotTo(BeNil())
				}

				// Validate Conditions
				if tc.expectedCondition != nil {
					Expect(tftestutil.CheckCondition(tfJob, *tc.expectedCondition, tc.expectedConditionReason)).Should(BeTrue())
				}
			}
		})
	})

	Context("TFJob with suspend semantics", func() {
		const name = "test-job"
		var (
			ns         *corev1.Namespace
			job        *kubeflowv1.TFJob
			jobKey     types.NamespacedName
			chiefKey   types.NamespacedName
			worker0Key types.NamespacedName
			ctx        = context.Background()
		)
		BeforeEach(func() {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "tensorflow-test-",
				},
			}
			Expect(testK8sClient.Create(ctx, ns)).Should(Succeed())

			// chief=1, worker=1
			job = tftestutil.NewTFJobV2(1, 0, 0, 1, 0)
			job.SetName(name)
			job.SetNamespace(ns.Name)
			jobKey = client.ObjectKeyFromObject(job)
			chiefKey = types.NamespacedName{
				Name:      fmt.Sprintf("%s-chief-0", name),
				Namespace: ns.Name,
			}
			worker0Key = types.NamespacedName{
				Name:      fmt.Sprintf("%s-worker-0", name),
				Namespace: ns.Name,
			}
		})
		AfterEach(func() {
			Expect(testK8sClient.Delete(ctx, job)).Should(Succeed())
			Expect(testK8sClient.Delete(ctx, ns)).Should(Succeed())
		})

		It("Shouldn't create resources if TFJob is suspended", func() {
			By("By creating a new TFJob with suspend=true")
			job.Spec.RunPolicy.Suspend = ptr.To(true)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.TFJob{}
			chiefPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			chiefSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			By("Checking created TFJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			By("Checking created TFJob has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the pods and services aren't created")
			Consistently(func() bool {
				errChiefPod := testK8sClient.Get(ctx, chiefKey, chiefPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errChiefSvc := testK8sClient.Get(ctx, chiefKey, chiefSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errChiefPod) && errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errChiefSvc) && errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the TFJob has suspended condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("TFJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("TFJob %s is suspended.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))
		})

		It("Should delete resources after TFJob is suspended; Should resume TFJob after TFJob is unsuspended", func() {
			By("By creating a new TFJob")
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.TFJob{}
			chiefPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			chiefSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			// We'll need to retry getting this newly created TFJob, given that creation may not immediately happen.
			By("Checking created TFJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			var startTimeBeforeSuspended *metav1.Time
			Eventually(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				startTimeBeforeSuspended = created.Status.StartTime
				return startTimeBeforeSuspended
			}, testutil.Timeout, testutil.Interval).ShouldNot(BeNil())

			By("Checking the created pods and services")
			Eventually(func() bool {
				errChief := testK8sClient.Get(ctx, chiefKey, chiefPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errChief == nil && errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errChief := testK8sClient.Get(ctx, chiefKey, chiefSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errChief == nil && errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Updating the pod's phase with Running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, chiefKey, chiefPod)).Should(Succeed())
				chiefPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, chiefPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking the TFJob's condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("TFJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("TFJob %s/%s is running.", ns.Name, name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Updating the TFJob with suspend=true")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = ptr.To(true)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the pods and services are removed")
			Eventually(func() bool {
				errChief := testK8sClient.Get(ctx, chiefKey, chiefPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errors.IsNotFound(errChief) && errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errChief := testK8sClient.Get(ctx, chiefKey, chiefSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errChief) && errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				errChiefPod := testK8sClient.Get(ctx, chiefKey, chiefPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errChiefSvc := testK8sClient.Get(ctx, chiefKey, chiefSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errChiefPod) && errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errChiefSvc) && errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the TFJob has a suspended condition")
			Eventually(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeChief].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeChief].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Expect(created.Status.Conditions).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("TFJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionFalse,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("TFJob %s is suspended.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("TFJob %s is suspended.", name),
					Status:  corev1.ConditionTrue,
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Unsuspending the TFJob")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = ptr.To(false)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.Timeout, testutil.Interval).ShouldNot(BeNil())

			By("Check if the pods and services are created")
			Eventually(func() error {
				return testK8sClient.Get(ctx, chiefKey, chiefPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, chiefKey, chiefSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

			By("Updating Pod's condition with running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, chiefKey, chiefPod)).Should(Succeed())
				chiefPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, chiefPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the TFJob has resumed conditions")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("TFJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobResumedReason),
					Message: fmt.Sprintf("TFJob %s is resumed.", name),
					Status:  corev1.ConditionFalse,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.TFJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("TFJob %s/%s is running.", ns.Name, name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Checking if the startTime is updated")
			Expect(created.Status.StartTime).ShouldNot(Equal(startTimeBeforeSuspended))
		})

		It("Should not reconcile a job while managed by external controller", func() {
			By("Creating a TFJob managed by external controller")
			job.Spec.RunPolicy = kubeflowv1.RunPolicy{
				ManagedBy: ptr.To(kubeflowv1.MultiKueueController),
			}
			job.Spec.RunPolicy.Suspend = ptr.To(true)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.TFJob{}
			By("Checking created TFJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Checking created TFJob has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the pods and services aren't created")
			Consistently(func() bool {
				chiefPod := &corev1.Pod{}
				workerPod := &corev1.Pod{}
				chiefSvc := &corev1.Service{}
				workerSvc := &corev1.Service{}
				errMasterPod := testK8sClient.Get(ctx, chiefKey, chiefPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errMasterSvc := testK8sClient.Get(ctx, chiefKey, chiefSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMasterPod) && errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errMasterSvc) && errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue(), "pods and services should be created by external controller (here not existent)")

			By("Checking if the TFJob status was not updated")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition(nil)))

			By("Unsuspending the TFJob")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = ptr.To(false)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking created TFJob still has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the TFJob status was not updated, even after unsuspending")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition(nil)))
		})
	})
})
