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

package xgboost

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
	logger "github.com/kubeflow/common/pkg/util"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	trainingoperatorcommon "github.com/kubeflow/training-operator/pkg/common"
	"github.com/kubeflow/training-operator/pkg/common/util"
)

const (
	controllerName = "xgboostjob-controller"

	// Reasons for job events.
	FailedDeleteJobReason     = "FailedDeleteJob"
	SuccessfulDeleteJobReason = "SuccessfulDeleteJob"
	// xgboostJobCreatedReason is added in a job when it is created.
	xgboostJobCreatedReason = "XGBoostJobCreated"
	// xgboostJobSucceededReason is added in a job when it is succeeded.
	xgboostJobSucceededReason = "XGBoostJobSucceeded"
	// xgboostJobRunningReason is added in a job when it is running.
	xgboostJobRunningReason = "XGBoostJobRunning"
	// xgboostJobFailedReason is added in a job when it is failed.
	xgboostJobFailedReason = "XGBoostJobFailed"
	// xgboostJobRestartingReason is added in a job when it is restarting.
	xgboostJobRestartingReason = "XGBoostJobRestarting"
)

// NewReconciler creates a XGBoostJob Reconciler
func NewReconciler(mgr manager.Manager, scheduling bool) *XGBoostJobReconciler {
	r := &XGBoostJobReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor(controllerName),
		apiReader: mgr.GetAPIReader(),
		Log:       ctrl.Log.WithName("controllers").WithName(kubeflowv1.XGBoostJobKind),
	}

	// Create clients
	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)
	sharedInformers := informers.NewSharedInformerFactory(kubeClientSet, 0)
	priorityClassInformer := sharedInformers.Scheduling().V1beta1().PriorityClasses()

	// Initialize common job controller
	r.JobController = common.JobController{
		Controller:                  r,
		Expectations:                expectation.NewControllerExpectations(),
		Config:                      common.JobControllerConfiguration{EnableGangScheduling: false},
		WorkQueue:                   &util.FakeWorkQueue{},
		Recorder:                    r.recorder,
		KubeClientSet:               kubeClientSet,
		VolcanoClientSet:            volcanoClientSet,
		PriorityClassLister:         priorityClassInformer.Lister(),
		PriorityClassInformerSynced: priorityClassInformer.Informer().HasSynced,
		PodControl:                  control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		ServiceControl:              control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.recorder},
	}

	return r
}

// XGBoostJobReconciler reconciles a XGBoostJob object
type XGBoostJobReconciler struct {
	common.JobController
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	recorder  record.EventRecorder
	apiReader client.Reader
}

//+kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete

