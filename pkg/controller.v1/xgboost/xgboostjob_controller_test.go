package xgboost

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

var _ = Describe("XGBoost controller", func() {
	// Define utility constants for object names.
	const (
		expectedPort = int32(9999)
	)
	var (
		cleanPodPolicyAll = kubeflowv1.CleanPodPolicyAll
	)

	Context("", func() {
		const name = "test-job"
		var (
			ns         *corev1.Namespace
			job        *kubeflowv1.XGBoostJob
			jobKey     types.NamespacedName
			masterKey  types.NamespacedName
			worker0Key types.NamespacedName
			ctx        = context.Background()
		)
		BeforeEach(func() {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "xgboost-test-",
				},
			}
			Expect(testK8sClient.Create(ctx, ns)).Should(Succeed())

			job = newXGBoostForTest(name, ns.Name)
			jobKey = client.ObjectKeyFromObject(job)
			masterKey = types.NamespacedName{
				Name:      fmt.Sprintf("%s-master-0", name),
				Namespace: ns.Name,
			}
			worker0Key = types.NamespacedName{
				Name:      fmt.Sprintf("%s-worker-0", name),
				Namespace: ns.Name,
			}
			job.Spec.XGBReplicaSpecs = map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
				kubeflowv1.XGBoostJobReplicaTypeMaster: {
					Replicas: pointer.Int32(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.XGBoostJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.XGBoostJobDefaultPortName,
											ContainerPort: expectedPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
								},
							},
						},
					},
				},
				kubeflowv1.XGBoostJobReplicaTypeWorker: {
					Replicas: pointer.Int32(2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "test-image",
									Name:  kubeflowv1.XGBoostJobDefaultContainerName,
									Ports: []corev1.ContainerPort{
										{
											Name:          kubeflowv1.XGBoostJobDefaultPortName,
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
		})
		AfterEach(func() {
			Expect(testK8sClient.Delete(ctx, job)).Should(Succeed())
			Expect(testK8sClient.Delete(ctx, ns)).Should(Succeed())
		})
		It("Shouldn't create resources when XGBoostJob is suspended; Should create resources once XGBoostJob is unsuspended", func() {
			By("By creating a new XGBoostJob with suspend=true")
			job.Spec.RunPolicy.Suspend = pointer.Bool(true)
			job.Spec.RunPolicy.CleanPodPolicy = &cleanPodPolicyAll
			job.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeWorker].Replicas = pointer.Int32(1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.XGBoostJob{}
			masterPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			masterSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			By("Checking created XGBoostJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Checking if the pods and services aren't created")
			Consistently(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the XGBoostJob has suspended condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  "XGBoostJobCreated",
					Message: fmt.Sprintf("xgboostJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Status:  corev1.ConditionTrue,
					Reason:  commonutil.JobSuspendedReason,
					Message: fmt.Sprintf("XGBoostJob %s is suspended.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Unsuspending the XGBoostJob")
			created.Spec.RunPolicy.Suspend = pointer.Bool(false)
			Expect(testK8sClient.Update(ctx, created)).Should(Succeed())

			By("Check if the pods and services are created")
			Eventually(func() error {
				return testK8sClient.Get(ctx, masterKey, masterPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerPod)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, masterKey, masterSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())
			Eventually(func() error {
				return testK8sClient.Get(ctx, worker0Key, workerSvc)
			}, testutil.Timeout, testutil.Interval).Should(BeNil())

			By("Updating Pod's condition with running")
			masterPod.Status.Phase = corev1.PodRunning
			masterPod.ResourceVersion = ""
			Expect(testK8sClient.Status().Update(ctx, masterPod)).Should(Succeed())
			workerPod.Status.Phase = corev1.PodRunning
			workerPod.ResourceVersion = ""
			Expect(testK8sClient.Status().Update(ctx, workerPod)).Should(Succeed())

			By("Checking the XGBoostJob has resumed conditions")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  "XGBoostJobCreated",
					Message: fmt.Sprintf("xgboostJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Status:  corev1.ConditionFalse,
					Reason:  commonutil.JobResumedReason,
					Message: fmt.Sprintf("XGBoostJob %s is resumed.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  xgboostJobRunningReason,
					Message: fmt.Sprintf("XGBoostJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))
		})

		It("Should delete resources after XGBoostJob is suspended", func() {
			By("By creating a new XGBoostJob")
			job.Spec.RunPolicy.CleanPodPolicy = &cleanPodPolicyAll
			job.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeWorker].Replicas = pointer.Int32(1)
			Expect(testK8sClient.Create(ctx, job)).Should(Succeed())

			created := &kubeflowv1.XGBoostJob{}
			masterPod := &corev1.Pod{}
			workerPod := &corev1.Pod{}
			masterSvc := &corev1.Service{}
			workerSvc := &corev1.Service{}

			// We'll need to retry getting this newly created XGBoostJob, given that creation may not immediately happen.
			By("Checking created XGBoostJob")
			Eventually(func() bool {
				err := testK8sClient.Get(ctx, jobKey, created)
				return err == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() *metav1.Time {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.StartTime
			}, testutil.Timeout, testutil.Interval).ShouldNot(BeNil())

			By("Checking the created pods and services")
			Eventually(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errMaster == nil && errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errMaster == nil && errWorker == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())

			By("Updating the pod's phase with Running")
			masterPod.Status.Phase = corev1.PodRunning
			masterPod.ResourceVersion = ""
			Expect(testK8sClient.Status().Update(ctx, masterPod)).Should(Succeed())
			workerPod.Status.Phase = corev1.PodRunning
			workerPod.ResourceVersion = ""
			Expect(testK8sClient.Status().Update(ctx, workerPod)).Should(Succeed())

			By("Checking the XGBoostJob's condition")
			Eventually(func() []kubeflowv1.JobCondition {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.Conditions
			}, testutil.Timeout, testutil.Interval).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  "XGBoostJobCreated",
					Message: fmt.Sprintf("xgboostJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionTrue,
					Reason:  xgboostJobRunningReason,
					Message: fmt.Sprintf("XGBoostJob %s is running.", name),
				},
			}, testutil.IgnoreJobConditionsTimes))

			By("Updating the XGBoostJob with suspend=true")
			created.Spec.RunPolicy.Suspend = pointer.Bool(true)
			Expect(testK8sClient.Update(ctx, created)).Should(Succeed())

			By("Checking if the pods and services are removed")
			Eventually(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterPod)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerPod)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Eventually(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				errMaster := testK8sClient.Get(ctx, masterKey, masterSvc)
				errWorker := testK8sClient.Get(ctx, worker0Key, workerSvc)
				return errors.IsNotFound(errMaster) && errors.IsNotFound(errWorker)
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())

			By("Checking if the XGBoostJob has a suspended condition")
			Eventually(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.XGBoostJobReplicaTypeMaster].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.XGBoostJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime == nil
			}, testutil.Timeout, testutil.Interval).Should(BeTrue())
			Consistently(func() bool {
				Expect(testK8sClient.Get(ctx, jobKey, created)).Should(Succeed())
				return created.Status.ReplicaStatuses[kubeflowv1.XGBoostJobReplicaTypeMaster].Active == 0 &&
					created.Status.ReplicaStatuses[kubeflowv1.XGBoostJobReplicaTypeWorker].Active == 0 &&
					created.Status.StartTime == nil
			}, testutil.ConsistentDuration, testutil.Interval).Should(BeTrue())
			Expect(created.Status.Conditions).Should(BeComparableTo([]kubeflowv1.JobCondition{
				{
					Type:    kubeflowv1.JobCreated,
					Status:  corev1.ConditionTrue,
					Reason:  "XGBoostJobCreated",
					Message: fmt.Sprintf("xgboostJob %s is created.", name),
				},
				{
					Type:    kubeflowv1.JobRunning,
					Status:  corev1.ConditionFalse,
					Reason:  commonutil.JobSuspendedReason,
					Message: fmt.Sprintf("XGBoostJob %s is suspended.", name),
				},
				{
					Type:    kubeflowv1.JobSuspended,
					Reason:  commonutil.JobSuspendedReason,
					Message: fmt.Sprintf("XGBoostJob %s is suspended.", name),
					Status:  corev1.ConditionTrue,
				},
			}, testutil.IgnoreJobConditionsTimes))
		})
	})
})

func newXGBoostForTest(name, namespace string) *kubeflowv1.XGBoostJob {
	return &kubeflowv1.XGBoostJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
