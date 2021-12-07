package pytorch

import (
	"fmt"
	pytorchv1 "github.com/kubeflow/training-operator/pkg/apis/pytorch/v1"
	corev1 "k8s.io/api/core/v1"
	"strconv"
)

func setPodLabel(obj interface{}, podTemplateSpec *corev1.PodTemplateSpec, rtype, index string) error {
	pytorchjob, ok := obj.(*pytorchv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", obj)
	}
	if pytorchjob.Spec.ElasticPolicy != nil {
		totalReplicas := getTotalReplicas(pytorchjob)
		indexNum, _ := strconv.ParseInt(index, 10, 32)
		podTemplateSpec.Labels["volcano.sh/task-priority"] = fmt.Sprintf("%d", totalReplicas-int32(indexNum))
	}
	return nil
}
