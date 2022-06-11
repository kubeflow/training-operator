// Copyright 2021 The Kubeflow Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mpi

import (
	"context"
	"fmt"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/types"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/training/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	gpuResourceName         = "nvidia.com/gpu"
	extendedGPUResourceName = "vendor-domain/gpu"
)

func newMPIJobCommon(name string, startTime, completionTime *metav1.Time) *trainingv1.MPIJob {
	cleanPodPolicyAll := common.CleanPodPolicyAll
	mpiJob := &trainingv1.MPIJob{
		TypeMeta: metav1.TypeMeta{APIVersion: trainingv1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: trainingv1.MPIJobSpec{
			RunPolicy: common.RunPolicy{
				CleanPodPolicy: &cleanPodPolicyAll,
			},
			MPIReplicaSpecs: map[common.ReplicaType]*common.ReplicaSpec{
				trainingv1.MPIReplicaTypeWorker: {
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "foo",
									Image: "bar",
								},
							},
						},
					},
				},
				trainingv1.MPIReplicaTypeLauncher: {
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "foo",
									Image: "bar",
								},
							},
						},
					},
				},
			},
		},
		Status: common.JobStatus{},
	}

	if startTime != nil {
		mpiJob.Status.StartTime = startTime
	}
	if completionTime != nil {
		mpiJob.Status.CompletionTime = completionTime
	}

	return mpiJob
}

func newMPIJobOld(name string, replicas *int32, pusPerReplica int64, resourceName string, startTime, completionTime *metav1.Time) *trainingv1.MPIJob {
	mpiJob := newMPIJobCommon(name, startTime, completionTime)

	mpiJob.Spec.MPIReplicaSpecs[trainingv1.MPIReplicaTypeWorker].Replicas = replicas

	workerContainers := mpiJob.Spec.MPIReplicaSpecs[trainingv1.MPIReplicaTypeWorker].Template.Spec.Containers
	for i := range workerContainers {
		container := &workerContainers[i]
		container.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceName(resourceName): *resource.NewQuantity(pusPerReplica, resource.DecimalExponent),
			},
		}
	}

	return mpiJob
}

var newMPIJob = newMPIJobWithLauncher

func newMPIJobWithLauncher(name string, replicas *int32, pusPerReplica int64, resourceName string, startTime, completionTime *metav1.Time) *trainingv1.MPIJob {
	mpiJob := newMPIJobOld(name, replicas, pusPerReplica, resourceName, startTime, completionTime)

	mpiJob.Spec.MPIReplicaSpecs[trainingv1.MPIReplicaTypeLauncher].Replicas = int32Ptr(1)

	launcherContainers := mpiJob.Spec.MPIReplicaSpecs[trainingv1.MPIReplicaTypeLauncher].Template.Spec.Containers
	for i := range launcherContainers {
		container := &launcherContainers[i]
		container.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceName(resourceName): *resource.NewQuantity(pusPerReplica, resource.DecimalExponent),
			},
		}
	}

	return mpiJob
}

