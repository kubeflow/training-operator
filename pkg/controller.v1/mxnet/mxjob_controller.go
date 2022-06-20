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

package mxnet

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/common/pkg/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	trainingoperatorcommon "github.com/kubeflow/training-operator/pkg/common"
	"github.com/kubeflow/training-operator/pkg/common/util"
)

const (
	controllerName = "mxjob-controller"

	// mxJobSucceededReason is added in a mxjob when it is succeeded.
	mxJobSucceededReason = "MXJobSucceeded"
	// mxJobRunningReason is added in a mxjob when it is running.
	mxJobRunningReason = "MXJobRunning"
	// mxJobFailedReason is added in a mxjob when it is failed.
	mxJobFailedReason = "MXJobFailed"
	// mxJobRestarting is added in a mxjob when it is restarting.
	mxJobRestartingReason = "MXJobRestarting"
)

// NewReconciler creates a MXJob Reconciler
func NewReconciler(mgr manager.Manager, enableGangScheduling bool) *MXJobReconciler {
	r := &MXJobReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Recorder:  mgr.GetEventRecorderFor(controllerName),
		apiReader: mgr.GetAPIReader(),
		Log:       log.Log,
	}

	// Create clients.
	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)
	sharedInformers := informers.NewSharedInformerFactory(kubeClientSet, 0)
	priorityClassInformer := sharedInformers.Scheduling().V1beta1().PriorityClasses()

	// Initialize common job controller
	r.JobController = common.JobController{
		Controller:                  r,
		Expectations:                expectation.NewControllerExpectations(),
		Config:                      common.JobControllerConfiguration{EnableGangScheduling: enableGangScheduling},
		WorkQueue:                   &util.FakeWorkQueue{},
		Recorder:                    r.Recorder,
		KubeClientSet:               kubeClientSet,
		VolcanoClientSet:            volcanoClientSet,
		PriorityClassLister:         priorityClassInformer.Lister(),
		PriorityClassInformerSynced: priorityClassInformer.Informer().HasSynced,
		PodControl:                  control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.Recorder},
		ServiceControl:              control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.Recorder},
	}

	return r
}

// MXJobReconciler reconciles a MXJob object
type MXJobReconciler struct {
	common.JobController
	client.Client
	Log       logr.Logger
	Recorder  record.EventRecorder
	apiReader client.Reader
	Scheme    *runtime.Scheme
}

//+kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=mxjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete
func (r *MXJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues(kubeflowv1.MXJobSingular, req.NamespacedName)

	mxjob := &kubeflowv1.MXJob{}
	err := r.Get(ctx, req.NamespacedName, mxjob)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch MXJob", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = kubeflowv1.ValidateV1MXJobSpec(&mxjob.Spec); err != nil {
		logger.Info(err.Error(), "MXJob failed validation", req.NamespacedName.String())
	}

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
	r.Scheme.Default(mxjob)

	// Convert MX.Spec.MXReplicasSpecs to  map[commonv1.ReplicaType]*commonv1.ReplicaSpec
	replicas := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{}
	for k, v := range mxjob.Spec.MXReplicaSpecs {
		replicas[commonv1.ReplicaType(k)] = v
	}

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(mxjob, replicas, mxjob.Status, &mxjob.Spec.RunPolicy)
	if err != nil {
		logrus.Warnf("Reconcile MX Job error %v", err)
		return ctrl.Result{}, err
	}

	t, err := util.DurationUntilExpireTime(&mxjob.Spec.RunPolicy, mxjob.Status)
	if err != nil {
		logrus.Warnf("Reconcile MX Job error %v", err)
		return ctrl.Result{}, err
	}
	if t >= 0 {
		return ctrl.Result{Requeue: true, RequeueAfter: t}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MXJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler: r,
	})

	if err != nil {
		return err
	}

	// using onOwnerCreateFunc is easier to set defaults
	if err = c.Watch(&source.Kind{Type: &kubeflowv1.MXJob{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: r.onOwnerCreateFunc()},
	); err != nil {
		return err
	}

	// inject watching for job related pod
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubeflowv1.MXJob{},
	}, predicate.Funcs{
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}); err != nil {
		return err
	}

	// inject watching for job related service
	if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubeflowv1.MXJob{},
	}, predicate.Funcs{
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}); err != nil {
		return err
	}

	return nil
}

