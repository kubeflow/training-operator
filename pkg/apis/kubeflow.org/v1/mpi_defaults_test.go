package v1

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func expectedMPIJob(cleanPodPolicy CleanPodPolicy, restartPolicy RestartPolicy) *MPIJob {
	return &MPIJob{
		Spec: MPIJobSpec{
			CleanPodPolicy: &cleanPodPolicy,
			RunPolicy: RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
			},
			MPIReplicaSpecs: map[ReplicaType]*ReplicaSpec{
				MPIJobReplicaTypeLauncher: {
					Replicas:      pointer.Int32(1),
					RestartPolicy: restartPolicy,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  MPIJobDefaultContainerName,
									Image: testImage,
								},
							},
						},
					},
				},
				MPIJobReplicaTypeWorker: {
					Replicas:      pointer.Int32(0),
					RestartPolicy: restartPolicy,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  MPIJobDefaultContainerName,
									Image: testImage,
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestSetDefaults_MPIJob(t *testing.T) {
	customRestartPolicy := RestartPolicyAlways

	testCases := map[string]struct {
		original *MPIJob
		expected *MPIJob
	}{
		"set default replicas": {
			original: &MPIJob{
				Spec: MPIJobSpec{
					CleanPodPolicy: CleanPodPolicyPointer(CleanPodPolicyRunning),
					RunPolicy: RunPolicy{
						CleanPodPolicy: CleanPodPolicyPointer(CleanPodPolicyRunning),
					},
					MPIReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MPIJobReplicaTypeLauncher: {
							RestartPolicy: customRestartPolicy,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MPIJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
						MPIJobReplicaTypeWorker: {
							RestartPolicy: customRestartPolicy,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MPIJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedMPIJob(CleanPodPolicyRunning, customRestartPolicy),
		},
		"set default clean pod policy": {
			original: &MPIJob{
				Spec: MPIJobSpec{
					MPIReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MPIJobReplicaTypeLauncher: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MPIJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
						MPIJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MPIJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedMPIJob(CleanPodPolicyNone, MPIJobDefaultRestartPolicy),
		},
		"set default restart policy": {
			original: &MPIJob{
				Spec: MPIJobSpec{
					MPIReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MPIJobReplicaTypeLauncher: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MPIJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
						MPIJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MPIJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedMPIJob(CleanPodPolicyNone, MPIJobDefaultRestartPolicy),
		},
	}
	for name, tc := range testCases {
		SetDefaults_MPIJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, tc.expected, tc.original)
		}
	}
}
