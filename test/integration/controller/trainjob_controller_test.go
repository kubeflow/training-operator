/*
Copyright 2024 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	jobsetconsts "sigs.k8s.io/jobset/pkg/constants"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	trainer "github.com/kubeflow/trainer/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/pkg/constants"
	jobsetplugin "github.com/kubeflow/trainer/pkg/runtime/framework/plugins/jobset"
	testingutil "github.com/kubeflow/trainer/pkg/util/testing"
	"github.com/kubeflow/trainer/test/integration/framework"
	"github.com/kubeflow/trainer/test/util"
)

var _ = ginkgo.Describe("TrainJob controller", ginkgo.Ordered, func() {
	var ns *corev1.Namespace

	resRequests := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("4Gi"),
	}

	ginkgo.BeforeAll(func() {
		fwk = &framework.Framework{}
		cfg = fwk.Init()
		ctx, k8sClient = fwk.RunManager(cfg, true)
	})
	ginkgo.AfterAll(func() {
		fwk.Teardown()
	})

	ginkgo.BeforeEach(func() {
		ns = &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "trainjob-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())
	})

	ginkgo.When("Reconciling TrainJob", func() {
		var (
			trainJob        *trainer.TrainJob
			trainJobKey     client.ObjectKey
			trainingRuntime *trainer.TrainingRuntime
		)

		ginkgo.AfterEach(func() {
			gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.TrainJob{}, client.InNamespace(ns.Name))).Should(gomega.Succeed())
		})

		ginkgo.BeforeEach(func() {
			trainJob = testingutil.MakeTrainJobWrapper(ns.Name, "alpha").
				Suspend(true).
				RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), "alpha").
				SpecLabel("testingKey", "testingVal").
				SpecAnnotation("testingKey", "testingVal").
				Trainer(
					testingutil.MakeTrainJobTrainerWrapper().
						Container("test:trainJob", []string{"trainjob"}, []string{"trainjob"}, resRequests).
						Obj()).
				DatasetConfig(
					testingutil.MakeTrainJobDatasetConfigWrapper().
						StorageUri("hf://trainjob-dataset").
						Obj(),
				).
				ModelConfig(
					testingutil.MakeTrainJobModelConfigWrapper().
						StorageUri("hf://trainjob-model").
						Obj(),
				).
				Obj()
			trainJobKey = client.ObjectKeyFromObject(trainJob)

			trainingRuntime = testingutil.MakeTrainingRuntimeWrapper(ns.Name, "alpha").
				RuntimeSpec(
					testingutil.MakeTrainingRuntimeSpecWrapper(testingutil.MakeTrainingRuntimeWrapper(metav1.NamespaceDefault, "alpha").Spec).
						NumNodes(100).
						ContainerTrainer("test:runtime", []string{"runtime"}, []string{"runtime"}, resRequests).
						InitContainerDatasetModelInitializer("test:runtime", []string{"runtime"}, []string{"runtime"}, resRequests).
						PodGroupPolicyCoscheduling(&trainer.CoschedulingPodGroupPolicySource{ScheduleTimeoutSeconds: ptr.To[int32](100)}).
						Obj()).
				Obj()
		})

		// Integration tests for the PlainML Runtime.
		ginkgo.It("Should succeed to create TrainJob with TrainingRuntime", func() {
			ginkgo.By("Creating TrainingRuntime and TrainJob")
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).Should(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())

			ginkgo.By("Checking if the appropriate JobSet and PodGroup are created")
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				g.Expect(jobSet).Should(gomega.BeComparableTo(
					testingutil.MakeJobSetWrapper(ns.Name, trainJobKey.Name).
						Replicas(1).
						NumNodes(100).
						ContainerTrainer("test:trainJob", []string{"trainjob"}, []string{"trainjob"}, resRequests).
						InitContainerDatasetModelInitializer("test:runtime", []string{"runtime"}, []string{"runtime"}, resRequests).
						InitContainerDatasetInitializerEnv([]corev1.EnvVar{{Name: jobsetplugin.InitializerEnvStorageUri, Value: "hf://trainjob-dataset"}}).
						InitContainerModelInitializerEnv([]corev1.EnvVar{{Name: jobsetplugin.InitializerEnvStorageUri, Value: "hf://trainjob-model"}}).
						Suspend(true).
						Label("testingKey", "testingVal").
						Annotation("testingKey", "testingVal").
						PodLabel(schedulerpluginsv1alpha1.PodGroupLabel, trainJobKey.Name).
						ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
						Obj(),
					util.IgnoreObjectMetadata))
				pg := &schedulerpluginsv1alpha1.PodGroup{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, pg)).Should(gomega.Succeed())
				g.Expect(pg).Should(gomega.BeComparableTo(
					testingutil.MakeSchedulerPluginsPodGroup(ns.Name, trainJobKey.Name).
						MinMember(101). // 101 replicas = 100 Trainer nodes + 1 Initializer.
						MinResources(corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("101"), // 1 CPU and 4Gi per replica.
							corev1.ResourceMemory: resource.MustParse("404Gi"),
						}).
						SchedulingTimeout(100).
						ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
						Obj(),
					util.IgnoreObjectMetadata))
			}, util.Timeout, util.Interval).Should(gomega.Succeed())
		})

		ginkgo.It("Should succeeded to update JobSet only when TrainJob is suspended", func() {
			ginkgo.By("Creating TrainingRuntime and suspended TrainJob")
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).Should(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())

			ginkgo.By("Checking if JobSet and PodGroup are created")
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, &jobsetv1alpha2.JobSet{})).Should(gomega.Succeed())
				g.Expect(k8sClient.Get(ctx, trainJobKey, &schedulerpluginsv1alpha1.PodGroup{})).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Updating suspended TrainJob Trainer image")
			updatedImageName := "updated-trainer-image"
			originImageName := *trainJob.Spec.Trainer.Image
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, trainJob)).Should(gomega.Succeed())
				trainJob.Spec.Trainer.Image = &updatedImageName
				g.Expect(k8sClient.Update(ctx, trainJob)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Trainer image should be updated")
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				g.Expect(jobSet).Should(gomega.BeComparableTo(
					testingutil.MakeJobSetWrapper(ns.Name, trainJobKey.Name).
						Replicas(1).
						NumNodes(100).
						ContainerTrainer(updatedImageName, []string{"trainjob"}, []string{"trainjob"}, resRequests).
						InitContainerDatasetModelInitializer("test:runtime", []string{"runtime"}, []string{"runtime"}, resRequests).
						InitContainerDatasetInitializerEnv([]corev1.EnvVar{{Name: jobsetplugin.InitializerEnvStorageUri, Value: "hf://trainjob-dataset"}}).
						InitContainerModelInitializerEnv([]corev1.EnvVar{{Name: jobsetplugin.InitializerEnvStorageUri, Value: "hf://trainjob-model"}}).
						Suspend(true).
						Label("testingKey", "testingVal").
						Annotation("testingKey", "testingVal").
						PodLabel(schedulerpluginsv1alpha1.PodGroupLabel, trainJobKey.Name).
						ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
						Obj(),
					util.IgnoreObjectMetadata))
				pg := &schedulerpluginsv1alpha1.PodGroup{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, pg)).Should(gomega.Succeed())
				g.Expect(pg).Should(gomega.BeComparableTo(
					testingutil.MakeSchedulerPluginsPodGroup(ns.Name, trainJobKey.Name).
						MinMember(101).
						MinResources(corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("101"), // 1 CPU and 4Gi per 101 replica.
							corev1.ResourceMemory: resource.MustParse("404Gi"),
						}).
						SchedulingTimeout(100).
						ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
						Obj(),
					util.IgnoreObjectMetadata))
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Unsuspending TrainJob")
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, trainJob)).Should(gomega.Succeed())
				trainJob.Spec.Suspend = ptr.To(false)
				g.Expect(k8sClient.Update(ctx, trainJob)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				g.Expect(ptr.Deref(jobSet.Spec.Suspend, false)).Should(gomega.BeFalse())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Trying to restore Trainer image")
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, trainJob)).Should(gomega.Succeed())
				trainJob.Spec.Trainer.Image = &originImageName
				g.Expect(k8sClient.Update(ctx, trainJob)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Checking if JobSet keep having updated Trainer image")
			gomega.Consistently(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				for _, rJob := range jobSet.Spec.ReplicatedJobs {
					if rJob.Name == constants.JobTrainerNode {
						g.Expect(rJob.Template.Spec.Template.Spec.Containers[0].Image).Should(gomega.Equal(updatedImageName))
					}
				}
			}, util.ConsistentDuration, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Trying to re-suspend TrainJob and restore Trainer image")
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, trainJob))
				trainJob.Spec.Suspend = ptr.To(true)
				trainJob.Spec.Trainer.Image = &originImageName
				g.Expect(k8sClient.Update(ctx, trainJob)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Checking if JobSet image is restored")
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				g.Expect(jobSet.Spec.Suspend).ShouldNot(gomega.BeNil())
				g.Expect(*jobSet.Spec.Suspend).Should(gomega.BeTrue())
				for _, rJob := range jobSet.Spec.ReplicatedJobs {
					if rJob.Name == constants.JobTrainerNode {
						g.Expect(rJob.Template.Spec.Template.Spec.Containers[0].Image).Should(gomega.Equal(originImageName))
					}
				}
			}, util.Timeout, util.Interval).Should(gomega.Succeed())
		})

		// Integration tests for the Torch Runtime.
		ginkgo.It("Should succeed to create TrainJob with Torch TrainingRuntime", func() {
			ginkgo.By("Creating Torch TrainingRuntime and TrainJob")
			trainJob = testingutil.MakeTrainJobWrapper(ns.Name, "alpha").
				RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), "alpha").
				Trainer(
					testingutil.MakeTrainJobTrainerWrapper().
						Container("test:trainJob", []string{"trainjob"}, []string{"trainjob"}, resRequests).
						ContainerEnv([]corev1.EnvVar{{Name: "TRAIN_JOB", Value: "value"}}).
						Obj()).
				Obj()
			trainJobKey = client.ObjectKeyFromObject(trainJob)

			trainingRuntime = testingutil.MakeTrainingRuntimeWrapper(ns.Name, "alpha").
				RuntimeSpec(
					testingutil.MakeTrainingRuntimeSpecWrapper(testingutil.MakeTrainingRuntimeWrapper(metav1.NamespaceDefault, "alpha").Spec).
						TorchPolicy(100, intstr.FromString("auto")).
						ContainerTrainer("test:runtime", []string{"runtime"}, []string{"runtime"}, resRequests).
						Obj()).
				Obj()
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).Should(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())

			ginkgo.By("Checking if the appropriate JobSet is created")
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				g.Expect(jobSet).Should(gomega.BeComparableTo(
					testingutil.MakeJobSetWrapper(ns.Name, trainJobKey.Name).
						Replicas(1).
						NumNodes(100).
						Suspend(false).
						ContainerTrainer("test:trainJob", []string{"trainjob"}, []string{"trainjob"}, resRequests).
						ContainerTrainerPorts([]corev1.ContainerPort{{ContainerPort: constants.ContainerTrainerPort, Protocol: "TCP"}}).
						ContainerTrainerEnv(
							[]corev1.EnvVar{
								{
									Name:  "TRAIN_JOB",
									Value: "value",
								},
								{
									Name:  constants.TorchEnvNumNodes,
									Value: "100",
								},
								{
									Name:  constants.TorchEnvNumProcPerNode,
									Value: "auto",
								},
								{
									Name: constants.TorchEnvNodeRank,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: constants.JobCompletionIndexFieldPath,
										},
									},
								},
								{
									Name:  constants.TorchEnvMasterAddr,
									Value: fmt.Sprintf("alpha-%s-0-0.alpha", constants.JobTrainerNode),
								},
								{
									Name:  constants.TorchEnvMasterPort,
									Value: fmt.Sprintf("%d", constants.ContainerTrainerPort),
								},
							},
						).
						ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
						Obj(),
					util.IgnoreObjectMetadata))

			}, util.Timeout, util.Interval).Should(gomega.Succeed())
		})

		ginkgo.It("Should succeeded to reconcile TrainJob conditions with Complete condition", func() {
			ginkgo.By("Creating TrainingRuntime and suspended TrainJob")
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).Should(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())

			ginkgo.By("Checking if JobSet and PodGroup are created")
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, &jobsetv1alpha2.JobSet{})).Should(gomega.Succeed())
				g.Expect(k8sClient.Get(ctx, trainJobKey, &schedulerpluginsv1alpha1.PodGroup{})).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Checking if TrainJob has Suspended and Created conditions")
			gomega.Eventually(func(g gomega.Gomega) {
				gotTrainJob := &trainer.TrainJob{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, gotTrainJob)).Should(gomega.Succeed())
				g.Expect(gotTrainJob.Status.Conditions).Should(gomega.BeComparableTo([]metav1.Condition{
					{
						Type:    trainer.TrainJobSuspended,
						Status:  metav1.ConditionTrue,
						Reason:  trainer.TrainJobSuspendedReason,
						Message: constants.TrainJobSuspendedMessage,
					},
					{
						Type:    trainer.TrainJobCreated,
						Status:  metav1.ConditionTrue,
						Reason:  trainer.TrainJobJobsCreationSucceededReason,
						Message: constants.TrainJobJobsCreationSucceededMessage,
					},
				}, util.IgnoreConditions))
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Checking if the TrainJob has Resumed and Created conditions after unsuspended")
			gomega.Eventually(func(g gomega.Gomega) {
				gotTrainJob := &trainer.TrainJob{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, gotTrainJob)).Should(gomega.Succeed())
				gotTrainJob.Spec.Suspend = ptr.To(false)
				g.Expect(k8sClient.Update(ctx, gotTrainJob)).Should(gomega.Succeed())
				g.Expect(k8sClient.Get(ctx, trainJobKey, gotTrainJob)).Should(gomega.Succeed())
				g.Expect(gotTrainJob.Status.Conditions).Should(gomega.BeComparableTo([]metav1.Condition{
					{
						Type:    trainer.TrainJobSuspended,
						Status:  metav1.ConditionFalse,
						Reason:  trainer.TrainJobResumedReason,
						Message: constants.TrainJobResumedMessage,
					},
					{
						Type:    trainer.TrainJobCreated,
						Status:  metav1.ConditionTrue,
						Reason:  trainer.TrainJobJobsCreationSucceededReason,
						Message: constants.TrainJobJobsCreationSucceededMessage,
					},
				}, util.IgnoreConditions))
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Updating the JobSet condition with Completed")
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				meta.SetStatusCondition(&jobSet.Status.Conditions, metav1.Condition{
					Type:    string(jobsetv1alpha2.JobSetCompleted),
					Reason:  jobsetconsts.AllJobsCompletedReason,
					Message: jobsetconsts.AllJobsCompletedMessage,
					Status:  metav1.ConditionTrue,
				})
				g.Expect(k8sClient.Status().Update(ctx, jobSet)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Checking if the TranJob has Resumed, Created, and Completed conditions")
			gomega.Eventually(func(g gomega.Gomega) {
				gotTrainJob := &trainer.TrainJob{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, gotTrainJob)).Should(gomega.Succeed())
				g.Expect(gotTrainJob.Status.Conditions).Should(gomega.BeComparableTo([]metav1.Condition{
					{
						Type:    trainer.TrainJobSuspended,
						Status:  metav1.ConditionFalse,
						Reason:  trainer.TrainJobResumedReason,
						Message: constants.TrainJobResumedMessage,
					},
					{
						Type:    trainer.TrainJobCreated,
						Status:  metav1.ConditionTrue,
						Reason:  trainer.TrainJobJobsCreationSucceededReason,
						Message: constants.TrainJobJobsCreationSucceededMessage,
					},
					{
						Type:    trainer.TrainJobComplete,
						Status:  metav1.ConditionTrue,
						Reason:  jobsetconsts.AllJobsCompletedReason,
						Message: jobsetconsts.AllJobsCompletedMessage,
					},
				}, util.IgnoreConditions))
			}, util.Timeout, util.Interval).Should(gomega.Succeed())
		})

		ginkgo.It("Should succeeded to reconcile TrainJob conditions with Failed condition", func() {
			ginkgo.By("Creating TrainingRuntime and suspended TrainJob")
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).Should(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())

			ginkgo.By("Checking if JobSet and PodGroup are created")
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, &jobsetv1alpha2.JobSet{})).Should(gomega.Succeed())
				g.Expect(k8sClient.Get(ctx, trainJobKey, &schedulerpluginsv1alpha1.PodGroup{})).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Unsuspending the TrainJob")
			gomega.Eventually(func(g gomega.Gomega) {
				gotTrainJob := &trainer.TrainJob{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, gotTrainJob)).Should(gomega.Succeed())
				gotTrainJob.Spec.Suspend = ptr.To(false)
				g.Expect(k8sClient.Update(ctx, gotTrainJob)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Waiting for TrainJob Created=True and Suspended=False condition")
			gomega.Eventually(func(g gomega.Gomega) {
				gotTrainJob := &trainer.TrainJob{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, gotTrainJob)).Should(gomega.Succeed())
				g.Expect(gotTrainJob.Status.Conditions).Should(gomega.BeComparableTo([]metav1.Condition{
					{
						Type:    trainer.TrainJobSuspended,
						Status:  metav1.ConditionFalse,
						Reason:  trainer.TrainJobResumedReason,
						Message: constants.TrainJobResumedMessage,
					},
					{
						Type:    trainer.TrainJobCreated,
						Status:  metav1.ConditionTrue,
						Reason:  trainer.TrainJobJobsCreationSucceededReason,
						Message: constants.TrainJobJobsCreationSucceededMessage,
					},
				}, util.IgnoreConditions))
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Updating the JobSet condition with Failed")
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				meta.SetStatusCondition(&jobSet.Status.Conditions, metav1.Condition{
					Type:    string(jobsetv1alpha2.JobSetFailed),
					Reason:  jobsetconsts.FailedJobsReason,
					Message: jobsetconsts.FailedJobsMessage,
					Status:  metav1.ConditionTrue,
				})
				g.Expect(k8sClient.Status().Update(ctx, jobSet)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Checking if the TranJob has Resumed, Created, and Failed conditions")
			gomega.Eventually(func(g gomega.Gomega) {
				gotTrainJob := &trainer.TrainJob{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, gotTrainJob)).Should(gomega.Succeed())
				g.Expect(gotTrainJob.Status.Conditions).Should(gomega.BeComparableTo([]metav1.Condition{
					{
						Type:    trainer.TrainJobSuspended,
						Status:  metav1.ConditionFalse,
						Reason:  trainer.TrainJobResumedReason,
						Message: constants.TrainJobResumedMessage,
					},
					{
						Type:    trainer.TrainJobCreated,
						Status:  metav1.ConditionTrue,
						Reason:  trainer.TrainJobJobsCreationSucceededReason,
						Message: constants.TrainJobJobsCreationSucceededMessage,
					},
					{
						Type:    trainer.TrainJobFailed,
						Status:  metav1.ConditionTrue,
						Reason:  jobsetconsts.FailedJobsReason,
						Message: jobsetconsts.FailedJobsMessage,
					},
				}, util.IgnoreConditions))
			}, util.Timeout, util.Interval).Should(gomega.Succeed())
		})
	})
})

var _ = ginkgo.Describe("TrainJob marker validations and defaulting", ginkgo.Ordered, func() {
	var ns *corev1.Namespace
	runtimeName := "training-runtime"
	ginkgo.BeforeAll(func() {
		fwk = &framework.Framework{}
		cfg = fwk.Init()
		ctx, k8sClient = fwk.RunManager(cfg, false)
	})
	ginkgo.AfterAll(func() {
		fwk.Teardown()
	})

	ginkgo.BeforeEach(func() {
		ns = &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "trainjob-marker-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())

		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		baseClusterRuntimeWrapper := testingutil.MakeClusterTrainingRuntimeWrapper(runtimeName)
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(
				testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName).Spec).Obj()).Obj()
		clusterTrainingRuntime := baseClusterRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(
				testingutil.MakeClusterTrainingRuntimeWrapper(runtimeName).Spec).Obj()).Obj()
		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		gomega.Expect(k8sClient.Create(ctx, clusterTrainingRuntime)).To(gomega.Succeed())

		gomega.Eventually(func() error {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(trainingRuntime), trainingRuntime)
			if err != nil {
				return err
			}
			return nil
		}, util.Timeout, util.Interval).Should(gomega.Succeed())
		gomega.Eventually(func() error {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(clusterTrainingRuntime), clusterTrainingRuntime)
			if err != nil {
				return err
			}
			return nil
		}, util.Timeout, util.Interval).Should(gomega.Succeed())
	})
	ginkgo.AfterEach(func() {
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.TrainingRuntime{}, client.InNamespace(ns.Name))).Should(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.ClusterTrainingRuntime{})).Should(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.TrainJob{}, client.InNamespace(ns.Name))).Should(gomega.Succeed())
	})

	ginkgo.When("Creating TrainJob", func() {
		ginkgo.DescribeTable("Validate TrainJob on creation", func(trainJob func() *trainer.TrainJob, errorMatcher gomega.OmegaMatcher) {
			gomega.Expect(k8sClient.Create(ctx, trainJob())).Should(errorMatcher)
		},
			ginkgo.Entry("Should succeed to create TrainJob with 'managedBy: trainer.kubeflow.org/trainjob-conteroller'",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "managed-by-trainjob-controller").
						ManagedBy("trainer.kubeflow.org/trainjob-controller").
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should succeed to create TrainJob with 'managedBy: kueue.x-k8s.io/multukueue'",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "managed-by-trainjob-controller").
						ManagedBy("kueue.x-k8s.io/multikueue").
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail to create TrainJob with invalid managedBy",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "invalid-managed-by").
						ManagedBy("invalid").
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Obj()
				},
				testingutil.BeInvalidError()),
		)
		ginkgo.DescribeTable("Defaulting TrainJob on creation", func(trainJob func() *trainer.TrainJob, wantTrainJob func() *trainer.TrainJob) {
			created := trainJob()
			gomega.Expect(k8sClient.Create(ctx, created)).Should(gomega.Succeed())
			gomega.Expect(created).Should(gomega.BeComparableTo(wantTrainJob(), util.IgnoreObjectMetadata))
		},
			ginkgo.Entry("Should succeed to default suspend=false",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "null-suspend").
						ManagedBy("kueue.x-k8s.io/multikueue").
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.ClusterTrainingRuntimeKind), runtimeName).
						Obj()
				},
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "null-suspend").
						ManagedBy("kueue.x-k8s.io/multikueue").
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.ClusterTrainingRuntimeKind), runtimeName).
						Suspend(false).
						Obj()
				}),
			ginkgo.Entry("Should succeed to default managedBy=trainer.kubeflow.org/trainjob-controller",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "null-managed-by").
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Suspend(true).
						Obj()
				},
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "null-managed-by").
						ManagedBy("trainer.kubeflow.org/trainjob-controller").
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Suspend(true).
						Obj()
				}),
			ginkgo.Entry("Should succeed to default runtimeRef.apiGroup",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "empty-api-group").
						RuntimeRef(schema.GroupVersionKind{Group: "", Version: "", Kind: trainer.TrainingRuntimeKind}, runtimeName).
						Obj()
				},
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "empty-api-group").
						ManagedBy("trainer.kubeflow.org/trainjob-controller").
						RuntimeRef(trainer.SchemeGroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Suspend(false).
						Obj()
				}),
			ginkgo.Entry("Should succeed to default runtimeRef.kind",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "empty-kind").
						RuntimeRef(trainer.SchemeGroupVersion.WithKind(""), runtimeName).
						Obj()
				},
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "empty-kind").
						ManagedBy("trainer.kubeflow.org/trainjob-controller").
						RuntimeRef(trainer.SchemeGroupVersion.WithKind(trainer.ClusterTrainingRuntimeKind), runtimeName).
						Suspend(false).
						Obj()
				}),
		)
	})

	ginkgo.When("Updating TrainJob", func() {
		ginkgo.DescribeTable("Validate TrainJob on update", func(old func() *trainer.TrainJob, new func(*trainer.TrainJob) *trainer.TrainJob, errorMatcher gomega.OmegaMatcher) {
			oldTrainJob := old()
			gomega.Expect(k8sClient.Create(ctx, oldTrainJob)).Should(gomega.Succeed())
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(oldTrainJob), oldTrainJob)).Should(gomega.Succeed())
				g.Expect(k8sClient.Update(ctx, new(oldTrainJob))).Should(errorMatcher)
			}, util.Timeout, util.Interval).Should(gomega.Succeed())
		},
			ginkgo.Entry("Should fail to update TrainJob managedBy",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "valid-managed-by").
						ManagedBy("trainer.kubeflow.org/trainjob-controller").
						RuntimeRef(trainer.SchemeGroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Obj()
				},
				func(job *trainer.TrainJob) *trainer.TrainJob {
					job.Spec.ManagedBy = ptr.To("kueue.x-k8s.io/multikueue")
					return job
				},
				testingutil.BeInvalidError()),
			ginkgo.Entry("Should fail to update runtimeRef",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, "valid-runtimeref").
						RuntimeRef(trainer.SchemeGroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Obj()
				},
				func(job *trainer.TrainJob) *trainer.TrainJob {
					job.Spec.RuntimeRef.Name = "forbidden-update"
					return job
				},
				testingutil.BeInvalidError()),
		)
	})
})
