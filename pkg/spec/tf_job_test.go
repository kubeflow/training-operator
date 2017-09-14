package spec

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/jlewi/mlkube.io/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/pkg/api/v1"
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
			},
		},
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						TfReplicaType: PS,
						TfImage:       "tensorflow/tensorflow:1.3.",
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
										Image: "tensorflow/tensorflow:1.3.",
										Name:  "tensorflow",
										VolumeMounts: []v1.VolumeMount{
											v1.VolumeMount{
												Name:      "ps-config-volume",
												MountPath: "/ps-server",
											},
										},
										Command: []string{"python", "/ps-server/start_server.py"},
									},
								},
								Volumes: []v1.Volume{
									v1.Volume{
										Name: "ps-config-volume",
										VolumeSource: v1.VolumeSource{
											ConfigMap: &v1.ConfigMapVolumeSource{
												LocalObjectReference: v1.LocalObjectReference{
													Name: PSConfigMapName(),
												},
											},
										},
									},
								},
								RestartPolicy: v1.RestartPolicyOnFailure,
							},
						},
						TfReplicaType: PS,
						TfImage:       "tensorflow/tensorflow:1.3.",
					},
				},
			},
		},
	}

	for _, c := range testCases {
		if err := c.in.SetDefaults(); err != nil {
			t.Errorf("SetDefaults error; %v", err)
		}
		if !reflect.DeepEqual(c.in, c.expected) {
			t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(c.in))
		}
	}
}
