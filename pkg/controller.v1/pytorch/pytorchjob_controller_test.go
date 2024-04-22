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

package pytorch

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

var _ = Describe("PyTorchJob controller", func() {
	// Define utility constants for object names.
	const (
		expectedPort = int32(8080)
	)

	Context("When creating the PyTorchJob", func() {
		const name = "test-job"
		var (
			ns         *corev1.Namespace
			job        *kubeflowv1.PyTorchJob
			jobKey     types.NamespacedName
			masterKey  types.NamespacedName
			worker0Key types.NamespacedName
			ctx        = context.Background()
		)
		BeforeEach(func() {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "pytorch-test-",
				},
			}
			Expect(testK8sClient.Create(ctx, ns)).Should(Succeed())

			job = newPyTorchJobForTest(name, ns.Name)
			jobKey = client.ObjectKeyFromObject(job)
			masterKey = types.NamespacedName{
				Name:      fmt.Sprintf("%s-master-0", name),
				Namespace: ns.Name,
			}
			worker0Key = types.NamespacedName{
				Name:      fmt.Sprintf("%s-worker-0", name),
				Namespace: ns.Name,
			}
			job.Spec.NprocPerNode = nil
			job.Spec.PyTorchReplicaSpecs = map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
				kubeflowv1.PyTorchJobReplicaTypeMaster: {
					Replicas: ptr.To[int32](1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.PyTorchJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.PyTorchJobDefaultPortName,
											ContainerPort: expectedPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
								},
							},
						},
					},
				},
				kubeflowv1.PyTorchJobReplicaTypeWorker: {
					Replicas: ptr.To[int32](2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.PyTorchJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.PyTorchJobDefaultPortName,
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
			By("By creating a new PyTorchJob")
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PyTorchJob{}

			// We'll need to retry getting this newly created PyTorchJob, given that creation may not immediately happen.
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
				Name:          kubeflowv1.PyTorchJobDefaultPortName,
				ContainerPort: expectedPort,
				Protocol:      corev1.ProtocolTCP}))
			// Check env variable
			Expect(masterPod.Spec.Containers[0].Env).To(ContainElements(corev1.EnvVar{
				Name:  EnvMasterPort,
				Value: fmt.Sprintf("%d", masterSvc.Spec.Ports[0].Port),
			}, corev1.EnvVar{
				Name:  EnvMasterAddr,
				Value: masterSvc.Name,
			}, corev1.EnvVar{
				Name:  EnvNprocPerNode,
				Value: kubeflowv1.DefaultNprocPerNode,
			}))
			// Check service port.
			Expect(masterSvc.Spec.Ports[0].Port).To(Equal(expectedPort))
			// Check owner reference.
			trueVal := true
			Expect(masterPod.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PyTorchJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))
			Expect(masterSvc.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PyTorchJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))

			// Test job status.
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, masterKey, masterPod)).Should(Succeed())
				masterPod.Status.Phase = corev1.PodSucceeded
				return testK8sClient.Status().Update(ctx, masterPod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				if err != nil {
					return false
				}
				return created.Status.ReplicaStatuses != nil && created.Status.
					ReplicaStatuses[kubeflowv1.PyTorchJobReplicaTypeMaster].Succeeded == 1
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			// Check if the job is succeeded.
			cond := getCondition(created.Status, kubeflowv1.JobSucceeded)
			Expect(cond.Status).To(Equal(corev1.ConditionTrue))
		})
		It("Shouldn't be updated resources if spec.runPolicy.schedulingPolicy.queue is changed after the job is created", func() {
			By("Creating a PyTorchJob with a specific queue")
			job.Spec.RunPolicy.SchedulingPolicy = &kubeflowv1.SchedulingPolicy{}
			job.Spec.RunPolicy.SchedulingPolicy.Queue = "initial-queue"
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			By("Attempting to update the PyTorchJob with a different queue value")
			Eventually(func(g Gomega) {
				updatedJob := &kubeflowv1.PyTorchJob{}
				g.Expect(testK8sClient.Get(ctx, jobKey, updatedJob)).Should(Succeed(), "Failed to get PyTorchJob")
				updatedJob.Spec.RunPolicy.SchedulingPolicy.Queue = "test"
				err := testK8sClient.Update(ctx, updatedJob)
				g.Expect(err).To(HaveOccurred(), "Expected an error when updating the queue, but update succeeded")
				By("Checking that the queue update fails")
				Expect(err).To(MatchError(ContainSubstring("spec.runPolicy.schedulingPolicy.queue is immutable"), "The error message did not contain the expected message"))

			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Validating the queue was not updated")
			freshJob := &kubeflowv1.PyTorchJob{}
			Expect(testK8sClient.Get(ctx, client.ObjectKeyFromObject(job), freshJob)).Should(Succeed(), "Failed to get PyTorchJob after update attempt")
			Expect(freshJob.Spec.RunPolicy.SchedulingPolicy.Queue).To(Equal("initial-queue"), "The queue should remain as the initial value since it should be immutable")
		})

		It("Shouldn't create resources if PyTorchJob is suspended", func() {
			By("By creating a new PyTorchJob with suspend=true")
			job.Spec.RunPolicy.Suspend = ptr.To(true)
			job.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeWorker].Replicas = ptr.To[int32](1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PyTorchJob{}
			masterPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			masterSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			By("Checking created PyTorchJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			By("Checking created PyTorchJob has a nil startTime")
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

			By("Checking if the PyTorchJob has suspended condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PyTorchJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("PyTorchJob %s is suspended.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))
		})

		It("Should delete resources after PyTorchJob is suspended; Should resume PyTorchJob after PyTorchJob is unsuspended", func() {
			By("By creating a new PyTorchJob")
			job.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeWorker].Replicas = ptr.To[int32](1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PyTorchJob{}
			masterPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			masterSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			// We'll need to retry getting this newly created PyTorchJob, given that creation may not immediately happen.
			By("Checking created PyTorchJob")
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

			By("Checking the PyTorchJob's condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PyTorchJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("PyTorchJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Updating the PyTorchJob with suspend=true")
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

			By("Checking if the PyTorchJob has a suspended condition")
			Eventually(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.PyTorchJobReplicaTypeMaster].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.PyTorchJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.PyTorchJobReplicaTypeMaster].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.PyTorchJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime.Equal(startTimeBeforeSuspended)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Expect(created.Status.Conditions).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PyTorchJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionFalse,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("PyTorchJob %s is suspended.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobSuspendedReason),
					Message: fmt.Sprintf("PyTorchJob %s is suspended.", name),
					Status:  corev1.ConditionTrue,
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Unsuspending the PyTorchJob")
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

			By("Checking if the PyTorchJob has resumed conditions")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobCreatedReason),
					Message: fmt.Sprintf("PyTorchJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobResumedReason),
					Message: fmt.Sprintf("PyTorchJob %s is resumed.", name),
					Status:  corev1.ConditionFalse,
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.NewReason(kubeflowv1.PyTorchJobKind, commonutil.JobRunningReason),
					Message: fmt.Sprintf("PyTorchJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Checking if the startTime is updated")
			Expect(created.Status.StartTime).ShouldNot(Equal(startTimeBeforeSuspended))
		})
	})

	Context("When creating the elastic PyTorchJob", func() {
		const name = "elastic-job"
		var (
			ctx         = context.Background()
			ns          *corev1.Namespace
			job         *kubeflowv1.PyTorchJob
			jobKey      types.NamespacedName
			workerKey   types.NamespacedName
			backendC10D = kubeflowv1.BackendC10D
			minReplicas = int32(1)
			maxReplicas = int32(3)
			maxRestarts = int32(3)
		)
		BeforeEach(func() {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "elastic-pytorch-test-",
				},
			}
			Expect(testK8sClient.Create(ctx, ns))

			job = newPyTorchJobForTest(name, ns.Name)
			jobKey = client.ObjectKeyFromObject(job)
			workerKey = types.NamespacedName{
				Name:      fmt.Sprintf("%s-worker-0", name),
				Namespace: ns.Name,
			}
			// Define the expected elastic policy.
			job.Spec.ElasticPolicy = &kubeflowv1.ElasticPolicy{
				RDZVBackend: &backendC10D,
				MinReplicas: &minReplicas,
				MaxReplicas: &maxReplicas,
				MaxRestarts: &maxRestarts,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:         autoscalingv2.UtilizationMetricType,
								AverageValue: resource.NewQuantity(80, resource.DecimalSI),
							},
						},
					},
				},
			}
			job.Spec.PyTorchReplicaSpecs = map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
				kubeflowv1.PyTorchJobReplicaTypeWorker: {
					Replicas: ptr.To[int32](1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.PyTorchJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.PyTorchJobDefaultPortName,
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
		// TODO(gaocegege): Test with more than 1 worker.
		It("Should get the corresponding resources successfully", func() {
			By("By creating a new PyTorchJob")
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PyTorchJob{}

			// We'll need to retry getting this newly created PyTorchJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			pod := &corev1.Pod{}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, workerKey, pod)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			svc := &corev1.Service{}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, workerKey, svc)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return testK8sClient.Get(ctx, jobKey, hpa)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

			// Check pod port.
			Expect(pod.Spec.Containers[0].Ports).To(ContainElement(corev1.ContainerPort{
				Name:          kubeflowv1.PyTorchJobDefaultPortName,
				ContainerPort: expectedPort,
				Protocol:      corev1.ProtocolTCP}))
			// Check environment variables.
			Expect(pod.Spec.Containers[0].Env).To(ContainElements(corev1.EnvVar{
				Name:  EnvRDZVBackend,
				Value: string(backendC10D),
			}, corev1.EnvVar{
				Name:  EnvNnodes,
				Value: fmt.Sprintf("%d:%d", minReplicas, maxReplicas),
			}, corev1.EnvVar{
				Name:  EnvRDZVEndpoint,
				Value: fmt.Sprintf("%s:%d", svc.Name, expectedPort),
			}, corev1.EnvVar{
				Name:  EnvMaxRestarts,
				Value: fmt.Sprintf("%d", maxRestarts),
			}))
			Expect(svc.Spec.Ports[0].Port).To(Equal(expectedPort))
			// Check owner references.
			trueVal := true
			Expect(pod.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PyTorchJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))
			Expect(svc.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PyTorchJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))

			// Test job status.
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, workerKey, pod)).Should(Succeed())
				pod.Status.Phase = corev1.PodSucceeded
				return testK8sClient.Status().Update(ctx, pod)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				if err != nil {
					return false
				}
				return created.Status.ReplicaStatuses != nil && created.Status.
					ReplicaStatuses[kubeflowv1.PyTorchJobReplicaTypeWorker].Succeeded == 1
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			// Check if the job is succeeded.
			cond := getCondition(created.Status, kubeflowv1.JobSucceeded)
			Expect(cond.Status).To(Equal(corev1.ConditionTrue))
		})
		It("Should delete HPA once the PyTorchJob is suspended", func() {
			By("By creating a new PyTorchJob")
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.PyTorchJob{}
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}

			By("Checking if the PyTorchJob and HPA are created")
			Eventually(func() error {
				return testK8sClient.Get(ctx, jobKey, created)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, jobKey, hpa)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

			By("Suspending PyTorchJob")
			Eventually(func() error {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				created.Spec.RunPolicy.Suspend = ptr.To(true)
				return testK8sClient.Update(ctx, created)
			}, testutil.Timeout, testutil.Interval).Should(Succeed())

			By("Checking if the HPA is deleted")
			Eventually(func() bool {
				return errors.IsNotFound(testK8sClient.Get(ctx, jobKey, hpa))
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
		})
	})
})

func newPyTorchJobForTest(name, namespace string) *kubeflowv1.PyTorchJob {
	return &kubeflowv1.PyTorchJob{
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
