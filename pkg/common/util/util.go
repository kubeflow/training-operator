package util

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
)

// ConvertServiceList convert service list to service point list
func ConvertServiceList(list []corev1.Service) []*corev1.Service {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Service, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

// ConvertPodList convert pod list to pod pointer list
func ConvertPodList(list []corev1.Pod) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

func GetReplicaTypes(specs map[commonv1.ReplicaType]*commonv1.ReplicaSpec) []commonv1.ReplicaType {
	keys := make([]commonv1.ReplicaType, 0, len(specs))
	for k := range specs {
		keys = append(keys, k)
	}
	return keys
}
