// Copyright 2022 The Kubeflow Authors
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

package paddle

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	commonv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

var _ = Describe("PaddleJob controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		expectedPort = int32(8080)
	)

	Context("When creating the PaddleJob", func() {
		It("Should get the corresponding resources successfully", func() {
			const (
				namespace = "default"
				name      = "test-job"
			)
			By("By creating a new PaddleJob")
			ctx := context.Background()
			job := newPaddleJobForTest(name, namespace)
			job.Spec.PaddleReplicaSpecs = map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				kubeflowv1.PaddleJobReplicaTypeMaster: {
					Replicas: pointer.Int32(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.PaddleJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.PaddleJobDefaultPortName,
											ContainerPort: expectedPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
								},
							},
						},
					},
				},
				kubeflowv1.PaddleJobReplicaTypeWorker: {
					Replicas: pointer.Int32(2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.PaddleJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.PaddleJobDefaultPortName,
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

			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			key := types.NamespacedName{Name: name, Namespace: namespace}
			created := &kubeflowv1.PaddleJob{}

			// We'll need to retry getting this newly created PaddleJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, key, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			masterKey := types.NamespacedName{Name: fmt.Sprintf("%s-master-0", name), Namespace: namespace}
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
				Name:          kubeflowv1.PaddleJobDefaultPortName,
				ContainerPort: expectedPort,
				Protocol:      corev1.ProtocolTCP}))
			// Check env variable
			Expect(masterPod.Spec.Containers[0].Env).To(ContainElements(corev1.EnvVar{
				Name:  EnvMasterEndpoint,
				Value: fmt.Sprintf("$(POD_IP_DUMMY):%d", expectedPort),
			}))
			// Check service port.
			Expect(masterSvc.Spec.Ports[0].Port).To(Equal(expectedPort))
			// Check owner reference.
			trueVal := true
			Expect(masterPod.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PaddleJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))
			Expect(masterSvc.OwnerReferences).To(ContainElement(metav1.OwnerReference{
				APIVersion:         kubeflowv1.SchemeGroupVersion.String(),
				Kind:               kubeflowv1.PaddleJobKind,
				Name:               name,
				UID:                created.UID,
				Controller:         &trueVal,
				BlockOwnerDeletion: &trueVal,
			}))

			// Test job status.
			masterPod.Status.Phase = corev1.PodSucceeded
			masterPod.ResourceVersion = ""
			Expect(testK8sClient.Status().Update(ctx, masterPod)).Should(Succeed())
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, key, created)
				if err != nil {
					return false
				}
				return created.Status.ReplicaStatuses != nil && created.Status.
					ReplicaStatuses[kubeflowv1.PaddleJobReplicaTypeMaster].Succeeded == 1
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			// Check if the job is succeeded.
			cond := getCondition(created.Status, commonv1.JobSucceeded)
			Expect(cond.Status).To(Equal(corev1.ConditionTrue))
			By("Deleting the PaddleJob")
			Expect(testK8sClient.Delete(ctx, job)).Should(Succeed())
		})
	})
})

func newPaddleJobForTest(name, namespace string) *kubeflowv1.PaddleJob {
	return &kubeflowv1.PaddleJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status commonv1.JobStatus, condType commonv1.JobConditionType) *commonv1.JobCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}
