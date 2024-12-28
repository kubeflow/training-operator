package jax

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestSetPodEnv(t *testing.T) {
	// Define some helper variables/constants for the test cases
	validPort := int32(6666)
	validIndex := "0"
	invalidIndex := "invalid"

	// Define a valid JAXJob structure
	validJAXJob := &kubeflowv1.JAXJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-jaxjob"},
		Spec: kubeflowv1.JAXJobSpec{
			JAXReplicaSpecs: map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
				kubeflowv1.JAXJobReplicaTypeWorker: {
					Replicas: ptr.To[int32](1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "jax",
								Image: "docker.io/kubeflow/jaxjob-dist-spmd-mnist:latest",
								Ports: []corev1.ContainerPort{{
									Name:          kubeflowv1.JAXJobDefaultPortName,
									ContainerPort: validPort,
								}},
								ImagePullPolicy: corev1.PullAlways,
								Command: []string{
									"python",
									"train.py",
								},
							}},
						},
					},
				},
			},
		},
	}

	// Define the test cases
	cases := map[string]struct {
		jaxJob         *kubeflowv1.JAXJob
		podTemplate    *corev1.PodTemplateSpec
		rtype          kubeflowv1.ReplicaType
		index          string
		wantPodEnvVars []corev1.EnvVar
		wantErr        error
	}{
		"successful environment variable setup": {
			jaxJob: validJAXJob,
			podTemplate: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			rtype: kubeflowv1.JAXJobReplicaTypeWorker,
			index: validIndex,
			wantPodEnvVars: []corev1.EnvVar{
				{Name: "PYTHONUNBUFFERED", Value: "1"},
				{Name: "COORDINATOR_PORT", Value: strconv.Itoa(int(validPort))},
				{Name: "COORDINATOR_ADDRESS", Value: "test-jaxjob-worker-0"},
				{Name: "NUM_PROCESSES", Value: "1"},
				{Name: "PROCESS_ID", Value: validIndex},
			},
			wantErr: nil,
		},
		"invalid index for PROCESS_ID": {
			jaxJob: validJAXJob,
			podTemplate: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			rtype:   kubeflowv1.JAXJobReplicaTypeWorker,
			index:   invalidIndex,
			wantErr: errorFailedToRecognizeRank,
		},
		"missing container port in JAXJob": {
			jaxJob: &kubeflowv1.JAXJob{
				Spec: kubeflowv1.JAXJobSpec{
					JAXReplicaSpecs: map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{
						kubeflowv1.JAXJobReplicaTypeWorker: {
							Replicas: ptr.To[int32](1),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name: "jax",
										Ports: []corev1.ContainerPort{
											{Name: "wrong-port", ContainerPort: 0},
										},
									}},
								},
							},
						},
					},
				},
			},
			podTemplate: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			rtype:   kubeflowv1.JAXJobReplicaTypeWorker,
			index:   validIndex,
			wantErr: errorDefaulContainerPortNotExposed,
		},
	}

	// Execute the test cases
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := setPodEnv(tc.jaxJob, tc.podTemplate, string(tc.rtype), tc.index)

			// Check if an error was expected
			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got):\n%s", diff)
			}

			for i, container := range tc.podTemplate.Spec.Containers {
				if diff := cmp.Diff(tc.wantPodEnvVars, container.Env); diff != "" {
					t.Errorf("Unexpected env vars for container %d (-want,+got):\n%s", i, diff)
				}
			}

		})
	}
}
