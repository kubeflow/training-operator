// Copyright 2023 The Kubeflow Authors
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
	"github.com/kubeflow/training-operator/pkg/util/k8sutil"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
)

// SchedulerFrameworkReconciler defines a gang-scheduling reconciler for Kubernetes Scheduler Framework
type SchedulerFrameworkReconciler struct {
	BaseGangReconciler
	ReconcilerUtilInterface
	client.Client
	SchedulerName string
}

func BareSchedulerFrameworkReconciler(client client.Client, bgReconciler *BaseGangReconciler, enabled bool) *SchedulerFrameworkReconciler {
	if bgReconciler == nil {
		bgReconciler = &BaseGangReconciler{}
	}
	bgReconciler.Enabled = enabled
	return &SchedulerFrameworkReconciler{
		BaseGangReconciler: *bgReconciler,
		Client:             client,
	}
}

func (r *SchedulerFrameworkReconciler) OverrideForGangSchedulingInterface(ui ReconcilerUtilInterface) {
	if ui != nil {
		r.ReconcilerUtilInterface = ui
	}
}

// GetGangSchedulerName returns the name of Gang Scheduler will be used.
func (r *SchedulerFrameworkReconciler) GetGangSchedulerName() string {
	return r.SchedulerName
}

// GangSchedulingEnabled returns if gang-scheduling is enabled for all jobs
func (r *SchedulerFrameworkReconciler) GangSchedulingEnabled() bool {
	return r.BaseGangReconciler.GangSchedulingEnabled()
}

// GetPodGroupName returns the name of PodGroup for this job
func (r *SchedulerFrameworkReconciler) GetPodGroupName(job client.Object) string {
	return r.BaseGangReconciler.GetPodGroupName(job)
}

// GetPodGroupForJob returns the PodGroup associated with this job
func (r *SchedulerFrameworkReconciler) GetPodGroupForJob(ctx context.Context, job client.Object) (client.Object, error) {
	var pg *schedulerpluginsv1alpha1.PodGroup = nil
	err := r.Get(ctx, types.NamespacedName{
		Namespace: job.GetNamespace(),
		Name:      r.GetPodGroupName(job),
	}, pg)
	return pg, err
}

// DeletePodGroup delete the PodGroup associated with this job
func (r *SchedulerFrameworkReconciler) DeletePodGroup(ctx context.Context, job client.Object) error {
	pg := &schedulerpluginsv1alpha1.PodGroup{}
	pg.SetNamespace(job.GetNamespace())
	pg.SetName(r.GetPodGroupName(job))
	return client.IgnoreNotFound(r.Delete(ctx, pg))
}

// ReconcilePodGroup reconciles the PodGroup resource for this job
func (r *SchedulerFrameworkReconciler) ReconcilePodGroup(
	ctx context.Context,
	job client.Object,
	runPolicy *commonv1.RunPolicy,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
) error {
	minMember := k8sutil.GetTotalReplicas(replicas)
	var scheduleTimeoutSeconds *int32
	var minResources *corev1.ResourceList

	if runPolicy.SchedulingPolicy != nil {
		if minAvailable := runPolicy.SchedulingPolicy.MinAvailable; minAvailable != nil {
			minMember = *minAvailable
		}
		if timeout := runPolicy.SchedulingPolicy.ScheduleTimeoutSeconds; timeout != nil {
			scheduleTimeoutSeconds = timeout
		}
		if mr := runPolicy.SchedulingPolicy.MinResources; mr != nil {
			minResources = (*corev1.ResourceList)(mr)
		}
	}

	if minResources == nil {
		minResources = r.calcPGMinResources(minMember, replicas)
	}

	pgSpec := schedulerpluginsv1alpha1.PodGroupSpec{
		MinMember:              minMember,
		MinResources:           *minResources,
		ScheduleTimeoutSeconds: scheduleTimeoutSeconds,
	}

	// Check if exist
	pg := &schedulerpluginsv1alpha1.PodGroup{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      r.GetPodGroupName(job),
		Namespace: job.GetNamespace(),
	}, pg)
	// If created, check updates, otherwise create it.
	if err == nil {
		pg.Spec = pgSpec
		err = r.Update(ctx, pg)
	}

	if errors.IsNotFound(err) {
		pg.TypeMeta = metav1.TypeMeta{
			APIVersion: schedulerpluginsv1alpha1.SchemeGroupVersion.String(),
			Kind:       "PodGroup",
		}
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
		log.Warnf("Sync PodGroup %v: %v", klog.KObj(pg), err)
		return err
	}
	return nil
}

// DecoratePodForGangScheduling decorates the podTemplate before it's used to generate a pod with information for gang-scheduling
func (r *SchedulerFrameworkReconciler) DecoratePodForGangScheduling(
	_ string,
	podTemplate *corev1.PodTemplateSpec,
	job client.Object,
) {
	if podTemplate.Spec.SchedulerName == "" {
		podTemplate.Spec.SchedulerName = r.GetGangSchedulerName()
	}

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	podTemplate.Labels[schedulerpluginsv1alpha1.PodGroupLabel] = job.GetName()
}

// calcPGMinResources calculates the minimal resources needed for this job. The value will be embedded into the associated PodGroup
func (r *SchedulerFrameworkReconciler) calcPGMinResources(
	minMember int32,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
) *corev1.ResourceList {
	return controllerv1.CalcPGMinResources(minMember, replicas,
		func(pc string) (*schedulingv1.PriorityClass, error) {
			priorityClass := &schedulingv1.PriorityClass{}
			err := r.Get(context.TODO(), types.NamespacedName{Name: pc}, priorityClass)
			return priorityClass, err
		})
}
