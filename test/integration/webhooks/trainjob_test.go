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

package webhooks

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	trainer "github.com/kubeflow/trainer/pkg/apis/trainer/v1alpha1"
	testingutil "github.com/kubeflow/trainer/pkg/util/testing"
	"github.com/kubeflow/trainer/test/integration/framework"
	"github.com/kubeflow/trainer/test/util"
)

var _ = ginkgo.Describe("TrainJob Webhook", ginkgo.Ordered, func() {
	var ns *corev1.Namespace
	var trainingRuntime *trainer.TrainingRuntime
	var clusterTrainingRuntime *trainer.ClusterTrainingRuntime
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
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.TrainingRuntime{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.ClusterTrainingRuntime{})).To(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.TrainJob{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
	})

	ginkgo.When("Creating TrainJob", func() {
		ginkgo.DescribeTable("Validate TrainJob on creation", func(trainJob func() *trainer.TrainJob, errorMatcher gomega.OmegaMatcher) {
			gomega.Expect(k8sClient.Create(ctx, trainJob())).Should(errorMatcher)
		},
			ginkgo.Entry("Should succeed in creating trainJob with namespace scoped trainingRuntime",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail in creating trainJob referencing trainingRuntime not present in the namespace",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), "invalid").
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should succeed in creating trainJob with namespace scoped trainingRuntime",
				func() *trainer.TrainJob {
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.ClusterTrainingRuntimeKind), runtimeName).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail in creating trainJob with pre-trained model config when referencing a trainingRuntime without an initializer",
				func() *trainer.TrainJob {
					trainingRuntime.Spec.Template = trainer.JobSetTemplateSpec{}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						ModelConfig(&trainer.ModelConfig{Input: &trainer.InputModel{}}).
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should fail in creating trainJob with invalid trainer config for mpi runtime",
				func() *trainer.TrainJob {
					trainingRuntime.Spec.MLPolicy = &trainer.MLPolicy{MLPolicySource: trainer.MLPolicySource{MPI: &trainer.MPIMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Trainer(&trainer.Trainer{NumProcPerNode: ptr.To(intstr.FromString("invalid"))}).
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should fail in creating trainJob with invalid trainer config for torch runtime",
				func() *trainer.TrainJob {
					trainingRuntime.Spec.MLPolicy = &trainer.MLPolicy{MLPolicySource: trainer.MLPolicySource{Torch: &trainer.TorchMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Trainer(&trainer.Trainer{NumProcPerNode: ptr.To(intstr.FromString("invalid"))}).
						Obj()
				},
				testingutil.BeForbiddenError()),
			ginkgo.Entry("Should succeed in creating trainJob with valid trainer config for torch runtime",
				func() *trainer.TrainJob {
					trainingRuntime.Spec.MLPolicy = &trainer.MLPolicy{MLPolicySource: trainer.MLPolicySource{Torch: &trainer.TorchMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Trainer(&trainer.Trainer{NumProcPerNode: ptr.To(intstr.FromString("auto"))}).
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail in creating trainJob with trainer config having envs with PET_ prefix",
				func() *trainer.TrainJob {
					trainingRuntime.Spec.MLPolicy = &trainer.MLPolicy{MLPolicySource: trainer.MLPolicySource{Torch: &trainer.TorchMLPolicySource{}}}
					gomega.Expect(k8sClient.Update(ctx, trainingRuntime)).To(gomega.Succeed())
					return testingutil.MakeTrainJobWrapper(ns.Name, jobName).
						RuntimeRef(trainer.GroupVersion.WithKind(trainer.TrainingRuntimeKind), runtimeName).
						Trainer(&trainer.Trainer{Env: []corev1.EnvVar{{Name: "PET_X", Value: "test"}}}).
						Obj()
				},
				testingutil.BeForbiddenError()),
		)
	})
})
