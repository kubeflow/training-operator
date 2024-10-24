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
	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
	"github.com/kubeflow/training-operator/test/integration/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var _ = ginkgo.Describe("TrainJob Webhook", ginkgo.Ordered, func() {
	var ns *corev1.Namespace
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
	})

	ginkgo.AfterEach(func() {
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainingRuntime{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.ClusterTrainingRuntime{})).To(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainJob{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
	})

	ginkgo.It("Should succeed in creating trainJob with namespace scoped trainingRuntime", func() {

		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
				Replicas(1).
				Obj()).Obj()
		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())

		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())
		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).To(gomega.Succeed())
	})

	ginkgo.It("Should fail in creating trainJob referencing trainingRuntime not present in the namespace", func() {

		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())
		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).To(testingutil.BeForbiddenError())
	})

	ginkgo.It("Should succeed in creating trainJob with ClusterTrainingRuntime", func() {

		baseRuntimeWrapper := testingutil.MakeClusterTrainingRuntimeWrapper(runtimeName)
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
				Replicas(1).
				Obj()).Obj()
		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())

		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())
		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).To(gomega.Succeed())
	})

	ginkgo.It("Should fail in creating trainJob with pre-trained model config when referencing "+
		"a trainingRuntime without an initializer", func() {

		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
				Replicas(1).
				Obj()).Obj()
		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())

		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).
			ModelConfig(&kubeflowv2.ModelConfig{Input: &kubeflowv2.InputModel{}}).
			Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).
			To(testingutil.BeForbiddenError())
	})

	ginkgo.It("Should fail in creating trainJob with output model config when referencing a trainingRuntime"+
		" without an exporter", func() {
		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
				Replicas(1).
				Obj()).Obj()
		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())

		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).
			ModelConfig(&kubeflowv2.ModelConfig{Output: &kubeflowv2.OutputModel{}}).
			Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).
			To(testingutil.BeForbiddenError())
	})

	ginkgo.It("Should fail in creating trainJob with podSpecOverrides when referencing a trainingRuntime doesnt "+
		"have the job specified in the override", func() {
		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
				Replicas(1).
				Obj()).Obj()
		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())

		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).Obj()
		trainJob.Spec.PodSpecOverrides = []kubeflowv2.PodSpecOverride{
			{TargetJobs: []kubeflowv2.PodSpecOverrideTargetJob{{Name: "valid"}, {Name: "invalid"}}},
		}

		gomega.Expect(k8sClient.Create(ctx, trainJob)).
			To(testingutil.BeForbiddenError())
	})

	ginkgo.It("Should fail in creating trainJob with podSpecOverrides when referencing a trainingRuntime doesnt "+
		"have the job specified in the override", func() {
		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(
			testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
				Replicas(1).
				Obj()).Obj()
		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())

		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).Obj()
		trainJob.Spec.PodSpecOverrides = []kubeflowv2.PodSpecOverride{
			{TargetJobs: []kubeflowv2.PodSpecOverrideTargetJob{{Name: "valid"}, {Name: "invalid"}}},
		}

		gomega.Expect(k8sClient.Create(ctx, trainJob)).
			To(testingutil.BeForbiddenError())
	})

	ginkgo.It("Should fail in creating trainJob with invalid trainer config for mpi runtime", func() {
		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		runtimeSpec := testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
			Replicas(1).
			MLPolicyNumNodes(1).
			Obj()
		runtimeSpec.MLPolicy.MLPolicySource = kubeflowv2.MLPolicySource{MPI: &kubeflowv2.MPIMLPolicySource{}}
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(runtimeSpec).Obj()

		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())

		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).
			Trainer(&kubeflowv2.Trainer{NumProcPerNode: ptr.To("invalid")}).
			Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).
			To(testingutil.BeForbiddenError())
	})

	ginkgo.It("Should fail in creating trainJob with invalid trainer config for torch runtime", func() {
		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		runtimeSpec := testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
			Replicas(1).
			MLPolicyNumNodes(1).
			Obj()
		runtimeSpec.MLPolicy.MLPolicySource = kubeflowv2.MLPolicySource{Torch: &kubeflowv2.TorchMLPolicySource{}}
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(runtimeSpec).Obj()

		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())

		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).
			Trainer(&kubeflowv2.Trainer{NumProcPerNode: ptr.To("invalid")}).
			Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).
			To(testingutil.BeForbiddenError())
	})

	ginkgo.It("Should succeed in creating trainJob with valid trainer config for torch runtime", func() {
		baseRuntimeWrapper := testingutil.MakeTrainingRuntimeWrapper(ns.Name, runtimeName)
		runtimeSpec := testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntimeWrapper.Spec).
			Replicas(1).
			MLPolicyNumNodes(1).
			Obj()
		runtimeSpec.MLPolicy.MLPolicySource = kubeflowv2.MLPolicySource{Torch: &kubeflowv2.TorchMLPolicySource{}}
		trainingRuntime := baseRuntimeWrapper.RuntimeSpec(runtimeSpec).Obj()

		gomega.Expect(k8sClient.Create(ctx, trainingRuntime)).To(gomega.Succeed())
		baseTrainJobWrapper := testingutil.MakeTrainJobWrapper(ns.Name, jobName)
		gvk, _ := apiutil.GVKForObject(baseRuntimeWrapper.Obj(), k8sClient.Scheme())

		trainJob := baseTrainJobWrapper.RuntimeRef(gvk, runtimeName).
			Trainer(&kubeflowv2.Trainer{NumProcPerNode: ptr.To("auto")}).
			Obj()

		gomega.Expect(k8sClient.Create(ctx, trainJob)).To(gomega.Succeed())
	})
})
