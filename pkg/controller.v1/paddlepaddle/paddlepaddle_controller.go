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
	"strings"
	"time"

	commonv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	trainingoperatorcommon "github.com/kubeflow/training-operator/pkg/common"
	"github.com/kubeflow/training-operator/pkg/common/util"
	"github.com/kubeflow/training-operator/pkg/controller.v1/common"
	"github.com/kubeflow/training-operator/pkg/controller.v1/control"
	"github.com/kubeflow/training-operator/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/training-operator/pkg/util"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

const (
	controllerName = "paddlejob-controller"
)

// NewReconciler creates a PaddleJob Reconciler
func NewReconciler(mgr manager.Manager, gangSchedulingSetupFunc common.GangSchedulingSetupFunc) *PaddleJobReconciler {
	r := &PaddleJobReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor(controllerName),
		apiReader: mgr.GetAPIReader(),
		Log:       log.Log,
	}

	// Create clients
	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	sharedInformers := informers.NewSharedInformerFactory(kubeClientSet, 0)
	priorityClassInformer := sharedInformers.Scheduling().V1().PriorityClasses()

	// Initialize common job controller
	r.JobController = common.JobController{
		Controller:                  r,
		Expectations:                expectation.NewControllerExpectations(),
		WorkQueue:                   &util.FakeWorkQueue{},
		Recorder:                    r.recorder,
		KubeClientSet:               kubeClientSet,
		PriorityClassLister:         priorityClassInformer.Lister(),
		PriorityClassInformerSynced: priorityClassInformer.Informer().HasSynced,
		PodControl:                  control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		ServiceControl:              control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.recorder},
	}

	gangSchedulingSetupFunc(&r.JobController)

	return r
}

// PaddleJobReconciler reconciles a PaddleJob object
type PaddleJobReconciler struct {
	common.JobController
	client.Client
	Scheme    *runtime.Scheme
	Log       logr.Logger
	recorder  record.EventRecorder
	apiReader client.Reader
}