// Reconcile reads that state of the cluster for a XGBoostJob object and makes changes based on the state read
// and what is in the XGBoostJob.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
func (r *XGBoostJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues(kubeflowv1.XGBoostJobSingular, req.NamespacedName)

	xgboostjob := &kubeflowv1.XGBoostJob{}
	err := r.Get(ctx, req.NamespacedName, xgboostjob)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch XGBoostJob", req.NamespacedName.String())
		// Object not found, return.  Created objects are automatically garbage collected.
		// For additional cleanup logic use finalizers.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = kubeflowv1.ValidateXGBoostJobSpec(&xgboostjob.Spec); err != nil {
		logger.Info(err.Error(), "XGBoostJob failed validation", req.NamespacedName.String())
	}

	// Check reconcile is required.
	jobKey, err := common.KeyFunc(xgboostjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get jobKey for job object %#v: %v", xgboostjob, err))
	}

	replicaTypes := util.GetReplicaTypes(xgboostjob.Spec.XGBReplicaSpecs)
	needSync := util.SatisfiedExpectations(r.Expectations, jobKey, replicaTypes)

	if !needSync || xgboostjob.GetDeletionTimestamp() != nil {
		logger.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", xgboostjob.GetDeletionTimestamp() != nil)
		return reconcile.Result{}, nil
	}

	// Set default priorities for xgboost job
	r.Scheme.Default(xgboostjob)

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(xgboostjob, xgboostjob.Spec.XGBReplicaSpecs, xgboostjob.Status, &xgboostjob.Spec.RunPolicy)
	if err != nil {
		logger.V(1).Error(err, "Reconcile XGBoost Job error")
		return ctrl.Result{}, err
	}

	t, err := util.DurationUntilExpireTime(&xgboostjob.Spec.RunPolicy, xgboostjob.Status)
	if err != nil {
		logrus.Warnf("Reconcile XGBoost Job error %v", err)
		return ctrl.Result{}, err
	}
	if t >= 0 {
		return ctrl.Result{Requeue: true, RequeueAfter: t}, nil
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *XGBoostJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler: r,
	})

	if err != nil {
		return err
	}

	// using onOwnerCreateFunc is easier to set defaults
	if err = c.Watch(&source.Kind{Type: &kubeflowv1.XGBoostJob{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: r.onOwnerCreateFunc()},
	); err != nil {
		return err
	}

	// inject watching for job related pod
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubeflowv1.XGBoostJob{},
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
		OwnerType:    &kubeflowv1.XGBoostJob{},
	}, predicate.Funcs{
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}); err != nil {
		return err
	}
	// skip watching podgroup if podgroup is not installed
	_, err = mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: v1beta1.SchemeGroupVersion.Group, Kind: "PodGroup"},
		v1beta1.SchemeGroupVersion.Version)
	if err == nil {
		// inject watching for job related podgroup
		if err = c.Watch(&source.Kind{Type: &v1beta1.PodGroup{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &kubeflowv1.XGBoostJob{},
		}, predicate.Funcs{
			CreateFunc: util.OnDependentCreateFunc(r.Expectations),
			UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
			DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *XGBoostJobReconciler) ControllerName() string {
	return controllerName
}

func (r *XGBoostJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return kubeflowv1.GroupVersion.WithKind(kubeflowv1.XGBoostJobKind)
}

func (r *XGBoostJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return kubeflowv1.GroupVersion
}

func (r *XGBoostJobReconciler) GetGroupNameLabelValue() string {
	return kubeflowv1.GroupVersion.Group
}

// GetJobFromInformerCache returns the Job from Informer Cache
func (r *XGBoostJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &kubeflowv1.XGBoostJob{}
	// Default reader for XGBoostJob is cache reader.
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "xgboost job not found", "namespace", namespace, "name", name)
		} else {
			r.Log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// GetJobFromAPIClient returns the Job from API server
func (r *XGBoostJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &kubeflowv1.XGBoostJob{}

	err := r.apiReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "xgboost job not found", "namespace", namespace, "name", name)
		} else {
			r.Log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *XGBoostJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
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

// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
// i.e. all services created by the job will come with label "job-name" = <this_job_name>
func (r *XGBoostJobReconciler) GetServicesForJob(obj interface{}) ([]*corev1.Service, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, fmt.Errorf("%+v is not a type of XGBoostJob", job)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList, client.MatchingLabels(r.GenLabels(job.GetName())), client.InNamespace(job.GetNamespace()))
	if err != nil {
		return nil, err
	}

	ret := util.ConvertServiceList(serviceList.Items)
	return ret, nil
}

// DeleteJob deletes the job
func (r *XGBoostJobReconciler) DeleteJob(job interface{}) error {
	xgboostjob, ok := job.(*kubeflowv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}
	if err := r.Delete(context.Background(), xgboostjob); err != nil {
		r.recorder.Eventf(xgboostjob, corev1.EventTypeWarning, FailedDeleteJobReason, "Error deleting: %v", err)
		r.Log.Error(err, "failed to delete job", "namespace", xgboostjob.Namespace, "name", xgboostjob.Name)
		return err
	}
	r.recorder.Eventf(xgboostjob, corev1.EventTypeNormal, SuccessfulDeleteJobReason, "Deleted job: %v", xgboostjob.Name)
	r.Log.Info("job deleted", "namespace", xgboostjob.Namespace, "name", xgboostjob.Name)
	trainingoperatorcommon.DeletedJobsCounterInc(xgboostjob.Namespace, kubeflowv1.XGBoostJobFrameworkName)
	return nil
}

// UpdateJobStatus updates the job status and job conditions
func (r *XGBoostJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	xgboostJob, ok := job.(*kubeflowv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of xgboostJob", xgboostJob)
	}

	xgboostJobKey, err := common.KeyFunc(xgboostJob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for xgboostjob object %#v: %v", xgboostJob, err))
		return err
	}

	// Set StartTime.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		if xgboostJob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
			logger.LoggerForJob(xgboostJob).Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *xgboostJob.Spec.RunPolicy.ActiveDeadlineSeconds)
			r.WorkQueue.AddAfter(xgboostJobKey, time.Duration(*xgboostJob.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
		}
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logrus.Infof("XGBoostJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			xgboostJob.Name, rtype, expected, running, succeeded, failed)

		if rtype == commonv1.ReplicaType(kubeflowv1.XGBoostJobReplicaTypeMaster) {
			if running > 0 {
				msg := fmt.Sprintf("XGBoostJob %s is running.", xgboostJob.Name)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, xgboostJobRunningReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
			}
			// when master is succeed, the job is finished.
			if expected == 0 {
				msg := fmt.Sprintf("XGBoostJob %s is successfully completed.", xgboostJob.Name)
				logrus.Info(msg)
				r.Recorder.Event(xgboostJob, corev1.EventTypeNormal, xgboostJobSucceededReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, xgboostJobSucceededReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
				trainingoperatorcommon.SuccessfulJobsCounterInc(xgboostJob.Namespace, kubeflowv1.XGBoostJobFrameworkName)
				return nil
			}
		}
		if failed > 0 {
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				msg := fmt.Sprintf("XGBoostJob %s is restarting because %d %s replica(s) failed.", xgboostJob.Name, failed, rtype)
				r.Recorder.Event(xgboostJob, corev1.EventTypeWarning, xgboostJobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, xgboostJobRestartingReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
				trainingoperatorcommon.RestartedJobsCounterInc(xgboostJob.Namespace, kubeflowv1.XGBoostJobFrameworkName)
			} else {
				msg := fmt.Sprintf("XGBoostJob %s is failed because %d %s replica(s) failed.", xgboostJob.Name, failed, rtype)
				r.Recorder.Event(xgboostJob, corev1.EventTypeNormal, xgboostJobFailedReason, msg)
				if xgboostJob.Status.CompletionTime == nil {
					now := metav1.Now()
					xgboostJob.Status.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, xgboostJobFailedReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
				trainingoperatorcommon.FailedJobsCounterInc(xgboostJob.Namespace, kubeflowv1.XGBoostJobFrameworkName)
			}
		}
	}

	// Some workers are still running, leave a running condition.
	msg := fmt.Sprintf("XGBoostJob %s is running.", xgboostJob.Name)
	logger.LoggerForJob(xgboostJob).Infof(msg)

	if err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, xgboostJobRunningReason, msg); err != nil {
		logger.LoggerForJob(xgboostJob).Error(err, "failed to update XGBoost Job conditions")
		return err
	}

	return nil
}

// UpdateJobStatusInApiServer updates the job status in to cluster.
func (r *XGBoostJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = map[commonv1.ReplicaType]*commonv1.ReplicaStatus{}
	}

	xgboostjob, ok := job.(*kubeflowv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !reflect.DeepEqual(&xgboostjob.Status, jobStatus) {
		xgboostjob = xgboostjob.DeepCopy()
		xgboostjob.Status = *jobStatus.DeepCopy()
	}

	result := r.Status().Update(context.Background(), xgboostjob)

	if result != nil {
		logger.LoggerForJob(xgboostjob).Error(result, "failed to update XGBoost Job conditions in the API server")
		return result
	}

	return nil
}

// SetClusterSpec sets the cluster spec for the pod
func (r *XGBoostJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, rtype, index)
}

func (r *XGBoostJobReconciler) GetDefaultContainerName() string {
	return kubeflowv1.XGBoostJobDefaultContainerName
}

func (r *XGBoostJobReconciler) GetDefaultContainerPortName() string {
	return kubeflowv1.XGBoostJobDefaultPortName
}

func (r *XGBoostJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(kubeflowv1.XGBoostJobReplicaTypeMaster)
}

// onOwnerCreateFunc modify creation condition.
func (r *XGBoostJobReconciler) onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		xgboostJob, ok := e.Object.(*kubeflowv1.XGBoostJob)
		if !ok {
			return true
		}
		r.Scheme.Default(xgboostJob)
		msg := fmt.Sprintf("xgboostJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		trainingoperatorcommon.CreatedJobsCounterInc(xgboostJob.Namespace, kubeflowv1.XGBoostJobFrameworkName)
		if err := commonutil.UpdateJobConditions(&xgboostJob.Status, commonv1.JobCreated, xgboostJobCreatedReason, msg); err != nil {
			log.Log.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
