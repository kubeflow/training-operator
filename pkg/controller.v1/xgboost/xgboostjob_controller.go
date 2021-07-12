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

package xgboost

import (
	"context"
	"fmt"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/tf-operator/pkg/common/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xgboostv1 "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
)

const (
	controllerName      = "xgboostjob-operator"
	labelXGBoostJobRole = "xgboostjob-job-role"
)

var (
	jobOwnerKey           = ".metadata.controller"
	defaultTTLSeconds     = int32(100)
	defaultCleanPodPolicy = commonv1.CleanPodPolicyNone
)

func NewReconciler(mgr manager.Manager) *XGBoostJobReconciler {
	r := &XGBoostJobReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("XGBoostJob"),
		Scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())

	// Create clients.
	kubeClientSet, _, volcanoClientSet, err := createClientSets(ctrl.GetConfigOrDie())
	if err != nil {
		r.Log.Info("Error building kubeclientset: %s", err.Error())
	}

	// Initialize common job controller
	r.JobController = common.JobController{
		Controller:   r,
		Expectations: expectation.NewControllerExpectations(),
		// TODO: add batch scheduler check later.
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

// Reconcile reads that state of the cluster for a XGBoostJob object and makes changes based on the state read
// and what is in the XGBoostJob.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=xgboostjobs/finalizers,verbs=update

func (r *XGBoostJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("xgboostjob", req.NamespacedName)

	xgboostjob := &xgboostv1.XGBoostJob{}
	err := r.Get(context.Background(), req.NamespacedName, xgboostjob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Check reconcile is required.
	jobKey, err := common.KeyFunc(xgboostjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get jobKey for job object %#v: %v", xgboostjob, err))
	}

	replicaTypes := util.GetReplicaTypes(xgboostjob.Spec.XGBReplicaSpecs)
	needSync := util.SatisfiedExpectations(r.Expectations, jobKey, replicaTypes)

	if !needSync || xgboostjob.DeletionTimestamp != nil {
		logger.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", xgboostjob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}

	// Set default priorities for xgboost job
	scheme.Scheme.Default(xgboostjob)

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(xgboostjob, xgboostjob.Spec.XGBReplicaSpecs, xgboostjob.Status.JobStatus, &xgboostjob.Spec.RunPolicy)
	if err != nil {
		logger.V(2).Error(err, "Reconcile XGBoost Job error")
		return ctrl.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *XGBoostJobReconciler) ControllerName() string {
	return controllerName
}

func (r *XGBoostJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return xgboostv1.SchemeBuilder.GroupVersion.WithKind(xgboostv1.Kind)
}

func (r *XGBoostJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return xgboostv1.GroupVersion
}

func (r *XGBoostJobReconciler) GetGroupNameLabelValue() string {
	return xgboostv1.GroupName
}

func (r *XGBoostJobReconciler) GetDefaultContainerName() string {
	return xgboostv1.DefaultContainerName
}

func (r *XGBoostJobReconciler) GetDefaultContainerPortName() string {
	return xgboostv1.DefaultContainerPortName
}

func (r *XGBoostJobReconciler) GetJobRoleKey() string {
	return labelXGBoostJobRole
}

func (r *XGBoostJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return string(rtype) == string(xgboostv1.XGBoostReplicaTypeMaster)
}

// SetClusterSpec sets the cluster spec for the pod
func (r *XGBoostJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return SetPodEnv(job, podTemplate, rtype, index)
}

// SetupWithManager sets up the controller with the Manager.
func (r *XGBoostJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		For(&xgboostv1.XGBoostJob{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
