package spec

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/tensorflow/k8s/pkg/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestAddAccelertor(t *testing.T) {
	type testCase struct {
		in       *TfJobSpec
		expected *TfJobSpec
		config   map[string]AcceleratorConfig
	}

	testCases := []testCase{
		// Case 1 checks that we look at requests.
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
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
						TfReplicaType: PS,
					},
				},
			},
			expected: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
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
										VolumeMounts: []v1.VolumeMount{
											{
												Name:      "cuda-lib",
												MountPath: "/usr/local/cuda",
											},
										},
									},
								},
								Volumes: []v1.Volume{
									{
										Name: "cuda-lib",
										VolumeSource: v1.VolumeSource{
											HostPath: &v1.HostPathVolumeSource{
												Path: "/home/cuda",
											},
										},
									},
								},
							},
						},
						TfReplicaType: PS,
					},
				},
			},
			config: map[string]AcceleratorConfig{
				"nvidia-gpu": AcceleratorConfig{
					Volumes: []AcceleratorVolume{
						{
							Name:      "cuda-lib",
							HostPath:  "/home/cuda",
							MountPath: "/usr/local/cuda",
						},
					},
				},
			},
		},
		// Case 2 checks that we look at limit.
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Replicas: proto.Int32(2),
						TfPort:   proto.Int32(10),
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "tensorflow",
										Resources: v1.ResourceRequirements{
											Limits: map[v1.ResourceName]resource.Quantity{
												"nvidia-gpu": resource.MustParse("1"),
											},
										},
									},
								},
							},
						},
						TfReplicaType: PS,
					},
				},
			},
			expected: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Replicas: proto.Int32(2),
						TfPort:   proto.Int32(10),
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "tensorflow",
										Resources: v1.ResourceRequirements{
											Limits: map[v1.ResourceName]resource.Quantity{
												"nvidia-gpu": resource.MustParse("1"),
											},
										},
										VolumeMounts: []v1.VolumeMount{
											{
												Name:      "cuda-lib",
												MountPath: "/usr/local/cuda",
											},
										},
									},
								},
								Volumes: []v1.Volume{
									{
										Name: "cuda-lib",
										VolumeSource: v1.VolumeSource{
											HostPath: &v1.HostPathVolumeSource{
												Path: "/home/cuda",
											},
										},
									},
								},
							},
						},
						TfReplicaType: PS,
					},
				},
			},
			config: map[string]AcceleratorConfig{
				"nvidia-gpu": AcceleratorConfig{
					Volumes: []AcceleratorVolume{
						{
							Name:      "cuda-lib",
							HostPath:  "/home/cuda",
							MountPath: "/usr/local/cuda",
						},
					},
				},
			},
		},
		// Case 3 no GPUs
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
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
						TfReplicaType: PS,
					},
				},
			},
			expected: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
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
						TfReplicaType: PS,
					},
				},
			},
			config: map[string]AcceleratorConfig{
				"nvidia-gpu": AcceleratorConfig{
					Volumes: []AcceleratorVolume{
						{
							Name:      "cuda-lib",
							HostPath:  "/home/cuda",
							MountPath: "/usr/local/cuda",
						},
					},
				},
			},
		},
	}

	for _, c := range testCases {
		if err := c.in.ConfigureAccelerators(c.config); err != nil {
			t.Errorf("ConfigureAccelerators error; %v", err)
		}
		if !reflect.DeepEqual(c.in, c.expected) {
			t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(c.in))
		}
	}
}

func TestSetDefaults(t *testing.T) {
	type testCase struct {
		in       *TfJobSpec
		expected *TfJobSpec
	}

	testCases := []testCase{
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "tensorflow",
									},
								},
							},
						},
					},
				},
				TfImage: "tensorflow/tensorflow:1.3.0",
			},
			expected: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Replicas: proto.Int32(1),
						TfPort:   proto.Int32(2222),
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "tensorflow",
									},
								},
							},
						},
						TfReplicaType: MASTER,
					},
				},
				TfImage: "tensorflow/tensorflow:1.3.0",
				TerminationPolicy: &TerminationPolicySpec{
					Chief: &ChiefSpec{
						ReplicaName:  "MASTER",
						ReplicaIndex: 0,
					},
				},
			},
		},
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						TfReplicaType: PS,
					},
				},
				TfImage: "tensorflow/tensorflow:1.4.0",
			},
			expected: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Replicas: proto.Int32(1),
						TfPort:   proto.Int32(2222),
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "tensorflow/tensorflow:1.4.0",
										Name:  "tensorflow",
										VolumeMounts: []v1.VolumeMount{
											v1.VolumeMount{
												Name:      "ps-config-volume",
												MountPath: "/ps-server",
											},
										},
									},
								},
								RestartPolicy: v1.RestartPolicyOnFailure,
							},
						},
						TfReplicaType: PS,
						IsDefaultPS:   true,
					},
				},
				TfImage: "tensorflow/tensorflow:1.4.0",
				TerminationPolicy: &TerminationPolicySpec{
					Chief: &ChiefSpec{
						ReplicaName:  "MASTER",
						ReplicaIndex: 0,
					},
				},
			},
		},
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						TfReplicaType: PS,
					},
				},
			},
			expected: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Replicas: proto.Int32(1),
						TfPort:   proto.Int32(2222),
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "tensorflow/tensorflow:1.3.0",
										Name:  "tensorflow",
										VolumeMounts: []v1.VolumeMount{
											v1.VolumeMount{
												Name:      "ps-config-volume",
												MountPath: "/ps-server",
											},
										},
									},
								},
								RestartPolicy: v1.RestartPolicyOnFailure,
							},
						},
						TfReplicaType: PS,
						IsDefaultPS:   true,
					},
				},
				TfImage: "tensorflow/tensorflow:1.3.0",
				TerminationPolicy: &TerminationPolicySpec{
					Chief: &ChiefSpec{
						ReplicaName:  "MASTER",
						ReplicaIndex: 0,
					},
				},
			},
		},
	}

	for i, c := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if err := c.in.SetDefaults(c.in.TfImage); err != nil {
				t.Errorf("SetDefaults error; %v", err)
			}
			if !reflect.DeepEqual(c.in, c.expected) {
				t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(c.in))
			}
		})
	}
}

func TestValidate(t *testing.T) {
	type testCase struct {
		in             *TfJobSpec
		expectingError bool
	}

	testCases := []testCase{
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "tensorflow",
									},
								},
							},
						},
						TfReplicaType: MASTER,
						Replicas:      proto.Int32(1),
					},
				},
				TfImage: "tensorflow/tensorflow:1.3.0",
			},
			expectingError: false,
		},
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "tensorflow",
									},
								},
							},
						},
						TfReplicaType: WORKER,
						Replicas:      proto.Int32(1),
					},
				},
				TfImage: "tensorflow/tensorflow:1.3.0",
			},
			expectingError: true,
		},
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "tensorflow",
									},
								},
							},
						},
						TfReplicaType: WORKER,
						Replicas:      proto.Int32(1),
					},
				},
				TfImage: "tensorflow/tensorflow:1.3.0",
				TerminationPolicy: &TerminationPolicySpec{
					Chief: &ChiefSpec{
						ReplicaName:  "WORKER",
						ReplicaIndex: 0,
					},
				},
			},
			expectingError: false,
		},
	}

	for _, c := range testCases {
		c.in.SetDefaults("")
		if err := c.in.Validate(); (err != nil) != c.expectingError {
			t.Errorf("unexpected validation result: %v", err)
		}
	}
}
