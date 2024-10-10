/*
Copyright 2024 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package coscheduling

import (
	"context"
	"errors"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
	runtimeindexer "github.com/kubeflow/training-operator/pkg/runtime.v2/indexer"
)

type CoScheduling struct {
	client     client.Client
	restMapper meta.RESTMapper
	scheme     *apiruntime.Scheme
}

var _ framework.EnforcePodGroupPolicyPlugin = (*CoScheduling)(nil)
var _ framework.WatchExtensionPlugin = (*CoScheduling)(nil)
var _ framework.ComponentBuilderPlugin = (*CoScheduling)(nil)

var (
	ErrorCanNotSetupTrainingRuntimeRuntimeClassIndexer        = errors.New("setting index on runtimeClass for TrainingRuntime")
	ErrorCanNotSetupClusterTrainingRuntimeRuntimeClassIndexer = errors.New("setting index on runtimeClass for ClusterTrainingRuntime")
)

const Name = "CoScheduling"

func New(ctx context.Context, c client.Client, indexer client.FieldIndexer) (framework.Plugin, error) {
	if err := indexer.IndexField(ctx, &kubeflowv2.TrainingRuntime{}, TrainingRuntimeContainerRuntimeClassKey,
		IndexTrainingRuntimeContainerRuntimeClass); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrorCanNotSetupTrainingRuntimeRuntimeClassIndexer, err)
	}
	if err := indexer.IndexField(ctx, &kubeflowv2.ClusterTrainingRuntime{}, ClusterTrainingRuntimeContainerRuntimeClassKey,
		IndexClusterTrainingRuntimeContainerRuntimeClass); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrorCanNotSetupClusterTrainingRuntimeRuntimeClassIndexer, err)
	}
	return &CoScheduling{
		client:     c,
		restMapper: c.RESTMapper(),
		scheme:     c.Scheme(),
	}, nil
}

func (c *CoScheduling) Name() string {
	return Name
}

func (c *CoScheduling) EnforcePodGroupPolicy(trainJob *kubeflowv2.TrainJob, info *runtime.Info) error {
	if info == nil || info.PodGroupPolicy == nil || trainJob == nil {
		return nil
	}
	if info.PodLabels == nil {
		info.PodLabels = make(map[string]string, 1)
	}
	info.PodLabels[schedulerpluginsv1alpha1.PodGroupLabel] = trainJob.Name
	return nil
}

func (c *CoScheduling) Build(ctx context.Context, info *runtime.Info, trainJob *kubeflowv2.TrainJob) (client.Object, error) {
	if info == nil || info.PodGroupPolicy == nil || info.PodGroupPolicy.Coscheduling == nil || trainJob == nil {
		return nil, nil
	}

	var totalMembers int32
	totalResources := make(corev1.ResourceList)
	for _, resourceRequests := range info.TotalRequests {
		totalMembers += resourceRequests.Replicas
		for resName, quantity := range resourceRequests.PodRequests {
			quantity.Mul(int64(resourceRequests.Replicas))
			current := totalResources[resName]
			current.Add(quantity)
			totalResources[resName] = current
		}
	}
	newPG := &schedulerpluginsv1alpha1.PodGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: schedulerpluginsv1alpha1.SchemeGroupVersion.String(),
			Kind:       "PodGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      trainJob.Name,
			Namespace: trainJob.Namespace,
		},
		Spec: schedulerpluginsv1alpha1.PodGroupSpec{
			ScheduleTimeoutSeconds: info.PodGroupPolicy.Coscheduling.ScheduleTimeoutSeconds,
			MinMember:              totalMembers,
			MinResources:           totalResources,
		},
	}
	if err := ctrlutil.SetControllerReference(trainJob, newPG, c.scheme); err != nil {
		return nil, err
	}
	oldPG := &schedulerpluginsv1alpha1.PodGroup{}
	if err := c.client.Get(ctx, client.ObjectKeyFromObject(newPG), oldPG); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		oldPG = nil
	}
	if needsCreateOrUpdate(oldPG, newPG, ptr.Deref(trainJob.Spec.Suspend, false)) {
		return newPG, nil
	}
	return nil, nil
}

func needsCreateOrUpdate(old, new *schedulerpluginsv1alpha1.PodGroup, suspended bool) bool {
	return old == nil ||
		suspended && (!equality.Semantic.DeepEqual(old.Spec, new.Spec) || !maps.Equal(old.Labels, new.Labels) || !maps.Equal(old.Annotations, new.Annotations))
}

type PodGroupRuntimeClassHandler struct {
	client client.Client
}

var _ handler.EventHandler = (*PodGroupRuntimeClassHandler)(nil)

func (h *PodGroupRuntimeClassHandler) Create(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	containerRuntimeClass, ok := e.Object.(*nodev1.RuntimeClass)
	if !ok {
		return
	}
	log := ctrl.LoggerFrom(ctx).WithValues("runtimeClass", klog.KObj(containerRuntimeClass))
	if err := h.queueSuspendedTrainJob(ctx, containerRuntimeClass, q); err != nil {
		log.Error(err, "could not queue suspended TrainJob to reconcile queue")
	}
}

func (h *PodGroupRuntimeClassHandler) Update(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	_, ok := e.ObjectOld.(*nodev1.RuntimeClass)
	if !ok {
		return
	}
	newContainerRuntimeClass, ok := e.ObjectNew.(*nodev1.RuntimeClass)
	if !ok {
		return
	}
	log := ctrl.LoggerFrom(ctx).WithValues("runtimeClass", klog.KObj(newContainerRuntimeClass))
	if err := h.queueSuspendedTrainJob(ctx, newContainerRuntimeClass, q); err != nil {
		log.Error(err, "could not queue suspended TrainJob to reconcile queue")
	}
}

func (h *PodGroupRuntimeClassHandler) Delete(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	containerRuntimeClass, ok := e.Object.(*nodev1.RuntimeClass)
	if !ok {
		return
	}
	log := ctrl.LoggerFrom(ctx).WithValues("runtimeClass", klog.KObj(containerRuntimeClass))
	if err := h.queueSuspendedTrainJob(ctx, containerRuntimeClass, q); err != nil {
		log.Error(err, "could not queue suspended TrainJob to reconcile queue")
	}
}

func (h *PodGroupRuntimeClassHandler) Generic(context.Context, event.GenericEvent, workqueue.RateLimitingInterface) {
}

func (h *PodGroupRuntimeClassHandler) queueSuspendedTrainJob(ctx context.Context, runtimeClass *nodev1.RuntimeClass, q workqueue.RateLimitingInterface) error {
	var trainingRuntimes kubeflowv2.TrainingRuntimeList
	if err := h.client.List(ctx, &trainingRuntimes, client.MatchingFields{TrainingRuntimeContainerRuntimeClassKey: runtimeClass.Name}); err != nil {
		return err
	}
	var clusterTrainingRuntimes kubeflowv2.ClusterTrainingRuntimeList
	if err := h.client.List(ctx, &clusterTrainingRuntimes, client.MatchingFields{ClusterTrainingRuntimeContainerRuntimeClassKey: runtimeClass.Name}); err != nil {
		return err
	}

	var runtimeNames []string
	for _, trainingRuntime := range trainingRuntimes.Items {
		runtimeNames = append(runtimeNames, trainingRuntime.Name)
	}
	for _, clusterTrainingRuntime := range clusterTrainingRuntimes.Items {
		runtimeNames = append(runtimeNames, clusterTrainingRuntime.Name)
	}
	for _, runtimeName := range runtimeNames {
		var trainJobs kubeflowv2.TrainJobList
		if err := h.client.List(ctx, &trainJobs, client.MatchingFields{runtimeindexer.TrainJobTrainingRuntimeRefKey: runtimeName}); err != nil {
			return err
		}
		for _, trainJob := range trainJobs.Items {
			if ptr.Deref(trainJob.Spec.Suspend, false) {
				q.Add(client.ObjectKeyFromObject(&trainJob))
			}
		}
	}
	return nil
}

type PodGroupLimitRangeHandler struct {
	client client.Client
}

var _ handler.EventHandler = (*PodGroupLimitRangeHandler)(nil)

func (h *PodGroupLimitRangeHandler) Create(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	limitRange, ok := e.Object.(*corev1.LimitRange)
	if !ok {
		return
	}
	log := ctrl.LoggerFrom(ctx).WithValues("limitRange", klog.KObj(limitRange))
	if err := h.queueSuspendedTrainJob(ctx, limitRange.Namespace, q); err != nil {
		log.Error(err, "could not queue suspended TrainJob to reconcile queue")
	}
}

func (h *PodGroupLimitRangeHandler) Update(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	_, ok := e.ObjectOld.(*corev1.LimitRange)
	if !ok {
		return
	}
	newLimitRange, ok := e.ObjectNew.(*corev1.LimitRange)
	if !ok {
		return
	}
	log := ctrl.LoggerFrom(ctx).WithValues("limitRange", klog.KObj(newLimitRange))
	if err := h.queueSuspendedTrainJob(ctx, newLimitRange.Namespace, q); err != nil {
		log.Error(err, "could not queue suspended TrainJob to reconcile queue")
	}
}

func (h *PodGroupLimitRangeHandler) Delete(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	limitRange, ok := e.Object.(*corev1.LimitRange)
	if !ok {
		return
	}
	log := ctrl.LoggerFrom(ctx).WithValues("limitRange", klog.KObj(limitRange))
	if err := h.queueSuspendedTrainJob(ctx, limitRange.Namespace, q); err != nil {
		log.Error(err, "could not queue suspended TrainJob to reconcile queue")
	}
}

func (h *PodGroupLimitRangeHandler) Generic(context.Context, event.GenericEvent, workqueue.RateLimitingInterface) {
}

func (h *PodGroupLimitRangeHandler) queueSuspendedTrainJob(ctx context.Context, ns string, q workqueue.RateLimitingInterface) error {
	var trainJobs kubeflowv2.TrainJobList
	if err := h.client.List(ctx, &trainJobs, client.InNamespace(ns)); err != nil {
		return err
	}
	for _, trainJob := range trainJobs.Items {
		if ptr.Deref(trainJob.Spec.Suspend, false) {
			q.Add(client.ObjectKeyFromObject(&trainJob))
		}
	}
	return nil
}

func (c *CoScheduling) ReconcilerBuilders() []runtime.ReconcilerBuilder {
	if _, err := c.restMapper.RESTMapping(
		schema.GroupKind{Group: schedulerpluginsv1alpha1.SchemeGroupVersion.Group, Kind: "PodGroup"},
		schedulerpluginsv1alpha1.SchemeGroupVersion.Version,
	); err != nil {
		return nil
	}
	return []runtime.ReconcilerBuilder{
		func(b *builder.Builder, c client.Client) *builder.Builder {
			return b.Owns(&schedulerpluginsv1alpha1.PodGroup{})
		},
		func(b *builder.Builder, c client.Client) *builder.Builder {
			return b.Watches(&corev1.LimitRange{}, &PodGroupLimitRangeHandler{
				client: c,
			})
		},
		func(b *builder.Builder, c client.Client) *builder.Builder {
			return b.Watches(&nodev1.RuntimeClass{}, &PodGroupRuntimeClassHandler{
				client: c,
			})
		},
	}
}
