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
	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
	"github.com/kubeflow/training-operator/test/integration/framework"
	"github.com/kubeflow/training-operator/test/util"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("TrainingRuntime marker validations and defaulting", ginkgo.Ordered, func() {
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
				GenerateName: "training-runtime-marker-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())
	})
	ginkgo.AfterEach(func() {
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainingRuntime{}, client.InNamespace(ns.Name))).Should(gomega.Succeed())
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.ClusterTrainingRuntime{})).Should(gomega.Succeed())
	})

	ginkgo.When("Creating TrainingRuntime", func() {
		ginkgo.DescribeTable("Validate TrainingRuntime on creation", func(trainingRuntime func() *kubeflowv2.TrainingRuntime, errorMatcher gomega.OmegaMatcher) {
			gomega.Expect(k8sClient.Create(ctx, trainingRuntime())).Should(errorMatcher)
		},
			ginkgo.Entry("Should succeed to create trainingRuntime",
				func() *kubeflowv2.TrainingRuntime {
					return testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").
						Obj()
				},
				gomega.Succeed()),
			ginkgo.Entry("Should fail to create trainingRuntime with both MPI and Torch runtimes",
				func() *kubeflowv2.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &kubeflowv2.MLPolicy{
						MLPolicySource: kubeflowv2.MLPolicySource{
							Torch: &kubeflowv2.TorchMLPolicySource{},
							MPI:   &kubeflowv2.MPIMLPolicySource{},
						},
					}
					return runtime
				},
				testingutil.BeInvalidError()),
			ginkgo.Entry("Should fail to create trainingRuntime with minNodes and torch.elasticPolicy",
				func() *kubeflowv2.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &kubeflowv2.MLPolicy{
						NumNodes: ptr.To(int32(2)),
						MLPolicySource: kubeflowv2.MLPolicySource{
							Torch: &kubeflowv2.TorchMLPolicySource{
								ElasticPolicy: &kubeflowv2.TorchElasticPolicy{},
							},
						},
					}
					return runtime
				},
				testingutil.BeInvalidError()),
		)
		ginkgo.DescribeTable("Defaulting TrainingRuntime on creation", func(trainingRuntime func() *kubeflowv2.TrainingRuntime, wantTrainingRuntime func() *kubeflowv2.TrainingRuntime) {
			created := trainingRuntime()
			gomega.Expect(k8sClient.Create(ctx, created)).Should(gomega.Succeed())
			gomega.Expect(created).Should(gomega.BeComparableTo(wantTrainingRuntime(), util.IgnoreObjectMetadata))
		},
			ginkgo.Entry("Should succeed to default TorchMLPolicySource.NumProcPerNode=auto",
				func() *kubeflowv2.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &kubeflowv2.MLPolicy{
						MLPolicySource: kubeflowv2.MLPolicySource{
							Torch: &kubeflowv2.TorchMLPolicySource{},
						},
					}
					return runtime
				},
				func() *kubeflowv2.TrainingRuntime {
					runtime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, "runtime").Obj()
					runtime.Spec.MLPolicy = &kubeflowv2.MLPolicy{
						MLPolicySource: kubeflowv2.MLPolicySource{
							Torch: &kubeflowv2.TorchMLPolicySource{
								NumProcPerNode: ptr.To("auto"),
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
