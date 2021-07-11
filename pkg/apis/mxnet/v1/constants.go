package v1

import commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

const (
	// DefaultPortName is name of the port used to communicate between scheduler and
	// servers & workers.
	DefaultPortName = "mxjob-port"
	// DefaultContainerName is the name of the MXJob container.
	DefaultContainerName = "mxnet"
	// DefaultPort is default value of the port.
	DefaultPort = 9091
	// DefaultRestartPolicy is default RestartPolicy for MXReplicaSpec.
	DefaultRestartPolicy = commonv1.RestartPolicyNever
)
