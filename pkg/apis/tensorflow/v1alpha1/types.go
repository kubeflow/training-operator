package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CRDKind       = "tfjob"
	CRDKindPlural = "tfjobs"
	CRDGroup      = "kubeflow.org"
	CRDVersion    = "v1alpha1"
	// Value of the APP label that gets applied to a lot of entities.
	AppLabel = "tensorflow-job"
	// Defaults for the Spec
	TFPort   = 2222
	Replicas = 1
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=tfjob

// TFJob describes tfjob info
type TFJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TFJobSpec   `json:"spec"`
	Status            TFJobStatus `json:"status"`
}

type TFJobSpec struct {
	// TODO(jlewi): Can we we get rid of this and use some value from Kubernetes or a random ide.
	RuntimeId string

	//TensorBoardSpec specifies the configuration to start a TensorBoard deployment
	TensorBoard *TensorBoardSpec `json:"tensorboard"`

	// ReplicaSpecs specifies the TF replicas to run.
	ReplicaSpecs []*TFReplicaSpec `json:"replicaSpecs"`

	// TFImage defines the tensorflow docker image that should be used for Tensorboard
	// and the default parameter server
	TFImage string `json:"tfImage,omitempty"`

	// TerminationPolicy specifies the condition that the tfjob should be considered finished.
	TerminationPolicy *TerminationPolicySpec `json:"terminationPolicy,omitempty"`
}

type TerminationPolicySpec struct {
	// Chief policy waits for a particular process (which is the chief) to exit.
	Chief *ChiefSpec `json:"chief,omitempty"`
}

type ChiefSpec struct {
	ReplicaName  string `json:"replicaName"`
	ReplicaIndex int    `json:"replicaIndex"`
}

// TFReplicaType determines how a set of TF processes are handled.
type TFReplicaType string

const (
	MASTER TFReplicaType = "MASTER"
	PS     TFReplicaType = "PS"
	WORKER TFReplicaType = "WORKER"
)

// ContainerName is an enum for expected containers.
type ContainerName string

const (
	TENSORFLOW     ContainerName = "tensorflow"
	DefaultTFImage string        = "tensorflow/tensorflow:1.3.0"
)

// TODO(jlewi): We probably want to add a name field. This would allow us to have more than 1 type of each worker.
// This might be useful if you wanted to have a separate set of workers to do eval.
type TFReplicaSpec struct {
	// Replicas is the number of desired replicas.
	// This is a pointer to distinguish between explicit zero and unspecified.
	// Defaults to 1.
	// More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller
	// +optional
	Replicas *int32              `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	Template *v1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`
	// TFPort is the port to use for TF services.
	TFPort        *int32 `json:"tfPort,omitempty" protobuf:"varint,1,opt,name=tfPort"`
	TFReplicaType `json:"tfReplicaType"`
}

type TensorBoardSpec struct {
	//Location of TensorFlow event files
	LogDir       string           `json:"logDir"`
	Volumes      []v1.Volume      `json:"volumes"`
	VolumeMounts []v1.VolumeMount `json:"volumeMounts"`
	ServiceType  v1.ServiceType   `json:"serviceType"`
}

type TFJobPhase string

const (
	TFJobPhaseNone     TFJobPhase = ""
	TFJobPhaseCreating TFJobPhase = "Creating"
	TFJobPhaseRunning  TFJobPhase = "Running"
	TFJobPhaseCleanUp  TFJobPhase = "CleanUp"
	TFJobPhaseFailed   TFJobPhase = "Failed"
	TFJobPhaseDone     TFJobPhase = "Done"
)

type State string

const (
	StateUnknown   State = "Unknown"
	StateRunning   State = "Running"
	StateSucceeded State = "Succeeded"
	StateFailed    State = "Failed"
)

type TFJobStatus struct {
	// Phase is the TFJob running phase
	Phase  TFJobPhase `json:"phase"`
	Reason string     `json:"reason"`

	// State indicates the state of the job.
	State State `json:"state"`

	// ReplicaStatuses specifies the status of each TF replica.
	ReplicaStatuses []*TFReplicaStatus `json:"replicaStatuses"`
}

type ReplicaState string

const (
	ReplicaStateUnknown   ReplicaState = "Unknown"
	ReplicaStateRunning   ReplicaState = "Running"
	ReplicaStateFailed    ReplicaState = "Failed"
	ReplicaStateSucceeded ReplicaState = "Succeeded"
)

type TFReplicaStatus struct {
	TFReplicaType `json:"tf_replica_type"`

	// State is the overall state of the replica
	State ReplicaState `json:"state"`

	// ReplicasStates provides the number of replicas in each status.
	ReplicasStates map[ReplicaState]int
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=tfjobs

// TFJobList is a list of TFJobs clusters.
type TFJobList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of TFJobs
	Items []TFJob `json:"items"`
}

type ControllerConfig struct {
	// Accelerators is a map from the name of the accelerator to the config for that accelerator.
	// This should match the value specified as a container limit.
	// e.g. alpha.kubernetes.io/nvidia-gpu
	Accelerators map[string]AcceleratorConfig

	// Path to the file containing the grpc server source
	GrpcServerFilePath string
}

// AcceleratorVolume represents a host path that must be mounted into
// each container that needs to use GPUs.
type AcceleratorVolume struct {
	Name      string
	HostPath  string
	MountPath string
}

type AcceleratorConfig struct {
	Volumes []AcceleratorVolume
	EnvVars []EnvironmentVariableConfig
}

type EnvironmentVariableConfig struct {
	Name  string
	Value string
}
