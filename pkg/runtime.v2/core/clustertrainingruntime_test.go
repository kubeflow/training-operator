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

package core

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
)

func TestClusterTrainingRuntimeNewObjects(t *testing.T) {
	baseRuntime := testingutil.MakeClusterTrainingRuntimeWrapper("test-runtime").
		Clone()

	cases := map[string]struct {
		trainJob               *kubeflowv2.TrainJob
		clusterTrainingRuntime *kubeflowv2.ClusterTrainingRuntime
		wantObjs               []client.Object
		wantError              error
	}{
		"succeeded to build JobSet and PodGroup": {
			trainJob: testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test-job").
				Suspend(true).
				UID("uid").
				RuntimeRef(kubeflowv2.SchemeGroupVersion.WithKind(kubeflowv2.ClusterTrainingRuntimeKind), "test-runtime").
				Trainer(
					testingutil.MakeTrainJobTrainerWrapper().
						ContainerImage("test:trainjob").
						Obj(),
				).
				Obj(),
			clusterTrainingRuntime: baseRuntime.RuntimeSpec(
				testingutil.MakeTrainingRuntimeSpecWrapper(baseRuntime.Spec).
					ContainerImage("test:runtime").
					PodGroupPolicyCoschedulingSchedulingTimeout(120).
					MLPolicyNumNodes(20).
					ResourceRequests(0, corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					}).
					ResourceRequests(1, corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("2"),
					}).
					Obj(),
			).Obj(),
			wantObjs: []client.Object{
				testingutil.MakeJobSetWrapper(metav1.NamespaceDefault, "test-job").
					Suspend(true).
					PodLabel(schedulerpluginsv1alpha1.PodGroupLabel, "test-job").
					ContainerImage(ptr.To("test:trainjob")).
					JobCompletionMode(batchv1.IndexedCompletion).
					ResourceRequests(0, corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					}).
					ResourceRequests(1, corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("2"),
					}).
					ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind("TrainJob"), "test-job", "uid").
					Obj(),
				testingutil.MakeSchedulerPluginsPodGroup(metav1.NamespaceDefault, "test-job").
					ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind("TrainJob"), "test-job", "uid").
					MinMember(40).
					SchedulingTimeout(120).
					MinResources(corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("60"),
					}).
					Obj(),
			},
		},
		"missing trainingRuntime resource": {
			trainJob: testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test-job").
				UID("uid").
				RuntimeRef(kubeflowv2.SchemeGroupVersion.WithKind(kubeflowv2.ClusterTrainingRuntimeKind), "test-runtime").
				Trainer(
					testingutil.MakeTrainJobTrainerWrapper().
						ContainerImage("test:trainjob").
						Obj(),
				).
				Obj(),
			wantError: errorNotFoundSpecifiedClusterTrainingRuntime,
		},
	}
	cmpOpts := []cmp.Option{
		cmpopts.SortSlices(func(a, b client.Object) bool {
			return a.GetObjectKind().GroupVersionKind().String() < b.GetObjectKind().GroupVersionKind().String()
		}),
		cmpopts.EquateEmpty(),
		cmpopts.SortMaps(func(a, b string) bool { return a < b }),
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			clientBuilder := testingutil.NewClientBuilder()
			if tc.clusterTrainingRuntime != nil {
				clientBuilder.WithObjects(tc.clusterTrainingRuntime)
			}

			trainingRuntime, err := NewTrainingRuntime(ctx, clientBuilder.Build(), testingutil.AsIndex(clientBuilder))
			if err != nil {
				t.Fatal(err)
			}
			var ok bool
			trainingRuntimeFactory, ok = trainingRuntime.(*TrainingRuntime)
			if !ok {
				t.Fatal("Failed type assertion from Runtime interface to TrainingRuntime")
			}

			clTrainingRuntime, err := NewClusterTrainingRuntime(ctx, clientBuilder.Build(), testingutil.AsIndex(clientBuilder))
			if err != nil {
				t.Fatal(err)
			}
			objs, err := clTrainingRuntime.NewObjects(ctx, tc.trainJob)
			if diff := cmp.Diff(tc.wantError, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantObjs, objs, cmpOpts...); len(diff) != 0 {
				t.Errorf("Unexpected objects (-want,+got):\n%s", diff)
			}
		})
	}
}
