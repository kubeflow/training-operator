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

package control

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/generator"
	"github.com/kubeflow/tf-operator/pkg/util/testutil"
)

func TestClaimServices(t *testing.T) {
	controllerUID := "123"

	type test struct {
		name     string
		manager  *ServiceControllerRefManager
		services []*v1.Service
		filters  []func(*v1.Service) bool
		claimed  []*v1.Service
		released []*v1.Service
	}
	var tests = []test{
		func() test {
			tfJob := testutil.NewTFJob(1, 0)
			tfJobLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: generator.GenLabels(tfJob.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			testService := testutil.NewBaseService("service2", tfJob, nil)
			testService.Labels[generator.LabelGroupName] = "testing"

			return test{
				name: "Claim services with correct label",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					tfJob,
					tfJobLabelSelector,
					tfv1alpha2.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutil.NewBaseService("service1", tfJob, t), testService},
				claimed:  []*v1.Service{testutil.NewBaseService("service1", tfJob, t)},
			}
		}(),
		func() test {
			controller := testutil.NewTFJob(1, 0)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: generator.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			testService1 := testutil.NewBaseService("service1", controller, t)
			testService1.SetOwnerReferences([]metav1.OwnerReference{})
			testService2 := testutil.NewBaseService("service2", controller, t)
			testService2.SetOwnerReferences([]metav1.OwnerReference{})
			return test{
				name: "Controller marked for deletion can not claim services",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					tfv1alpha2.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testService1, testService2},
				claimed:  nil,
			}
		}(),
		func() test {
			controller := testutil.NewTFJob(1, 0)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: generator.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			testService2 := testutil.NewBaseService("service2", controller, t)
			testService2.SetOwnerReferences([]metav1.OwnerReference{})
			return test{
				name: "Controller marked for deletion can not claim new services",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					tfv1alpha2.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutil.NewBaseService("service1", controller, t), testService2},
				claimed:  []*v1.Service{testutil.NewBaseService("service1", controller, t)},
			}
		}(),
		func() test {
			controller := testutil.NewTFJob(1, 0)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: generator.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller2 := testutil.NewTFJob(1, 0)
			controller.UID = types.UID(controllerUID)
			controller2.UID = types.UID("AAAAA")
			return test{
				name: "Controller can not claim services owned by another controller",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					tfv1alpha2.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutil.NewBaseService("service1", controller, t), testutil.NewBaseService("service2", controller2, t)},
				claimed:  []*v1.Service{testutil.NewBaseService("service1", controller, t)},
			}
		}(),
		func() test {
			controller := testutil.NewTFJob(1, 0)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: generator.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			testService2 := testutil.NewBaseService("service2", controller, t)
			testService2.Labels[generator.LabelGroupName] = "testing"
			return test{
				name: "Controller releases claimed services when selector doesn't match",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					tfv1alpha2.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testutil.NewBaseService("service1", controller, t), testService2},
				claimed:  []*v1.Service{testutil.NewBaseService("service1", controller, t)},
			}
		}(),
		func() test {
			controller := testutil.NewTFJob(1, 0)
			controllerLabelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: generator.GenLabels(controller.Name),
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			controller.UID = types.UID(controllerUID)
			testService1 := testutil.NewBaseService("service1", controller, t)
			testService2 := testutil.NewBaseService("service2", controller, t)
			testService2.Labels[generator.LabelGroupName] = "testing"
			now := metav1.Now()
			testService1.DeletionTimestamp = &now
			testService2.DeletionTimestamp = &now

			return test{
				name: "Controller does not claim orphaned services marked for deletion",
				manager: NewServiceControllerRefManager(&FakeServiceControl{},
					controller,
					controllerLabelSelector,
					tfv1alpha2.SchemeGroupVersionKind,
					func() error { return nil }),
				services: []*v1.Service{testService1, testService2},
				claimed:  []*v1.Service{testService1},
			}
		}(),
	}
	for _, test := range tests {
		claimed, err := test.manager.ClaimServices(test.services)
		if err != nil {
			t.Errorf("Test case `%s`, unexpected error: %v", test.name, err)
		} else if !reflect.DeepEqual(test.claimed, claimed) {
			t.Errorf("Test case `%s`, claimed wrong services. Expected %v, got %v", test.name, serviceToStringSlice(test.claimed), serviceToStringSlice(claimed))
		}

	}
}

func serviceToStringSlice(services []*v1.Service) []string {
	var names []string
	for _, service := range services {
		names = append(names, service.Name)
	}
	return names
}
