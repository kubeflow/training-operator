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
	"os"
	"time"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/common/util/v1/testutil"
)

var _ = Describe("TFJob controller", func() {
	const (
		timeout  = 10 * time.Second
		interval = 1000 * time.Millisecond
	)

	Context("Test ClusterSpec", func() {
		It("should generate desired cluster spec", func() {
			type tc struct {
				tfJob               *kubeflowv1.TFJob
				rt                  string
				index               string
				customClusterDomain string
				expectedClusterSpec string
			}
			testCase := []tc{
				{
					tfJob:               testutil.NewTFJobWithNamespace(1, 0, "ns0"),
					rt:                  "worker",
					index:               "0",
					customClusterDomain: "",
					expectedClusterSpec: "",
				},
				{
					tfJob:               testutil.NewTFJobWithNamespace(1, 0, "ns1"),
					rt:                  "worker",
					index:               "0",
					customClusterDomain: "tf.training.com",
					expectedClusterSpec: "",
				},
				{
					tfJob:               testutil.NewTFJobWithNamespace(1, 1, "ns2"),
					rt:                  "worker",
					index:               "0",
					customClusterDomain: "tf.training.org",
					expectedClusterSpec: `{"cluster":{"ps":["` + testutil.TestTFJobName +
						`-ps-0.ns2.svc.tf.training.org:2222"],"worker":["` + testutil.TestTFJobName +
						`-worker-0.ns2.svc.tf.training.org:2222"]},"task":{"type":"worker","index":0},"environment":"cloud"}`,
				},
				{
					tfJob:               testutil.NewTFJobWithEvaluatorAndNamespace(1, 1, 1, "ns3"),
					rt:                  "worker",
					index:               "0",
					customClusterDomain: "tf.training.io",
					expectedClusterSpec: `{"cluster":{"evaluator":["` + testutil.TestTFJobName +
						`-evaluator-0.ns3.svc.tf.training.io:2222"],"ps":["` + testutil.TestTFJobName +
						`-ps-0.ns3.svc.tf.training.io:2222"],"worker":["` + testutil.TestTFJobName +
						`-worker-0.ns3.svc.tf.training.io:2222"]},"task":{"type":"worker","index":0},"environment":"cloud"}`,
				},
				{
					tfJob:               testutil.NewTFJobWithEvaluatorAndNamespace(1, 1, 1, "ns3"),
					rt:                  "worker",
					index:               "0",
					customClusterDomain: "",
					expectedClusterSpec: `{"cluster":{"evaluator":["` + testutil.TestTFJobName +
						`-evaluator-0.ns3.svc:2222"],"ps":["` + testutil.TestTFJobName +
						`-ps-0.ns3.svc:2222"],"worker":["` + testutil.TestTFJobName +
						`-worker-0.ns3.svc:2222"]},"task":{"type":"worker","index":0},"environment":"cloud"}`,
				},
			}

			for _, c := range testCase {
				c.tfJob.SetName("test-tfjob")
				c.tfJob.SetUID(uuid.NewUUID())
				_ = os.Setenv(EnvCustomClusterDomain, c.customClusterDomain)

				podTemplate := c.tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].Template.DeepCopy()

				podTemplate.Name = core.GenGeneralName(c.tfJob.GetName(), c.rt, c.index)

				if podTemplate.Labels == nil {
					podTemplate.Labels = map[string]string{}
				}

				jobName := c.tfJob.GetName()
				labels := reconciler.GenLabels(jobName)
				labels[commonv1.ReplicaTypeLabelDeprecated] = c.rt
				labels[commonv1.ReplicaTypeLabel] = c.rt
				labels[commonv1.ReplicaIndexLabelDeprecated] = c.index
				labels[commonv1.ReplicaIndexLabel] = c.index

				Expect(reconciler.SetClusterSpec(c.tfJob, podTemplate, c.rt, c.index)).Should(Succeed())

				if c.expectedClusterSpec == "" {
					Expect(len(podTemplate.Spec.Containers[0].Env)).Should(Equal(0))
				} else {
					actual := podTemplate.Spec.Containers[0].Env[0].Value
					reconciler.Log.Info("printing cluster spec", "expected", c.expectedClusterSpec, "actual pod", podTemplate)
					Expect(actual).Should(Equal(c.expectedClusterSpec))
				}
			}
		})
	})

	Context("Test IsDistributed", func() {
		It("should returns correctly", func() {
			type tc struct {
				tfJob    *kubeflowv1.TFJob
				expected bool
			}
			testCase := []tc{
				{
					tfJob:    testutil.NewTFJob(1, 0),
					expected: false,
				},
				{
					tfJob:    testutil.NewTFJob(1, 1),
					expected: true,
				},
				{
					tfJob:    testutil.NewTFJob(0, 1),
					expected: false,
				},
				{
					tfJob:    testutil.NewTFJobWithChief(1, 0),
					expected: true,
				},
			}
			for _, c := range testCase {
				Expect(isDistributed(c.tfJob)).To(Equal(c.expected))
			}
		})
	})

	Context("Test Restart Policy", func() {
		It("should assign proper restart policy to pod", func() {
			type tc struct {
				tfJob                 *kubeflowv1.TFJob
				expectedRestartPolicy corev1.RestartPolicy
				expectedType          commonv1.ReplicaType
			}
			testCase := []tc{
				func() tc {
					tfJob := testutil.NewTFJob(1, 0)
					specRestartPolicy := commonv1.RestartPolicyExitCode
					tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].RestartPolicy = specRestartPolicy
					return tc{
						tfJob:                 tfJob,
						expectedRestartPolicy: corev1.RestartPolicyNever,
						expectedType:          kubeflowv1.TFJobReplicaTypeWorker,
					}
				}(),
				func() tc {
					tfJob := testutil.NewTFJob(1, 0)
					specRestartPolicy := commonv1.RestartPolicyNever
					tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].RestartPolicy = specRestartPolicy
					return tc{
						tfJob:                 tfJob,
						expectedRestartPolicy: corev1.RestartPolicyNever,
						expectedType:          kubeflowv1.TFJobReplicaTypeWorker,
					}
				}(),
				func() tc {
					tfJob := testutil.NewTFJob(1, 0)
					specRestartPolicy := commonv1.RestartPolicyAlways
					tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].RestartPolicy = specRestartPolicy
					return tc{
						tfJob:                 tfJob,
						expectedRestartPolicy: corev1.RestartPolicyAlways,
						expectedType:          kubeflowv1.TFJobReplicaTypeWorker,
					}
				}(),
				func() tc {
					tfJob := testutil.NewTFJob(1, 0)
					specRestartPolicy := commonv1.RestartPolicyOnFailure
					tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].RestartPolicy = specRestartPolicy
					return tc{
						tfJob:                 tfJob,
						expectedRestartPolicy: corev1.RestartPolicyOnFailure,
						expectedType:          kubeflowv1.TFJobReplicaTypeWorker,
					}
				}(),
			}
			for _, c := range testCase {
				spec := c.tfJob.Spec.TFReplicaSpecs[c.expectedType]
				podTemplate := spec.Template
				setRestartPolicy(&podTemplate, spec)
				Expect(podTemplate.Spec.RestartPolicy).To(Equal(c.expectedRestartPolicy))
			}
		})
	})

	Context("Test Exit Code", func() {
		It("should delete designated Pod", func() {
			By("Creating TFJob \"test-exit-code\" with 1 worker only")
			ctx := context.Background()

			tfJob := testutil.NewTFJob(1, 0)
			tfJob.SetName("test-exit-code")
			tfJob.SetUID(uuid.NewUUID())
			tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker].RestartPolicy = commonv1.RestartPolicyExitCode

			refs := []metav1.OwnerReference{
				*reconciler.GenOwnerReference(tfJob),
			}
			By("creating worker Pod")
			pod := testutil.NewPod(tfJob, testutil.LabelWorker, 0, refs)
			basicLabels := reconciler.GenLabels(tfJob.GetName())
			for k, v := range basicLabels {
				pod.Labels[k] = v
			}
			pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
				Name:  kubeflowv1.TFJobDefaultContainerName,
				Image: testutil.DummyContainerImage,
			})
			Expect(testK8sClient.Create(ctx, pod)).Should(Succeed())

			created := &corev1.Pod{}
			key := types.NamespacedName{Namespace: metav1.NamespaceDefault, Name: pod.GetName()}
			Expect(testK8sClient.Get(ctx, key, created)).Should(Succeed())
			created.Status.Phase = corev1.PodFailed
			created.Status.ContainerStatuses = append(created.Status.ContainerStatuses, corev1.ContainerStatus{
				Name: kubeflowv1.TFJobDefaultContainerName,
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 130,
					},
				},
			})
			Expect(testK8sClient.Status().Update(ctx, created))

			// Make sure the version of pod created is updated with desired status
			Eventually(func() error {
				updated := &corev1.Pod{}
				if err := testK8sClient.Get(ctx, key, updated); err != nil {
					return err
				}
				if updated.Status.Phase != corev1.PodFailed {
					return fmt.Errorf("pod status is not Failed")
				}
				return nil
			}, timeout, interval).Should(BeNil())

			_ = reconciler.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy)

			Eventually(func() bool {
				noPod := &corev1.Pod{}
				err := testK8sClient.Get(ctx, key, noPod)
				if err == nil {
					reconciler.Log.Info("still got pod", "jobName", tfJob.GetName(), "pod", noPod)
					return noPod.GetDeletionTimestamp() != nil
				}
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe("Test Scale Down", func() {
		It("should delete redundant Pods", func() {
			ctx := context.Background()

			tfJob := testutil.NewTFJob(2, 0)
			//tfJob.SelfLink = "/api/v1/namespaces/default/tfjob/test-tfjob"
			tfJob.SetName("test-scale-down")
			tfJob.SetUID(uuid.NewUUID())
			tfJob.Spec.EnableDynamicWorker = true

			refs := []metav1.OwnerReference{*reconciler.GenOwnerReference(tfJob)}

			pods := []*corev1.Pod{
				testutil.NewPod(tfJob, testutil.LabelWorker, 0, refs),
				testutil.NewPod(tfJob, testutil.LabelWorker, 1, refs),
				testutil.NewPod(tfJob, testutil.LabelWorker, 2, refs),
			}

			for i := range pods {
				pod := pods[i]
				for k, v := range reconciler.GenLabels(tfJob.GetName()) {
					pod.Labels[k] = v
				}
				Expect(testK8sClient.Create(ctx, pod)).Should(Succeed())
			}

			// Ensure the created Pods are all in cache
			Eventually(func() error {
				podList := &corev1.PodList{}
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: reconciler.GenLabels(tfJob.GetName()),
				})
				if err != nil {
					return err
				}
				listOpt := client.MatchingLabelsSelector{
					Selector: selector,
				}
				err = testK8sClient.List(ctx, podList, listOpt)
				if err != nil {
					return err
				}
				if len(podList.Items) != 3 {
					return fmt.Errorf("expecting %d Pods while got %d", 3, len(podList.Items))
				}
				return nil
			}, timeout, interval).Should(BeNil())

			_ = reconciler.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy)

			noKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      pods[2].GetName(),
			}
			Eventually(func() bool {
				noPod := &corev1.Pod{}
				err := testK8sClient.Get(ctx, noKey, noPod)
				if err == nil {
					return false
				}
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe("Test Scale Up", func() {
		It("should create missing Pods", func() {
			ctx := context.Background()

			tfJob := testutil.NewTFJob(3, 0)
			tfJob.SetName("test-scale-up")
			tfJob.SetUID(uuid.NewUUID())
			tfJob.Spec.EnableDynamicWorker = true

			refs := []metav1.OwnerReference{*reconciler.GenOwnerReference(tfJob)}

			pods := []*corev1.Pod{
				testutil.NewPod(tfJob, testutil.LabelWorker, 0, refs),
			}

			for i := range pods {
				pod := pods[i]
				for k, v := range reconciler.GenLabels(tfJob.GetName()) {
					pod.Labels[k] = v
				}
				Expect(testK8sClient.Create(ctx, pod)).Should(Succeed())
			}

			// Ensure the created Pods are all in cache
			Eventually(func() error {
				podList := &corev1.PodList{}
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: reconciler.GenLabels(tfJob.GetName()),
				})
				if err != nil {
					return err
				}
				listOpt := client.MatchingLabelsSelector{
					Selector: selector,
				}
				err = testK8sClient.List(ctx, podList, listOpt)
				if err != nil {
					return err
				}
				if len(podList.Items) != 1 {
					return fmt.Errorf("before reconciling, expecting %d Pods while got %d", 1, len(podList.Items))
				}
				return nil
			}, timeout, interval).Should(BeNil())

			_ = reconciler.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy)

			// Check if there are two more Pods created
			Eventually(func() error {
				podList := &corev1.PodList{}
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: reconciler.GenLabels(tfJob.GetName()),
				})
				if err != nil {
					return err
				}
				listOpt := client.MatchingLabelsSelector{
					Selector: selector,
				}
				err = testK8sClient.List(ctx, podList, listOpt)
				if err != nil {
					return err
				}
				if len(podList.Items) != 3 {
					return fmt.Errorf("after reconciling, expecting %d Pods while got %d", 3, len(podList.Items))
				}
				return nil
			}, timeout, interval).Should(BeNil())
		})
	})

	Describe("TestIsWorker0Completed", func() {
		It("should match expected result", func() {
			newInt32 := func(in int32) *int32 {
				return &in
			}
			tests := []struct {
				// worker failed, succeeded, running num
				workers     [3]int32
				tfJob       *kubeflowv1.TFJob
				replicas    map[commonv1.ReplicaType]*commonv1.ReplicaSpec
				expected    bool
				expectedErr bool
			}{
				{
					workers:     [3]int32{0, 0, 1},
					tfJob:       testutil.NewTFJobV2(1, 1, 0, 0, 0),
					expected:    false,
					expectedErr: false,
					replicas: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.TFJobReplicaTypeWorker: {
							Replicas: newInt32(1),
							Template: testutil.NewTFReplicaSpecTemplate(),
						},
						kubeflowv1.TFJobReplicaTypePS: {
							Replicas: newInt32(1),
							Template: testutil.NewTFReplicaSpecTemplate(),
						},
					},
				},
				{
					workers:     [3]int32{0, 1, 0},
					tfJob:       testutil.NewTFJobV2(1, 0, 0, 0, 0),
					expected:    true,
					expectedErr: false,
					replicas: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.TFJobReplicaTypeWorker: {
							Replicas: newInt32(1),
							Template: testutil.NewTFReplicaSpecTemplate(),
						},
					},
				},
				{
					workers:     [3]int32{0, 0, 0},
					tfJob:       testutil.NewTFJobV2(0, 0, 1, 0, 0),
					expected:    true,
					expectedErr: false,
					replicas: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.TFJobReplicaTypeMaster: {
							Replicas: newInt32(1),
							Template: testutil.NewTFReplicaSpecTemplate(),
						},
					},
				},
				{
					workers:     [3]int32{0, 0, 0},
					tfJob:       testutil.NewTFJobV2(0, 0, 0, 1, 0),
					expected:    true,
					expectedErr: false,
					replicas: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.TFJobReplicaTypeChief: {
							Replicas: newInt32(1),
							Template: testutil.NewTFReplicaSpecTemplate(),
						},
					},
				},
				{
					workers:     [3]int32{1, 1, 0},
					tfJob:       testutil.NewTFJobV2(2, 0, 0, 0, 0),
					expected:    true,
					expectedErr: false,
					replicas: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.TFJobReplicaTypeWorker: {
							Replicas: newInt32(2),
							Template: testutil.NewTFReplicaSpecTemplate(),
						},
					},
				},
				{
					workers:     [3]int32{1, 0, 1},
					tfJob:       testutil.NewTFJobV2(2, 0, 0, 0, 0),
					expected:    false,
					expectedErr: false,
					replicas: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						kubeflowv1.TFJobReplicaTypeWorker: {
							Replicas: newInt32(2),
							Template: testutil.NewTFReplicaSpecTemplate(),
						},
					},
				},
			}

			jobNameTemplate := "test-worker0-complete-%d"
			for i, tt := range tests {
				tt.tfJob.SetName(fmt.Sprintf(jobNameTemplate, i))
				tt.tfJob.SetUID(uuid.NewUUID())
				// only related to worker status
				initializeReplicaStatuses(&tt.tfJob.Status, kubeflowv1.TFJobReplicaTypeWorker)
				// set status and add pod to indexer
				setStatusForTest(tt.tfJob, kubeflowv1.TFJobReplicaTypeWorker, tt.workers[0], tt.workers[1], tt.workers[2], false, true, testK8sClient)

				// Adding this section to make sure all pods are created and cached
				Eventually(func() error {
					podList := &corev1.PodList{}
					selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
						MatchLabels: reconciler.GenLabels(tt.tfJob.GetName()),
					})
					if err != nil {
						return err
					}
					listOpt := client.MatchingLabelsSelector{
						Selector: selector,
					}
					err = testK8sClient.List(context.Background(), podList, listOpt)
					if err != nil {
						return nil
					}
					totalExpectedPodCount := tt.workers[0] + tt.workers[1] + tt.workers[2]
					if len(podList.Items) != int(totalExpectedPodCount) {
						return fmt.Errorf("pod number (%d) for %s not match for expected pod number %d",
							len(podList.Items), tt.tfJob.GetName(), totalExpectedPodCount)
					}
					return nil
				}, timeout, interval).Should(BeNil())

				got, err := reconciler.IsWorker0Completed(tt.tfJob, tt.replicas)

				if err != nil {
					Expect(err).To(Equal(tt.expectedErr))
				} else {
					Expect(got).To(Equal(tt.expected))
				}
			}
		})
	})
})
