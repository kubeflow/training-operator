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

package webhookv2

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
	"github.com/kubeflow/training-operator/test/integration/framework"
	"github.com/kubeflow/training-operator/test/util"
)

var _ = ginkgo.Describe("TrainJob Webhook", ginkgo.Ordered, func() {
	var ns *corev1.Namespace
	var trainingRuntime *kubeflowv2.TrainingRuntime
	var clusterTrainingRuntime *kubeflowv2.ClusterTrainingRuntime
	runtimeName := "training-runtime"
	jobName := "train-job"

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
				GenerateName: "trainjob-webhook-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())

		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		baseClusterRuntimeWrapper := testingutil.MakeClusterTrainingRuntimeWrapper(runtimeName)
		trainingRuntime = baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(
				testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName).Spec).Obj()).Obj()
		clusterTrainingRuntime = baseClusterRuntimeWrapper.RuntimeSpec(
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
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainingRuntime{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.ClusterTrainingRuntime{})).To(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainJob{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
	})

	ginkgo.When("Creating TrainJob", func() {
		ginkgo.DescribeTable("Validate TrainJob on creation", func(trainJob func() *kubeflowv2.TrainJob, errorMatcher gomega.OmegaMatcher) {
			gomega.Expect(k8sClient.Create(ctx, trainJob())).Should(errorMatcher)
		},
			ginkgo.Entry("Should succeed in creating trainJob with namespace scoped trainingRuntime",
				func() *kubeflowv2.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), runtimeName).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail in creating trainJob referencing trainingRuntime not present in the namespace",
				func() *kubeflowv2.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), "invalid").
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should succeed in creating trainJob with namespace scoped trainingRuntime",
				func() *kubeflowv2.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.ClusterTrainingRuntimeKind), runtimeName).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail in creating trainJob with pre-trained model config when referencing a trainingRuntime without an initializer",
				func() *kubeflowv2.TrainJob {
					trainingRuntime.Spec.Template = kubeflowv2.JobSetTemplateSpec{}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), runtimeName).
						ModelConfig(&kubeflowv2.ModelConfig{Input: &kubeflowv2.InputModel{}}).
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should fail in creating trainJob with podSpecOverrides when referencing a trainingRuntime doesnt have the job specified in the override",
				func() *kubeflowv2.TrainJob {
					trainJob := testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), runtimeName).
						Obj()
					trainJob.Spec.PodSpecOverrides = []kubeflowv2.PodSpecOverride{
						{TargetJobs: []kubeflowv2.PodSpecOverrideTargetJob{{Name: "valid"}, {Name: "invalid"}}},
					}
					return trainJob
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should fail in creating trainJob with invalid trainer config for mpi runtime",
				func() *kubeflowv2.TrainJob {
					trainingRuntime.Spec.MLPolicy = &kubeflowv2.MLPolicy{MLPolicySource: kubeflowv2.MLPolicySource{MPI: &kubeflowv2.MPIMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), runtimeName).
						Trainer(&kubeflowv2.Trainer{NumProcPerNode: ptr.To("invalid")}).
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should fail in creating trainJob with invalid trainer config for torch runtime",
				func() *kubeflowv2.TrainJob {
					trainingRuntime.Spec.MLPolicy = &kubeflowv2.MLPolicy{MLPolicySource: kubeflowv2.MLPolicySource{Torch: &kubeflowv2.TorchMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), runtimeName).
						Trainer(&kubeflowv2.Trainer{NumProcPerNode: ptr.To("invalid")}).
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should succeed in creating trainJob with valid trainer config for torch runtime",
				func() *kubeflowv2.TrainJob {
					trainingRuntime.Spec.MLPolicy = &kubeflowv2.MLPolicy{MLPolicySource: kubeflowv2.MLPolicySource{Torch: &kubeflowv2.TorchMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), runtimeName).
						Trainer(&kubeflowv2.Trainer{NumProcPerNode: ptr.To("auto")}).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail in creating trainJob with trainer config having envs with PET_ prefix",
				func() *kubeflowv2.TrainJob {
					trainingRuntime.Spec.MLPolicy = &kubeflowv2.MLPolicy{MLPolicySource: kubeflowv2.MLPolicySource{Torch: &kubeflowv2.TorchMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(kubeflowv2.GroupVersion.WithKind(kubeflowv2.TrainingRuntimeKind), runtimeName).
						Trainer(&kubeflowv2.Trainer{Env: []corev1.EnvVar{{Name: "PET_X", Value: "test"}}}).
						Obj()
				},
				testingutil.BeForbiddenError()),
		)
	})
})
