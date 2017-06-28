package trainer

import (
	"testing"

	"github.com/golang/protobuf/proto"

	tpb "mlkube.io/pkg/trainer/protos"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"reflect"
	"time"
	"fmt"
)

func TestTFReplicaSet(t *testing.T) {
	clientSet := fake.NewSimpleClientset()

	code := &tpb.TrainingCode{
		PythonModule: proto.String("main_module"),
		PackageUris:  []string{"pkg1"},
	}

	cluster := TFCluster{
		RuntimeId: "some-runtime",
	}

	process := tpb.TensorFlowProcess{
		Image:       proto.String("image"),
		NumReplicas: proto.Int32(2),
		Type:        tpb.TensorFlowProcess_PS.Enum(),
	}

	replica, err := NewTFReplicaSet(clientSet, *code, process, cluster)

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
			t.Fatalf("Service Labels; Got %v Want: %v", expectedLabels, s.ObjectMeta.Labels)
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

func TestTFReplicaSet_CreatePublicServices(t *testing.T) {
	clientSet := fake.NewSimpleClientset(&v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "ps-some-runtime-0",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort {
				{
					Name: "tf-port",
					Port: 2222,
				},
			},
		},
		Status: v1.ServiceStatus {
			LoadBalancer: v1.LoadBalancerStatus {
				Ingress: []v1.LoadBalancerIngress {
					{
						IP: "123.123.123.123",
					},
				},
			},
		},
	})

	code := &tpb.TrainingCode{
		PythonModule: proto.String("main_module"),
		PackageUris:  []string{"pkg1"},
	}

	cluster := TFCluster{
		RuntimeId: "some-runtime",
	}

	process := tpb.TensorFlowProcess{
		Image:       proto.String("image"),
		NumReplicas: proto.Int32(1),
		Type:        tpb.TensorFlowProcess_PS.Enum(),
		AssignExternalIp: proto.Bool(true),
	}

	replica, err := NewTFReplicaSet(clientSet, *code, process, cluster)

	if err != nil {
		t.Fatalf("NewTFReplicaSet failed: %v", err)
	}

	otherClientSet := fake.NewSimpleClientset()
	if err := replica.CreatePublicServices(otherClientSet); err != nil {
		t.Fatalf("replica.CreatePublicServices() error; %v", err)
	}

	for index := 0; index < 1; index ++ {
		// Expected labels
		expectedLabels := map[string]string{
			"cloud_ml":   "",
			"task_index": fmt.Sprintf("%v", index),
			"job_type":   "PS",
			"runtime_id": "some-runtime",
		}

		// Check that a service was created.
		sList, err := otherClientSet.CoreV1().Services(NAMESPACE).List(meta_v1.ListOptions{})
		if err != nil {
			t.Fatalf("List services error; %v", err)
		}

		if len(sList.Items) != 1 {
			t.Fatalf("Expected 1 service got %v", len(sList.Items))
		}

		s := sList.Items[index]

		if !reflect.DeepEqual(expectedLabels, s.ObjectMeta.Labels) {
			t.Fatalf("Service Labels; Got %v Want: %v", expectedLabels, s.ObjectMeta.Labels)
		}

		name := fmt.Sprintf("ps-some-runtime-%v", index)
		if s.ObjectMeta.Name != name {
			t.Fatalf("Job.ObjectMeta.Name = %v; want %v", s.ObjectMeta.Name, name)
		}

		// Check that an endpoints object was created.

		endpoint, err := otherClientSet.Core().Endpoints(NAMESPACE).Get(name, meta_v1.GetOptions{})

		if err != nil {
			t.Fatalf("Endpoints(%v).Get(%v) error; %v", NAMESPACE, name, err)
		}

		if endpoint.Subsets[0].Addresses[0].IP != "123.123.123.123"  {
			t.Fatalf("endpoint.Subsets[0].Addresses[0].IP want 123.123.123.123 got %v", endpoint.Subsets[0].Addresses[0].IP)
		}
	}

	// Delete the services.
	if err := replica.DeletePublicServices(otherClientSet); err != nil {
		t.Fatalf("replica.DeleteServices() error; %v", err)
	}
}

func TestTFReplicaSetFromPodList(t *testing.T) {
	type TestCase struct {
		PodList  v1.PodList
		Name     string
		Expected tpb.TFReplicaSetStatus_State
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
		  Expected: tpb.TFReplicaSetStatus_RUNNING,
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
		  Expected: tpb.TFReplicaSetStatus_SUCCEEDED,
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
		  Expected: tpb.TFReplicaSetStatus_SUCCEEDED,
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
		  Expected: tpb.TFReplicaSetStatus_FAILED,
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
			Expected: tpb.TFReplicaSetStatus_FAILED,
		},
	}

	for _, c := range cases {
		status := replicaStatusFromPodList(c.PodList, c.Name)
		if status != c.Expected {
			t.Errorf("replicaStatusFromPodList(%+v, %v)=%v ; want %v", c.PodList, c.Name, status, c.Expected)
		}
	}
}
