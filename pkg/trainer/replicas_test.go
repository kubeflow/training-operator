package trainer

import (
  "testing"

  "github.com/golang/protobuf/proto"

  meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/client-go/kubernetes/fake"
  "k8s.io/client-go/pkg/api/v1"
  "reflect"
  "time"
  "fmt"
  "mlkube.io/pkg/spec"
  tfJobFake "mlkube.io/pkg/util/k8sutil/fake"
  "sync"
)

func TestTFReplicaSet(t *testing.T) {
  clientSet := fake.NewSimpleClientset()

  jobSpec := &spec.TfJob {
    Spec: spec.TfJobSpec {
      RuntimeId: "some-runtime",
      ReplicaSpecs: []*spec.TfReplicaSpec{
        {
          Replicas: proto.Int32(2),
          TfPort: proto.Int32(10),
          Template: &v1.PodTemplateSpec{},
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

  if err := replica.Create(); err != nil {
    t.Fatalf("replica.Create() error; %v", err)
  }

  for index := 0; index < 2; index ++ {
    // Expected labels
    expectedLabels := map[string]string{
      "cloud_ml":   "",
      "task_index": fmt.Sprintf("%v", index),
      "job_type":   "PS",
      "runtime_id": "some-runtime",
    }

    // Check that a service was created.
    sList, err := clientSet.CoreV1().Services(NAMESPACE).List(meta_v1.ListOptions{})
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
    l, err := clientSet.BatchV1().Jobs(NAMESPACE).List(meta_v1.ListOptions{})
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
      PodList: v1.PodList {
        Items: []v1.Pod {
          {
            Status: v1.PodStatus {
              ContainerStatuses: []v1.ContainerStatus {
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
      Name: "master",
      Expected: spec.ReplicaStateRunning,
    },
    {
      PodList: v1.PodList {
        Items: []v1.Pod {
          {
            Status: v1.PodStatus {
              ContainerStatuses: []v1.ContainerStatus {
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
      Name: "master",
      Expected: spec.ReplicaStateSucceeded,
    },
    {
      // Multiple containers; make sure we match by name.
      PodList: v1.PodList {
        Items: []v1.Pod {
          {
            Status: v1.PodStatus {
              ContainerStatuses: []v1.ContainerStatus {
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
      Name: "master",
      Expected: spec.ReplicaStateSucceeded,
    },
    {
      // Container failed with permanent error and then got restarted.
      PodList: v1.PodList {
        Items: []v1.Pod {
          {
            Status: v1.PodStatus {
              ContainerStatuses: []v1.ContainerStatus {
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
      Name: "master",
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
                      Message:   "some reason",
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
