package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/core/v1"
)

const (
	CRDKind       = "TfJob"
	CRDKindPlural = "tfjobs"
	CRDGroup      = "tensorflow.org"
	CRDVersion    = "v1alpha1"
	// Value of the APP label that gets applied to a lot of entities.
	AppLabel = "tensorflow-job"
	// Defaults for the Spec
	TfPort   = 2222
	Replicas = 1
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=tfjob

// TFJob describes tfjob info
type TfJob struct {
	metav1.TypeMeta    `json:",inline"`
	metav1.ObjectMeta  `json:"metadata,omitempty"`
	Spec   TfJobSpec   `json:"spec"`
	Status TfJobStatus `json:"status"`
}

type TfJobSpec struct {
	// TODO(jlewi): Can we we get rid of this and use some value from Kubernetes or a random ide.
	RuntimeId string

	//TensorBoardSpec specifies the configuration to start a TensorBoard deployment
	TensorBoard *TensorBoardSpec `json:"tensorboard"`

	// ReplicaSpecs specifies the TF replicas to run.
	ReplicaSpecs []*TfReplicaSpec `json:"replicaSpecs"`

	// TfImage defines the tensorflow docker image that should be used for Tensorboard
	// and the default parameter server
	TfImage string `json:"tfImage,omitempty"`

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


// TfReplicaType determines how a set of TF processes are handled.
type TfReplicaType string

const (
	MASTER TfReplicaType = "MASTER"
	PS     TfReplicaType = "PS"
	WORKER TfReplicaType = "WORKER"
)

// ContainerName is an enum for expected containers.
type ContainerName string

const (
	TENSORFLOW     ContainerName = "tensorflow"
	DefaultTFImage string        = "tensorflow/tensorflow:1.3.0"
)

// TODO(jlewi): We probably want to add a name field. This would allow us to have more than 1 type of each worker.
// This might be useful if you wanted to have a separate set of workers to do eval.
type TfReplicaSpec struct {
	// Replicas is the number of desired replicas.
	// This is a pointer to distinguish between explicit zero and unspecified.
	// Defaults to 1.
	// More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller
	// +optional
	Replicas *int32              `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	Template *v1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`
	// TfPort is the port to use for TF services.
	TfPort *int32 `json:"tfPort,omitempty" protobuf:"varint,1,opt,name=tfPort"`
	TfReplicaType `json:"tfReplicaType"`
	// IsDefaultPS denotes if the parameter server should use the default grpc_tensorflow_server
	IsDefaultPS bool
}

type TensorBoardSpec struct {
	//Location of TensorFlow event files
	LogDir       string           `json:"logDir"`
	Volumes      []v1.Volume      `json:"volumes"`
	VolumeMounts []v1.VolumeMount `json:"volumeMounts"`
	ServiceType  v1.ServiceType   `json:"serviceType"`
}

type TfJobPhase string

const (
	TfJobPhaseNone     TfJobPhase = ""
	TfJobPhaseCreating TfJobPhase = "Creating"
	TfJobPhaseRunning  TfJobPhase = "Running"
	TfJobPhaseCleanUp  TfJobPhase = "CleanUp"
	TfJobPhaseFailed   TfJobPhase = "Failed"
	TfJobPhaseDone     TfJobPhase = "Done"
)

type State string

const (
	StateUnknown   State = "Unknown"
	StateRunning   State = "Running"
	StateSucceeded State = "Succeeded"
	StateFailed    State = "Failed"
)

type TfJobStatus struct {
	// Phase is the TfJob running phase
	Phase  TfJobPhase `json:"phase"`
	Reason string     `json:"reason"`

	// State indicates the state of the job.
	State State `json:"state"`

	// ReplicaStatuses specifies the status of each TF replica.
	ReplicaStatuses []*TfReplicaStatus `json:"replicaStatuses"`
}

type ReplicaState string

const (
	ReplicaStateUnknown   ReplicaState = "Unknown"
	ReplicaStateStarting  ReplicaState = "Starting"
	ReplicaStateRunning   ReplicaState = "Running"
	ReplicaStateFailed    ReplicaState = "Failed"
	ReplicaStateSucceeded ReplicaState = "Succeeded"
)

type TfReplicaStatus struct {
	TfReplicaType `json:"tf_replica_type"`

	// State is the overall state of the replica
	State ReplicaState `json:"state"`

	// ReplicasStates provides the number of replicas in each status.
	ReplicasStates map[ReplicaState]int
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=tfjobs

// TfJobList is a list of TfJobs clusters.
type TfJobList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	Metadata metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of TfJobs
	Items []TfJob `json:"items"`
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
