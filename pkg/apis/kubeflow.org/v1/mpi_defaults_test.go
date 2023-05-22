package v1

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
)

func expectedMPIJob(cleanPodPolicy commonv1.CleanPodPolicy, restartPolicy commonv1.RestartPolicy) *MPIJob {
	return &MPIJob{
		Spec: MPIJobSpec{
			CleanPodPolicy: &cleanPodPolicy,
			RunPolicy: commonv1.RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
			},
			MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
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
	customRestartPolicy := commonv1.RestartPolicyAlways
	customCleanPodPolicy := commonv1.CleanPodPolicyRunning

	testCases := map[string]struct {
		original *MPIJob
		expected *MPIJob
	}{
		"set default replicas": {
			original: &MPIJob{
				Spec: MPIJobSpec{
					CleanPodPolicy: &customCleanPodPolicy,
					RunPolicy: commonv1.RunPolicy{
						CleanPodPolicy: &customCleanPodPolicy,
					},
					MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
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
			expected: expectedMPIJob(customCleanPodPolicy, customRestartPolicy),
		},
		"set default clean pod policy": {
			original: &MPIJob{
				Spec: MPIJobSpec{
					MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
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
			expected: expectedMPIJob(commonv1.CleanPodPolicyNone, MPIJobDefaultRestartPolicy),
		},
		"set default restart policy": {
			original: &MPIJob{
				Spec: MPIJobSpec{
					MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
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
			expected: expectedMPIJob(commonv1.CleanPodPolicyNone, MPIJobDefaultRestartPolicy),
		},
	}
	for name, tc := range testCases {
		SetDefaults_MPIJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, util.Pformat(tc.expected), util.Pformat(tc.original))
		}
	}
}
