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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	testingutil "github.com/kubeflow/training-operator/pkg/util.v2/testing"
)

func TestNewInfo(t *testing.T) {
	jobSetBase := testingutil.MakeJobSetWrapper(metav1.NamespaceDefault, "test-job").
		Clone()

	cases := map[string]struct {
		obj      client.Object
		infoOpts []InfoOption
		wantInfo *Info
	}{
		"all arguments are specified": {
			obj: jobSetBase.Obj(),
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
				WithPodSpecReplicas("Leader", 1, corev1.PodSpec{
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
				WithPodSpecReplicas("Worker", 10, corev1.PodSpec{
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
				Obj: jobSetBase.Obj(),
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
					"Leader": {
						Replicas: 1,
						PodRequests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("15"),
						},
					},
					"Worker": {
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
			info := NewInfo(tc.obj, tc.infoOpts...)
			if diff := cmp.Diff(tc.wantInfo, info, cmpOpts...); len(diff) != 0 {
				t.Errorf("Unexpected runtime.Info (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	jobSetBase := testingutil.MakeJobSetWrapper(metav1.NamespaceDefault, "test-job").
		Clone()

	cases := map[string]struct {
		info      *Info
		obj       client.Object
		wantInfo  *Info
		wantError error
	}{
		"gvk is different between old and new objects": {
			info: &Info{
				Obj: jobSetBase.Obj(),
			},
			obj: testingutil.MakeTrainJobWrapper(metav1.NamespaceDefault, "test-job").
				Obj(),
			wantInfo: &Info{
				Obj: jobSetBase.Obj(),
			},
			wantError: errorDifferentGVK,
		},
		"old object is nil": {
			info:      &Info{},
			obj:       jobSetBase.Obj(),
			wantInfo:  &Info{},
			wantError: errorObjectsAreNil,
		},
		"new object is nil": {
			info: &Info{
				Obj: jobSetBase.Obj(),
			},
			wantInfo: &Info{
				Obj: jobSetBase.Obj(),
			},
			wantError: errorObjectsAreNil,
		},
		"update object with the appropriate parameter": {
			info: &Info{
				Obj: jobSetBase.Obj(),
			},
			obj: jobSetBase.ContainerImage(ptr.To("test:latest")).Obj(),
			wantInfo: &Info{
				Obj: jobSetBase.ContainerImage(ptr.To("test:latest")).Obj(),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.info != nil {
				err := tc.info.Update(tc.obj)
				if diff := cmp.Diff(tc.wantError, err, cmpopts.EquateErrors()); len(diff) != 0 {
					t.Errorf("Unexpected error (-want,+got):\n%s", diff)
				}
			}
			if diff := cmp.Diff(tc.wantInfo, tc.info); len(diff) != 0 {
				t.Errorf("Unexpected runtime.Info (-want,+got):\n%s", diff)
			}
		})
	}
}
