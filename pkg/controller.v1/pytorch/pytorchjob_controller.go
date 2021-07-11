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
	"github.com/kubeflow/tf-operator/pkg/common/util"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

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
	ControllerName = "pytorchjob-operator"
)

// NewReconciler creates a PyTorchJob Reconciler
func NewReconciler(mgr manager.Manager) *PyTorchJobReconciler {
	r := &PyTorchJobReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor(ControllerName),
		Log:      log.Log,
	}

	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)

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
	recorder record.EventRecorder
	Log      logr.Logger
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

	clientReader, err := getDelegatingClientFromClient(r.Client)
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

func (r *PyTorchJobReconciler) DeleteJob(job interface{}) error {
	pytorchjob, ok := job.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", job)
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

//+kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=pytorchjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
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

	// Convert PyTorch.Spec.PyTorchReplicasSpecs to  map[commonv1.ReplicaType]*commonv1.ReplicaSpec
	replicas := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{}
	for k, v := range pytorchjob.Spec.PyTorchReplicaSpecs {
		replicas[commonv1.ReplicaType(k)] = v
	}

	// Construct RunPolicy based on PyTorchJob.Spec
	runPolicy := &commonv1.RunPolicy{
		CleanPodPolicy:          pytorchjob.Spec.CleanPodPolicy,
		TTLSecondsAfterFinished: pytorchjob.Spec.TTLSecondsAfterFinished,
		ActiveDeadlineSeconds:   pytorchjob.Spec.ActiveDeadlineSeconds,
		BackoffLimit:            pytorchjob.Spec.BackoffLimit,
		SchedulingPolicy:        nil,
	}

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(pytorchjob, replicas, pytorchjob.Status, runPolicy)
	if err != nil {
		logrus.Warnf("Reconcile PyTorch Job error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PyTorchJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//// Create a new Controller
	//c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	//if err != nil {
	//	return err
	//}
	//
	//// Watch for changes to PyTorchJob
	//err = c.Watch(&source.Kind{Type: &pytorchv1.PyTorchJob{}}, &handler.EnqueueRequestForObject{},
	//	predicate.Funcs{CreateFunc: onOwnerCreateFunc(r)},
	//)
	//if err != nil {
	//	return err
	//}
	//
	////inject watching for pytorchjob related pod
	//err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	//	IsController: true,
	//	OwnerType:    &pytorchv1.PyTorchJob{},
	//},
	//	predicate.Funcs{CreateFunc: onDependentCreateFunc(r), DeleteFunc: onDependentDeleteFunc(r)},
	//)
	//if err != nil {
	//	return err
	//}
	//
	////inject watching for xgboostjob related service
	//err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
	//	IsController: true,
	//	OwnerType:    &pytorchv1.PyTorchJob{},
	//},
	//	&predicate.Funcs{CreateFunc: onDependentCreateFunc(r), DeleteFunc: onDependentDeleteFunc(r)},
	//)
	//if err != nil {
	//	return err
	//}
	//
	//return nil

	return ctrl.NewControllerManagedBy(mgr).
		For(&pytorchv1.PyTorchJob{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *PyTorchJobReconciler) ControllerName() string {
	return ControllerName
}

func (r *PyTorchJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return pytorchv1.GroupVersion.WithKind(pytorchv1.Kind)
}

func (r *PyTorchJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return pytorchv1.GroupVersion
}

func (r *PyTorchJobReconciler) GetGroupNameLabelValue() string {
	return pytorchv1.GroupName
}

func (r *PyTorchJobReconciler) GetDefaultContainerName() string {
	return pytorchv1.DefaultContainerName
}

func (r *PyTorchJobReconciler) GetDefaultContainerPortName() string {
	return pytorchv1.DefaultPortName
}

func (r *PyTorchJobReconciler) GetJobRoleKey() string {
	return "pytorchjob-job-role"
}

func (r *PyTorchJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(pytorchv1.PyTorchReplicaTypeMaster)
}

// SetClusterSpec sets the cluster spec for the pod
func (r *PyTorchJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, rtype, index)
}
