package trainer

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/tensorflow/k8s/pkg/spec"
	tfJobFake "github.com/tensorflow/k8s/pkg/util/k8sutil/fake"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"encoding/json"
)

func TestTFReplicaSet(t *testing.T) {
	clientSet := fake.NewSimpleClientset()

	jobSpec := &spec.TfJob{
		Spec: spec.TfJobSpec{
			RuntimeId: "some-runtime",
			ReplicaSpecs: []*spec.TfReplicaSpec{
				{
					Replicas:      proto.Int32(2),
					TfPort:        proto.Int32(10),
					Template:      &v1.PodTemplateSpec{
						Spec: v1.PodSpec {
							Containers: []v1.Container{
								{
									Name: "tensorflow",
								},
							},
						},
					},
					TfReplicaType: spec.PS,
				},
			},
		},
	}

	stopC := make(chan struct{})

	wg := &sync.WaitGroup{}
	job, err := initJob(clientSet, &tfJobFake.TfJobClientFake{}, jobSpec, stopC, wg)

	if err != nil {
		t.Fatalf("initJob failed: %v", err)
	}

	replica, err := NewTFReplicaSet(clientSet, *jobSpec.Spec.ReplicaSpecs[0], job)

	if err != nil {
		t.Fatalf("NewTFReplicaSet failed: %v", err)
	}

	if err := replica.Create(&spec.ControllerConfig{}); err != nil {
		t.Fatalf("replica.Create() error; %v", err)
	}

	for index := 0; index < 2; index++ {
		// Expected labels
		expectedLabels := map[string]string{
			"mlkube.io":  "",
			"task_index": fmt.Sprintf("%v", index),
			"job_type":   "PS",
			"runtime_id": "some-runtime",
		}

		// Check that a service was created.
		sList, err := clientSet.CoreV1().Services(replica.Job.job.Metadata.Namespace).List(meta_v1.ListOptions{})
		if err != nil {
			t.Fatalf("List services error; %v", err)
		}

		if len(sList.Items) != 2 {
			t.Fatalf("Expected 1 service got %v", len(sList.Items))
		}

		s := sList.Items[index]

		if !reflect.DeepEqual(expectedLabels, s.ObjectMeta.Labels) {
			t.Fatalf("Service Labels; Got %v Want: %v", s.ObjectMeta.Labels, expectedLabels)
		}

		name := fmt.Sprintf("ps-some-runtime-%v", index)
		if s.ObjectMeta.Name != name {
			t.Fatalf("Job.ObjectMeta.Name = %v; want %v", s.ObjectMeta.Name, name)
		}

		// Check that a job was created.
		l, err := clientSet.BatchV1().Jobs(replica.Job.job.Metadata.Namespace).List(meta_v1.ListOptions{})
		if err != nil {
			t.Fatalf("List jobs error; %v", err)
		}

		if len(l.Items) != 2 {
			t.Fatalf("Expected 1 job got %v", len(l.Items))
		}

		j := l.Items[index]

		if !reflect.DeepEqual(expectedLabels, j.ObjectMeta.Labels) {
			t.Fatalf("Job Labels; Got %v Want: %v", expectedLabels, j.ObjectMeta.Labels)
		}

		if j.ObjectMeta.Name != name {
			t.Fatalf("Job.ObjectMeta.Name = %v; want %v", j.ObjectMeta.Name, name)
		}

		if len(j.Spec.Template.Spec.Containers) != 1 {
			t.Fatalf("Expected 1 container got %v", len(j.Spec.Template.Spec.Containers))
		}

		c := j.Spec.Template.Spec.Containers[0]
		if len(c.Env) != 1 {
			t.Fatalf("Expected 1 environment variable got %v", len(c.Env))
		}

		actualTFConfig := &TfConfig{}
		if err := json.Unmarshal([]byte(c.Env[0].Value), actualTFConfig); err != nil {
			t.Fatalf("Could not unmarshal TfConfig %v", err)
		}

		expectedTfConfig := &TfConfig {
			Cluster: ClusterSpec{},
			Task: TaskSpec{
				Type:  "ps",
				Index: index,
			},
			Environment: "cloud",
		}

		if !reflect.DeepEqual(expectedTfConfig, actualTFConfig) {
			t.Fatalf("Got %v, Want %v", actualTFConfig, expectedTfConfig)
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
		Expected spec.ReplicaState
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
			Expected: spec.ReplicaStateRunning,
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
			Expected: spec.ReplicaStateSucceeded,
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
			Expected: spec.ReplicaStateSucceeded,
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
			Expected: spec.ReplicaStateFailed,
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
			Expected: spec.ReplicaStateFailed,
		},
	}

	for _, c := range cases {
		status := replicaStatusFromPodList(c.PodList, spec.ContainerName(c.Name))
		if status != c.Expected {
			t.Errorf("replicaStatusFromPodList(%+v, %v)=%v ; want %v", c.PodList, c.Name, status, c.Expected)
		}
	}
}

func TestTransformClusterSpecForDefaultPS(t *testing.T) {

	cs := ClusterSpec{
		"master": {"master-0:2222"},
		"worker": {"worker-0:2222", "worker-1:2222"},
		"ps":     {"localhost:2222", "ps-1:2222"},
	}
	expected := "master|master-0:2222,ps|localhost:2222;ps-1:2222,worker|worker-0:2222;worker-1:2222"

	tx := transformClusterSpecForDefaultPS(cs)

	if tx != expected {
		t.Errorf("transformClusterSpecForDefaultPS() expected: %v, received: %v", expected, tx)
	}
}
