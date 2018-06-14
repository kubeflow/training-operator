package testutil

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

func NewTFJobWithChief(worker, ps int) *tfv1alpha2.TFJob {
	tfJob := NewTFJob(worker, ps)
	tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeChief] = &tfv1alpha2.TFReplicaSpec{
		Template: NewTFReplicaSpecTemplate(),
	}
	return tfJob
}

func NewTFJob(worker, ps int) *tfv1alpha2.TFJob {
	tfJob := &tfv1alpha2.TFJob{
		TypeMeta: metav1.TypeMeta{
			Kind: tfv1alpha2.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestTFJobName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: tfv1alpha2.TFJobSpec{
			TFReplicaSpecs: make(map[tfv1alpha2.TFReplicaType]*tfv1alpha2.TFReplicaSpec),
		},
	}

	if worker > 0 {
		worker := int32(worker)
		workerReplicaSpec := &tfv1alpha2.TFReplicaSpec{
			Replicas: &worker,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypeWorker] = workerReplicaSpec
	}

	if ps > 0 {
		ps := int32(ps)
		psReplicaSpec := &tfv1alpha2.TFReplicaSpec{
			Replicas: &ps,
			Template: NewTFReplicaSpecTemplate(),
		}
		tfJob.Spec.TFReplicaSpecs[tfv1alpha2.TFReplicaTypePS] = psReplicaSpec
	}
	return tfJob
}

func NewTFReplicaSpecTemplate() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  tfv1alpha2.DefaultContainerName,
					Image: TestImageName,
					Args:  []string{"Fake", "Fake"},
					Ports: []v1.ContainerPort{
						v1.ContainerPort{
							Name:          tfv1alpha2.DefaultPortName,
							ContainerPort: tfv1alpha2.DefaultPort,
						},
					},
				},
			},
		},
	}
}
