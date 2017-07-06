package trainer

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"mlkube.io/pkg/spec"
	"sync"
)

func TestIsRetryableTerminationState(t *testing.T) {
	type TestCase struct {
		State    v1.ContainerStateTerminated
		Expected bool
	}

	cases := []TestCase{
		{
      // Since reason is empty we don't trust the exit code.
			State: v1.ContainerStateTerminated{
				ExitCode: 0,
			},
      Expected: true,
		},
    {
      State: v1.ContainerStateTerminated{
        ExitCode: 0,
        Message: "some reason",
      },
      Expected: false,
    },
    {
      State: v1.ContainerStateTerminated{
        ExitCode: 1,
        Message: "some reason",
      },
      Expected: false,
    },
    {
      // Since Reason is empty we don't trust the exit code.
      State: v1.ContainerStateTerminated{
        ExitCode: 1,
      },
      Expected: true,
    },
    {
      State: v1.ContainerStateTerminated{
        ExitCode: 244,
        Message: "some reason",
      },
      Expected: true,
    },
    {
      State: v1.ContainerStateTerminated{
        ExitCode: 244,
        Reason: "OOMKilled",
      },
      Expected: false,
    },
	}

  for _, c := range cases {
    actual := isRetryableTerminationState(&c.State)
		if actual != c.Expected {
      t.Errorf("isRetryableTerminationState(%+v)=%v want %v", c.State, actual, c.Expected)
    }
  }
}

func TestClusterSpec(t *testing.T) {
	type TestCase struct {
		Spec     *spec.TfJob
		Expected map[string][]string
	}

	cases := []TestCase{
		{
			Spec: &spec.TfJob{
				Spec: spec.TfJobSpec{
					RuntimeId: "runtime",
					ReplicaSpecs: []*spec.TfReplicaSpec{
						{
							Replicas:      proto.Int32(2),
							TfPort:        proto.Int32(22),
							Template:      &v1.PodTemplateSpec{},
							TfReplicaType: spec.PS,
						},
						{
							Replicas:      proto.Int32(1),
							TfPort:        proto.Int32(42),
							Template:      &v1.PodTemplateSpec{},
							TfReplicaType: spec.MASTER,
						},
            {
              Replicas:      proto.Int32(3),
              TfPort:        proto.Int32(40),
              Template:      &v1.PodTemplateSpec{},
              TfReplicaType: spec.WORKER,
            },
					},
				},
			},

			Expected: map[string][]string{
				"ps":     []string{"ps-runtime-0:22", "ps-runtime-1:22"},
				"master": []string{"master-runtime-0:42"},
        "worker":     []string{"worker-runtime-0:40", "worker-runtime-1:40", "worker-runtime-2:40"},
			},
		},
	}

	for _, c := range cases {

		clientSet := fake.NewSimpleClientset()

		stopC := make(chan struct{})

		wg := &sync.WaitGroup{}
		job, err := initJob(clientSet, c.Spec, stopC, wg)

		if err != nil {
			t.Fatalf("initJob failed: %v", err)
		}

		actual := job.ClusterSpec()

		for k, v := range c.Expected {
			actualV, ok := actual[k]
			if !ok {
				t.Errorf("Actual cluster spec is missing key: %v", k)
				continue
			}
			if !reflect.DeepEqual(actualV, v) {
				t.Errorf("Key %v got %v want %v", k, actualV, v)
			}
		}
	}
}
