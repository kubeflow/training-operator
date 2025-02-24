/*
Copyright 2025 The Kubeflow Authors.

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

const trainingRuntimeName = "test-trainingruntime"

var _ = ginkgo.Describe("TrainingRuntime Webhook", ginkgo.Ordered, func() {
	var ns *corev1.Namespace

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
				GenerateName: "trainingruntime-webhook-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())
	})

	ginkgo.AfterEach(func() {
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.TrainingRuntime{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
	})

	ginkgo.When("Creating TrainingRuntime", func() {
		ginkgo.DescribeTable("Validate TrainingRuntime on creation", func(runtime func() *trainer.TrainingRuntime) {
			gomega.Expect(k8sClient.Create(ctx, runtime())).Should(gomega.Succeed())
		},
			ginkgo.Entry("Should succeed to create TrainingRuntime",
				func() *trainer.TrainingRuntime {
					baseRuntime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, trainingRuntimeName)
					return baseRuntime.
						RuntimeSpec(
							testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntime.Spec).
								Obj()).
						Obj()
				}),
		)
	})
})

var _ = ginkgo.Describe("TrainingRuntime marker validations and defaulting", ginkgo.Ordered, func() {
	var ns *corev1.Namespace

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
				GenerateName: "training-runtime-marker-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())
	})
	ginkgo.AfterEach(func() {
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.TrainingRuntime{}, client.InNamespace(ns.Name))).Should(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &trainer.ClusterTrainingRuntime{})).Should(gomega.Succeed())
	})

	ginkgo.When("Creating TrainingRuntime", func() {
		ginkgo.DescribeTable("Validate TrainingRuntime on creation", func(trainingRuntime func() *trainer.TrainingRuntime, errorMatcher gomega.OmegaMatcher) {
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime())).Should(errorMatcher)
		},
			ginkgo.Entry("Should succeed to create trainingRuntime",
				func() *trainer.TrainingRuntime {
					return testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail to create trainingRuntime with both MPI and Torch runtimes",
				func() *trainer.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &trainer.MLPolicy{
						MLPolicySource: trainer.MLPolicySource{
							Torch: &trainer.TorchMLPolicySource{},
							MPI:   &trainer.MPIMLPolicySource{},
						},
					}
					return runtime
				},
				testingutil.BeInvalidError()),
			ginkgo.Entry("Should fail to create trainingRuntime with minNodes and torch.elasticPolicy",
				func() *trainer.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &trainer.MLPolicy{
						NumNodes: ptr.To(int32(2)),
						MLPolicySource: trainer.MLPolicySource{
							Torch: &trainer.TorchMLPolicySource{
								ElasticPolicy: &trainer.TorchElasticPolicy{},
							},
						},
					}
					return runtime
				},
				testingutil.BeInvalidError()),
		)
		ginkgo.DescribeTable("Defaulting TrainingRuntime on creation", func(trainingRuntime func() *trainer.TrainingRuntime, wantTrainingRuntime func() *trainer.TrainingRuntime) {
			created := trainingRuntime()
			gomega.Expect(k8sClient.Create(ctx, created)).Should(gomega.Succeed())
			gomega.Expect(created).Should(gomega.BeComparableTo(wantTrainingRuntime(), util.IgnoreObjectMetadata))
		},
			ginkgo.Entry("Should succeed to default torch.NumProcPerNode=auto",
				func() *trainer.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &trainer.MLPolicy{
						MLPolicySource: trainer.MLPolicySource{
							Torch: &trainer.TorchMLPolicySource{},
						},
					}
					return runtime
				},
				func() *trainer.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &trainer.MLPolicy{
						MLPolicySource: trainer.MLPolicySource{
							Torch: &trainer.TorchMLPolicySource{
								NumProcPerNode: ptr.To(intstr.FromString("auto")),
							},
						},
					}
					runtime.Spec.Template.Spec = testingutil.MakeJobSetWrapper(ns.Name, "runtime").
						Replicas(1).Obj().Spec
					return runtime
				}),

			ginkgo.Entry("Should succeed to default mpi.mpiImplementation=OpenMPI",
				func() *trainer.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &trainer.MLPolicy{
						MLPolicySource: trainer.MLPolicySource{
							MPI: &trainer.MPIMLPolicySource{},
						},
					}
					return runtime
				},
				func() *trainer.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &trainer.MLPolicy{
						MLPolicySource: trainer.MLPolicySource{
							MPI: &trainer.MPIMLPolicySource{
								MPIImplementation: trainer.MPIImplementationOpenMPI,
								RunLauncherAsNode: ptr.To(false),
							},
						},
					}
					runtime.Spec.Template.Spec = testingutil.MakeJobSetWrapper(ns.Name, "runtime").
						Replicas(1).Obj().Spec
					return runtime
				}),
		)
	})
})
