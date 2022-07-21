// Copyright 2021 The Kubeflow Authors
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

package tensorflow

import (
	"context"
	"fmt"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/common/util/v1/testutil"
)

var _ = Describe("TFJob controller", func() {
	Context("Test Normal Path", func() {
		It("should create desired Pods and Services", func() {
			var (
				tfJobRunning   = commonv1.JobRunning
				tfJobSucceeded = commonv1.JobSucceeded
			)

			testCases := map[string]struct {
				worker int
				ps     int

				// pod setup
				// ControllerError error
				// jobKeyForget    bool

				pendingWorkerPods   int32
				activeWorkerPods    int32
				succeededWorkerPods int32
				failedWorkerPods    int32

				pendingPSPods   int32
				activePSPods    int32
				succeededPSPods int32
				failedPSPods    int32

				activeWorkerServices int32
				activePSServices     int32

				// expectations
				expectedPodCreations     int32
				expectedPodDeletions     int32
				expectedServiceCreations int32

				expectedActiveWorkerPods    int32
				expectedSucceededWorkerPods int32
				expectedFailedWorkerPods    int32

				expectedActivePSPods    int32
				expectedSucceededPSPods int32
				expectedFailedPSPods    int32

				expectedCondition       *commonv1.JobConditionType
				expectedConditionReason string

				// There are some cases that should not check start time since the field should be set in the previous sync loop.
				needCheckStartTime bool
			}{
				"Local TFJob is created": {
					1, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0,
					1, 0, 1,
					0, 0, 0,
					0, 0, 0,
					// We can not check if it is created since the condition is set in addTFJob.
					nil, "",
					false,
				},
				"Distributed TFJob (4 workers, 2 PS) is created": {
					4, 2,
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0,
					6, 0, 6,
					0, 0, 0,
					0, 0, 0,
					nil, "",
					false,
				},
				"Distributed TFJob (4 workers, 2 PS) is created and all replicas are pending": {
					4, 2,
					4, 0, 0, 0,
					2, 0, 0, 0,
					4, 2,
					0, 0, 0,
					0, 0, 0,
					0, 0, 0,
					nil, "",
					false,
				},
				"Distributed TFJob (4 workers, 2 PS) is created and all replicas are running": {
					4, 2,
					0, 4, 0, 0,
					0, 2, 0, 0,
					4, 2,
					0, 0, 0,
					4, 0, 0,
					2, 0, 0,
					&tfJobRunning, tfJobRunningReason,
					true,
				},
				"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending": {
					4, 2,
					2, 0, 0, 0,
					1, 0, 0, 0,
					2, 1,
					3, 0, 3,
					0, 0, 0,
					0, 0, 0,
					nil, "",
					false,
				},
				"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is running": {
					4, 2,
					2, 1, 0, 0,
					1, 0, 0, 0,
					3, 1,
					2, 0, 2,
					1, 0, 0,
					0, 0, 0,
					&tfJobRunning, tfJobRunningReason,
					false,
				},
				"Distributed TFJob (4 workers, 2 PS) is created, 2 workers, 1 PS are pending, 1 worker is succeeded": {
					4, 2,
					2, 0, 1, 0,
					1, 0, 0, 0,
					3, 1,
					2, 0, 2,
					0, 1, 0,
					0, 0, 0,
					nil, "",
					false,
				},
				"Distributed TFJob (4 workers, 2 PS) is succeeded": {
					4, 2,
					0, 0, 4, 0,
					0, 0, 2, 0,
					4, 2,
					0, 0, 0,
					0, 4, 0,
					0, 2, 0,
					&tfJobSucceeded, tfJobSucceededReason,
					false,
				},
			}

			jobNameTemplate := "test-case-norm-%d"
			caseIdx := 0
			for name, tc := range testCases {
				By(name)
				ctx := context.Background()
				jobName := fmt.Sprintf(jobNameTemplate, caseIdx)
				caseIdx++

				tfJob := testutil.NewTFJob(tc.worker, tc.ps)
				tfJob.SetName(jobName)
				tfJob.SetUID(uuid.NewUUID())

				refs := []metav1.OwnerReference{*reconciler.GenOwnerReference(tfJob)}
				basicLabels := reconciler.GenLabels(tfJob.GetName())

				testutil.SetPodsStatuses(testK8sClient, tfJob, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, nil, refs, basicLabels)
				testutil.SetPodsStatuses(testK8sClient, tfJob, testutil.LabelPS, tc.pendingPSPods, tc.activePSPods, tc.succeededPSPods, tc.failedPSPods, nil, refs, basicLabels)

				testutil.SetServices(testK8sClient, tfJob, testutil.LabelWorker, tc.activeWorkerServices, refs, basicLabels)
				testutil.SetServices(testK8sClient, tfJob, testutil.LabelPS, tc.activePSServices, refs, basicLabels)

				totalPodNumber := int(tc.pendingWorkerPods + tc.activeWorkerPods + tc.succeededWorkerPods + tc.failedWorkerPods + tc.pendingPSPods + tc.activePSPods + tc.succeededPSPods + tc.failedPSPods)
				totalServiceNumber := int(tc.activeWorkerServices + tc.activePSServices)

				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: reconciler.GenLabels(tfJob.GetName())})
				Expect(err).Should(BeNil())
				listOpt := client.MatchingLabelsSelector{Selector: selector}
				Eventually(func() error {
					podList := &corev1.PodList{}
					svcList := &corev1.ServiceList{}

					err = testK8sClient.List(ctx, podList, listOpt)
					if err != nil {
						return err
					}
					if len(podList.Items) != totalPodNumber {
						return fmt.Errorf("expected %d Pods, got %d", totalPodNumber, len(podList.Items))
					}

					err = testK8sClient.List(ctx, svcList, listOpt)
					if err != nil {
						return err
					}
					if len(svcList.Items) != totalServiceNumber {
						return fmt.Errorf("expected %d Services, got %d", totalServiceNumber, len(svcList.Items))
					}
					return nil
				}).Should(BeNil())

				_ = reconciler.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy)

				// Check the number of Pods and Services
				//var pods []*corev1.Pod = nil
				//var svcs []*corev1.Service = nil
				Eventually(func() error {
					podList := &corev1.PodList{}
					svcList := &corev1.ServiceList{}

					err = testK8sClient.List(ctx, podList, listOpt)
					if err != nil {
						return err
					}
					podCreatedNumber := 0
					if len(podList.Items) > totalPodNumber {
						podCreatedNumber = len(podList.Items) - totalPodNumber
					}
					podDeletedNumber := 0
					if len(podList.Items) < totalPodNumber {
						podDeletedNumber = totalPodNumber - len(podList.Items)
					}
					if podCreatedNumber != int(tc.expectedPodCreations) {
						return fmt.Errorf("%s: unexpected number of pod creates.  Expected %d, saw %d\n", name, tc.expectedPodCreations, podCreatedNumber)
					}
					if podDeletedNumber != int(tc.expectedPodDeletions) {
						return fmt.Errorf("%s: unexpected number of service creates.  Expected %d, saw %d\n", name, tc.expectedServiceCreations, podDeletedNumber)
					}
					// check controller references for all pods
					for _, p := range podList.Items {
						for _, ref := range p.GetOwnerReferences() {
							if ref.APIVersion != kubeflowv1.SchemeGroupVersion.String() {
								return fmt.Errorf("controllerRef.APIVersion = %q, want %q", ref.APIVersion, kubeflowv1.SchemeGroupVersion.String())
							}
							if ref.Kind != kubeflowv1.TFJobKind {
								return fmt.Errorf("controllerRef.MPIKind = %q, want %q", ref.Kind, kubeflowv1.TFJobKind)
							}
							if ref.Name != tfJob.GetName() {
								return fmt.Errorf("controllerRef.Name = %q, want %q", ref.Name, tfJob.GetName())
							}
							if ref.UID != tfJob.GetUID() {
								return fmt.Errorf("controllerRef.UID = %q, want %q", ref.UID, tfJob.GetUID())
							}
						}
					}

					err = testK8sClient.List(ctx, svcList, listOpt)
					if err != nil {
						return err
					}
					serviceCreatedNumber := 0
					if len(svcList.Items) > totalServiceNumber {
						serviceCreatedNumber = len(svcList.Items) - totalServiceNumber
					}
					if serviceCreatedNumber != int(tc.expectedServiceCreations) {
						return fmt.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", name, tc.expectedPodDeletions, serviceCreatedNumber)
					}
					// check controller reference for all services
					for _, s := range svcList.Items {
						for _, ref := range s.GetOwnerReferences() {
							if ref.APIVersion != kubeflowv1.SchemeGroupVersion.String() {
								return fmt.Errorf("controllerRef.APIVersion = %q, want %q", ref.APIVersion, kubeflowv1.SchemeGroupVersion.String())
							}
							if ref.Kind != kubeflowv1.TFJobKind {
								return fmt.Errorf("controllerRef.MPIKind = %q, want %q", ref.Kind, kubeflowv1.TFJobKind)
							}
							if ref.Name != tfJob.GetName() {
								return fmt.Errorf("controllerRef.Name = %q, want %q", ref.Name, tfJob.GetName())
							}
							if ref.UID != tfJob.GetUID() {
								return fmt.Errorf("controllerRef.UID = %q, want %q", ref.UID, tfJob.GetUID())
							}
						}
					}
					return nil
				}).Should(BeNil())

				// Validate Worker status
				if tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker] != nil {
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Active).To(Equal(tc.expectedActiveWorkerPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Succeeded).To(Equal(tc.expectedSucceededWorkerPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypeWorker].Failed).To(Equal(tc.expectedFailedWorkerPods))
				}
				// Validate PS status
				if tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS] != nil {
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS].Active).To(Equal(tc.expectedActivePSPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS].Succeeded).To(Equal(tc.expectedSucceededPSPods))
					Expect(tfJob.Status.ReplicaStatuses[kubeflowv1.TFJobReplicaTypePS].Failed).To(Equal(tc.expectedFailedPSPods))
				}

				// Validate StartTime
				if tc.needCheckStartTime {
					Expect(tfJob.Status.StartTime).NotTo(BeNil())
				}

				// Validate Conditions
				if tc.expectedCondition != nil {
					Expect(testutil.CheckCondition(tfJob, *tc.expectedCondition, tc.expectedConditionReason)).Should(BeTrue())
				}
			}
		})
	})
})
