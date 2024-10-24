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
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
	"github.com/kubeflow/training-operator/test/integration/framework"
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
		gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainingRuntime{}, client.InNamespace(ns.Name))).To(gomega.Succeed())
	})

	ginkgo.When("Creating TrainingRuntime", func() {
		ginkgo.DescribeTable("Validate TrainingRuntime on creation", func(runtime func() *kubeflowv2.TrainingRuntime) {
			gomega.Expect(k8sClient.Create(ctx, runtime())).Should(gomega.Succeed())
		},
			ginkgo.Entry("Should succeed to create TrainingRuntime",
				func() *kubeflowv2.TrainingRuntime {
					baseRuntime := testingutil.MakeTrainingRuntimeWrapper(ns.Name, trainingRuntimeName).Clone()
					return baseRuntime.
						RuntimeSpec(
							testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntime.Spec).
								Replicas(1).
								Obj()).
						Obj()
				}),
		)
	})
})
