// Copyright YEAR The Kubeflow Authors
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

package mxnet

import (
	"context"
	"fmt"
	"time"

	"reflect"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/common/pkg/util"
	mxjobv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	"github.com/kubeflow/tf-operator/pkg/common/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controllerName = "mxnet-operator"

	// mxJobCreatedReason is added in a mxjob when it is created.
	mxJobCreatedReason = "MXJobCreated"
	// mxJobSucceededReason is added in a mxjob when it is succeeded.
	mxJobSucceededReason = "MXJobSucceeded"
	// mxJobRunningReason is added in a mxjob when it is running.
	mxJobRunningReason = "MXJobRunning"
	// mxJobFailedReason is added in a mxjob when it is failed.
	mxJobFailedReason = "MXJobFailed"
	// mxJobRestarting is added in a mxjob when it is restarting.
	mxJobRestartingReason = "MXJobRestarting"
)

var (
	jobOwnerKey = ".metadata.controller"
	// DefaultMXControllerConfiguration is the suggested mxnet-operator configuration for production.
	DefaultMXControllerConfiguration = common.JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: metav1.Duration{Duration: 15 * time.Second},
		EnableGangScheduling:     false,
	}
	// DefaultCleanPodPolicy is the default clean pod policy controller assign the new Job if not exist
	DefaultCleanPodPolicy = commonv1.CleanPodPolicyNone
)

// NewReconciler creates a PyTorchJob Reconciler
func NewReconciler(mgr manager.Manager) *MXJobReconciler {
	r := &MXJobReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor(controllerName),
		Log:      log.Log,
	}

	// Create clients.
	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)

	// Initialize common job controller
	r.JobController = common.JobController{
		Controller:       r,
		Expectations:     expectation.NewControllerExpectations(),
		Config:           common.JobControllerConfiguration{EnableGangScheduling: false},
		WorkQueue:        &util.FakeWorkQueue{},
		Recorder:         r.Recorder,
		KubeClientSet:    kubeClientSet,
		VolcanoClientSet: volcanoClientSet,
		PodControl:       control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.Recorder},
		ServiceControl:   control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.Recorder},
	}

	return r
}

