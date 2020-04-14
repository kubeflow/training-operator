// Copyright 2018 The Kubeflow Authors
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

package v1

import (
	"time"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testjobv1 "github.com/kubeflow/common/test_job/apis/test_job/v1"
)

func NewTestJob(worker int) *testjobv1.TestJob {
	testJob := &testjobv1.TestJob{
		TypeMeta: metav1.TypeMeta{
			Kind: testjobv1.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestJobName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: testjobv1.TestJobSpec{
			TestReplicaSpecs: make(map[testjobv1.TestReplicaType]*apiv1.ReplicaSpec),
		},
	}

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &apiv1.ReplicaSpec{
			Replicas: &worker,
			Template: NewTestReplicaSpecTemplate(),
		}
		testJob.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker] = workerReplicaSpec
	}

	return testJob
}

func NewTestReplicaSpecTemplate() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  testjobv1.DefaultContainerName,
					Image: TestImageName,
					Args:  []string{"Fake", "Fake"},
					Ports: []v1.ContainerPort{
						v1.ContainerPort{
							Name:          testjobv1.DefaultPortName,
							ContainerPort: testjobv1.DefaultPort,
						},
					},
				},
			},
		},
	}
}

func SetTestJobCompletionTime(testJob *testjobv1.TestJob) {
	now := metav1.Time{Time: time.Now()}
	testJob.Status.CompletionTime = &now
}