// Below is ControllerInterface's implementation
func (r *MXJobReconciler) ControllerName() string {
	return controllerName
}

func (r *MXJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return kubeflowv1.GroupVersion.WithKind(kubeflowv1.MXJobKind)
}

func (r *MXJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return kubeflowv1.GroupVersion
}

func (r *MXJobReconciler) GetGroupNameLabelValue() string {
	return kubeflowv1.GroupVersion.Group
}

func (r *MXJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &kubeflowv1.MXJob{}
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
	job := &kubeflowv1.MXJob{}

	err := r.apiReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
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
	err = r.List(context.Background(), podlist, client.MatchingLabels(r.GenLabels(job.GetName())), client.InNamespace(job.GetNamespace()))
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
	err = r.List(context.Background(), serviceList, client.MatchingLabels(r.GenLabels(mxJob.GetName())), client.InNamespace(mxJob.GetNamespace()))
	if err != nil {
		return nil, err
	}

	ret := util.ConvertServiceList(serviceList.Items)
	return ret, nil
}

func (r *MXJobReconciler) DeleteJob(job interface{}) error {
	mxjob, ok := job.(*kubeflowv1.MXJob)
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
	trainingoperatorcommon.DeletedJobsCounterInc(mxjob.Namespace, kubeflowv1.MXJobFrameworkName)
	return nil
}

func (r *MXJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	mxjob, ok := job.(*kubeflowv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxjob)
	}

	mxjobKey, err := common.KeyFunc(mxjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for mxjob object %#v: %v", mxjob, err))
		return err
	}

	if mxjob.Status.StartTime == nil {
		now := metav1.Now()
		mxjob.Status.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		if mxjob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
			logrus.Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *mxjob.Spec.RunPolicy.ActiveDeadlineSeconds)
			r.WorkQueue.AddAfter(mxjobKey, time.Duration(*mxjob.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
		}
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

		if rtype == commonv1.ReplicaType(kubeflowv1.MXJobReplicaTypeScheduler) {
			if running > 0 {
				msg := fmt.Sprintf("MXJob %s is running.", mxjob.Name)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, mxJobRunningReason, msg)
				if err != nil {
					logrus.Infof("Append mxjob condition error: %v", err)
					return err
				}
			}
			// when scheduler is succeeded, the job is finished.
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
				trainingoperatorcommon.SuccessfulJobsCounterInc(mxjob.Namespace, kubeflowv1.MXJobFrameworkName)
				return nil
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
				trainingoperatorcommon.RestartedJobsCounterInc(mxjob.Namespace, kubeflowv1.MXJobFrameworkName)
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
				trainingoperatorcommon.FailedJobsCounterInc(mxjob.Namespace, kubeflowv1.MXJobFrameworkName)
			}
		}
	}

	return nil
}

// UpdateJobStatusInApiServer updates the status of the given MXJob.
func (r *MXJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = map[commonv1.ReplicaType]*commonv1.ReplicaStatus{}
	}

	mxJob, ok := job.(*kubeflowv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	if !reflect.DeepEqual(&mxJob.Status, jobStatus) {
		mxJob = mxJob.DeepCopy()
		mxJob.Status = *jobStatus.DeepCopy()
	}

	if err := r.Status().Update(context.Background(), mxJob); err != nil {
		logrus.Error(err, " failed to update MxJob conditions in the API server")
		return err
	}

	return nil
}

func (r *MXJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, rtype, index)
}

func (r *MXJobReconciler) GetDefaultContainerName() string {
	return kubeflowv1.MXJobDefaultContainerName
}

func (r *MXJobReconciler) GetDefaultContainerPortName() string {
	return kubeflowv1.MXJobDefaultPortName
}

func (r *MXJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(kubeflowv1.MXJobReplicaTypeServer)
}

// onOwnerCreateFunc modify creation condition.
func (r *MXJobReconciler) onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		mxJob, ok := e.Object.(*kubeflowv1.MXJob)
		if !ok {
			return true
		}

		// Use defaulters registered in scheme.
		r.Scheme.Default(mxJob)
		msg := fmt.Sprintf("MXJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		trainingoperatorcommon.CreatedJobsCounterInc(mxJob.Namespace, kubeflowv1.MXJobFrameworkName)
		if err := commonutil.UpdateJobConditions(&mxJob.Status, commonv1.JobCreated, "MXJobCreated", msg); err != nil {
			logrus.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
