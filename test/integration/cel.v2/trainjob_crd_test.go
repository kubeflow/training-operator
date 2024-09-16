package cel_v2

import (
	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeflow/training-operator/test/integration/framework"
)

var _ = ginkgo.Describe("TrainJob CRD", ginkgo.Ordered, func() {
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
				GenerateName: "trainjob-validation-",
			},
		}
		gomega.Expect(k8sClient.Create(ctx, ns)).To(gomega.Succeed())
	})

	ginkgo.When("TrainJob CR Validation", func() {
		ginkgo.AfterEach(func() {
			gomega.Expect(k8sClient.DeleteAllOf(ctx, &kubeflowv2.TrainJob{}, client.InNamespace(ns.Name))).Should(gomega.Succeed())
		})

		ginkgo.It("Should succeed in creating TrainJob", func() {

			apiGroup := "kubeflow.org"
			kind := "TrainingRuntime"
			managedBy := "kubeflow.org/trainjob-controller"

			trainingRuntimeRef := kubeflowv2.TrainingRuntimeRef{
				Name:     "InvalidRuntimeRef",
				APIGroup: &apiGroup,
				Kind:     &kind,
			}
			jobSpec := kubeflowv2.TrainJobSpec{
				TrainingRuntimeRef: trainingRuntimeRef,
				ManagedBy:          &managedBy,
			}
			trainJob := &kubeflowv2.TrainJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubeflowv2.SchemeGroupVersion.String(),
					Kind:       "TrainJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alpha",
					Namespace: ns.Name,
				},
				Spec: jobSpec,
			}

			err := k8sClient.Create(ctx, trainJob)
			gomega.Expect(err).Should(gomega.Succeed())
		})

		ginkgo.It("Should fail in creating TrainJob with invalid spec.trainingRuntimeRef", func() {

			apiGroup := "kubeflow.org"
			kind := "InvalidRuntime"

			trainingRuntimeRef := kubeflowv2.TrainingRuntimeRef{
				Name:     "InvalidRuntimeRef",
				APIGroup: &apiGroup,
				Kind:     &kind,
			}
			jobSpec := kubeflowv2.TrainJobSpec{
				TrainingRuntimeRef: trainingRuntimeRef,
			}
			trainJob := &kubeflowv2.TrainJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubeflowv2.SchemeGroupVersion.String(),
					Kind:       "TrainJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-trainjob",
					Namespace: ns.Name,
				},
				Spec: jobSpec,
			}
			gomega.Expect(k8sClient.Create(ctx, trainJob)).To(gomega.HaveOccurred())
		})

		ginkgo.It("Should fail in creating TrainJob with invalid spec.managedBy", func() {
			managedBy := "invalidManagedBy"
			jobSpec := kubeflowv2.TrainJobSpec{
				ManagedBy: &managedBy,
			}
			trainJob := &kubeflowv2.TrainJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubeflowv2.SchemeGroupVersion.String(),
					Kind:       "TrainJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-trainjob",
					Namespace: ns.Name,
				},
				Spec: jobSpec,
			}
			gomega.Expect(k8sClient.Create(ctx, trainJob)).To(gomega.HaveOccurred())
		})

		ginkgo.It("Should fail in updating spec.managedBy", func() {

			apiGroup := "kubeflow.org"
			kind := "TrainingRuntime"
			managedBy := "kubeflow.org/trainjob-controller"

			trainingRuntimeRef := kubeflowv2.TrainingRuntimeRef{
				Name:     "InvalidRuntimeRef",
				APIGroup: &apiGroup,
				Kind:     &kind,
			}
			jobSpec := kubeflowv2.TrainJobSpec{
				TrainingRuntimeRef: trainingRuntimeRef,
				ManagedBy:          &managedBy,
			}
			trainJob := &kubeflowv2.TrainJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubeflowv2.SchemeGroupVersion.String(),
					Kind:       "TrainJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alpha",
					Namespace: ns.Name,
				},
				Spec: jobSpec,
			}

			gomega.Expect(k8sClient.Create(ctx, trainJob)).Should(gomega.Succeed())
			updatedManagedBy := "kueue.x-k8s.io/multikueue"
			jobSpec.ManagedBy = &updatedManagedBy
			trainJob.Spec = jobSpec
			gomega.Expect(k8sClient.Update(ctx, trainJob)).To(gomega.HaveOccurred())
		})
	})
})
