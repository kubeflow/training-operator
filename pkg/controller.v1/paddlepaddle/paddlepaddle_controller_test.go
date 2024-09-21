// Copyright 2022 The Kubeflow Authors
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

package paddle

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

var _ = Describe("PaddleJob controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		expectedPort = int32(8080)
	)
	Context("When creating the PaddleJob", func() {
		const name = "test-job"
		var (
			ctx        = context.Background()
			ns         *corev1.Namespace
			job        *kubeflowv1.PaddleJob
			jobKey     types.NamespacedName
			masterKey  types.NamespacedName
			worker0Key types.NamespacedName
		)
		BeforeEach(func() {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "paddle-test-",
				},
			}
			Expect(testK8sClient.Create(ctx, ns)).Should(Succeed())

			job = newPaddleJobForTest(name, ns.Name)
			jobKey = client.ObjectKeyFromObject(job)
			masterKey = types.NamespacedName{
				Name:      fmt.Sprintf("%s-master-0", name),
				Namespace: ns.Name,
			}
			worker0Key = types.NamespacedName{
				Name:      fmt.Sprintf("%s-worker-0", name),
				Namespace: ns.Name,
			}
			job.Spec.PaddleReplicaSpecs = map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
				kubeflowv1.PaddleJobReplicaTypeMaster: {
					Replicas: ptr.To[int32](1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.PaddleJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.PaddleJobDefaultPortName,
											ContainerPort: expectedPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
								},
							},
						},
					},
				},
				kubeflowv1.PaddleJobReplicaTypeWorker: {
					Replicas: ptr.To[int32](2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.PaddleJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.PaddleJobDefaultPortName,
											ContainerPort: expectedPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
								},
							},
						},
					},
				},
			}
		})
		AfterEach(func() {
			Expect(testK8sClient.Delete(ctx, job)).Should(Succeed())
			Expect(testK8sClient.Delete(ctx, ns)).Should(Succeed())
		})
		It("Should get the corresponding resources successfully", func() {
			By("By creating a new PaddleJob")
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PaddleJob{}

			// We'll need to retry getting this newly created PaddleJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			masterPod := &corev1.Pod{}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, masterKey, masterPod)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			masterSvc := &corev1.Service{}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, masterKey, masterSvc)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			// Check the pod port.
			Expect(masterPod.Spec.Containers[0].Ports).To(ContainElement(corev1.ContainerPort{
				Name:          kubeflowv1.PaddleJobDefaultPortName,
				ContainerPort: expectedPort,
				Protocol:      corev1.ProtocolTCP}))
			// Check env variable
			Expect(masterPod.Spec.Containers[0].Env).To(ContainElements(corev1.EnvVar{
				Name:  EnvMasterEndpoint,
				Value: fmt.Sprintf("$(POD_IP_DUMMY):%d", expectedPort),
			}))
			// Check service port.
			Expect(masterSvc.Spec.Ports[0].Port).To(Equal(expectedPort))
			// Check owner reference.
			trueVal := true
			Expect(masterPod.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PaddleJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))
			Expect(masterSvc.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PaddleJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))

			// Test job status.
			masterPod.Status.Phase = corev1.PodSucceeded
			masterPod.ResourceVersion = ""
			Expect(testK8sClient.Status().Update(ctx, masterPod)).Should(Succeed())
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				if err != nil {
					return false
				}
				return created.Status.ReplicaStatuses != nil && created.Status.
					ReplicaStatuses[kubeflowv1.PaddleJobReplicaTypeMaster].Succeeded == 1
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			// Check if the job is succeeded.
			cond := getCondition(created.Status, kubeflowv1.JobSucceeded)
			Expect(cond.Status).To(Equal(corev1.ConditionTrue))
		})
		It("Shouldn't create resources if PaddleJob is suspended", func() {
			By("By creating a new PaddleJob with suspend=true")
			job.Spec.RunPolicy.Suspend = ptr.To(true)
			job.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeWorker].Replicas = ptr.To[int32](1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PaddleJob{}
			masterPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			masterSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			By("Checking created PaddleJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			By("Checking created PaddleJob has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the pods and services aren't created")
			Consistently(func() bool {
				errMasterPod := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errMasterSvc := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMasterPod) && errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errMasterSvc) && errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the PaddleJob has suspended condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PaddleJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("PaddleJob %s is suspended.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))
		})

		It("Should delete resources after PaddleJob is suspended; Should resume PaddleJob after PaddleJob is unsuspended", func() {
			By("By creating a new PaddleJob")
			job.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeWorker].Replicas = ptr.To[int32](1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PaddleJob{}
			masterPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			masterSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			// We'll need to retry getting this newly created PaddleJob, given that creation may not immediately happen.
			By("Checking created PaddleJob")
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
				errMaster := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errMaster == nil && errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errMaster == nil && errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Updating the pod's phase with Running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, masterKey, masterPod)).Should(Succeed())
				masterPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, masterPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking the PaddleJob's condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PaddleJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("PaddleJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Updating the PaddleJob with suspend=true")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = ptr.To(true)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the pods and services are removed")
			Eventually(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				errMasterPod := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errMasterSvc := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMasterPod) && errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errMasterSvc) && errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the PaddleJob has a suspended condition")
			Eventually(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.PaddleJobReplicaTypeMaster].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.PaddleJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.PaddleJobReplicaTypeMaster].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.PaddleJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Expect(created.Status.Conditions).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PaddleJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionFalse,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("PaddleJob %s is suspended.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("PaddleJob %s is suspended.", name),
					Status:  corev1.ConditionTrue,
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Unsuspending the PaddleJob")
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
				return testK8sClient.Get(ctx, masterKey, masterPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, masterKey, masterSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

			By("Updating Pod's condition with running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, masterKey, masterPod)).Should(Succeed())
				masterPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, masterPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the PaddleJob has resumed conditions")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PaddleJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobResumedReason),
					Message: fmt.Sprintf("PaddleJob %s is resumed.", name),
					Status:  corev1.ConditionFalse,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PaddleJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("PaddleJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Checking if the startTime is updated")
			Expect(created.Status.StartTime).ShouldNot(Equal(startTimeBeforeSuspended))
		})

		It("Should not reconcile a job while managed by external controller", func() {
			By("Creating a PaddleJob managed by external controller")
			job.Spec.RunPolicy = kubeflowv1.RunPolicy{
				ManagedBy: ptr.To(kubeflowv1.MultiKueueController),
			}
			job.Spec.RunPolicy.Suspend = ptr.To(true)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PaddleJob{}
			By("Checking created PaddleJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Checking created PaddleJob has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the pods and services aren't created")
			Consistently(func() bool {
				masterPod := &corev1.Pod{}
				workerPod := &corev1.Pod{}
				masterSvc := &corev1.Service{}
				workerSvc := &corev1.Service{}
				errMasterPod := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errMasterSvc := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMasterPod) && errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errMasterSvc) && errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue(), "pods and services should be created by external controller (here not existent)")

			By("Checking if the PaddleJob status was not updated")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition(nil)))

			By("Unsuspending the PaddleJob")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = ptr.To(false)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking created PaddleJob still has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the PaddleJob status was not updated, even after unsuspending")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition(nil)))
		})
	})
})

func newPaddleJobForTest(name, namespace string) *kubeflowv1.PaddleJob {
	return &kubeflowv1.PaddleJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status kubeflowv1.JobStatus, condType kubeflowv1.JobConditionType) *kubeflowv1.JobCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}
