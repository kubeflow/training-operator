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

package trainer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	tfv1alpha1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha1"
	tfJobFake "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/fake"
	"github.com/kubeflow/tf-operator/pkg/util"
)

var (
	groupVersionKind = schema.GroupVersionKind{
		Group:   tfv1alpha1.GroupName,
		Version: tfv1alpha1.GroupVersion,
		Kind:    tfv1alpha1.TFJobResourceKind,
	}
)

func TestTFReplicaSet(t *testing.T) {
	clientSet := fake.NewSimpleClientset()

	testSchedulerName := "test-scheduler"

	jobSpec := &tfv1alpha1.TFJob{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "some-job",
			UID:  "some-uid",
		},
		Spec: tfv1alpha1.TFJobSpec{
			RuntimeId: "some-runtime",
			ReplicaSpecs: []*tfv1alpha1.TFReplicaSpec{
				{
					Replicas: proto.Int32(2),
					TFPort:   proto.Int32(10),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "tensorflow",
								},
							},
						},
					},
					TFReplicaType: tfv1alpha1.PS,
				},
			},
			SchedulerName: testSchedulerName,
		},
	}

	jobSpec.Spec.ReplicaSpecs[0].Template.Labels = map[string]string{"some-label": "some-value"}
	jobSpec.Spec.ReplicaSpecs[0].Template.Annotations = map[string]string{"some-anno": "some-value"}
	recorder := record.NewFakeRecorder(100)
	job, err := initJob(clientSet, &tfJobFake.Clientset{}, recorder, jobSpec)

	if err != nil {
		t.Fatalf("initJob failed: %v", err)
	}

	replica, err := NewTFReplicaSet(clientSet, recorder, *jobSpec.Spec.ReplicaSpecs[0], job)

	if err != nil {
		t.Fatalf("NewTFReplicaSet failed: %v", err)
	}

	if err := replica.SyncPods(); err != nil {
		t.Fatalf("replica.SyncPods() error; %v", err)
	}

	if err := replica.SyncServices(); err != nil {
		t.Fatalf("replica.SyncServices() error; %v", err)
	}

	trueVal := true
	expectedOwnerReference := meta_v1.OwnerReference{
		APIVersion:         groupVersionKind.GroupVersion().String(),
		Kind:               groupVersionKind.Kind,
		Name:               "some-job",
		UID:                "some-uid",
		Controller:         &trueVal,
		BlockOwnerDeletion: &trueVal,
	}

	for index := 0; index < 2; index++ {
		// Expected labels
		expectedServiceLabels := map[string]string{
			"kubeflow.org": "",
			"task_index":   fmt.Sprintf("%v", index),
			"job_type":     "PS",
			"runtime_id":   "some-runtime",
			"tf_job_name":  "some-job",
		}
		expectedPodLabels := map[string]string{
			"kubeflow.org": "",
			"task_index":   fmt.Sprintf("%v", index),
			"job_type":     "PS",
			"runtime_id":   "some-runtime",
			"tf_job_name":  "some-job",
			"some-label":   "some-value",
		}

		expectedPodAnnotations := map[string]string{
			"some-anno": "some-value",
		}

		// Check that a service was created.
		sList, err := clientSet.CoreV1().Services(replica.Job.job.ObjectMeta.Namespace).List(meta_v1.ListOptions{})
		if err != nil {
			t.Fatalf("List services error; %v", err)
		}

		if len(sList.Items) != 2 {
			t.Fatalf("Expected 2 services got %v", len(sList.Items))
		}

		s := sList.Items[index]

		if !reflect.DeepEqual(expectedServiceLabels, s.ObjectMeta.Labels) {
			t.Fatalf("Service Labels; Got %v Want: %v", s.ObjectMeta.Labels, expectedServiceLabels)
		}

		name := fmt.Sprintf("some-job-ps-some-runtime-%v", index)
		if s.ObjectMeta.Name != name {
			t.Fatalf("Job.ObjectMeta.Name = %v; want %v", s.ObjectMeta.Name, name)
		}

		if len(s.ObjectMeta.OwnerReferences) != 1 {
			t.Fatalf("Expected 1 owner reference got %v", len(s.ObjectMeta.OwnerReferences))
		}

		if !reflect.DeepEqual(s.ObjectMeta.OwnerReferences[0], expectedOwnerReference) {
			t.Fatalf("Service.Metadata.OwnerReferences; Got %v; want %v", util.Pformat(s.ObjectMeta.OwnerReferences[0]), util.Pformat(expectedOwnerReference))
		}

		// Check that a pod was created.
		l, err := clientSet.CoreV1().Pods(replica.Job.job.ObjectMeta.Namespace).List(meta_v1.ListOptions{})
		if err != nil {
			t.Fatalf("List pods error; %v", err)
		}

		if len(l.Items) != 2 {
			t.Fatalf("Expected 1 pod got %v", len(l.Items))
		}

		p := l.Items[index]

		if !reflect.DeepEqual(expectedPodLabels, p.ObjectMeta.Labels) {
			t.Fatalf("Pod Labels; Got %v Want: %v", p.ObjectMeta.Labels, expectedPodLabels)
		}

		if !reflect.DeepEqual(expectedPodAnnotations, p.ObjectMeta.Annotations) {
			t.Fatalf("Pod Annotations; Got %v Want: %v", p.ObjectMeta.Annotations, expectedPodAnnotations)
		}

		if len(p.Spec.Containers) != 1 {
			t.Fatalf("Expected 1 container got %v", len(p.Spec.Containers))
		}

		if len(p.ObjectMeta.OwnerReferences) != 1 {
			t.Fatalf("Expected 1 owner reference got %v", len(p.ObjectMeta.OwnerReferences))
		}

		if !reflect.DeepEqual(p.ObjectMeta.OwnerReferences[0], expectedOwnerReference) {
			t.Fatalf("Pod.Metadata.OwnerReferences; Got %v; want %v", util.Pformat(p.ObjectMeta.OwnerReferences[0]), util.Pformat(expectedOwnerReference))
		}

		c := p.Spec.Containers[0]
		if len(c.Env) != 1 {
			t.Fatalf("Expected 1 environment variable got %v", len(c.Env))
		}

		if strings.Compare(p.Spec.SchedulerName, testSchedulerName) != 0 {
			t.Fatalf("p.Spec.Template.Spec.SchedulerName; Got %v; want %v", p.Spec.SchedulerName, testSchedulerName)
		}

		actualTFConfig := &TFConfig{}
		if err := json.Unmarshal([]byte(c.Env[0].Value), actualTFConfig); err != nil {
			t.Fatalf("Could not unmarshal TFConfig %v", err)
		}

		expectedTFConfig := &TFConfig{
			Cluster: ClusterSpec{},
			Task: TaskSpec{
				Type:  "ps",
				Index: index,
			},
			Environment: "cloud",
		}

		if !reflect.DeepEqual(expectedTFConfig, actualTFConfig) {
			t.Fatalf("Got %v, Want %v", actualTFConfig, expectedTFConfig)
		}
	}
	// Delete the job.
	// N.B it doesn't look like the Fake clientset is sophisticated enough to delete jobs in response to a
	// DeleteCollection request (deleting individual jobs does appear to work with the Fake). So if we were to list
	// the jobs after calling Delete we'd still see the job. So we will rely on E2E tests to verify Delete works
	// correctly.
	if err := replica.Delete(); err != nil {
		t.Fatalf("replica.Delete() error; %v", err)
	}
}

