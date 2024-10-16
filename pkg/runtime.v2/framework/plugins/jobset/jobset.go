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

package jobset

import (
	"context"
	"fmt"
	"maps"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
)

type JobSet struct {
	client     client.Client
	restMapper meta.RESTMapper
	scheme     *apiruntime.Scheme
	logger     logr.Logger
}

var _ framework.WatchExtensionPlugin = (*JobSet)(nil)
var _ framework.ComponentBuilderPlugin = (*JobSet)(nil)

const Name = "JobSet"

func New(ctx context.Context, c client.Client, _ client.FieldIndexer) (framework.Plugin, error) {
	return &JobSet{
		client:     c,
		restMapper: c.RESTMapper(),
		scheme:     c.Scheme(),
		logger:     ctrl.LoggerFrom(ctx).WithValues("pluginName", "JobSet"),
	}, nil
}

func (j *JobSet) Name() string {
	return Name
}

func (j *JobSet) Build(ctx context.Context, info *runtime.Info, trainJob *kubeflowv2.TrainJob) (client.Object, error) {
	if info == nil || info.Obj == nil || trainJob == nil {
		return nil, fmt.Errorf("runtime info or object is missing")
	}
	raw, ok := info.Obj.(*jobsetv1alpha2.JobSet)
	if !ok {
		return nil, nil
	}
	jobSetBuilder := NewBuilder(client.ObjectKeyFromObject(trainJob), kubeflowv2.JobSetTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      info.Labels,
			Annotations: info.Annotations,
		},
		Spec: raw.Spec,
	})
	// TODO (tenzen-y): We should support all field propagation in builder.
	jobSet := jobSetBuilder.
		ContainerImage(trainJob.Spec.Trainer.Image).
		JobCompletionMode(batchv1.IndexedCompletion).
		PodLabels(info.PodLabels).
		Build()
	if err := ctrlutil.SetControllerReference(trainJob, jobSet, j.scheme); err != nil {
		return nil, err
	}
	oldJobSet := &jobsetv1alpha2.JobSet{}
	if err := j.client.Get(ctx, client.ObjectKeyFromObject(jobSet), oldJobSet); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		oldJobSet = nil
	}
	if err := info.Update(jobSet); err != nil {
		return nil, err
	}
	if needsCreateOrUpdate(oldJobSet, jobSet, ptr.Deref(trainJob.Spec.Suspend, false)) {
		return jobSet, nil
	}
	return nil, nil
}

func needsCreateOrUpdate(old, new *jobsetv1alpha2.JobSet, suspended bool) bool {
	return old == nil ||
		suspended && (!equality.Semantic.DeepEqual(old.Spec, new.Spec) || !maps.Equal(old.Labels, new.Labels) || !maps.Equal(old.Annotations, new.Annotations))
}

func (j *JobSet) ReconcilerBuilders() []runtime.ReconcilerBuilder {
	if _, err := j.restMapper.RESTMapping(
		schema.GroupKind{Group: jobsetv1alpha2.GroupVersion.Group, Kind: "JobSet"},
		jobsetv1alpha2.SchemeGroupVersion.Version,
	); err != nil {
		// TODO (tenzen-y): After we provide the Configuration API, we should return errors based on the enabled plugins.
		j.logger.Error(err, "JobSet CRDs must be installed in advance")
	}
	return []runtime.ReconcilerBuilder{
		func(b *builder.Builder, c client.Client) *builder.Builder {
			return b.Owns(&jobsetv1alpha2.JobSet{})
		},
	}
}
