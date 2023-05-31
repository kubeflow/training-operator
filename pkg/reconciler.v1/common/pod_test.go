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

package common_test

import (
	"testing"

	commonv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	testjobv1 "github.com/kubeflow/training-operator/test_job/apis/test_job/v1"
	"github.com/kubeflow/training-operator/test_job/reconciler.v1/test_job"
	testutilv1 "github.com/kubeflow/training-operator/test_job/test_util/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenPodName(t *testing.T) {
	type tc struct {
		testJob      *testjobv1.TestJob
		testRType    string
		testIndex    string
		expectedName string
	}
	testCase := []tc{
		func() tc {
			tj := testutilv1.NewTestJob(1)
			tj.SetName("hello-world")
			return tc{
				testJob:      tj,
				testRType:    string(testjobv1.TestReplicaTypeWorker),
				testIndex:    "1",
				expectedName: "hello-world-worker-1",
			}
		}(),
	}

	testReconciler := test_job.NewTestReconciler()

	for _, c := range testCase {
		na := testReconciler.GenPodName(c.testJob.GetName(), c.testRType, c.testIndex)
		if na != c.expectedName {
			t.Errorf("Expected %s, got %s", c.expectedName, na)
		}
	}
}

func PodInSlice(pod *corev1.Pod, pods []*corev1.Pod) bool {
	for _, p := range pods {
		if p.GetNamespace() == pod.GetNamespace() && p.GetName() == pod.GetName() {
			return true
		}
	}
	return false
}

func TestFilterPodsForReplicaType(t *testing.T) {
	type tc struct {
		testPods     []*corev1.Pod
		testRType    string
		expectedPods []*corev1.Pod
	}
	testCase := []tc{
		func() tc {
			tj := testutilv1.NewTestJob(3)
			tj.SetName("hello-world")

			pod0 := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod0",
					Namespace: "default",
					Labels: map[string]string{
						commonv1.ReplicaTypeLabel: string(testjobv1.TestReplicaTypeMaster),
					},
				},
				Spec:   corev1.PodSpec{},
				Status: corev1.PodStatus{},
			}

			pod1 := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
					Labels: map[string]string{
						commonv1.ReplicaTypeLabel: string(testjobv1.TestReplicaTypeWorker),
					},
				},
				Spec:   corev1.PodSpec{},
				Status: corev1.PodStatus{},
			}

			pod2 := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "default",
					Labels: map[string]string{
						commonv1.ReplicaTypeLabel: string(testjobv1.TestReplicaTypeWorker),
					},
				},
				Spec:   corev1.PodSpec{},
				Status: corev1.PodStatus{},
			}

			allPods := []*corev1.Pod{pod0, pod1, pod2}
			filteredPods := []*corev1.Pod{pod1, pod2}

			return tc{
				testPods:     allPods,
				testRType:    string(testjobv1.TestReplicaTypeWorker),
				expectedPods: filteredPods,
			}
		}(),
	}

	testReconciler := test_job.NewTestReconciler()

	for _, c := range testCase {
		filtered, err := testReconciler.FilterPodsForReplicaType(c.testPods, c.testRType)
		if err != nil {
			t.Errorf("FilterPodsForReplicaType returns error %v", err)
		}
		for _, ep := range c.expectedPods {
			if !PodInSlice(ep, filtered) {
				t.Errorf("Cannot found expected pod %s", ep.GetName())
			}
		}

	}
}
