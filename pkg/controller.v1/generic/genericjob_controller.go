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
// limitations under the License

package generic

import (
	"context"

	"github.com/kubeflow/common/pkg/reconciler.v1/common"
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type KubeflowReconciler struct {
	common.ReconcilerUtil
	common.ServiceReconciler
	common.PodReconciler
	common.JobReconciler
	common.VolcanoReconciler

	client.Client
}

func NewReonciler(mgr manager.Manager, enableGangScheduling bool) *KubeflowReconciler {
	r := &KubeflowReconciler{Client: mgr.GetClient()}

	jobR := common.BareJobReconciler(mgr.GetClient())
	jobR.OverrideForJobInterface(r, r, r, r)

	podR := common.BarePodReconciler(mgr.GetClient())
	podR.OverrideForPodInterface(r, r, r)

	svcR := common.BareServiceReconciler(mgr.GetClient())
	svcR.OverrideForServiceInterface(r, r, r)

	gangR := common.BareVolcanoReconciler(mgr.GetClient(), nil, enableGangScheduling)
	gangR.OverrideForGangSchedulingInterface(r)

	utilR := common.BareUtilReconciler(mgr.GetEventRecorderFor(r.GetReconcilerName()), mgr.GetLogger(), mgr.GetScheme())

	r.JobReconciler = *jobR
	r.PodReconciler = *podR
	r.ServiceReconciler = *svcR
	r.VolcanoReconciler = *gangR
	r.ReconcilerUtil = *utilR

	return r
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KubeflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	job, err := r.GetJob(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger := r.GetLogger(job)

	if job.GetDeletionTimestamp() != nil {
		logger.Info(common.MsgReconcileCancelled, common.ReasonKey, common.ReasonJobDeleted)
		return ctrl.Result{}, nil
	}

	scheme.Scheme.Default(job)

	// Get rid of SatisfiedExpectation
	replicasSpec, err := r.ExtractReplicasSpec(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	runPolicy, err := r.ExtractRunPolicy(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	status, err := r.ExtractJobStatus(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.ReconcileJob(ctx, job, replicasSpec, status, runPolicy)
	if err != nil {
		logger.Info("Reconcile Generic Job failed",
			"job", req.NamespacedName.String(), "error", err.Error())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeflowReconciler) SetupWithManager(mgr ctrl.Manager, obj client.Object) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(obj).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
