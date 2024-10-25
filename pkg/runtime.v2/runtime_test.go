/*
Copyright 2024 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtimev2

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	"github.com/kubeflow/training-operator/pkg/constants"
)

func TestNewInfo(t *testing.T) {
	// jobSetBase := testingutil.MakeJobSetWrapper(metav1.NamespaceDefault, "test-job").
	// 	Clone()

	cases := map[string]struct {
		infoOpts []InfoOption
		wantInfo *Info
	}{
		"all arguments are specified": {
			infoOpts: []InfoOption{
				WithLabels(map[string]string{
					"labelKey": "labelValue",
				}),
				WithAnnotations(map[string]string{
					"annotationKey": "annotationValue",
				}),
				WithPodGroupPolicy(&kubeflowv2.PodGroupPolicy{
					PodGroupPolicySource: kubeflowv2.PodGroupPolicySource{
						Coscheduling: &kubeflowv2.CoschedulingPodGroupPolicySource{
							ScheduleTimeoutSeconds: ptr.To[int32](300),
						},
					},
				}),
				WithMLPolicy(&kubeflowv2.MLPolicy{
					NumNodes: ptr.To[int32](100),
					MLPolicySource: kubeflowv2.MLPolicySource{
						Torch: &kubeflowv2.TorchMLPolicySource{
							NumProcPerNode: ptr.To("8"),
						},
					},
				}),
				WithPodSpecReplicas(constants.JobInitializer, 1, corev1.PodSpec{
					InitContainers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("5"),
							},
						},
						RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
					}},
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("10"),
							},
						},
					}},
				}),
				WithPodSpecReplicas(constants.JobTrainerNode, 10, corev1.PodSpec{
					InitContainers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("15"),
							},
						},
						RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
					}},
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("25"),
							},
						},
					}},
				}),
			},
			wantInfo: &Info{
				Labels: map[string]string{
					"labelKey": "labelValue",
				},
				Annotations: map[string]string{
					"annotationKey": "annotationValue",
				},
				Policy: Policy{
					MLPolicy: &kubeflowv2.MLPolicy{
						NumNodes: ptr.To[int32](100),
						MLPolicySource: kubeflowv2.MLPolicySource{
							Torch: &kubeflowv2.TorchMLPolicySource{
								NumProcPerNode: ptr.To("8"),
							},
						},
					},
					PodGroupPolicy: &kubeflowv2.PodGroupPolicy{
						PodGroupPolicySource: kubeflowv2.PodGroupPolicySource{
							Coscheduling: &kubeflowv2.CoschedulingPodGroupPolicySource{
								ScheduleTimeoutSeconds: ptr.To[int32](300),
							},
						},
					},
				},
				TotalRequests: map[string]TotalResourceRequest{
					constants.JobInitializer: {
						Replicas: 1,
						PodRequests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("15"),
						},
					},
					constants.JobTrainerNode: {
						Replicas: 10,
						PodRequests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("40"),
						},
					},
				},
			},
		},
		"all arguments are not specified": {
			wantInfo: &Info{},
		},
	}
	cmpOpts := []cmp.Option{
		cmpopts.SortMaps(func(a, b string) bool { return a < b }),
		cmpopts.EquateEmpty(),
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			info := NewInfo(tc.infoOpts...)
			if diff := cmp.Diff(tc.wantInfo, info, cmpOpts...); len(diff) != 0 {
				t.Errorf("Unexpected runtime.Info (-want,+got):\n%s", diff)
			}
		})
	}
}