func TestTFReplicaSetStatusFromPodList(t *testing.T) {
	type TestCase struct {
		PodList  v1.PodList
		Name     string
		Expected tfv1alpha1.ReplicaState
	}

	cases := []TestCase{
		{
			PodList: v1.PodList{
				Items: []v1.Pod{
					{
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name: "master",
									State: v1.ContainerState{
										Running: &v1.ContainerStateRunning{},
									},
								},
							},
						},
					},
				},
			},
			Name:     "master",
			Expected: tfv1alpha1.ReplicaStateRunning,
		},
		{
			PodList: v1.PodList{
				Items: []v1.Pod{
					{
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name: "master",
									State: v1.ContainerState{
										Terminated: &v1.ContainerStateTerminated{
											ExitCode: 0,
										},
									},
								},
							},
						},
					},
				},
			},
			Name:     "master",
			Expected: tfv1alpha1.ReplicaStateSucceeded,
		},
		{
			// Multiple containers; make sure we match by name.
			PodList: v1.PodList{
				Items: []v1.Pod{
					{
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name: "other",
									State: v1.ContainerState{
										Running: &v1.ContainerStateRunning{},
									},
								},
								{
									Name: "master",
									State: v1.ContainerState{
										Terminated: &v1.ContainerStateTerminated{
											ExitCode: 0,
										},
									},
								},
							},
						},
					},
				},
			},
			Name:     "master",
			Expected: tfv1alpha1.ReplicaStateSucceeded,
		},
		{
			// Container failed with permanent error and then got restarted.
			PodList: v1.PodList{
				Items: []v1.Pod{
					{
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name: "master",
									State: v1.ContainerState{
										Running: &v1.ContainerStateRunning{},
									},
									LastTerminationState: v1.ContainerState{
										Terminated: &v1.ContainerStateTerminated{
											ExitCode: 100,
											Message:  "some reason",
										},
									},
								},
							},
						},
					},
				},
			},
			Name:     "master",
			Expected: tfv1alpha1.ReplicaStateFailed,
		},
		{
			// Multiple Pods; check we get the most recent.
			PodList: v1.PodList{
				Items: []v1.Pod{
					{
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name: "master",
									State: v1.ContainerState{
										Running: &v1.ContainerStateRunning{},
									},
								},
							},
							StartTime: &meta_v1.Time{
								Time: time.Date(2017, 0, 0, 0, 0, 0, 0, time.UTC),
							},
						},
					},
					{
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name: "master",
									State: v1.ContainerState{
										Terminated: &v1.ContainerStateTerminated{
											ExitCode: 100,
											Message:  "some reason",
										},
									},
								},
							},
							StartTime: &meta_v1.Time{
								Time: time.Date(2018, 0, 0, 0, 0, 0, 0, time.UTC),
							},
						},
					},
				},
			},
			Name:     "master",
			Expected: tfv1alpha1.ReplicaStateFailed,
		},
	}

	for _, c := range cases {
		status := replicaStatusFromPodList(c.PodList, c.Name)
		if status != c.Expected {
			t.Errorf("replicaStatusFromPodList(%+v, %v)=%v ; want %v", c.PodList, c.Name, status, c.Expected)
		}
	}
}
