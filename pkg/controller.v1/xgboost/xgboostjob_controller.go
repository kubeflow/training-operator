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

	"github.com/kubeflow/tf-operator/pkg/apis/xgboost/validation"

	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"reflect"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/common/pkg/util"
	logger "github.com/kubeflow/common/pkg/util"
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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xgboostv1 "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
)

const (
	controllerName = "xgboostjob-operator"

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

var (
	jobOwnerKey           = ".metadata.controller"
	defaultTTLSeconds     = int32(100)
	defaultCleanPodPolicy = commonv1.CleanPodPolicyNone
)

func NewReconciler(mgr manager.Manager) *XGBoostJobReconciler {
	r := &XGBoostJobReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor(controllerName),
		Log:      ctrl.Log.WithName("controllers").WithName(xgboostv1.Kind),
	}

	// Create clients
	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)

	// Initialize common job controller
	r.JobController = common.JobController{
		Controller:       r,
		Expectations:     expectation.NewControllerExpectations(),
		Config:           common.JobControllerConfiguration{EnableGangScheduling: false},
		WorkQueue:        &util.FakeWorkQueue{},
		Recorder:         r.recorder,
		KubeClientSet:    kubeClientSet,
		VolcanoClientSet: volcanoClientSet,
		PodControl:       control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		ServiceControl:   control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.recorder},
	}

	return r
}

// XGBoostJobReconciler reconciles a XGBoostJob object
type XGBoostJobReconciler struct {
	common.JobController
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
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
	logger := r.Log.WithValues(xgboostv1.Singular, req.NamespacedName)

	xgboostjob := &xgboostv1.XGBoostJob{}
	err := r.Get(ctx, req.NamespacedName, xgboostjob)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch XGBoostJob", req.NamespacedName.String())
		// Object not found, return.  Created objects are automatically garbage collected.
		// For additional cleanup logic use finalizers.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = validation.ValidateV1XGBoostJobSpec(&xgboostjob.Spec); err != nil {
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
	scheme.Scheme.Default(xgboostjob)

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(xgboostjob, xgboostjob.Spec.XGBReplicaSpecs, xgboostjob.Status, &xgboostjob.Spec.RunPolicy)
	if err != nil {
		logger.V(2).Error(err, "Reconcile XGBoost Job error")
		return ctrl.Result{}, err
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
	if err = c.Watch(&source.Kind{Type: &xgboostv1.XGBoostJob{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: onOwnerCreateFunc()},
	); err != nil {
		return err
	}

	// inject watching for job related pod
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &xgboostv1.XGBoostJob{},
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
		OwnerType:    &xgboostv1.XGBoostJob{},
	}, predicate.Funcs{
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}); err != nil {
		return err
	}

	return nil
}

func (r *XGBoostJobReconciler) ControllerName() string {
	return controllerName
}

func (r *XGBoostJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return xgboostv1.GroupVersion.WithKind(xgboostv1.Kind)
}

func (r *XGBoostJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return xgboostv1.GroupVersion
}

func (r *XGBoostJobReconciler) GetGroupNameLabelValue() string {
	return xgboostv1.GroupVersion.Group
}

// GetJobFromInformerCache returns the Job from Informer Cache
func (r *XGBoostJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &xgboostv1.XGBoostJob{}
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
	job := &xgboostv1.XGBoostJob{}

	clientReader, err := util.GetDelegatingClientFromClient(r.Client)
	err = clientReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
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
	err = r.List(context.Background(), podlist, client.MatchingLabels(r.GenLabels(job.GetName())))
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
	err = r.List(context.Background(), serviceList, client.MatchingLabels(r.GenLabels(job.GetName())))
	if err != nil {
		return nil, err
	}

	ret := util.ConvertServiceList(serviceList.Items)
	return ret, nil
}

// DeleteJob deletes the job
func (r *XGBoostJobReconciler) DeleteJob(job interface{}) error {
	xgboostjob, ok := job.(*xgboostv1.XGBoostJob)
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
	return nil
}

// UpdateJobStatus updates the job status and job conditions
func (r *XGBoostJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	xgboostJob, ok := job.(*xgboostv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of xgboostJob", xgboostJob)
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logrus.Infof("XGBoostJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			xgboostJob.Name, rtype, expected, running, succeeded, failed)

		if rtype == commonv1.ReplicaType(xgboostv1.XGBoostReplicaTypeMaster) {
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
	xgboostjob, ok := job.(*xgboostv1.XGBoostJob)
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
	return xgboostv1.DefaultContainerName
}

func (r *XGBoostJobReconciler) GetDefaultContainerPortName() string {
	return xgboostv1.DefaultPortName
}

func (r *XGBoostJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(xgboostv1.XGBoostReplicaTypeMaster)
}

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		xgboostJob, ok := e.Object.(*xgboostv1.XGBoostJob)
		if !ok {
			return true
		}
		xgboostv1.SetDefaults_XGBoostJob(xgboostJob)
		// TODO: figure out why default funcs are not registered successfully.
		scheme.Scheme.Default(xgboostJob)
		msg := fmt.Sprintf("xgboostJob %s is created.", e.Object.GetName())
		logrus.Info(msg)

		if err := commonutil.UpdateJobConditions(&xgboostJob.Status, commonv1.JobCreated, xgboostJobCreatedReason, msg); err != nil {
			log.Log.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
