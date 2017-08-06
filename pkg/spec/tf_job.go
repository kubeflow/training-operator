package spec

import (
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"mlkube.io/pkg/util"
)

const (
	TPRKind        = "tf-job"
	TPRKindPlural  = "tfjobs"
	TPRGroup       = "mlkube.io"
	TPRVersion     = "v1beta1"
	TPRDescription = "TensorFlow training job"

	// Value of the APP label that gets applied to a lot of entities.
	AppLabel = "tensorflow-job"
)

var ()

func TPRName() string {
	return fmt.Sprintf("%s.%s", TPRKind, TPRGroup)
}

type TfJob struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec            TfJobSpec         `json:"spec"`
	Status          TfJobStatus       `json:"status"`
}

func (c *TfJob) AsOwner() metav1.OwnerReference {
	trueVar := true
	// TODO: In 1.6 this is gonna be "k8s.io/kubernetes/pkg/apis/meta/v1"
	// Both api.OwnerReference and metatypes.OwnerReference are combined into that.
	return metav1.OwnerReference{
		APIVersion: c.APIVersion,
		Kind:       c.Kind,
		Name:       c.Metadata.Name,
		UID:        c.Metadata.UID,
		Controller: &trueVar,
	}
}

// TODO(jlewi): Need to define the actual configuration for the TensorFlow TfJob.
type TfJobSpec struct {
	// TODO(jlewi): Can we we get rid of this and use some value from Kubernetes or a random ide.
	RuntimeId string

	//TensorBoardSpec specifies the configuration to start a TensorBoard deployment
	TensorBoard *TensorBoardSpec `json:"tensorboard"`

	// ReplicaSpecs specifies the TF replicas to run.
	ReplicaSpecs []*TfReplicaSpec `json:"replicaSpecs"`
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
	TENSORFLOW ContainerName = "tensorflow"
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
	TfPort        *int32 `json:"tfPort,omitempty" protobuf:"varint,1,opt,name=tfPort"`
	TfReplicaType `json:"tfReplicaType"`
}

type TensorBoardSpec struct {
	//Location of TensorFlow event files
	LogDir       string           `json:"logDir"`
	Volumes      []v1.Volume      `json:"volumes"`
	VolumeMounts []v1.VolumeMount `json:"volumeMounts"`
	ServiceType  v1.ServiceType   `json:"serviceType"`
}

func (c *TfJobSpec) Validate() error {
	// Check that each replica has a TensorFlow container.
	for _, r := range c.ReplicaSpecs {
		found := false
		if r.Template == nil {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}
		for _, c := range r.Template.Spec.Containers {
			if c.Name == string(TENSORFLOW) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Replica type %v is missing a container named %v", r.TfReplicaType, TENSORFLOW)
		}
	}
	return nil
}

// ConfigureAccelerators adds any accelerator specific configuration to the pods.
func (c *TfJobSpec) ConfigureAccelerators(accelerators map[string]AcceleratorConfig) error {
	for _, r := range c.ReplicaSpecs {
		if r.Template == nil {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}
		for i, c := range r.Template.Spec.Containers {
			if c.Name == string(TENSORFLOW) {
				// Identify the accelerators attached to this container.
				a := map[string]AcceleratorConfig{}

				lists := []v1.ResourceList{c.Resources.Limits, c.Resources.Requests}
				for _, resources := range lists {
					for name, _ := range resources {

						if _, ok := accelerators[string(name)]; !ok {
							continue
						}

						// Add the expected mounts to the pods.
						a[string(name)] = accelerators[string(name)]
					}
				}

				// Add accelerator information to the pod.
				for _, config := range a {
					for _, v := range config.Volumes {
						r.Template.Spec.Volumes = append(r.Template.Spec.Volumes,
							v1.Volume{
								Name: v.Name,
								VolumeSource: v1.VolumeSource{
									HostPath: &v1.HostPathVolumeSource{
										Path: v.HostPath,
									},
								},
							})
						c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
							Name:      v.Name,
							MountPath: v.MountPath,
						})
					}

					for _, envVar := range config.EnvVars {
						c.Env = append(c.Env, v1.EnvVar{
							Name:  envVar.Name,
							Value: envVar.Value,
						})
					}
				}
				r.Template.Spec.Containers[i] = c
				break
			}
		}
	}
	return nil
}

// Cleanup cleans up user passed spec, e.g. defaulting, transforming fields.
// TODO: move this to admission controller
func (c *TfJobSpec) Cleanup() {
	// TODO(jlewi): Add logic to cleanup user provided spec; e.g. by filling in defaults.
	// We should have default container images so user doesn't have to provide these.
}

type TfJobPhase string

