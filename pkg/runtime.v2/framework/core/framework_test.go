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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
	fwkplugins "github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins/coscheduling"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins/jobset"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins/mpi"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins/plainml"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins/torch"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
)

// TODO: We should introduce mock plugins and use plugins in this framework testing.
// After we migrate the actual plugins to mock one for testing data,
// we can delegate the actual plugin testing to each plugin directories, and implement detailed unit testing.

func TestNew(t *testing.T) {
	cases := map[string]struct {
		registry                                                               fwkplugins.Registry
		emptyCoSchedulingIndexerTrainingRuntimeContainerRuntimeClassKey        bool
		emptyCoSchedulingIndexerClusterTrainingRuntimeContainerRuntimeClassKey bool
		wantFramework                                                          *Framework
		wantError                                                              error
	}{
		"positive case": {
			registry: fwkplugins.NewRegistry(),
			wantFramework: &Framework{
				registry: fwkplugins.NewRegistry(),
				plugins: map[string]framework.Plugin{
					coscheduling.Name: &coscheduling.CoScheduling{},
					mpi.Name:          &mpi.MPI{},
					plainml.Name:      &plainml.PlainML{},
					torch.Name:        &torch.Torch{},
					jobset.Name:       &jobset.JobSet{},
				},
				enforceMLPlugins: []framework.EnforceMLPolicyPlugin{
					&mpi.MPI{},
					&plainml.PlainML{},
					&torch.Torch{},
				},
				enforcePodGroupPolicyPlugins: []framework.EnforcePodGroupPolicyPlugin{
					&coscheduling.CoScheduling{},
				},
				customValidationPlugins: []framework.CustomValidationPlugin{
					&mpi.MPI{},
					&torch.Torch{},
					&jobset.JobSet{},
				},
				watchExtensionPlugins: []framework.WatchExtensionPlugin{
					&coscheduling.CoScheduling{},
					&jobset.JobSet{},
				},
				componentBuilderPlugins: []framework.ComponentBuilderPlugin{
					&coscheduling.CoScheduling{},
					&jobset.JobSet{},
				},
			},
		},
		"indexer key for trainingRuntime and runtimeClass is an empty": {
			registry: fwkplugins.Registry{
				coscheduling.Name: coscheduling.New,
			},
			emptyCoSchedulingIndexerTrainingRuntimeContainerRuntimeClassKey: true,
			wantError: coscheduling.ErrorCanNotSetupTrainingRuntimeRuntimeClassIndexer,
		},
		"indexer key for clusterTrainingRuntime and runtimeClass is an empty": {
			registry: fwkplugins.Registry{
				coscheduling.Name: coscheduling.New,
			},
			emptyCoSchedulingIndexerClusterTrainingRuntimeContainerRuntimeClassKey: true,
			wantError: coscheduling.ErrorCanNotSetupClusterTrainingRuntimeRuntimeClassIndexer,
		},
	}
	cmpOpts := []cmp.Option{
		cmp.AllowUnexported(Framework{}),
		cmpopts.IgnoreUnexported(coscheduling.CoScheduling{}, mpi.MPI{}, plainml.PlainML{}, torch.Torch{}, jobset.JobSet{}),
		cmpopts.IgnoreFields(coscheduling.CoScheduling{}, "client"),
		cmpopts.IgnoreFields(jobset.JobSet{}, "client"),
		cmpopts.IgnoreTypes(apiruntime.Scheme{}, meta.DefaultRESTMapper{}, fwkplugins.Registry{}),
		cmpopts.SortMaps(func(a, b string) bool { return a < b }),
		cmpopts.SortSlices(func(a, b framework.Plugin) bool { return a.Name() < b.Name() }),
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			if tc.emptyCoSchedulingIndexerTrainingRuntimeContainerRuntimeClassKey {
				originTrainingRuntimeRuntimeKey := coscheduling.TrainingRuntimeContainerRuntimeClassKey
				coscheduling.TrainingRuntimeContainerRuntimeClassKey = ""
				t.Cleanup(func() {
					coscheduling.TrainingRuntimeContainerRuntimeClassKey = originTrainingRuntimeRuntimeKey
				})
			}
			if tc.emptyCoSchedulingIndexerClusterTrainingRuntimeContainerRuntimeClassKey {
				originClusterTrainingRuntimeKey := coscheduling.ClusterTrainingRuntimeContainerRuntimeClassKey
				coscheduling.ClusterTrainingRuntimeContainerRuntimeClassKey = ""
				t.Cleanup(func() {
					coscheduling.ClusterTrainingRuntimeContainerRuntimeClassKey = originClusterTrainingRuntimeKey
				})
			}
			clientBuilder := testingutil.NewClientBuilder()
			fwk, err := New(ctx, clientBuilder.Build(), tc.registry, testingutil.AsIndex(clientBuilder))
			if diff := cmp.Diff(tc.wantError, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected errors (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantFramework, fwk, cmpOpts...); len(diff) != 0 {
				t.Errorf("Unexpected framework (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestRunEnforceMLPolicyPlugins(t *testing.T) {
	cases := map[string]struct {
		registry        fwkplugins.Registry
		runtimeInfo     *runtime.Info
		wantRuntimeInfo *runtime.Info
		wantError       error
	}{
		"plainml MLPolicy is applied to runtime.Info": {
			registry: fwkplugins.NewRegistry(),
			runtimeInfo: &runtime.Info{
				Policy: runtime.Policy{
					MLPolicy: &kubeflowv2.MLPolicy{
						NumNodes: ptr.To[int32](100),
					},
				},
				TotalRequests: map[string]runtime.TotalResourceRequest{
					"Coordinator": {Replicas: 1},
					"Worker":      {Replicas: 10},
				},
			},
			wantRuntimeInfo: &runtime.Info{
				Policy: runtime.Policy{
					MLPolicy: &kubeflowv2.MLPolicy{
						NumNodes: ptr.To[int32](100),
					},
				},
				TotalRequests: map[string]runtime.TotalResourceRequest{
					"Coordinator": {Replicas: 100},
					"Worker":      {Replicas: 100},
				},
			},
		},
		"registry is empty": {
			runtimeInfo: &runtime.Info{
				Policy: runtime.Policy{
					MLPolicy: &kubeflowv2.MLPolicy{
						NumNodes: ptr.To[int32](100),
					},
				},
				TotalRequests: map[string]runtime.TotalResourceRequest{
					"Coordinator": {Replicas: 1},
					"Worker":      {Replicas: 10},
				},
			},
			wantRuntimeInfo: &runtime.Info{
				Policy: runtime.Policy{
					MLPolicy: &kubeflowv2.MLPolicy{
						NumNodes: ptr.To[int32](100),
					},
				},
				TotalRequests: map[string]runtime.TotalResourceRequest{
					"Coordinator": {Replicas: 1},
					"Worker":      {Replicas: 10},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			clientBuilder := testingutil.NewClientBuilder()

			fwk, err := New(ctx, clientBuilder.Build(), tc.registry, testingutil.AsIndex(clientBuilder))
			if err != nil {
				t.Fatal(err)
			}
			err = fwk.RunEnforceMLPolicyPlugins(tc.runtimeInfo)
			if diff := cmp.Diff(tc.wantError, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got): %s", diff)
			}
			if diff := cmp.Diff(tc.wantRuntimeInfo, tc.runtimeInfo, cmpopts.EquateEmpty()); len(diff) != 0 {
				t.Errorf("Unexpected runtime.Info (-want,+got): %s", diff)
			}
		})
	}
}

func TestRunEnforcePodGroupPolicyPlugins(t *testing.T) {
	cases := map[string]struct {
		trainJob        *kubeflowv2.TrainJob
		registry        fwkplugins.Registry
		runtimeInfo     *runtime.Info
		wantRuntimeInfo *runtime.Info
		wantError       error
	}{
		"coscheduling plugin is applied to runtime.Info": {
			trainJob: &kubeflowv2.TrainJob{ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: metav1.NamespaceDefault}},
			registry: fwkplugins.NewRegistry(),
			runtimeInfo: &runtime.Info{
				PodLabels: make(map[string]string),
				Policy: runtime.Policy{
					PodGroupPolicy: &kubeflowv2.PodGroupPolicy{},
				},
			},
			wantRuntimeInfo: &runtime.Info{
				PodLabels: map[string]string{
					schedulerpluginsv1alpha1.PodGroupLabel: "test-job",
				},
				Policy: runtime.Policy{
					PodGroupPolicy: &kubeflowv2.PodGroupPolicy{},
				},
			},
		},
		"an empty registry": {
			trainJob: &kubeflowv2.TrainJob{ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: metav1.NamespaceDefault}},
			runtimeInfo: &runtime.Info{
				Policy: runtime.Policy{
					PodGroupPolicy: &kubeflowv2.PodGroupPolicy{},
				},
			},
			wantRuntimeInfo: &runtime.Info{
				Policy: runtime.Policy{
					PodGroupPolicy: &kubeflowv2.PodGroupPolicy{},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			clientBuilder := testingutil.NewClientBuilder()

			fwk, err := New(ctx, clientBuilder.Build(), tc.registry, testingutil.AsIndex(clientBuilder))
			if err != nil {
				t.Fatal(err)
			}
			err = fwk.RunEnforcePodGroupPolicyPlugins(tc.trainJob, tc.runtimeInfo)
			if diff := cmp.Diff(tc.wantError, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got): %s", diff)
			}
			if diff := cmp.Diff(tc.wantRuntimeInfo, tc.runtimeInfo); len(diff) != 0 {
				t.Errorf("Unexpected runtime.Info (-want,+got): %s", diff)
			}
		})
	}
}

func TestRunCustomValidationPlugins(t *testing.T) {
	cases := map[string]struct {
		registry     fwkplugins.Registry
		oldObj       *kubeflowv2.TrainJob
		newObj       *kubeflowv2.TrainJob
		wantWarnings admission.Warnings
		wantError    field.ErrorList
	}{
		// Need to implement more detail testing after we implement custom validator in any plugins.
		"there are not any custom validations": {
			registry: fwkplugins.NewRegistry(),
			oldObj:   testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Obj(),
			newObj:   testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Obj(),
		},
		"an empty registry": {
			oldObj: testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Obj(),
			newObj: testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Obj(),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			clientBuildr := testingutil.NewClientBuilder()

			fwk, err := New(ctx, clientBuildr.Build(), tc.registry, testingutil.AsIndex(clientBuildr))
			if err != nil {
				t.Fatal(err)
			}
			runtimeInfo := runtime.NewInfo(testingutil.MakeJobSetWrapper(metav1.NamespaceDefault, "test").Obj())
			warnings, errs := fwk.RunCustomValidationPlugins(tc.oldObj, tc.newObj, runtimeInfo)
			if diff := cmp.Diff(tc.wantWarnings, warnings, cmpopts.SortSlices(func(a, b string) bool { return a < b })); len(diff) != 0 {
				t.Errorf("Unexpected warninigs (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantError, errs, cmpopts.IgnoreFields(field.Error{}, "Detail", "BadValue")); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestRunComponentBuilderPlugins(t *testing.T) {
	jobSetBase := testingutil.MakeJobSetWrapper(metav1.NamespaceDefault, "test-job").
		ResourceRequests(0, corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		}).
		ResourceRequests(1, corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		})
	jobSetWithPropagatedTrainJobParams := jobSetBase.
		Clone().
		JobCompletionMode(batchv1.IndexedCompletion).
		ContainerImage(ptr.To("foo:bar")).
		ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind("TrainJob"), "test-job", "uid")

	cases := map[string]struct {
		runtimeInfo     *runtime.Info
		trainJob        *kubeflowv2.TrainJob
		registry        fwkplugins.Registry
		wantError       error
		wantRuntimeInfo *runtime.Info
		wantObjs        []client.Object
	}{
		"coscheduling and jobset are performed": {
			trainJob: testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test-job").
				UID("uid").
				Trainer(
					testingutil.MakeTrainJobTrainerWrapper().
						ContainerImage("foo:bar").
						Obj(),
				).
				Obj(),
			runtimeInfo: &runtime.Info{
				Obj: jobSetBase.
					Clone().
					Obj(),
				Policy: runtime.Policy{
					MLPolicy: &kubeflowv2.MLPolicy{
						NumNodes: ptr.To[int32](10),
					},
					PodGroupPolicy: &kubeflowv2.PodGroupPolicy{
						PodGroupPolicySource: kubeflowv2.PodGroupPolicySource{
							Coscheduling: &kubeflowv2.CoschedulingPodGroupPolicySource{
								ScheduleTimeoutSeconds: ptr.To[int32](300),
							},
						},
					},
				},
				TotalRequests: map[string]runtime.TotalResourceRequest{
					"Coordinator": {
						Replicas: 1,
						PodRequests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
					"Worker": {
						Replicas: 1,
						PodRequests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
			},
			registry: fwkplugins.NewRegistry(),
			wantObjs: []client.Object{
				testingutil.MakeSchedulerPluginsPodGroup(metav1.NamespaceDefault, "test-job").
					SchedulingTimeout(300).
					MinMember(20).
					MinResources(corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("30"),
						corev1.ResourceMemory: resource.MustParse("60Gi"),
					}).
					ControllerReference(kubeflowv2.SchemeGroupVersion.WithKind("TrainJob"), "test-job", "uid").
					Obj(),
				jobSetWithPropagatedTrainJobParams.
					Clone().
					Obj(),
			},
			wantRuntimeInfo: &runtime.Info{
				Obj: jobSetWithPropagatedTrainJobParams.
					Clone().
					Obj(),
				Policy: runtime.Policy{
					MLPolicy: &kubeflowv2.MLPolicy{
						NumNodes: ptr.To[int32](10),
					},
					PodGroupPolicy: &kubeflowv2.PodGroupPolicy{
						PodGroupPolicySource: kubeflowv2.PodGroupPolicySource{
							Coscheduling: &kubeflowv2.CoschedulingPodGroupPolicySource{
								ScheduleTimeoutSeconds: ptr.To[int32](300),
							},
						},
					},
				},
				TotalRequests: map[string]runtime.TotalResourceRequest{
					"Coordinator": {
						Replicas: 10,
						PodRequests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
					"Worker": {
						Replicas: 10,
						PodRequests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
			},
		},
		"an empty registry": {},
	}
	cmpOpts := []cmp.Option{
		cmpopts.SortSlices(func(a, b client.Object) bool {
			return a.GetObjectKind().GroupVersionKind().String() < b.GetObjectKind().GroupVersionKind().String()
		}),
		cmpopts.EquateEmpty(),
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			clientBuilder := testingutil.NewClientBuilder()

			fwk, err := New(ctx, clientBuilder.Build(), tc.registry, testingutil.AsIndex(clientBuilder))
			if err != nil {
				t.Fatal(err)
			}
			if err = fwk.RunEnforceMLPolicyPlugins(tc.runtimeInfo); err != nil {
				t.Fatal(err)
			}
			objs, err := fwk.RunComponentBuilderPlugins(ctx, tc.runtimeInfo, tc.trainJob)
			if diff := cmp.Diff(tc.wantError, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected errors (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantRuntimeInfo, tc.runtimeInfo); len(diff) != 0 {
				t.Errorf("Unexpected runtime.Info (-want,+got)\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantObjs, objs, cmpOpts...); len(diff) != 0 {
				t.Errorf("Unexpected objects (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestRunExtensionPlugins(t *testing.T) {
	cases := map[string]struct {
		registry    fwkplugins.Registry
		wantPlugins []framework.WatchExtensionPlugin
	}{
		"coscheding and jobset are performed": {
			registry: fwkplugins.NewRegistry(),
			wantPlugins: []framework.WatchExtensionPlugin{
				&coscheduling.CoScheduling{},
				&jobset.JobSet{},
			},
		},
		"an empty registry": {
			wantPlugins: nil,
		},
	}
	cmpOpts := []cmp.Option{
		cmpopts.SortSlices(func(a, b framework.Plugin) bool { return a.Name() < b.Name() }),
		cmpopts.IgnoreUnexported(coscheduling.CoScheduling{}, jobset.JobSet{}),
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			clientBuilder := testingutil.NewClientBuilder()

			fwk, err := New(ctx, clientBuilder.Build(), tc.registry, testingutil.AsIndex(clientBuilder))
			if err != nil {
				t.Fatal(err)
			}
			plugins := fwk.WatchExtensionPlugins()
			if diff := cmp.Diff(tc.wantPlugins, plugins, cmpOpts...); len(diff) != 0 {
				t.Errorf("Unexpected plugins (-want,+got):\n%s", diff)
			}
		})
	}
}
