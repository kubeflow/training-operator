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

package controllers

import (
	"context"
	"fmt"
	"reflect"

	commonutil "github.com/kubeflow/common/pkg/util"
	"github.com/kubeflow/tf-operator/pkg/common/util"
	"k8s.io/apimachinery/pkg/api/meta"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/go-logr/logr"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	pytorchv1 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1"
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"
)

const (
	controllerName = "pytorchjob-operator"
)

var (
	jobOwnerKey           = ".metadata.controller"
	defaultCleanPodPolicy = commonv1.CleanPodPolicyNone
)

// NewReconciler creates a PyTorchJob Reconciler
func NewReconciler(mgr manager.Manager) *PyTorchJobReconciler {
	r := &PyTorchJobReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor(controllerName),
		Log:      log.Log,
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

// PyTorchJobReconciler reconciles a PyTorchJob object
type PyTorchJobReconciler struct {
	common.JobController
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the PyTorchJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PyTorchJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues(pytorchv1.Singular, req.NamespacedName)

	pytorchjob := &pytorchv1.PyTorchJob{}
	err := r.Get(ctx, req.NamespacedName, pytorchjob)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch PyTorchJob", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	pytorchv1.SetDefaults_PyTorchJob(pytorchjob)

	// Check if reconciliation is needed
	jobKey, err := common.KeyFunc(pytorchjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get jobKey for job object %#v: %v", pytorchjob, err))
	}

	replicaTypes := util.GetReplicaTypes(pytorchjob.Spec.PyTorchReplicaSpecs)
	needReconcile := util.SatisfiedExpectations(r.Expectations, jobKey, replicaTypes)

	if !needReconcile || pytorchjob.GetDeletionTimestamp() != nil {
		logger.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needReconcile, "deleted", pytorchjob.GetDeletionTimestamp() != nil)
		return ctrl.Result{}, nil
	}

	// Set default priorities to pytorch job
	scheme.Scheme.Default(pytorchjob)

	// Construct RunPolicy based on PyTorchJob.Spec
	runPolicy := &commonv1.RunPolicy{
		CleanPodPolicy:          pytorchjob.Spec.CleanPodPolicy,
		TTLSecondsAfterFinished: pytorchjob.Spec.TTLSecondsAfterFinished,
		ActiveDeadlineSeconds:   pytorchjob.Spec.ActiveDeadlineSeconds,
		BackoffLimit:            pytorchjob.Spec.BackoffLimit,
		SchedulingPolicy:        nil,
	}

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(pytorchjob, pytorchjob.Spec.PyTorchReplicaSpecs, pytorchjob.Status, runPolicy)
	if err != nil {
		logrus.Warnf("Reconcile PyTorch Job error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PyTorchJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// setup FieldIndexer to inform the manager that this controller owns pods and services,
	// so that it will automatically call Reconcile on the underlying XGBoostJob when a Pod or Service changes, is deleted, etc.
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
		For(&pytorchv1.PyTorchJob{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *PyTorchJobReconciler) ControllerName() string {
	return controllerName
}

func (r *PyTorchJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return pytorchv1.GroupVersion.WithKind(pytorchv1.Kind)
}

func (r *PyTorchJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return pytorchv1.GroupVersion
}

func (r *PyTorchJobReconciler) GetGroupNameLabelValue() string {
	return pytorchv1.GroupVersion.Group
}

func (r *PyTorchJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &pytorchv1.PyTorchJob{}
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Error(err, "pytorch job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

func (r *PyTorchJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &pytorchv1.PyTorchJob{}

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

func (r *PyTorchJobReconciler) GetPodsForJob(obj interface{}) ([]*corev1.Pod, error) {
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

func (r *PyTorchJobReconciler) GetServicesForJob(obj interface{}) ([]*corev1.Service, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
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

func (r *PyTorchJobReconciler) DeleteJob(job interface{}) error {
	pytorchjob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", job)
	}
	if err := r.Delete(context.Background(), pytorchjob); err != nil {
		r.recorder.Eventf(pytorchjob, corev1.EventTypeWarning, control.FailedDeletePodReason, "Error deleting: %v", err)
		logrus.Error(err, "failed to delete job", "namespace", pytorchjob.Namespace, "name", pytorchjob.Name)
		return err
	}
	r.recorder.Eventf(pytorchjob, corev1.EventTypeNormal, control.SuccessfulDeletePodReason, "Deleted job: %v", pytorchjob.Name)
	logrus.Info("job deleted", "namespace", pytorchjob.Namespace, "name", pytorchjob.Name)
	return nil
}

// UpdateJobStatus updates the job status and job conditions
func (r *PyTorchJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	pytorchjob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", job)
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logrus.Infof("XGBoostJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			pytorchjob.Name, rtype, expected, running, succeeded, failed)

		if rtype == commonv1.ReplicaType(pytorchv1.PyTorchReplicaTypeMaster) {
			if running > 0 {
				msg := fmt.Sprintf("XGBoostJob %s is running.", pytorchjob.Name)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
			}
			// when master is succeed, the job is finished.
			if expected == 0 {
				msg := fmt.Sprintf("XGBoostJob %s is successfully completed.", pytorchjob.Name)
				logrus.Info(msg)
				r.Recorder.Event(pytorchjob, corev1.EventTypeNormal, commonutil.JobSucceededReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, commonutil.JobSucceededReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
				return nil
			}
		}
		if failed > 0 {
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				msg := fmt.Sprintf("XGBoostJob %s is restarting because %d %s replica(s) failed.", pytorchjob.Name, failed, rtype)
				r.Recorder.Event(pytorchjob, corev1.EventTypeWarning, commonutil.JobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, commonutil.JobRestartingReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
			} else {
				msg := fmt.Sprintf("XGBoostJob %s is failed because %d %s replica(s) failed.", pytorchjob.Name, failed, rtype)
				r.Recorder.Event(pytorchjob, corev1.EventTypeNormal, commonutil.JobFailedReason, msg)
				if pytorchjob.Status.CompletionTime == nil {
					now := metav1.Now()
					pytorchjob.Status.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, commonutil.JobFailedReason, msg)
				if err != nil {
					commonutil.LoggerForJob(pytorchjob).Infof("Append job condition error: %v", err)
					return err
				}
			}
		}
	}

	// Some workers are still running, leave a running condition.
	msg := fmt.Sprintf("PyTorchJob %s is running.", pytorchjob.Name)
	commonutil.LoggerForJob(pytorchjob).Infof(msg)

	if err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, commonutil.JobRunningReason, msg); err != nil {
		commonutil.LoggerForJob(pytorchjob).Error(err, "failed to update XGBoost Job conditions")
		return err
	}

	return nil
}

// UpdateJobStatusInApiServer updates the job status in to cluster.
func (r *PyTorchJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	pytorchjob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", job)
	}

	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !reflect.DeepEqual(&pytorchjob.Status, jobStatus) {
		pytorchjob = pytorchjob.DeepCopy()
		pytorchjob.Status = *jobStatus.DeepCopy()
	}

	result := r.Update(context.Background(), pytorchjob)

	if result != nil {
		r.Log.WithValues("pytorchjob", types.NamespacedName{
			Namespace: pytorchjob.GetNamespace(),
			Name:      pytorchjob.GetName(),
		})
		return result
	}

	return nil
}

// SetClusterSpec sets the cluster spec for the pod
func (r *PyTorchJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, rtype, index)
}

func (r *PyTorchJobReconciler) GetDefaultContainerName() string {
	return pytorchv1.DefaultContainerName
}

func (r *PyTorchJobReconciler) GetDefaultContainerPortName() string {
	return pytorchv1.DefaultPortName
}

func (r *PyTorchJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(pytorchv1.PyTorchReplicaTypeMaster)
}

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		pytorchjob, ok := e.Object.(*pytorchv1.PyTorchJob)
		if !ok {
			return true
		}
		scheme.Scheme.Default(pytorchjob)
		msg := fmt.Sprintf("PyTorchJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		//specific the run policy

		if pytorchjob.Spec.CleanPodPolicy == nil {
			pytorchjob.Spec.CleanPodPolicy = new(commonv1.CleanPodPolicy)
			pytorchjob.Spec.CleanPodPolicy = &defaultCleanPodPolicy
		}

		if err := commonutil.UpdateJobConditions(&pytorchjob.Status, commonv1.JobCreated, "PyTorchJobCreated", msg); err != nil {
			logrus.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