const (
	TfJobPhaseNone     TfJobPhase = ""
	TfJobPhaseCreating            = "Creating"
	TfJobPhaseRunning             = "Running"
	TfJobPhaseCleanUp             = "CleanUp"
	TfJobPhaseFailed              = "Failed"
	TfJobPhaseDone                = "Done"
)

type TfJobCondition struct {
	Type TfJobConditionType `json:"type"`

	Reason string `json:"reason"`

	TransitionTime string `json:"transitionTime"`
}

type TfJobConditionType string

// TODO(jlewi): Need to define appropriate conditions and get rid of the ones we don't need.
const (
	TfJobConditionReady = "Ready"

	TfJobConditionRemovingDeadMember = "RemovingDeadMember"

	TfJobConditionRecovering = "Recovering"

	TfJobConditionScalingUp   = "ScalingUp"
	TfJobConditionScalingDown = "ScalingDown"

	TfJobConditionUpgrading = "Upgrading"
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

	// ControlPuased indicates the operator pauses the control of the cluster.
	// TODO(jlewi): I think we can get rid of ControlPaued.
	ControlPaused bool `json:"controlPaused"`

	// Condition keeps ten most recent cluster conditions
	Conditions []TfJobCondition `json:"conditions"`

	// State indicates the state of the job.
	State State `json:"state"`

	// ReplicaStatuses specifies the status of each TF replica.
	ReplicaStatuses []*TfReplicaStatus `json:"replicaStatuses"`
}

type ReplicaState string

const (
	ReplicaStateUnknown   ReplicaState = "Unknown"
	ReplicaStateStarting               = "Starting"
	ReplicaStateRunning                = "Running"
	ReplicaStateFailed                 = "Failed"
	ReplicaStateSucceeded              = "Succeeded"
)

type TfReplicaStatus struct {
	TfReplicaType `json:"tf_replica_type"`
	// State is the overall state of the replica
	State ReplicaState `json:"state"`

	// ReplicasStates provides the number of replicas in each status.
	ReplicasStates map[ReplicaState]int
}

func (cs TfJobStatus) Copy() TfJobStatus {
	newCS := TfJobStatus{}
	b, err := json.Marshal(cs)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &newCS)
	if err != nil {
		panic(err)
	}
	return newCS
}

func (cs *TfJobStatus) IsFailed() bool {
	if cs == nil {
		return false
	}
	return cs.State == StateFailed
}

func (cs *TfJobStatus) SetPhase(p TfJobPhase) {
	cs.Phase = p
}

func (cs *TfJobStatus) PauseControl() {
	cs.ControlPaused = true
}

func (cs *TfJobStatus) Control() {
	cs.ControlPaused = false
}

func (cs *TfJobStatus) SetReason(r string) {
	cs.Reason = r
}

func (cs *TfJobStatus) SetState(s State) {
	cs.State = s
}

// TODO(jlewi): Get rid of the append methods that we don't need
func (cs *TfJobStatus) AppendScalingDownCondition(from, to int) {
	c := TfJobCondition{
		Type:           TfJobConditionScalingDown,
		Reason:         scalingReason(from, to),
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	cs.appendCondition(c)
}

func (cs *TfJobStatus) AppendRecoveringCondition() {
	c := TfJobCondition{
		Type:           TfJobConditionRecovering,
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	cs.appendCondition(c)
}

func (cs *TfJobStatus) AppendUpgradingCondition(to string, member string) {
	reason := fmt.Sprintf("upgrading cluster member %s version to %v", member, to)

	c := TfJobCondition{
		Type:           TfJobConditionUpgrading,
		Reason:         reason,
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	cs.appendCondition(c)
}

func (cs *TfJobStatus) AppendRemovingDeadMember(name string) {
	reason := fmt.Sprintf("removing dead member %s", name)

	c := TfJobCondition{
		Type:           TfJobConditionRemovingDeadMember,
		Reason:         reason,
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	cs.appendCondition(c)
}

func (cs *TfJobStatus) SetReadyCondition() {
	c := TfJobCondition{
		Type:           TfJobConditionReady,
		TransitionTime: time.Now().Format(time.RFC3339),
	}

	if len(cs.Conditions) == 0 {
		cs.appendCondition(c)
		return
	}

	lastc := cs.Conditions[len(cs.Conditions)-1]
	if lastc.Type == TfJobConditionReady {
		return
	}
	cs.appendCondition(c)
}

func (cs *TfJobStatus) appendCondition(c TfJobCondition) {
	cs.Conditions = append(cs.Conditions, c)
	if len(cs.Conditions) > 10 {
		cs.Conditions = cs.Conditions[1:]
	}
}

func scalingReason(from, to int) string {
	return fmt.Sprintf("Current cluster size: %d, desired cluster size: %d", from, to)
}
