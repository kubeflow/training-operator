package trainer

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/tensorflow/k8s/pkg/spec"
	tfJobFake "github.com/tensorflow/k8s/pkg/util/k8sutil/fake"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
			Expected: false,
		},
		{
			State: v1.ContainerStateTerminated{
				ExitCode: 0,
				Message:  "some reason",
			},
			Expected: false,
		},
		{
			State: v1.ContainerStateTerminated{
				ExitCode: 1,
				Message:  "some reason",
			},
			Expected: false,
		},
		{
			State: v1.ContainerStateTerminated{
				ExitCode: 1,
			},
			Expected: false,
		},
		{
			State: v1.ContainerStateTerminated{
				ExitCode: 244,
				Message:  "some reason",
			},
			Expected: true,
		},
		{
			State: v1.ContainerStateTerminated{
				ExitCode: 244,
				Reason:   "OOMKilled",
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
				Metadata: metav1.ObjectMeta{
					Name: "myjob",
				},
				Spec: spec.TfJobSpec{
					RuntimeId: "runtime",
					ReplicaSpecs: []*spec.TfReplicaSpec{
						{
							Replicas: proto.Int32(2),
							TfPort:   proto.Int32(22),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
										},
									},
								},
							},
							TfReplicaType: spec.PS,
						},
						{
							Replicas: proto.Int32(1),
							TfPort:   proto.Int32(42),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
										},
									},
								},
							},
							TfReplicaType: spec.MASTER,
						},
						{
							Replicas: proto.Int32(3),
							TfPort:   proto.Int32(40),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
										},
									},
								},
							},
							TfReplicaType: spec.WORKER,
						},
					},
				},
			},

			Expected: map[string][]string{
				"ps":     []string{"myjob-ps-runtime-0:22", "myjob-ps-runtime-1:22"},
				"master": []string{"myjob-master-runtime-0:42"},
				"worker": []string{"myjob-worker-runtime-0:40", "myjob-worker-runtime-1:40", "myjob-worker-runtime-2:40"},
			},
		},
	}

	for _, c := range cases {

		clientSet := fake.NewSimpleClientset()

		stopC := make(chan struct{})

		wg := &sync.WaitGroup{}
		job, err := initJob(clientSet, &tfJobFake.TfJobClientFake{}, c.Spec, stopC, wg)

		if err != nil {
			t.Fatalf("initJob failed: %v", err)
		}

		job.setup(&spec.ControllerConfig{})

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

func TestJobSetup(t *testing.T) {
	// Verify the setup will fill in the RuntimeId.
	clientSet := fake.NewSimpleClientset()

	type testCase struct {
		jobSpec      *spec.TfJob
		expectMounts int
		expectPhase  spec.TfJobPhase
		expectReason string
		expectState  spec.State
	}

	testCases := []testCase{
		{
			jobSpec: &spec.TfJob{
				Spec: spec.TfJobSpec{
					ReplicaSpecs: []*spec.TfReplicaSpec{
						{
							Replicas: proto.Int32(2),
							TfPort:   proto.Int32(10),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
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
			},
			expectMounts: 0,
			expectPhase:  spec.TfJobPhaseCreating,
			expectState:  spec.StateRunning,
		},
		{
			jobSpec: &spec.TfJob{
				Spec: spec.TfJobSpec{
					ReplicaSpecs: []*spec.TfReplicaSpec{
						{
							Replicas: proto.Int32(2),
							TfPort:   proto.Int32(10),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
											Resources: v1.ResourceRequirements{
												Requests: map[v1.ResourceName]resource.Quantity{
													"nvidia-gpu": resource.MustParse("1"),
												},
											},
										},
									},
								},
							},
							TfReplicaType: spec.PS,
						},
					},
				},
			},
			expectMounts: 1,
			expectPhase:  spec.TfJobPhaseCreating,
			expectState:  spec.StateRunning,
		},
		{
			// The job should fail setup because the spec is invalid.
			jobSpec: &spec.TfJob{
				Spec: spec.TfJobSpec{
					ReplicaSpecs: []*spec.TfReplicaSpec{
						{
							Replicas: proto.Int32(2),
							TfPort:   proto.Int32(10),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
											Resources: v1.ResourceRequirements{
												Requests: map[v1.ResourceName]resource.Quantity{
													"nvidia-gpu": resource.MustParse("1"),
												},
											},
										},
									},
								},
							},
							TfReplicaType: spec.PS,
						},
					},
					TensorBoard: &spec.TensorBoardSpec{},
				},
			},
			expectMounts: 0,
			expectPhase:  spec.TfJobPhaseFailed,
			expectState:  spec.StateFailed,
			expectReason: "tbReplicaSpec.LogDir must be specified",
		},
	}

	config := &spec.ControllerConfig{
		Accelerators: map[string]spec.AcceleratorConfig{
			"nvidia-gpu": spec.AcceleratorConfig{
				Volumes: []spec.AcceleratorVolume{
					{
						Name:      "cuda-lib",
						HostPath:  "/home/cuda",
						MountPath: "/usr/local/cuda",
					},
				},
			},
		},
	}

	for _, c := range testCases {
		stopC := make(chan struct{})
		wg := &sync.WaitGroup{}
		job, err := initJob(clientSet, &tfJobFake.TfJobClientFake{}, c.jobSpec, stopC, wg)

		job.setup(config)

		if err != nil {
			t.Errorf("j.setup error: %v", err)
		}

		if job.status.Phase != c.expectPhase {
			t.Errorf("job.job.Status.Phase Want: %v Got:%v ", c.expectPhase, job.status.Phase)
		}

		if job.status.Reason != c.expectReason {
			t.Errorf("job.job.Status.Reason Want: %v Got:%v ", c.expectReason, job.status.Reason)
		}

		if job.status.State != c.expectState {
			t.Errorf("job.job.Status.State Want: %v Got:%v ", c.expectState, job.status.State)
		}

		// Make sure the runtime id is set if the job didn't fail.
		if c.expectState != spec.StateFailed && job.job.Spec.RuntimeId == "" {
			t.Errorf("RuntimeId should not be empty after calling setup.")
		}

		if len(job.job.Spec.ReplicaSpecs[0].Template.Spec.Volumes) != c.expectMounts {
			t.Errorf("Expect %v Volumes got %v", c.expectMounts, len(job.job.Spec.ReplicaSpecs[0].Template.Spec.Volumes))
		}

		if len(job.job.Spec.ReplicaSpecs[0].Template.Spec.Containers[0].VolumeMounts) != c.expectMounts {
			t.Errorf("Expect %v VolumeMounts got %v", c.expectMounts, len(job.job.Spec.ReplicaSpecs[0].Template.Spec.Containers[0].VolumeMounts))
		}
	}
}
