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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controllerName = "mxnet-operator"

	// labels for pods and servers.
	mxReplicaTypeLabel  = "mxnet-replica-type"
	mxReplicaIndexLabel = "mxnet-replica-index"
	labelGroupName      = "group-name"
	labelMXJobName      = "mxnet-job-name"
	labelMXJobRole      = "mxnet-job-role"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

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

	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)

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

	// mxjobv1.SetDefaults(mxjob)

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

	// Set default priorities to pytorch job
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
	// Create a new Controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to PyTorchJob
	err = c.Watch(&source.Kind{Type: &mxjobv1.MXJob{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: onOwnerCreateFunc()},
	)
	if err != nil {
		return err
	}

	//inject watching for pytorchjob related pod
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mxjobv1.MXJob{},
	},
		predicate.Funcs{CreateFunc: util.OnDependentCreateFunc(r.Expectations), DeleteFunc: util.OnDependentDeleteFunc(r.Expectations)},
	)
	if err != nil {
		return err
	}

	//inject watching for xgboostjob related service
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mxjobv1.MXJob{},
	},
		&predicate.Funcs{CreateFunc: util.OnDependentCreateFunc(r.Expectations), DeleteFunc: util.OnDependentDeleteFunc(r.Expectations)},
	)
	if err != nil {
		return err
	}

	return nil
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

// satisfiedExpectations returns true if the required adds/dels for the given mxjob have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.

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

func (r *MXJobReconciler) GetDefaultContainerName() string {
	return mxjobv1.DefaultContainerName
}

func (r *MXJobReconciler) GetDefaultContainerPortName() string {
	return mxjobv1.DefaultPortName
}

func (r *MXJobReconciler) GetJobRoleKey() string {
	return labelMXJobRole
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