// MXJobReconciler reconciles a MXJob object
type MXJobReconciler struct {
	common.JobController
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

//+kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete
func (r *MXJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues(mxjobv1.Singular, req.NamespacedName)

	mxjob := &mxjobv1.MXJob{}
	err := r.Get(ctx, req.NamespacedName, mxjob)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch PyTorchJob", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//mxjobv1.SetDefaults_MXJob(mxjob)

	// Check if reconciliation is needed
	jobKey, err := common.KeyFunc(mxjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get jobKey for job object %#v: %v", mxjob, err))
	}

	replicaTypes := util.GetReplicaTypes(mxjob.Spec.MXReplicaSpecs)
	needReconcile := util.SatisfiedExpectations(r.Expectations, jobKey, replicaTypes)

	if !needReconcile || mxjob.GetDeletionTimestamp() != nil {
		logger.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needReconcile, "deleted", mxjob.GetDeletionTimestamp() != nil)
		return ctrl.Result{}, nil
	}

	// Set default priorities to mxnet job
	scheme.Scheme.Default(mxjob)

	// Convert PyTorch.Spec.PyTorchReplicasSpecs to  map[commonv1.ReplicaType]*commonv1.ReplicaSpec
	replicas := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{}
	for k, v := range mxjob.Spec.MXReplicaSpecs {
		replicas[commonv1.ReplicaType(k)] = v
	}

	// Construct RunPolicy based on PyTorchJob.Spec
	runPolicy := &commonv1.RunPolicy{
		CleanPodPolicy:          mxjob.Spec.RunPolicy.CleanPodPolicy,
		TTLSecondsAfterFinished: mxjob.Spec.RunPolicy.TTLSecondsAfterFinished,
		ActiveDeadlineSeconds:   mxjob.Spec.RunPolicy.ActiveDeadlineSeconds,
		BackoffLimit:            mxjob.Spec.RunPolicy.BackoffLimit,
		SchedulingPolicy:        nil,
	}

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(mxjob, replicas, mxjob.Status, runPolicy)
	if err != nil {
		logrus.Warnf("Reconcile PyTorch Job error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MXJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// setup FieldIndexer to inform the manager that this controller owns pods and services,
	// so that it will automatically call Reconcile on the underlying MXJob when a Pod or Service changes, is deleted, etc.
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, jobOwnerKey, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}

		// Make sure owner is XGBoostJob Controller.
		if owner.APIVersion != r.GetAPIGroupVersion().Version || owner.Kind != r.GetAPIGroupVersionKind().Kind {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, jobOwnerKey, func(rawObj client.Object) []string {
		svc := rawObj.(*corev1.Service)
		owner := metav1.GetControllerOf(svc)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != r.GetAPIGroupVersion().Version || owner.Kind != r.GetAPIGroupVersionKind().Kind {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&mxjobv1.MXJob{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// Below is ControllerInterface's implementation
func (r *MXJobReconciler) ControllerName() string {
	return controllerName
}

func (r *MXJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return mxjobv1.GroupVersion.WithKind(mxjobv1.Kind)
}

func (r *MXJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return mxjobv1.GroupVersion
}

func (r *MXJobReconciler) GetGroupNameLabelValue() string {
	return mxjobv1.GroupVersion.Group
}

func (r *MXJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &mxjobv1.MXJob{}
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Error(err, "mxnet job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *MXJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &mxjobv1.MXJob{}

	clientReader, err := util.GetDelegatingClientFromClient(r.Client)
	if err != nil {
		return nil, err
	}
	err = clientReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Error(err, "xgboost job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *MXJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, fmt.Errorf("%v is not a type of MXJob", job)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist, client.MatchingLabels(r.GenLabels(job.GetName())))
	if err != nil {
		return nil, err
	}
	return util.ConvertPodList(podlist.Items), nil
}

func (r *MXJobReconciler) GetServicesForJob(job interface{}) ([]*corev1.Service, error) {
	mxJob, err := meta.Accessor(job)
	if err != nil {
		return nil, fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	// List all services to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList, client.MatchingLabels(r.GenLabels(mxJob.GetName())))
	if err != nil {
		return nil, err
	}

	ret := util.ConvertServiceList(serviceList.Items)
	return ret, nil
}

func (r *MXJobReconciler) DeleteJob(job interface{}) error {
	mxjob, ok := job.(*mxjobv1.MXJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", job)
	}
	if err := r.Delete(context.Background(), mxjob); err != nil {
		r.Recorder.Eventf(mxjob, corev1.EventTypeWarning, control.FailedDeletePodReason, "Error deleting: %v", err)
		logrus.Error(err, "failed to delete job", "namespace", mxjob.Namespace, "name", mxjob.Name)
		return err
	}
	r.Recorder.Eventf(mxjob, corev1.EventTypeNormal, control.SuccessfulDeletePodReason, "Deleted job: %v", mxjob.Name)
	logrus.Info("job deleted", "namespace", mxjob.Namespace, "name", mxjob.Name)
	return nil
}

func (r *MXJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	mxjob, ok := job.(*mxjobv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxjob)
	}

	mxjobKey, err := common.KeyFunc(mxjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for mxjob object %#v: %v", mxjob, err))
		return err
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]

		// Expect to have `replicas - succeeded` pods alive.
		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		r.Log.Info(fmt.Sprintf("MXJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			mxjob.Name, rtype, expected, running, succeeded, failed))

		if mxjob.Status.StartTime == nil {
			now := metav1.Now()
			mxjob.Status.StartTime = &now
			// enqueue a sync to check if job past ActiveDeadlineSeconds
			if mxjob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
				logrus.Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *mxjob.Spec.RunPolicy.ActiveDeadlineSeconds)
				r.WorkQueue.AddAfter(mxjobKey, time.Duration(*mxjob.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
			}
		}

		if running > 0 {
			msg := fmt.Sprintf("MXJob %s is running.", mxjob.Name)
			err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, mxJobRunningReason, msg)
			if err != nil {
				logrus.Infof("Append mxjob condition error: %v", err)
				return err
			}
		}
		if expected == 0 {
			msg := fmt.Sprintf("MXJob %s is successfully completed.", mxjob.Name)
			r.Recorder.Event(mxjob, corev1.EventTypeNormal, mxJobSucceededReason, msg)
			if mxjob.Status.CompletionTime == nil {
				now := metav1.Now()
				mxjob.Status.CompletionTime = &now
			}
			err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, mxJobSucceededReason, msg)
			if err != nil {
				logrus.Infof("Append mxjob condition error: %v", err)
				return err
			}
		}

		if failed > 0 {
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				msg := fmt.Sprintf("mxjob %s is restarting because %d %s replica(s) failed.", mxjob.Name, failed, rtype)
				r.Recorder.Event(mxjob, corev1.EventTypeWarning, mxJobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, mxJobRestartingReason, msg)
				if err != nil {
					logrus.Infof("Append job condition error: %v", err)
					return err
				}
			} else {
				msg := fmt.Sprintf("mxjob %s is failed because %d %s replica(s) failed.", mxjob.Name, failed, rtype)
				r.Recorder.Event(mxjob, corev1.EventTypeNormal, mxJobFailedReason, msg)
				if mxjob.Status.CompletionTime == nil {
					now := metav1.Now()
					mxjob.Status.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, mxJobFailedReason, msg)
				if err != nil {
					logrus.Infof("Append job condition error: %v", err)
					return err
				}
			}
		}
	}

	return nil
}

// UpdateJobStatusInApiServer updates the status of the given MXJob.
func (r *MXJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	mxJob, ok := job.(*mxjobv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	if !reflect.DeepEqual(&mxJob.Status, jobStatus) {
		mxJob = mxJob.DeepCopy()
		mxJob.Status = *jobStatus.DeepCopy()
	}

	if err := r.Update(context.Background(), mxJob); err != nil {
		logrus.Error(err, " failed to update MxJob conditions in the API server")
		return err
	}

	return nil
}

func (r *MXJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, rtype, index)
}

func (r *MXJobReconciler) GetDefaultContainerName() string {
	return mxjobv1.DefaultContainerName
}

func (r *MXJobReconciler) GetDefaultContainerPortName() string {
	return mxjobv1.DefaultPortName
}

func (r *MXJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(mxjobv1.MXReplicaTypeServer)
}

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		mxjob, ok := e.Object.(*mxjobv1.MXJob)
		if !ok {
			return true
		}

		// TODO: check default setting
		scheme.Scheme.Default(mxjob)
		msg := fmt.Sprintf("xgboostJob %s is created.", e.Object.GetName())
		logrus.Info(msg)

		// TODO: should we move defaulter somewhere else, like pass a default func here to call
		//specific the run policy
		if mxjob.Spec.RunPolicy.CleanPodPolicy == nil {
			mxjob.Spec.RunPolicy.CleanPodPolicy = new(commonv1.CleanPodPolicy)
			mxjob.Spec.RunPolicy.CleanPodPolicy = &DefaultCleanPodPolicy
		}

		if err := commonutil.UpdateJobConditions(&mxjob.Status, commonv1.JobCreated, "MXJobCreated", msg); err != nil {
			logrus.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
