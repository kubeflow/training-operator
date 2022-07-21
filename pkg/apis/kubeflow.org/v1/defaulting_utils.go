package v1

import (
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func getDefaultContainerIndex(spec *corev1.PodSpec, defaultContainerName string) int {
	for i, container := range spec.Containers {
		if container.Name == defaultContainerName {
			return i
		}
	}
	return 0
}

func hasDefaultPort(spec *corev1.PodSpec, containerIndex int, defaultPortName string) bool {
	for _, port := range spec.Containers[containerIndex].Ports {
		if port.Name == defaultPortName {
			return true
		}
	}
	return false
}

func setDefaultPort(spec *corev1.PodSpec, defaultPortName string, defaultPort int32, defaultContainerIndex int) {
	spec.Containers[defaultContainerIndex].Ports = append(spec.Containers[defaultContainerIndex].Ports,
		corev1.ContainerPort{
			Name:          defaultPortName,
			ContainerPort: defaultPort,
		})
}

func setDefaultRestartPolicy(replicaSpec *commonv1.ReplicaSpec, defaultRestartPolicy commonv1.RestartPolicy) {
	if replicaSpec != nil && replicaSpec.RestartPolicy == "" {
		replicaSpec.RestartPolicy = defaultRestartPolicy
	}
}

func setDefaultReplicas(replicaSpec *commonv1.ReplicaSpec, replicas int32) {
	if replicaSpec != nil && replicaSpec.Replicas == nil {
		replicaSpec.Replicas = pointer.Int32(replicas)
	}
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from server to Server; from WORKER to Worker.
func setTypeNameToCamelCase(replicaSpecs map[commonv1.ReplicaType]*commonv1.ReplicaSpec, typ commonv1.ReplicaType) {
	for t := range replicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := replicaSpecs[t]
			delete(replicaSpecs, t)
			replicaSpecs[typ] = spec
			return
		}
	}
}
