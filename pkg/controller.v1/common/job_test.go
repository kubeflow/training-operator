/*
Copyright 2023 The Kubeflow Authors.

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

package common

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/controller.v1/control"
	testjobv1 "github.com/kubeflow/training-operator/test_job/apis/test_job/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
)

func TestDeletePodsAndServices(T *testing.T) {
	pods := []runtime.Object{
		newPod("runningPod", corev1.PodRunning),
		newPod("succeededPod", corev1.PodSucceeded),
	}
	services := []runtime.Object{
		newService("runningPod"),
		newService("succeededPod"),
	}

	cases := map[string]struct {
		cleanPodPolicy apiv1.CleanPodPolicy
		jobCondition   apiv1.JobConditionType
		wantPods       *corev1.PodList
		wantService    *corev1.ServiceList
	}{
		"Succeeded job and cleanPodPolicy is Running": {
			cleanPodPolicy: apiv1.CleanPodPolicyRunning,
			jobCondition:   apiv1.JobSucceeded,
			wantPods: &corev1.PodList{
				Items: []corev1.Pod{
					*pods[1].(*corev1.Pod),
				},
			},
			wantService: &corev1.ServiceList{
				Items: []corev1.Service{
					*services[1].(*corev1.Service),
				},
			},
		},
		"Suspended job and cleanPodPolicy is Running": {
			cleanPodPolicy: apiv1.CleanPodPolicyRunning,
			jobCondition:   apiv1.JobSuspended,
			wantPods:       &corev1.PodList{},
			wantService:    &corev1.ServiceList{},
		},
		"Finished job and cleanPodPolicy is All": {
			cleanPodPolicy: apiv1.CleanPodPolicyAll,
			jobCondition:   apiv1.JobSucceeded,
			wantPods:       &corev1.PodList{},
			wantService:    &corev1.ServiceList{},
		},
		"Finished job and cleanPodPolicy is None": {
			cleanPodPolicy: apiv1.CleanPodPolicyNone,
			jobCondition:   apiv1.JobFailed,
			wantPods: &corev1.PodList{
				Items: []corev1.Pod{
					*pods[0].(*corev1.Pod),
					*pods[1].(*corev1.Pod),
				},
			},
			wantService: &corev1.ServiceList{
				Items: []corev1.Service{
					*services[0].(*corev1.Service),
					*services[1].(*corev1.Service),
				},
			},
		},
		"Suspended job and cleanPodPolicy is None": {
			cleanPodPolicy: apiv1.CleanPodPolicyNone,
			jobCondition:   apiv1.JobSuspended,
			wantPods:       &corev1.PodList{},
			wantService:    &corev1.ServiceList{},
		},
	}
	for name, tc := range cases {
		T.Run(name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(append(pods, services...)...)
			jobController := JobController{
				PodControl:     control.RealPodControl{KubeClient: fakeClient, Recorder: &record.FakeRecorder{}},
				ServiceControl: control.RealServiceControl{KubeClient: fakeClient, Recorder: &record.FakeRecorder{}},
			}

			var inPods []*corev1.Pod
			for i := range pods {
				inPods = append(inPods, pods[i].(*corev1.Pod))
			}
			runPolicy := &apiv1.RunPolicy{
				CleanPodPolicy: &tc.cleanPodPolicy,
			}
			jobStatus := apiv1.JobStatus{
				Conditions: []apiv1.JobCondition{
					{
						Type:   tc.jobCondition,
						Status: corev1.ConditionTrue,
					},
				},
			}
			if err := jobController.DeletePodsAndServices(&testjobv1.TestJob{}, runPolicy, jobStatus, inPods); err != nil {
				T.Errorf("Failed to delete pods and services: %v", err)
			}
			gotPods, err := fakeClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Errorf("Failed to list pods: %v", err)
			}
			if diff := cmp.Diff(tc.wantPods, gotPods); len(diff) != 0 {
				t.Errorf("Unexpected pods after running DeletePodsAndServices (-want,+got):%s\n", diff)
			}
			gotServices, err := fakeClient.CoreV1().Services("").List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Errorf("Failed to list services: %v", err)
			}
			if diff := cmp.Diff(tc.wantService, gotServices); len(diff) != 0 {
				t.Errorf("Unexpected services after running DeletePodsAndServices (-want,+got):%s\n", diff)
			}
		})
	}
}

func TestPastBackoffLimit(T *testing.T) {
	backoffLimitExceededPod := newPod("runningPodWithBackoff", corev1.PodRunning)
	backoffLimitExceededPod.Status.ContainerStatuses = []corev1.ContainerStatus{
		{RestartCount: 3},
	}
	allPods := []*corev1.Pod{
		newPod("runningPod", corev1.PodRunning),
		newPod("succeededPod", corev1.PodSucceeded),
		backoffLimitExceededPod,
	}
	cases := map[string]struct {
		pods                 []*corev1.Pod
		backOffLimit         int32
		wantPastBackOffLimit bool
	}{
		"backOffLimit is 0": {
			pods:                 allPods[:2],
			backOffLimit:         0,
			wantPastBackOffLimit: false,
		},
		"backOffLimit is 3": {
			pods:                 allPods,
			backOffLimit:         3,
			wantPastBackOffLimit: true,
		},
	}
	for name, tc := range cases {
		T.Run(name, func(t *testing.T) {
			jobController := JobController{}
			runPolicy := &apiv1.RunPolicy{
				BackoffLimit: &tc.backOffLimit,
			}
			replica := map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
				"test": {RestartPolicy: apiv1.RestartPolicyOnFailure},
			}
			got, err := jobController.PastBackoffLimit("test-job", runPolicy, replica, tc.pods)
			if err != nil {
				t.Errorf("Failaed to do PastBackoffLimit: %v", err)
			}
			if tc.wantPastBackOffLimit != got {
				t.Errorf("Unexpected pastBackoffLimit: \nwant: %v\ngot: %v\n", tc.wantPastBackOffLimit, got)
			}
		})
	}
}

func TestPastActiveDeadline(T *testing.T) {
	cases := map[string]struct {
		activeDeadlineSeconds         int64
		wantPastActiveDeadlineSeconds bool
	}{
		"activeDeadlineSeconds is 0": {
			activeDeadlineSeconds:         0,
			wantPastActiveDeadlineSeconds: true,
		},
		"activeDeadlineSeconds is 2": {
			activeDeadlineSeconds:         2,
			wantPastActiveDeadlineSeconds: false,
		},
	}
	for name, tc := range cases {
		T.Run(name, func(t *testing.T) {
			jobController := JobController{}
			runPolicy := &apiv1.RunPolicy{
				ActiveDeadlineSeconds: &tc.activeDeadlineSeconds,
			}
			jobStatus := apiv1.JobStatus{
				StartTime: &metav1.Time{
					Time: time.Now(),
				},
			}
			if got := jobController.PastActiveDeadline(runPolicy, jobStatus); tc.wantPastActiveDeadlineSeconds != got {
				t.Errorf("Unexpected PastActiveDeadline: \nwant: %v\ngot: %v\n", tc.wantPastActiveDeadlineSeconds, got)
			}
		})
	}
}

func TestManagedBy(T *testing.T) {
	cases := map[string]struct {
		managedBy *string
	}{
		"managedBy is empty": {
			managedBy: ptr.To[string](""),
		},
		"managedBy is training-operator controller": {
			managedBy: ptr.To[string](apiv1.KubeflowJobsController),
		},
		"managedBy is not the training-operator controller": {
			managedBy: ptr.To[string]("kueue.x-k8s.io/multikueue"),
		},
		"managedBy is other value": {
			managedBy: ptr.To[string]("other-job-controller"),
		},
	}
	for name, tc := range cases {
		T.Run(name, func(t *testing.T) {
			jobController := JobController{}
			runPolicy := &apiv1.RunPolicy{
				ManagedBy: tc.managedBy,
			}

			if got := jobController.ManagedByExternalController(*runPolicy); got != nil {
				if diff := cmp.Diff(tc.managedBy, got); diff != "" {
					t.Errorf("Unexpected manager controller (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func newPod(name string, phase corev1.PodPhase) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				apiv1.ReplicaTypeLabel: "test",
			},
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
	return pod
}

func newService(name string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return service
}
