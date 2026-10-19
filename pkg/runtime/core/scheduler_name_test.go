/*
Copyright The Kubeflow Authors.

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
	"encoding/json"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/ktesting"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	trainer "github.com/kubeflow/trainer/v2/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/v2/pkg/constants"
	testingutil "github.com/kubeflow/trainer/v2/pkg/util/testing"
)

func TestRuntimeSchedulerNamePatch(t *testing.T) {
	tests := map[string]struct {
		runtimeScheduler string
		patch            string
		wantScheduler    string
	}{
		"omitted scheduler inherits the runtime": {
			runtimeScheduler: "runtime-scheduler",
			patch:            `{}`,
			wantScheduler:    "runtime-scheduler",
		},
		"override node scheduler": {
			runtimeScheduler: "default-scheduler",
			patch:            `{"schedulerName":"volcano"}`,
			wantScheduler:    "volcano",
		},
		"override with a qualified scheduler name": {
			runtimeScheduler: "default-scheduler",
			patch:            `{"schedulerName":"scheduler.example.com"}`,
			wantScheduler:    "scheduler.example.com",
		},
		"override back to the default scheduler": {
			runtimeScheduler: "volcano",
			patch:            `{"schedulerName":"default-scheduler"}`,
			wantScheduler:    "default-scheduler",
		},
	}

	for _, kind := range []string{trainer.TrainingRuntimeKind, trainer.ClusterTrainingRuntimeKind} {
		for name, tc := range tests {
			t.Run(kind+"/"+name, func(t *testing.T) {
				_, ctx := ktesting.NewTestContext(t)
				runtimeSpec := testingutil.MakeTrainingRuntimeWrapper(metav1.NamespaceDefault, "test-runtime").Spec
				for i := range runtimeSpec.Template.Spec.ReplicatedJobs {
					job := &runtimeSpec.Template.Spec.ReplicatedJobs[i]
					job.Template.Spec.Template.Spec.SchedulerName = "default-scheduler"
					if job.Name == constants.Node {
						job.Template.Spec.Template.Spec.SchedulerName = tc.runtimeScheduler
					}
				}

				// Decode the wire representation to also exercise the public patch API.
				trainJob := testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test-job").
					UID("test-uid").
					RuntimeRef(trainer.GroupVersion.WithKind(kind), "test-runtime").Obj()
				patchJSON := fmt.Sprintf(`[{"manager":"test.io/scheduler","trainingRuntimeSpec":{"template":{"spec":{"replicatedJobs":[{"name":"node","template":{"spec":{"template":{"spec":%s}}}}]}}}}]`, tc.patch)
				if err := json.Unmarshal([]byte(patchJSON), &trainJob.Spec.RuntimePatches); err != nil {
					t.Fatal(err)
				}

				clientBuilder := testingutil.NewClientBuilder()
				if kind == trainer.TrainingRuntimeKind {
					clientBuilder.WithObjects(testingutil.MakeTrainingRuntimeWrapper(metav1.NamespaceDefault, "test-runtime").RuntimeSpec(runtimeSpec).Obj())
				} else {
					clientBuilder.WithObjects(testingutil.MakeClusterTrainingRuntimeWrapper("test-runtime").RuntimeSpec(runtimeSpec).Obj())
				}
				c := clientBuilder.Build()
				rt, err := NewTrainingRuntime(ctx, c, testingutil.AsIndex(clientBuilder), nil)
				if err != nil {
					t.Fatal(err)
				}
				if kind == trainer.ClusterTrainingRuntimeKind {
					rt, err = NewClusterTrainingRuntime(ctx, c, testingutil.AsIndex(clientBuilder), nil)
				}
				if err != nil {
					t.Fatal(err)
				}
				objs, err := rt.NewObjects(ctx, trainJob)
				if err != nil {
					t.Fatal(err)
				}
				objects, err := testingutil.ToObject(c.Scheme(), objs...)
				if err != nil {
					t.Fatal(err)
				}
				foundNode := false
				for _, object := range objects {
					jobSet, ok := object.(*jobsetv1alpha2.JobSet)
					if !ok {
						continue
					}
					for _, job := range jobSet.Spec.ReplicatedJobs {
						want := "default-scheduler"
						if job.Name == constants.Node {
							foundNode = true
							want = tc.wantScheduler
						}
						if got := job.Template.Spec.Template.Spec.SchedulerName; got != want {
							t.Errorf("job %s schedulerName = %q, want %q", job.Name, got, want)
						}
					}
				}
				if !foundNode {
					t.Fatal("generated JobSet does not contain the node job")
				}
				for _, job := range runtimeSpec.Template.Spec.ReplicatedJobs {
					if job.Name == constants.Node && job.Template.Spec.Template.Spec.SchedulerName != tc.runtimeScheduler {
						t.Error("runtime schedulerName was mutated")
					}
				}
			})
		}
	}
}
