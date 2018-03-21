package v1alpha2

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func setDefaultPort(spec *TFReplicaSpec) {
	// TODO(gaocegege): Set ports for init containers, maybe.
	for _, container := range spec.Template.Spec.Containers {
		if len(container.Ports) == 0 {
			container.Ports = append(container.Ports, v1.ContainerPort{
				Name:          defaultPortName,
				ContainerPort: defaultPort,
			})
		}
	}
}

func setDefaultImage(spec *TFReplicaSpec) {
	for _, container := range spec.Template.Spec.Containers {
		if container.Image == "" {
			container.Image = defaultImage
		}
	}
}

func setDefaultRestartPolicy(spec *TFReplicaSpec) {
	if spec.RestartPolicy == RestartPolicy("") {
		spec.RestartPolicy = RestartPolicyAlways
	}
}

// SetDefaults_TFJob sets any unspecified values to defaults.
func SetDefaults_TFJob(tfjob *TFJob) {
	for typ, spec := range tfjob.Spec.TFReplicaSpecs {
		if spec.Replicas == nil {
			spec.Replicas = Int32(1)
		}
		setDefaultPort(spec)
		setDefaultImage(spec)
		setDefaultRestartPolicy(spec)
	}
}
