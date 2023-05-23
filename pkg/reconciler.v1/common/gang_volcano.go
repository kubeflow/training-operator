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

package common

import (
	"context"

	commonv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	controllerv1 "github.com/kubeflow/training-operator/pkg/controller.v1/common"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	"github.com/kubeflow/training-operator/pkg/util/k8sutil"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	volcano "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

// VolcanoReconciler defines a gang-scheduling reconciler for volcano.sh/volcano
type VolcanoReconciler struct {
	BaseGangReconciler
	ReconcilerUtilInterface
	client.Client
}

const (
	// VolcanoPodGroupAnnotation defines which PodGroup is linked to this Pod in annotation
	VolcanoPodGroupAnnotation = "scheduling.k8s.io/group-name"
)

// BareVolcanoReconciler returns a VolcanoReconciler pointer with minimal components defined
func BareVolcanoReconciler(client client.Client, bgReconciler *BaseGangReconciler, enabled bool) *VolcanoReconciler {
	if bgReconciler == nil {
		bgReconciler = &BaseGangReconciler{}
	}
	bgReconciler.Enabled = enabled
	return &VolcanoReconciler{
		BaseGangReconciler: *bgReconciler,
		Client:             client,
	}
}

// OverrideForGangSchedulingInterface reset ReconcilerUtilInterface used in this VolcanoReconciler
func (r *VolcanoReconciler) OverrideForGangSchedulingInterface(ui ReconcilerUtilInterface) {
	if ui != nil {
		r.ReconcilerUtilInterface = ui
	}
}

// GetGangSchedulerName returns the name of Gang Scheduler will be used, which is "volcano" for VolcanoReconciler
func (r *VolcanoReconciler) GetGangSchedulerName() string {
	return "volcano"
}

// GangSchedulingEnabled returns if gang-scheduling is enabled for all jobs
func (r *VolcanoReconciler) GangSchedulingEnabled() bool {
	return r.BaseGangReconciler.GangSchedulingEnabled()
}

// GetPodGroupName returns the name of PodGroup for this job
func (r *VolcanoReconciler) GetPodGroupName(job client.Object) string {
	return r.BaseGangReconciler.GetPodGroupName(job)
}

// GetPodGroupForJob returns the PodGroup associated with this job
func (r *VolcanoReconciler) GetPodGroupForJob(ctx context.Context, job client.Object) (client.Object, error) {
	var pg *volcano.PodGroup = nil
	err := r.Get(ctx, types.NamespacedName{
		Namespace: job.GetNamespace(),
		Name:      r.GetPodGroupName(job),
	}, pg)

	return pg, err
}

// DeletePodGroup delete the PodGroup associated with this job
func (r *VolcanoReconciler) DeletePodGroup(ctx context.Context, job client.Object) error {
	pg := &volcano.PodGroup{}
	pg.SetNamespace(job.GetNamespace())
	pg.SetName(r.GetPodGroupName(job))

	err := r.Delete(ctx, pg)
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

// ReconcilePodGroup reconciles the PodGroup resource for this job
func (r *VolcanoReconciler) ReconcilePodGroup(
	ctx context.Context,
	job client.Object,
	runPolicy *commonv1.RunPolicy,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {

	minMember := k8sutil.GetTotalReplicas(replicas)
	queue := ""
	priorityClass := ""
	var minResources *corev1.ResourceList

	if runPolicy.SchedulingPolicy != nil {
		if runPolicy.SchedulingPolicy.MinAvailable != nil {
			minMember = *runPolicy.SchedulingPolicy.MinAvailable
		}

		if runPolicy.SchedulingPolicy.Queue != "" {
			queue = runPolicy.SchedulingPolicy.Queue
		}

		if runPolicy.SchedulingPolicy.PriorityClass != "" {
			priorityClass = runPolicy.SchedulingPolicy.PriorityClass
		}

		if runPolicy.SchedulingPolicy.MinResources != nil {
			minResources = runPolicy.SchedulingPolicy.MinResources
		}
	}

	if minResources == nil {
		minResources = r.calcPGMinResources(minMember, replicas)
	}

	pgSpec := volcano.PodGroupSpec{
		MinMember:         minMember,
		Queue:             queue,
		PriorityClassName: priorityClass,
		MinResources:      minResources,
	}

	// Check if exist
	pg := &volcano.PodGroup{}
	err := r.Get(ctx, types.NamespacedName{Namespace: job.GetNamespace(), Name: r.GetPodGroupName(job)}, pg)
	// If Created, check updates, otherwise create it
	if err == nil {
		pg.Spec = pgSpec
		err = r.Update(ctx, pg)
	}

	if errors.IsNotFound(err) {
		pg.ObjectMeta = metav1.ObjectMeta{
			Name:      r.GetPodGroupName(job),
			Namespace: job.GetNamespace(),
		}
		pg.Spec = pgSpec
		err = controllerutil.SetControllerReference(job, pg, r.GetScheme())
		if err == nil {
			err = r.Create(ctx, pg)
		}
	}

	if err != nil {
		log.Warnf("Sync PodGroup %v: %v",
			types.NamespacedName{Namespace: job.GetNamespace(), Name: r.GetPodGroupName(job)}, err)
		return err
	}

	return nil
}

// DecoratePodForGangScheduling decorates the podTemplate before it's used to generate a pod with information for gang-scheduling
func (r *VolcanoReconciler) DecoratePodForGangScheduling(rtype string, podTemplate *corev1.PodTemplateSpec, job client.Object) {
	if podTemplate.Spec.SchedulerName == "" || podTemplate.Spec.SchedulerName == r.GetGangSchedulerName() {
		podTemplate.Spec.SchedulerName = r.GetGangSchedulerName()
	} else {
		warnMsg := "Another scheduler is specified when gang-scheduling is enabled and it will not be overwritten"
		commonutil.LoggerForReplica(job, rtype).Warn(warnMsg)
		r.GetRecorder().Event(job, corev1.EventTypeWarning, "PodTemplateSchedulerNameAlreadySet", warnMsg)
	}

	if podTemplate.Annotations == nil {
		podTemplate.Annotations = map[string]string{}
	}

	podTemplate.Annotations[VolcanoPodGroupAnnotation] = job.GetName()
}

// calcPGMinResources calculates the minimal resources needed for this job. The value will be embedded into the associated PodGroup
func (r *VolcanoReconciler) calcPGMinResources(minMember int32, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) *corev1.ResourceList {
	pcGetFunc := func(pc string) (*schedulingv1.PriorityClass, error) {
		priorityClass := &schedulingv1.PriorityClass{}
		err := r.Get(context.Background(), types.NamespacedName{Name: pc}, priorityClass)
		return priorityClass, err
	}

	return controllerv1.CalcPGMinResources(minMember, replicas, pcGetFunc)
}
