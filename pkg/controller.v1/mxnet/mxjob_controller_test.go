// Copyright 2023 The Kubeflow Authors
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

package mxnet

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

// TODO: we should implement more tests.
var _ = Describe("MXJob controller", func() {
	const (
		expectedPort = int32(9091)
	)
	Context("When creating the MXJob", func() {
		const name = "test-job"
		var (
			ns           *corev1.Namespace
			job          *kubeflowv1.MXJob
			jobKey       types.NamespacedName
			serverKey    types.NamespacedName
			worker0Key   types.NamespacedName
			schedulerKey types.NamespacedName
			ctx          = context.Background()
		)
		BeforeEach(func() {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "mxjob-test-",
				},
			}
			Expect(testK8sClient.Create(ctx, ns)).Should(Succeed())

			job = newMXJobForTest(name, ns.Name)
			jobKey = client.ObjectKeyFromObject(job)
			serverKey = types.NamespacedName{
				Name:      fmt.Sprintf("%s-server-0", name),
				Namespace: ns.Name,
			}
			worker0Key = types.NamespacedName{
				Name:      fmt.Sprintf("%s-worker-0", name),
				Namespace: ns.Name,
			}
			schedulerKey = types.NamespacedName{
				Name:      fmt.Sprintf("%s-scheduler-0", name),
				Namespace: ns.Name,
			}
			job.Spec.MXReplicaSpecs = map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
				kubeflowv1.MXJobReplicaTypeServer: {
					Replicas: pointer.Int32(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.MXJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.MXJobDefaultPortName,
											ContainerPort: expectedPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
								},
							},
						},
					},
				},
				kubeflowv1.MXJobReplicaTypeScheduler: {
					Replicas: pointer.Int32(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.MXJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.MXJobDefaultPortName,
											ContainerPort: expectedPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
								},
							},
						},
					},
				},
				kubeflowv1.MXJobReplicaTypeWorker: {
					Replicas: pointer.Int32(2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.MXJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.MXJobDefaultPortName,
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
		It("Shouldn't create resources when MXJob is suspended; Should create resources once MXJob is unsuspended", func() {
			By("By creating a new MXJob with suspend=true")
			job.Spec.RunPolicy.Suspend = pointer.Bool(true)
			job.Spec.MXReplicaSpecs[kubeflowv1.MXJobReplicaTypeWorker].Replicas = pointer.Int32(1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.MXJob{}
			serverPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			schedulerPod := &corev1.Pod{}
			serverSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}
			schedulerSvc := &corev1.Service{}

			By("Checking created MXJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			By("Checking created MXJob has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the pods and services aren't created")
			Consistently(func() bool {
				errServerPod := testK8sClient.Get(ctx, serverKey, serverPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errSchedulerPod := testK8sClient.Get(ctx, schedulerKey, schedulerPod)
				errServerSvc := testK8sClient.Get(ctx, serverKey, serverSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				errSchedulerSvc := testK8sClient.Get(ctx, schedulerKey, schedulerSvc)
				return errors.IsNotFound(errServerPod) && errors.IsNotFound(errWorkerPod) && errors.IsNotFound(errSchedulerPod) &&
					errors.IsNotFound(errServerSvc) && errors.IsNotFound(errWorkerSvc) && errors.IsNotFound(errSchedulerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the MXJob has suspended condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("MXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("MXJob %s is suspended.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))
		})

		It("Should delete resources after MXJob is suspended; Should resume MXJob after MXJob is unsuspended", func() {
			By("By creating a new MXJob")
			job.Spec.MXReplicaSpecs[kubeflowv1.MXJobReplicaTypeWorker].Replicas = pointer.Int32(1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.MXJob{}
			serverPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			schedulerPod := &corev1.Pod{}
			serverSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}
			schedulerSvc := &corev1.Service{}

			// We'll need to retry getting this newly created MXJob, given that creation may not immediately happen.
			By("Checking created MXJob")
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
				errServerPod := testK8sClient.Get(ctx, serverKey, serverPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errSchedulerPod := testK8sClient.Get(ctx, schedulerKey, schedulerPod)
				errServerSvc := testK8sClient.Get(ctx, serverKey, serverSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				errSchedulerSvc := testK8sClient.Get(ctx, schedulerKey, schedulerSvc)
				return errServerPod == nil && errWorkerPod == nil && errSchedulerPod == nil &&
					errServerSvc == nil && errWorkerSvc == nil && errSchedulerSvc == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Updating the pod's phase with Running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, serverKey, serverPod)).Should(Succeed())
				serverPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, serverPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, schedulerKey, schedulerPod)).Should(Succeed())
				schedulerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, schedulerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking the MXJob's condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("MXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("MXJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Updating the MXJob with suspend=true")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = pointer.Bool(true)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the pods and services are removed")
			Eventually(func() bool {
				errServer := testK8sClient.Get(ctx, serverKey, serverPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				errScheduler := testK8sClient.Get(ctx, schedulerKey, schedulerPod)
				return errors.IsNotFound(errServer) && errors.IsNotFound(errWorker) && errors.IsNotFound(errScheduler)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errServer := testK8sClient.Get(ctx, serverKey, serverSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				errScheduler := testK8sClient.Get(ctx, schedulerKey, schedulerSvc)
				return errors.IsNotFound(errServer) && errors.IsNotFound(errWorker) && errors.IsNotFound(errScheduler)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				errServerPod := testK8sClient.Get(ctx, serverKey, serverPod)
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errSchedulerPod := testK8sClient.Get(ctx, schedulerKey, schedulerPod)
				errServerSvc := testK8sClient.Get(ctx, serverKey, serverSvc)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				errSchedulerSvc := testK8sClient.Get(ctx, schedulerKey, schedulerSvc)
				return errors.IsNotFound(errServerPod) && errors.IsNotFound(errWorkerPod) && errors.IsNotFound(errSchedulerPod) &&
					errors.IsNotFound(errServerSvc) && errors.IsNotFound(errWorkerSvc) && errors.IsNotFound(errSchedulerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the MXJob has a suspended condition")
			Eventually(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.MXJobReplicaTypeServer].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.MXJobReplicaTypeWorker].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.MXJobReplicaTypeScheduler].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.MXJobReplicaTypeServer].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.MXJobReplicaTypeWorker].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.MXJobReplicaTypeScheduler].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Expect(created.Status.Conditions).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("MXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionFalse,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("MXJob %s is suspended.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("MXJob %s is suspended.", name),
					Status:  corev1.ConditionTrue,
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Unsuspending the MXJob")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = pointer.Bool(false)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.Timeout, testutil.Interval).ShouldNot(BeNil())

			By("Check if the pods and services are created")
			Eventually(func() error {
				return testK8sClient.Get(ctx, serverKey, serverPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, schedulerKey, schedulerPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, serverKey, serverSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, schedulerKey, schedulerSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

			By("Updating Pod's condition with running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, serverKey, serverPod)).Should(Succeed())
				serverPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, serverPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, schedulerKey, schedulerPod)).Should(Succeed())
				schedulerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, schedulerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the MXJob has resumed conditions")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("MXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobResumedReason),
					Message: fmt.Sprintf("MXJob %s is resumed.", name),
					Status:  corev1.ConditionFalse,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.MXJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("MXJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Checking if the startTime is updated")
			Expect(created.Status.StartTime).ShouldNot(Equal(startTimeBeforeSuspended))
		})
	})
})

func newMXJobForTest(name, namespace string) *kubeflowv1.MXJob {
	return &kubeflowv1.MXJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
