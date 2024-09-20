// Copyright 2024 The Kubeflow Authors
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

package jax

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

var _ = Describe("JAXJob controller", func() {
	// Define utility constants for object names.
	const (
		expectedPort = int32(6666)
	)

	Context("When creating the JAXJob", func() {
		const name = "test-job"
		var (
			ns         *corev1.Namespace
			job        *kubeflowv1.JAXJob
			jobKey     types.NamespacedName
			worker0Key types.NamespacedName
			ctx        = context.Background()
		)
		BeforeEach(func() {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "jax-test-",
				},
			}
			Expect(testK8sClient.Create(ctx, ns)).Should(Succeed())

			job = &kubeflowv1.JAXJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: ns.Name,
				},
			}
			jobKey = client.ObjectKeyFromObject(job)

			worker0Key = types.NamespacedName{
				Name:      fmt.Sprintf("%s-worker-0", name),
				Namespace: ns.Name,
			}
			job.Spec.JAXReplicaSpecs = map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
				kubeflowv1.JAXJobReplicaTypeWorker: {
					Replicas: ptr.To[int32](2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.JAXJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.JAXJobDefaultPortName,
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

		It("Shouldn't create resources if JAXJob is suspended", func() {
			By("By creating a new JAXJob with suspend=true")
			job.Spec.RunPolicy.Suspend = ptr.To(true)
			job.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker].Replicas = ptr.To[int32](1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.JAXJob{}
			workerPod := &corev1.Pod{}
			workerSvc := &corev1.Service{}

			By("Checking created JAXJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			By("Checking created JAXJob has a nil startTime")
			Consistently(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeNil())

			By("Checking if the pods and services aren't created")
			Consistently(func() bool {
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the JAXJob has suspended condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("JAXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("JAXJob %s is suspended.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))
		})

		It("Should delete resources after JAXJob is suspended; Should resume JAXJob after JAXJob is unsuspended", func() {
			By("By creating a new JAXJob")
			job.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker].Replicas = ptr.To[int32](1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.JAXJob{}
			workerPod := &corev1.Pod{}
			workerSvc := &corev1.Service{}

			// We'll need to retry getting this newly created JAXJob, given that creation may not immediately happen.
			By("Checking created JAXJob")
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
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Updating the pod's phase with Running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking the JAXJob's condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("JAXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("JAXJob %s/%s is running.", ns.Name, name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Updating the JAXJob with suspend=true")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = ptr.To(true)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the pods and services are removed")
			Eventually(func() bool {
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				errWorkerPod := testK8sClient.Get(ctx, worker0Key, workerPod)
				errWorkerSvc := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errWorkerPod) &&
					errors.IsNotFound(errWorkerSvc)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the JAXJob has a suspended condition")
			Eventually(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.JAXJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.JAXJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Expect(created.Status.Conditions).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("JAXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionFalse,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("JAXJob %s is suspended.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("JAXJob %s is suspended.", name),
					Status:  corev1.ConditionTrue,
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Unsuspending the JAXJob")
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
				return testK8sClient.Get(ctx, worker0Key, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

			By("Updating Pod's condition with running")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, worker0Key, workerPod)).Should(Succeed())
				workerPod.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the JAXJob has resumed conditions")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("JAXJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobResumedReason),
					Message: fmt.Sprintf("JAXJob %s is resumed.", name),
					Status:  corev1.ConditionFalse,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.JAXJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("JAXJob %s/%s is running.", ns.Name, name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Checking if the startTime is updated")
			Expect(created.Status.StartTime).ShouldNot(Equal(startTimeBeforeSuspended))
		})
	})
})
