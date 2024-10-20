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

package controllerv2

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
	"github.com/kubeflow/training-operator/test/integration/framework"
	"github.com/kubeflow/training-operator/test/util"
)

var _ = ginkgo.Describe("TrainJob controller", ginkgo.Ordered, func() {
	var ns *corev1.Namespace

	ginkgo.BeforeAll(func() {
		fwk = &framework.Framework{}
		cfg = fwk.Init()
		ctx, k8sClient = fwk.RunManager(cfg)
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
			trainJob        *kubeflowv2.TrainJob
			trainJobKey     client.ObjectKey
			trainingRuntime *kubeflowv2.TrainingRuntime
		)

		ginkgo.AfterEach(func() {
			gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainJob{}, client.InNamespace(ns.Name))).Should(gomega.Succeed())
		})

		ginkgo.BeforeEach(func() {
			trainJob = testingutil.MakeTrainJobWrapper(ns.Name, "alpha").
				Suspend(true).
				RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), "alpha").
				SpecLabel("testingKey", "testingVal").
				SpecAnnotation("testingKey", "testingVal").
				Trainer(
					testingutil.MakeTrainJobTrainerWrapper().
						ContainerImage("trainJob").
						Obj()).
				Obj()
			trainJobKey = client.ObjectKeyFromObject(trainJob)
			baseRuntime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "alpha")
			trainingRuntime = baseRuntime.Clone().
				RuntimeSpec(
					testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntime.Clone().Spec).
						ContainerImage("trainingRuntime").
						PodGroupPolicyCoscheduling(&kubeflowv2.CoschedulingPodGroupPolicySource{}).
						MLPolicyNumNodes(100).
						ResourceRequests(0, corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5"),
						}).
						ResourceRequests(1, corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("10"),
						}).
						Obj()).
				Obj()
		})

		ginkgo.It("Should succeed to create TrainJob with TrainingRuntime", func() {
			ginkgo.By("Creating TrainingRuntime and TrainJob")
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).Should(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())

			ginkgo.By("Checking if appropriately JobSet and PodGroup are created")
			gomega.Eventually(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				g.Expect(jobSet).Should(gomega.BeComparableTo(
					testingutil.MakeJobSetWrapper(ns.Name, trainJobKey.Name).
						Suspend(true).
						Label("testingKey", "testingVal").
						Annotation("testingKey", "testingVal").
						PodLabel(schedulerpluginsv1alpha1.PodGroupLabel, trainJobKey.Name).
						ContainerImage(ptr.To("trainJob")).
						JobCompletionMode(batchv1.IndexedCompletion).
						ResourceRequests(0, corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5"),
						}).
						ResourceRequests(1, corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("10"),
						}).
						ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind(kubeflowv2.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
						Obj(),
					util.IgnoreObjectMetadata))
				pg := &schedulerpluginsv1alpha1.PodGroup{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, pg)).Should(gomega.Succeed())
				g.Expect(pg).Should(gomega.BeComparableTo(
					testingutil.MakeSchedulerPluginsPodGroup(ns.Name, trainJobKey.Name).
						MinMember(200).
						MinResources(corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("1500"),
						}).
						ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind(kubeflowv2.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
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
						Suspend(true).
						Label("testingKey", "testingVal").
						Annotation("testingKey", "testingVal").
						PodLabel(schedulerpluginsv1alpha1.PodGroupLabel, trainJobKey.Name).
						ContainerImage(&updatedImageName).
						JobCompletionMode(batchv1.IndexedCompletion).
						ResourceRequests(0, corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5"),
						}).
						ResourceRequests(1, corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("10"),
						}).
						ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind(kubeflowv2.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
						Obj(),
					util.IgnoreObjectMetadata))
				pg := &schedulerpluginsv1alpha1.PodGroup{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, pg)).Should(gomega.Succeed())
				g.Expect(pg).Should(gomega.BeComparableTo(
					testingutil.MakeSchedulerPluginsPodGroup(ns.Name, trainJobKey.Name).
						MinMember(200).
						MinResources(corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("1500"),
						}).
						ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind(kubeflowv2.TrainJobKind), trainJobKey.Name, string(trainJob.UID)).
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

			ginkgo.By("Trying to restore trainer image")
			gomega.Eventually(func(g gomega.Gomega) {
				g.Expect(k8sClient.Get(ctx, trainJobKey, trainJob)).Should(gomega.Succeed())
				trainJob.Spec.Trainer.Image = &originImageName
				g.Expect(k8sClient.Update(ctx, trainJob)).Should(gomega.Succeed())
			}, util.Timeout, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Checking if JobSet keep having updated image")
			gomega.Consistently(func(g gomega.Gomega) {
				jobSet := &jobsetv1alpha2.JobSet{}
				g.Expect(k8sClient.Get(ctx, trainJobKey, jobSet)).Should(gomega.Succeed())
				for _, rJob := range jobSet.Spec.ReplicatedJobs {
					g.Expect(rJob.Template.Spec.Template.Spec.Containers[0].Image).Should(gomega.Equal(updatedImageName))
				}
			}, util.ConsistentDuration, util.Interval).Should(gomega.Succeed())

			ginkgo.By("Trying to re-suspend TrainJob and restore trainer image")
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
					g.Expect(rJob.Template.Spec.Template.Spec.Containers[0].Image).Should(gomega.Equal(originImageName))
				}
			}, util.Timeout, util.Interval).Should(gomega.Succeed())
		})
	})

	ginkgo.When("TrainJob CR Validation", func() {
		ginkgo.AfterEach(func() {
			gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainJob{}, client.InNamespace(ns.Name))).Should(
				gomega.Succeed())
		})

		ginkgo.It("Should succeed in creating TrainJob", func() {

			managedBy := "kubeflow.org/trainjob-controller"

			trainingRuntimeRef := kubeflowv2.RuntimeRef{
				Name:     "TorchRuntime",
				APIGroup: ptr.To(kubeflowv2.GroupVersion.Group),
				Kind:     ptr.To(kubeflowv2.TrainingRuntimeKind),
			}
			jobSpec := kubeflowv2.TrainJobSpec{
				RuntimeRef: trainingRuntimeRef,
				ManagedBy:  &managedBy,
			}
			trainJob := &kubeflowv2.TrainJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubeflowv2.SchemeGroupVersion.String(),
					Kind:       kubeflowv2.TrainJobKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "valid-trainjob-",
					Namespace:    ns.Name,
				},
				Spec: jobSpec,
			}

			err := k8sClient.Create(ctx, trainJob)
			gomega.Expect(err).Should(gomega.Succeed())
		})

		ginkgo.It("Should fail in creating TrainJob with invalid spec.managedBy", func() {
			managedBy := "invalidManagedBy"
			jobSpec := kubeflowv2.TrainJobSpec{
				ManagedBy: &managedBy,
			}
			trainJob := &kubeflowv2.TrainJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubeflowv2.SchemeGroupVersion.String(),
					Kind:       kubeflowv2.TrainJobKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-trainjob",
					Namespace: ns.Name,
				},
				Spec: jobSpec,
			}
			gomega.Expect(k8sClient.Create(ctx, trainJob)).To(gomega.MatchError(
				gomega.ContainSubstring("spec.managedBy: Invalid value")))
		})

		ginkgo.It("Should fail in updating spec.managedBy", func() {

			managedBy := "kubeflow.org/trainjob-controller"

			trainingRuntimeRef := kubeflowv2.RuntimeRef{
				Name:     "TorchRuntime",
				APIGroup: ptr.To(kubeflowv2.GroupVersion.Group),
				Kind:     ptr.To(kubeflowv2.TrainingRuntimeKind),
			}
			jobSpec := kubeflowv2.TrainJobSpec{
				RuntimeRef: trainingRuntimeRef,
				ManagedBy:  &managedBy,
			}
			trainJob := &kubeflowv2.TrainJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubeflowv2.SchemeGroupVersion.String(),
					Kind:       kubeflowv2.TrainJobKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-with-failed-update",
					Namespace: ns.Name,
				},
				Spec: jobSpec,
			}

			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())
			updatedManagedBy := "kueue.x-k8s.io/multikueue"
			jobSpec.ManagedBy = &updatedManagedBy
			trainJob.Spec = jobSpec
			gomega.Expect(k8sClient.Update(ctx, trainJob)).To(gomega.MatchError(
				gomega.ContainSubstring("ManagedBy value is immutable")))
		})
	})
})