var _ = Describe("MPIJob controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = 10 * time.Second
		interval = 1000 * time.Millisecond
	)

	Context("Test launcher is GPU launcher", func() {
		It("Should pass GPU Launcher verification", func() {
			By("By creating MPIJobs with various resource configuration")

			testCases := map[string]struct {
				gpu      string
				expected bool
			}{
				"isNvidiaGPU": {
					gpu:      gpuResourceName,
					expected: true,
				},
				"isExtendedGPU": {
					gpu:      extendedGPUResourceName,
					expected: true,
				},
				"notGPU": {
					gpu:      "vendor-domain/resourcetype",
					expected: false,
				},
			}

			startTime := metav1.Now()
			completionTime := metav1.Now()

			for testName, testCase := range testCases {
				mpiJob := newMPIJobWithLauncher("test-"+strings.ToLower(testName),
					int32Ptr(64), 1, testCase.gpu, &startTime, &completionTime)
				Expect(isGPULauncher(mpiJob) == testCase.expected).To(BeTrue())
			}
		})
	})

	Context("Test MPIJob with succeeded launcher Pod", func() {
		It("Should contains desired launcher ReplicaStatus", func() {
			By("By marking a launcher pod with Phase Succeeded")
			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			jobName := "test-launcher-succeeded"

			mpiJob := newMPIJobWithLauncher(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)
			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			launcher := reconciler.newLauncher(mpiJob, "kubectl-delivery", isGPULauncher(mpiJob))
			launcher.Status.Phase = corev1.PodSucceeded

			launcherKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      launcher.GetName(),
			}
			Eventually(func() error {
				launcherCreated := &corev1.Pod{}
				if err := testK8sClient.Get(ctx, launcherKey, launcherCreated); err != nil {
					return err
				}
				launcherCreated.Status.Phase = corev1.PodSucceeded
				return testK8sClient.Status().Update(ctx, launcherCreated)
			}, timeout, interval).Should(BeNil())

			created := &trainingv1.MPIJob{}
			launcherStatus := &common.ReplicaStatus{
				Active:    0,
				Succeeded: 1,
				Failed:    0,
			}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceDefault, Name: jobName}, created)
				if err != nil {
					return false
				}
				return ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeLauncher, launcherStatus)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Test MPIJob with failed launcher Pod", func() {
		It("Should contains desired launcher ReplicaStatus", func() {
			By("By marking a launcher pod with Phase Failed")
			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			jobName := "test-launcher-failed"

			mpiJob := newMPIJobWithLauncher(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)
			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			launcher := reconciler.newLauncher(mpiJob, "kubectl-delivery", isGPULauncher(mpiJob))
			launcherKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      launcher.GetName(),
			}
			Eventually(func() error {
				launcherCreated := &corev1.Pod{}
				if err := testK8sClient.Get(ctx, launcherKey, launcherCreated); err != nil {
					return err
				}
				launcherCreated.Status.Phase = corev1.PodFailed
				return testK8sClient.Status().Update(ctx, launcherCreated)
			}, timeout, interval).Should(BeNil())

			launcherStatus := &common.ReplicaStatus{
				Active:    0,
				Succeeded: 0,
				Failed:    1,
			}
			created := &trainingv1.MPIJob{}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceDefault, Name: jobName}, created)
				if err != nil {
					return false
				}
				return ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeLauncher, launcherStatus)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Test MPIJob with succeeded launcher pod", func() {
		It("Should contain desired ReplicaStatuses for worker", func() {
			By("By marking the launcher Pod as Succeeded")
			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			jobName := "test-launcher-succeeded2"

			mpiJob := newMPIJobWithLauncher(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)
			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			launcher := reconciler.newLauncher(mpiJob, "kubectl-delivery", isGPULauncher(mpiJob))
			launcher.Status.Phase = corev1.PodSucceeded

			launcherKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      launcher.GetName(),
			}
			Eventually(func() error {
				launcherCreated := &corev1.Pod{}
				if err := testK8sClient.Get(ctx, launcherKey, launcherCreated); err != nil {
					return err
				}
				launcherCreated.Status.Phase = corev1.PodSucceeded
				return testK8sClient.Status().Update(ctx, launcherCreated)
			}, timeout, interval).Should(BeNil())

			created := &trainingv1.MPIJob{}
			launcherStatus := &common.ReplicaStatus{
				Active:    0,
				Succeeded: 0,
				Failed:    0,
			}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceDefault, Name: jobName}, created)
				if err != nil {
					return false
				}
				return ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeWorker, launcherStatus)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Test MPIJob with Running launcher Pod and Pending worker Pods", func() {
		It("Should contain desired ReplicaStatuses", func() {
			By("By marking an active launcher pod and pending worker pods")

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			jobName := "test-launcher-running-worker-pending"

			var replicas int32 = 8
			mpiJob := newMPIJobWithLauncher(jobName, &replicas, 1, gpuResourceName, &startTime, &completionTime)
			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			launcher := reconciler.newLauncher(mpiJob, "kubectl-delivery", isGPULauncher(mpiJob))
			launcherKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      launcher.GetName(),
			}
			Eventually(func() error {
				launcherCreated := &corev1.Pod{}
				if err := testK8sClient.Get(ctx, launcherKey, launcherCreated); err != nil {
					return err
				}
				launcherCreated.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, launcherCreated)
			}, timeout, interval).Should(BeNil())

			for i := 0; i < int(replicas); i++ {
				name := fmt.Sprintf("%s-%d", mpiJob.Name+workerSuffix, i)
				worker := reconciler.newWorker(mpiJob, name)
				workerKey := types.NamespacedName{
					Namespace: metav1.NamespaceDefault,
					Name:      worker.GetName(),
				}
				Eventually(func() error {
					workerCreated := &corev1.Pod{}
					if err := testK8sClient.Get(ctx, workerKey, workerCreated); err != nil {
						return err
					}
					workerCreated.Status.Phase = corev1.PodPending
					return testK8sClient.Status().Update(ctx, workerCreated)
				}, timeout, interval).Should(BeNil())
			}

			key := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      jobName,
			}
			launcherStatus := &common.ReplicaStatus{
				Active:    1,
				Succeeded: 0,
				Failed:    0,
			}
			workerStatus := &common.ReplicaStatus{
				Active:    0,
				Succeeded: 0,
				Failed:    0,
			}
			Eventually(func() bool {
				created := &trainingv1.MPIJob{}
				err := testK8sClient.Get(ctx, key, created)
				if err != nil {
					return false
				}
				return ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeLauncher,
					launcherStatus) && ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeWorker,
					workerStatus)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Test MPIJob with Running launcher Pod and Running worker Pods", func() {
		It("Should contain desired ReplicaStatuses", func() {
			By("By creating an active launcher pod and active worker pods")

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			jobName := "test-launcher-running-worker-running"

			var replicas int32 = 8
			mpiJob := newMPIJob(jobName, &replicas, 1, gpuResourceName, &startTime, &completionTime)
			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			launcher := reconciler.newLauncher(mpiJob, "kubectl-delivery", isGPULauncher(mpiJob))
			launcherKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      launcher.GetName(),
			}
			Eventually(func() error {
				launcherCreated := &corev1.Pod{}
				if err := testK8sClient.Get(ctx, launcherKey, launcherCreated); err != nil {
					return err
				}
				launcherCreated.Status.Phase = corev1.PodRunning
				return testK8sClient.Status().Update(ctx, launcherCreated)
			}, timeout, interval).Should(BeNil())

			for i := 0; i < int(replicas); i++ {
				name := fmt.Sprintf("%s-%d", mpiJob.Name+workerSuffix, i)
				worker := reconciler.newWorker(mpiJob, name)
				workerKey := types.NamespacedName{
					Namespace: metav1.NamespaceDefault,
					Name:      worker.GetName(),
				}
				Eventually(func() error {
					workerCreated := &corev1.Pod{}
					if err := testK8sClient.Get(ctx, workerKey, workerCreated); err != nil {
						return err
					}
					workerCreated.Status.Phase = corev1.PodRunning
					return testK8sClient.Status().Update(ctx, workerCreated)
				}, timeout, interval).Should(BeNil())
			}

			key := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      jobName,
			}
			launcherStatus := &common.ReplicaStatus{
				Active:    1,
				Succeeded: 0,
				Failed:    0,
			}
			workerStatus := &common.ReplicaStatus{
				Active:    8,
				Succeeded: 0,
				Failed:    0,
			}
			Eventually(func() bool {
				created := &trainingv1.MPIJob{}
				err := testK8sClient.Get(ctx, key, created)
				if err != nil {
					return false
				}
				return ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeLauncher,
					launcherStatus) && ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeWorker,
					workerStatus)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Test MPIJob with Running worker Pods", func() {
		It("Should contain desired ReplicaStatuses and create a launcher pod", func() {
			By("By creating only active worker pods")

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			jobName := "test-worker-running"

			var replicas int32 = 16
			mpiJob := newMPIJob(jobName, &replicas, 1, gpuResourceName, &startTime, &completionTime)
			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			for i := 0; i < int(replicas); i++ {
				name := fmt.Sprintf("%s-%d", mpiJob.Name+workerSuffix, i)
				worker := reconciler.newWorker(mpiJob, name)
				workerKey := types.NamespacedName{
					Namespace: metav1.NamespaceDefault,
					Name:      worker.GetName(),
				}
				Eventually(func() error {
					workerCreated := &corev1.Pod{}
					if err := testK8sClient.Get(ctx, workerKey, workerCreated); err != nil {
						return err
					}
					workerCreated.Status.Phase = corev1.PodRunning
					return testK8sClient.Status().Update(ctx, workerCreated)
				}, timeout, interval).Should(BeNil())
			}

			launcherKey := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      mpiJob.Name + launcherSuffix,
			}
			launcher := &trainingv1.MPIJob{}
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, launcherKey, launcher)
				return err != nil
			}, timeout, interval).Should(BeTrue())

			key := types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      jobName,
			}
			launcherStatus := &common.ReplicaStatus{
				Active:    0,
				Succeeded: 0,
				Failed:    0,
			}
			workerStatus := &common.ReplicaStatus{
				Active:    16,
				Succeeded: 0,
				Failed:    0,
			}
			Eventually(func() bool {
				created := &trainingv1.MPIJob{}
				err := testK8sClient.Get(ctx, key, created)
				if err != nil {
					return false
				}
				return ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeLauncher,
					launcherStatus) && ReplicaStatusMatch(created.Status.ReplicaStatuses, trainingv1.MPIReplicaTypeWorker,
					workerStatus)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("MPIJob not found", func() {
		It("Should do nothing", func() {
			By("Calling Reconcile method")
			jobName := "test-not-exist"

			ctx := context.Background()

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      jobName,
			}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).Should(BeNil())
		})
	})

	Context("MPIJob with launcher Pod not controlled by itself", func() {
		It("Should return error", func() {
			By("Calling Reconcile method")
			jobName := "test-launcher-orphan"
			testKind := "Pod"

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			mpiJob := newMPIJob(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)

			launcher := reconciler.newLauncher(mpiJob, "kubectl-delivery", isGPULauncher(mpiJob))
			launcher.OwnerReferences = nil
			Expect(testK8sClient.Create(ctx, launcher)).Should(Succeed())

			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      mpiJob.GetName(),
			}}
			expectedErr := fmt.Errorf(MessageResourceExists, launcher.Name, testKind)
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, req)
				return err
			}, timeout, interval).Should(MatchError(expectedErr))
		})
	})

	Context("MPIJob with worker Pod not controlled by itself", func() {
		It("Should return error", func() {
			By("Calling Reconcile method")
			jobName := "test-worker-orphan"
			testKind := "Pod"

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			mpiJob := newMPIJob(jobName, int32Ptr(1), 1, gpuResourceName, &startTime, &completionTime)

			for i := 0; i < 1; i++ {
				name := fmt.Sprintf("%s-%d", mpiJob.Name+workerSuffix, i)
				worker := reconciler.newWorker(mpiJob, name)
				worker.OwnerReferences = nil
				Expect(testK8sClient.Create(ctx, worker)).Should(Succeed())
			}

			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      mpiJob.GetName(),
			}}
			expectedErr := fmt.Errorf(MessageResourceExists, fmt.Sprintf("%s-%d", mpiJob.Name+workerSuffix, 0), testKind)
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, req)
				return err
			}, timeout, interval).Should(MatchError(expectedErr))
		})
	})

	Context("MPIJob with ConfigMap not controlled by itself", func() {
		It("Should return error", func() {
			By("Calling Reconcile method")
			jobName := "test-cm-orphan"
			testKind := "ConfigMap"

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			mpiJob := newMPIJob(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)

			cm := newConfigMap(mpiJob, 64, isGPULauncher(mpiJob))
			cm.OwnerReferences = nil
			Expect(testK8sClient.Create(ctx, cm)).Should(Succeed())

			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      mpiJob.GetName(),
			}}
			expectedErr := fmt.Errorf(MessageResourceExists, cm.Name, testKind)
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, req)
				return err
			}, timeout, interval).Should(MatchError(expectedErr))
		})
	})

	Context("MPIJob with ServiceAccount not controlled by itself", func() {
		It("Should return error", func() {
			By("Calling Reconcile method")
			jobName := "test-sa-orphan"
			testKind := "ServiceAccount"

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			mpiJob := newMPIJob(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)

			sa := newLauncherServiceAccount(mpiJob)
			sa.OwnerReferences = nil
			Expect(testK8sClient.Create(ctx, sa)).Should(Succeed())

			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      mpiJob.GetName(),
			}}
			expectedErr := fmt.Errorf(MessageResourceExists, sa.Name, testKind)
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, req)
				return err
			}, timeout, interval).Should(MatchError(expectedErr))
		})
	})

	Context("MPIJob with Role not controlled by itself", func() {
		It("Should return error", func() {
			By("Calling Reconcile method")
			jobName := "test-role-orphan"
			testKind := "Role"

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			mpiJob := newMPIJob(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)

			role := newLauncherRole(mpiJob, 64)
			role.OwnerReferences = nil
			Expect(testK8sClient.Create(ctx, role)).Should(Succeed())

			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      mpiJob.GetName(),
			}}
			expectedErr := fmt.Errorf(MessageResourceExists, role.Name, testKind)
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, req)
				return err
			}, timeout, interval).Should(MatchError(expectedErr))
		})
	})

	Context("MPIJob with RoleBinding not controlled by itself", func() {
		It("Should return error", func() {
			By("Calling Reconcile method")
			jobName := "test-rb-orphan"
			testKind := "RoleBinding"

			ctx := context.Background()
			startTime := metav1.Now()
			completionTime := metav1.Now()

			mpiJob := newMPIJob(jobName, int32Ptr(64), 1, gpuResourceName, &startTime, &completionTime)

			rb := newLauncherRoleBinding(mpiJob)
			rb.OwnerReferences = nil
			Expect(testK8sClient.Create(ctx, rb)).Should(Succeed())

			Expect(testK8sClient.Create(ctx, mpiJob)).Should(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceDefault,
				Name:      mpiJob.GetName(),
			}}
			expectedErr := fmt.Errorf(MessageResourceExists, rb.Name, testKind)
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, req)
				return err
			}, timeout, interval).Should(MatchError(expectedErr))
		})
	})

})

func ReplicaStatusMatch(replicaStatuses map[common.ReplicaType]*common.ReplicaStatus,
	replicaType common.ReplicaType, status *common.ReplicaStatus) bool {

	result := true

	if replicaStatuses == nil {
		return false
	}
	if val, exist := replicaStatuses[replicaType]; !exist {
		return false
	} else {
		result = result && (val.Active == status.Active)
		result = result && (val.Succeeded == status.Succeeded)
		result = result && (val.Failed == status.Failed)
	}

	return result
}

func int32Ptr(i int32) *int32 { return &i }