//+kubebuilder:rbac:groups=kubeflow.org,resources=paddlejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=paddlejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=paddlejobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=scheduling.volcano.sh,resources=podgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduling.x-k8s.io,resources=podgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the PaddleJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PaddleJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues(kubeflowv1.PaddleJobSingular, req.NamespacedName)

	paddlejob := &kubeflowv1.PaddleJob{}
	err := r.Get(ctx, req.NamespacedName, paddlejob)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch PaddleJob", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = kubeflowv1.ValidateV1PaddleJob(paddlejob); err != nil {
		logger.Error(err, "PaddleJob failed validation")
		r.Recorder.Eventf(paddlejob, corev1.EventTypeWarning, commonutil.JobFailedValidationReason, "PaddleJob failed validation because %s", err)
		return ctrl.Result{}, err
	}

	// Check if reconciliation is needed
	jobKey, err := common.KeyFunc(paddlejob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get jobKey for job object %#v: %v", paddlejob, err))
	}

	replicaTypes := util.GetReplicaTypes(paddlejob.Spec.PaddleReplicaSpecs)
	needReconcile := util.SatisfiedExpectations(r.Expectations, jobKey, replicaTypes)

	if !needReconcile || paddlejob.GetDeletionTimestamp() != nil {
		logger.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needReconcile, "deleted", paddlejob.GetDeletionTimestamp() != nil)
		return ctrl.Result{}, nil
	}

	// Set default priorities to paddle job
	r.Scheme.Default(paddlejob)

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(paddlejob, paddlejob.Spec.PaddleReplicaSpecs, paddlejob.Status, &paddlejob.Spec.RunPolicy)
	if err != nil {
		logger.Error(err, "Reconcile PaddleJob error")
		return ctrl.Result{}, err
	}

	t, err := util.DurationUntilExpireTime(&paddlejob.Spec.RunPolicy, paddlejob.Status)
	if err != nil {
		logrus.Warnf("Reconcile PaddleJob error %v", err)
		return ctrl.Result{}, err
	}
	if t >= 0 {
		return ctrl.Result{Requeue: true, RequeueAfter: t}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PaddleJobReconciler) SetupWithManager(mgr ctrl.Manager, controllerThreads int) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: controllerThreads,
	})

	if err != nil {
		return err
	}

	// using onOwnerCreateFunc is easier to set defaults
	if err = c.Watch(&source.Kind{Type: &kubeflowv1.PaddleJob{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: r.onOwnerCreateFunc()},
	); err != nil {
		return err
	}

	// inject watching for job related pod
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubeflowv1.PaddleJob{},
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
		OwnerType:    &kubeflowv1.PaddleJob{},
	}, predicate.Funcs{
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}); err != nil {
		return err
	}

	// skip watching volcano PodGroup if volcano PodGroup is not installed
	_, err = mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: v1beta1.SchemeGroupVersion.Group, Kind: "PodGroup"},
		v1beta1.SchemeGroupVersion.Version)
	if err == nil {
		// inject watching for job related volcano PodGroup
		if err = c.Watch(&source.Kind{Type: &v1beta1.PodGroup{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &kubeflowv1.PaddleJob{},
		}, predicate.Funcs{
			CreateFunc: util.OnDependentCreateFuncGeneric(r.Expectations),
			UpdateFunc: util.OnDependentUpdateFuncGeneric(&r.JobController),
			DeleteFunc: util.OnDependentDeleteFuncGeneric(r.Expectations),
		}); err != nil {
			return err
		}
	}

	// skip watching scheduler-plugins PodGroup if scheduler-plugins PodGroup is not installed
	_, err = mgr.GetRESTMapper().RESTMapping(
		schema.GroupKind{Group: schedulerpluginsv1alpha1.SchemeGroupVersion.Group, Kind: "PodGroup"},
		schedulerpluginsv1alpha1.SchemeGroupVersion.Version)
	if err == nil {
		// inject watching for job related scheduler-plugins PodGroup
		if err = c.Watch(&source.Kind{Type: &schedulerpluginsv1alpha1.PodGroup{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &kubeflowv1.PaddleJob{},
		}, predicate.Funcs{
			CreateFunc: util.OnDependentCreateFuncGeneric(r.Expectations),
			UpdateFunc: util.OnDependentUpdateFuncGeneric(&r.JobController),
			DeleteFunc: util.OnDependentDeleteFuncGeneric(r.Expectations),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *PaddleJobReconciler) ControllerName() string {
	return controllerName
}

func (r *PaddleJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return kubeflowv1.GroupVersion.WithKind(kubeflowv1.PaddleJobKind)
}

func (r *PaddleJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return kubeflowv1.GroupVersion
}

func (r *PaddleJobReconciler) GetGroupNameLabelValue() string {
	return kubeflowv1.GroupVersion.Group
}

func (r *PaddleJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &kubeflowv1.PaddleJob{}
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Error(err, "paddle job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *PaddleJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &kubeflowv1.PaddleJob{}

	err := r.apiReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Error(err, "paddle job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *PaddleJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
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

	return util.JobControlledPodList(podlist.Items, job), nil
}

func (r *PaddleJobReconciler) GetServicesForJob(obj interface{}) ([]*corev1.Service, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
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

func (r *PaddleJobReconciler) DeleteJob(job interface{}) error {
	paddlejob, ok := job.(*kubeflowv1.PaddleJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PaddleJob", job)
	}
	if err := r.Delete(context.Background(), paddlejob); err != nil {
		r.recorder.Eventf(paddlejob, corev1.EventTypeWarning, control.FailedDeletePodReason, "Error deleting: %v", err)
		logrus.Error(err, "failed to delete job", "namespace", paddlejob.Namespace, "name", paddlejob.Name)
		return err
	}
	r.recorder.Eventf(paddlejob, corev1.EventTypeNormal, control.SuccessfulDeletePodReason, "Deleted job: %v", paddlejob.Name)
	logrus.Info("job deleted", "namespace", paddlejob.Namespace, "name", paddlejob.Name)
	trainingoperatorcommon.DeletedJobsCounterInc(paddlejob.Namespace, kubeflowv1.PaddleJobFrameworkName)
	return nil
}

func (jc *PaddleJobReconciler) GenLabelSelector(jobName string,
	rtype commonv1.ReplicaType) *metav1.LabelSelector {
	labels := jc.GenLabels(jobName)
	labels[commonv1.ReplicaTypeLabel] = strings.ToLower(string(rtype))

	return &metav1.LabelSelector{
		MatchLabels: labels,
	}
}

// UpdateJobStatus updates the job status and job conditions
func (r *PaddleJobReconciler) UpdateJobStatus(job interface{},
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	jobStatus *commonv1.JobStatus) error {
	paddlejob, ok := job.(*kubeflowv1.PaddleJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PaddleJob", job)
	}

	paddlejobKey, err := common.KeyFunc(paddlejob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for paddlejob object %#v: %v", paddlejob, err))
		return err
	}

	logger := commonutil.LoggerForJob(paddlejob)

	// Set StartTime.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		if paddlejob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
			logger.Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *paddlejob.Spec.RunPolicy.ActiveDeadlineSeconds)
			r.WorkQueue.AddAfter(paddlejobKey, time.Duration(*paddlejob.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
		}
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]
		// Generate the label selector.
		status.Selector = metav1.FormatLabelSelector(r.GenLabelSelector(paddlejob.Name, rtype))

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed
		specReplicas := *spec.Replicas

		logrus.Infof("PaddleJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d, failed=%d, Replicas=%d",
			paddlejob.Name, rtype, expected, running, succeeded, failed, specReplicas)

		if ContainsMasterSpec(replicas) {
			if rtype == commonv1.ReplicaType(kubeflowv1.PaddleJobReplicaTypeMaster) {
				if running > 0 {
					msg := fmt.Sprintf("PaddleJob %s is running.", paddlejob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						commonutil.LoggerForJob(paddlejob).Infof("Append job condition error: %v", err)
						return err
					}
				}
				// when master is succeed, the job is finished.
				if expected == 0 {
					msg := fmt.Sprintf("PaddleJob %s is successfully completed.", paddlejob.Name)
					logrus.Info(msg)
					r.Recorder.Event(paddlejob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, commonutil.JobSucceededReason, msg)
					if err != nil {
						commonutil.LoggerForJob(paddlejob).Infof("Append job condition error: %v", err)
						return err
					}
					trainingoperatorcommon.SuccessfulJobsCounterInc(paddlejob.Namespace, kubeflowv1.PaddleJobFrameworkName)
					return nil
				}
			}
		} else {
			if rtype == kubeflowv1.PaddleJobReplicaTypeWorker {
				// TODO(gaocegege): Support SuccessPolicy
				if expected == 0 {
					msg := fmt.Sprintf("PaddleJob %s/%s successfully completed.",
						paddlejob.Namespace, paddlejob.Name)
					r.recorder.Event(paddlejob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus,
						commonv1.JobSucceeded, commonutil.JobSucceededReason, msg)
					if err != nil {
						commonutil.LoggerForJob(paddlejob).Infof("Append paddlejob condition error: %v", err)
						return err
					}
					trainingoperatorcommon.SuccessfulJobsCounterInc(paddlejob.Namespace, kubeflowv1.PaddleJobFrameworkName)
				} else if running > 0 {
					// Some workers are still running, leave a running condition.
					msg := fmt.Sprintf("PaddleJob %s/%s is running.",
						paddlejob.Namespace, paddlejob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg)
					if err != nil {
						commonutil.LoggerForJob(paddlejob).Infof("Append paddlejob condition error: %v", err)
						return err
					}
				}
			}
		}

		if failed > 0 && (specReplicas > succeeded+running) {
			if spec.RestartPolicy != commonv1.RestartPolicyNever {
				msg := fmt.Sprintf("PaddleJob %s is restarting because %d %s replica(s) failed.", paddlejob.Name, failed, rtype)
				r.Recorder.Event(paddlejob, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					commonutil.LoggerForJob(paddlejob).Infof("Append job condition error: %v", err)
					return err
				}
				trainingoperatorcommon.RestartedJobsCounterInc(paddlejob.Namespace, kubeflowv1.PaddleJobFrameworkName)
			} else {
				msg := fmt.Sprintf("PaddleJob %s is failed because %d %s replica(s) failed.", paddlejob.Name, failed, rtype)
				r.Recorder.Event(paddlejob, corev1.EventTypeNormal, commonutil.JobFailedReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, commonutil.JobFailedReason, msg)
				if err != nil {
					commonutil.LoggerForJob(paddlejob).Infof("Append job condition error: %v", err)
					return err
				}
				trainingoperatorcommon.FailedJobsCounterInc(paddlejob.Namespace, kubeflowv1.PaddleJobFrameworkName)
			}
		}
	}

	return nil
}

// ContainsMasterSpec returns true if the paddlejob contains master spec.
func ContainsMasterSpec(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) bool {
	if _, ok := replicas[kubeflowv1.PaddleJobReplicaTypeMaster]; ok {
		return true
	}
	return false
}

// UpdateJobStatusInApiServer updates the job status in to cluster.
func (r *PaddleJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = map[commonv1.ReplicaType]*commonv1.ReplicaStatus{}
	}

	paddlejob, ok := job.(*kubeflowv1.PaddleJob)
	trainingoperatorcommon.ClearGeneratedFields(&paddlejob.ObjectMeta)
	if !ok {
		return fmt.Errorf("%+v is not a type of PaddleJob", job)
	}

	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !equality.Semantic.DeepEqual(&paddlejob.Status, jobStatus) {
		paddlejob = paddlejob.DeepCopy()
		paddlejob.Status = *jobStatus.DeepCopy()
	}

	result := r.Status().Update(context.Background(), paddlejob)

	if result != nil {
		r.Log.WithValues("paddlejob", types.NamespacedName{
			Namespace: paddlejob.GetNamespace(),
			Name:      paddlejob.GetName(),
		})
		return result
	}

	return nil
}

// SetClusterSpec sets the cluster spec and init container for the pod
func (r *PaddleJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	// TODO
	if err := setPodEnv(job, podTemplate, rtype, index); err != nil {
		return err
	}
	return nil
}

func (r *PaddleJobReconciler) GetDefaultContainerName() string {
	return kubeflowv1.PaddleJobDefaultContainerName
}

func (r *PaddleJobReconciler) GetDefaultContainerPortName() string {
	return kubeflowv1.PaddleJobDefaultPortName
}

func (r *PaddleJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(kubeflowv1.PaddleJobReplicaTypeMaster)
}

// onOwnerCreateFunc modify creation condition.
func (r *PaddleJobReconciler) onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		paddlejob, ok := e.Object.(*kubeflowv1.PaddleJob)
		if !ok {
			return true
		}
		r.Scheme.Default(paddlejob)
		msg := fmt.Sprintf("PaddleJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		trainingoperatorcommon.CreatedJobsCounterInc(paddlejob.Namespace, kubeflowv1.PaddleJobFrameworkName)
		if err := commonutil.UpdateJobConditions(&paddlejob.Status, commonv1.JobCreated, "PaddleJobCreated", msg); err != nil {
			logrus.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
