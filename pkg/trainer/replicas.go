package trainer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jlewi/mlkube.io/pkg/spec"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	batch "k8s.io/client-go/pkg/apis/batch/v1"
	"github.com/jlewi/mlkube.io/pkg/util"
)

// TFReplicaSet is a set of TF processes all acting as the same role (e.g. worker
type TFReplicaSet struct {
	ClientSet kubernetes.Interface
	// Job is a pointer to the TrainingJob to which this replica belongs.
	Job  *TrainingJob
	Spec spec.TfReplicaSpec
}

// TFReplicas is an interface for managing a set of replicas.
type TFReplicaSetInterface interface {
	Create() error
	Delete() error
	GetStatus() (spec.TfReplicaStatus, error)
}

// TFConfig is a struct representing the TensorFlow config. This struct is turned into an environment
// which is used by TensorFlow processes to configure themselves.
type TfConfig struct {
	// Cluster represents a TensorFlow ClusterSpec.
	// See: https://www.tensorflow.org/api_docs/python/tf/train/ClusterSpechttps://www.tensorflow.org/api_docs/python/tf/train/ClusterSpec
	Cluster ClusterSpec            `json:"cluster"`
	Task    map[string]interface{} `json:"task"`
}

func NewTFReplicaSet(clientSet kubernetes.Interface, tfReplicaSpec spec.TfReplicaSpec, job *TrainingJob) (*TFReplicaSet, error) {
	if tfReplicaSpec.TfReplicaType == spec.MASTER && *tfReplicaSpec.Replicas != 1 {
		return nil, errors.New("The MASTER must have Replicas = 1")
	}

	if tfReplicaSpec.TfPort == nil {
		return nil, errors.New("tfReplicaSpec.TfPort can't be nil.")
	}

	if tfReplicaSpec.Template == nil {
		return nil, errors.New("tfReplicaSpec.Template can't be nil.")
	}

	// Make sure the replica type is valid.
	validReplicaTypes := []spec.TfReplicaType{spec.MASTER, spec.PS, spec.WORKER}

	isValidReplicaType := false
	for _, t := range validReplicaTypes {
		if t == tfReplicaSpec.TfReplicaType {
			isValidReplicaType = true
			break
		}
	}

	if !isValidReplicaType {
		return nil, fmt.Errorf("tfReplicaSpec.TfReplicaType is %v but must be one of %v", tfReplicaSpec.TfReplicaType, validReplicaTypes)
	}
	return &TFReplicaSet{
		ClientSet: clientSet,
		Job:       job,
		Spec:      tfReplicaSpec,
	}, nil
}

// Labels returns the labels for this replica set.
func (s *TFReplicaSet) Labels() KubernetesLabels {
	return KubernetesLabels(map[string]string{
		"mlkube.io": "",
		"job_type":  string(s.Spec.TfReplicaType),
		// runtime_id is set by Job.setup, which is called after the TfReplicaSet is created.
		// this is why labels aren't a member variable.
		"runtime_id": s.Job.job.Spec.RuntimeId})
}

func (s *TFReplicaSet) Create() error {
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		taskLabels := s.Labels()
		taskLabels["task_index"] = fmt.Sprintf("%v", index)

		// Create the service.
		service := &v1.Service{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   s.jobName(index),
				Labels: taskLabels,
			},
			Spec: v1.ServiceSpec{
				Selector: taskLabels,
				Ports: []v1.ServicePort{
					{
						Name: "tf-port",
						Port: *s.Spec.TfPort,
					},
				},
			},
		}

		log.Infof("Creating Service: %v", service.ObjectMeta.Name)
		_, err := s.ClientSet.CoreV1().Services(NAMESPACE).Create(service)

		// If the job already exists do nothing.
		if err != nil {
			if k8s_errors.IsAlreadyExists(err) {
				log.Infof("Service %v already exists.", s.jobName(index))
			} else {
				return err
			}
		}

		// Configure the TFCONFIG environment variable.
		//
		// TODO(jlewi): We would need to add support for hyperparameter jobs to support CMLE
		// hyperparameter tuning.
		tfConfig := TfConfig{
			Cluster: s.Job.ClusterSpec(),
			Task: map[string]interface{}{
				"type":  strings.ToLower(string(s.Spec.TfReplicaType)),
				"index": index,
			},
		}
		tfConfigJson, err := json.Marshal(tfConfig)
		if err != nil {
			log.Errorf("Job: %v serializing tfConfig: %v return error; %v", s.Job.job.Metadata.Name, util.Pformat(tfConfig), err)
			return err
		}

		// Make a copy of the template because we will modify it below.
		// TODO(jlewi): I don't fully understand why this works but setting Template: *s.Spec.Template
		// leads to TF_CONFIG being added multiples as an environment variable.
		newPodSpecTemplate := *s.Spec.Template
		// TODO(jlewi): We need to set environment variable TF_CONFIG.
		newJ := &batch.Job{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   s.jobName(index),
				Labels: taskLabels,
			},
			Spec: batch.JobSpec{
				Completions: proto.Int32(1),
				Parallelism: proto.Int32(1),
				Template:    newPodSpecTemplate,
			},
		}

		if newJ.Spec.Template.ObjectMeta.Labels == nil {
			newJ.Spec.Template.ObjectMeta.Labels = make(map[string]string)
		}

		// Pods need to be tagged with the labels.
		for k, v := range taskLabels {
			newJ.Spec.Template.ObjectMeta.Labels[k] = v
		}

		// Add TF_CONFIG environment variable.
		for i, _ := range newJ.Spec.Template.Spec.Containers {
			// We can't get c in the loop variable because that would be by value so our modifications
			// wouldn't have any effect.
			c := &newJ.Spec.Template.Spec.Containers[i]
			if spec.ContainerName(c.Name) != spec.TENSORFLOW {
				continue
			}
			if len(c.Env) == 0 {
				c.Env = make([]v1.EnvVar, 0)
			}
			c.Env = append(c.Env, v1.EnvVar{
				Name:  "TF_CONFIG",
				Value: string(tfConfigJson),
			})
		}

		log.Infof("Creating Job: %v", newJ.ObjectMeta.Name)
		_, err = s.ClientSet.BatchV1().Jobs(NAMESPACE).Create(newJ)

		// If the job already exists do nothing.
		if err != nil {
			if k8s_errors.IsAlreadyExists(err) {
				log.Infof("%v already exists.", s.jobName(index))

			} else {
				return err
			}
		}
	}
	return nil
}

// Delete deletes the replicas
func (s *TFReplicaSet) Delete() error {
	selector, err := s.Labels().ToSelector()
	if err != nil {
		return err
	}

	failures := false

	options := meta_v1.ListOptions{
		LabelSelector: selector,
	}

	err = s.ClientSet.BatchV1().Jobs(NAMESPACE).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		log.Errorf("There was a problem deleting the jobs; %v", err)
		failures = true
	}

	// We need to delete the completed pods.
	err = s.ClientSet.CoreV1().Pods(NAMESPACE).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		log.Errorf("There was a problem deleting the pods; %v", err)
		failures = true
	}

	// Services doesn't support DeleteCollection so we delete them individually.
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		err = s.ClientSet.CoreV1().Services(NAMESPACE).Delete(s.jobName(index), &meta_v1.DeleteOptions{})

		if err != nil {
			log.Errorf("Error deleteing service %v; %v", s.jobName(index), err)
			failures = true
		}
	}

	if failures {
		return errors.New("Some of the replicas resources could not be deleted")
	}
	return nil
}

// replicaStatusFromPodList returns a status from a list of pods for a job.
func replicaStatusFromPodList(l v1.PodList, name spec.ContainerName) spec.ReplicaState {
	log.V(1).Infof("Get replicaStatus from PodList: %v", util.Pformat(l))
	var latest *v1.Pod
	for _, i := range l.Items {
		log.Infof("latest: %v", latest)
		if latest == nil {
			latest = &i
			continue
		}
		if latest.Status.StartTime.Before(*i.Status.StartTime) {
			latest = &i
		}
	}

	if latest == nil {
		return spec.ReplicaStateRunning
	}

	found := false
	var terminated v1.ContainerState

	for _, i := range latest.Status.ContainerStatuses {
		if i.Name != string(name) {
			continue
		}
		for _, s := range []v1.ContainerState{i.State, i.LastTerminationState} {
			if s.Terminated == nil {
				continue
			}

			if !found {
				found = true
				terminated = s
				continue
			}

			if terminated.Terminated.StartedAt.Before(s.Terminated.StartedAt) {
				terminated = s
			}
		}
	}

	if !found {
		log.Warningf("No container named: %v found for pod; assuming POD is running", name)
		return spec.ReplicaStateRunning
	}

	log.V(1).Infof("Terminated container: %v", terminated)

	if terminated.Terminated.ExitCode == 0 {
		return spec.ReplicaStateSucceeded
	}

	if isRetryableTerminationState(terminated.Terminated) {
		// Since its a retryable error just return RUNNING.
		// We can just let Kubernetes restart the container to retry.
		return spec.ReplicaStateRunning
	}

	return spec.ReplicaStateFailed
}

// Status returns the status of the replica set.
func (s *TFReplicaSet) GetStatus() (spec.TfReplicaStatus, error) {

	status := spec.TfReplicaStatus{
		TfReplicaType:  s.Spec.TfReplicaType,
		State:          spec.ReplicaStateUnknown,
		ReplicasStates: make(map[spec.ReplicaState]int),
	}

	increment := func(state spec.ReplicaState) {
		v, ok := status.ReplicasStates[state]
		if ok {
			status.ReplicasStates[state] = v + 1
		} else {
			status.ReplicasStates[state] = 1
		}
	}

	for index := int32(0); index < *s.Spec.Replicas; index++ {

		j, err := s.ClientSet.BatchV1().Jobs(NAMESPACE).Get(s.jobName(index), meta_v1.GetOptions{})

		if err != nil {
			increment(spec.ReplicaStateUnknown)
			continue
		}

		if j.Status.Succeeded >= 1 {
			increment(spec.ReplicaStateSucceeded)
			continue
		}

		labels := s.Labels()
		labels["task_index"] = fmt.Sprintf("%v", index)
		selector, err := labels.ToSelector()
		if err != nil {
			log.Errorf("labels.ToSelector() error; %v", err)
			increment(spec.ReplicaStateFailed)
			continue
		}

		log.V(1).Infof("Using filter %v", selector)
		// TODO(jlewi): Handle errors. We need to get the pod and looking at recent container exits.
		l, err := s.ClientSet.CoreV1().Pods(NAMESPACE).List(meta_v1.ListOptions{
			// TODO(jlewi): Why isn't the label selector working?
			LabelSelector: selector,
		})

		if err != nil {
			// TODO(jlewi): Are there errors that should be treated as retryable errors?
			increment(spec.ReplicaStateFailed)
			continue
		}

		status := replicaStatusFromPodList(*l, spec.TENSORFLOW)
		increment(status)
	}

	// Determine the overall status for the replica set based on the status of the individual
	// replicas.
	// If any of the replicas failed mark the set as failed.
	if _, ok := status.ReplicasStates[spec.ReplicaStateFailed]; ok {
		status.State = spec.ReplicaStateFailed
		return status, nil
	}

	// If any replicas are RUNNING mark it as RUNNING.
	if _, ok := status.ReplicasStates[spec.ReplicaStateRunning]; ok {
		status.State = spec.ReplicaStateRunning
		return status, nil
	}

	// If all of the replicas succeeded consider it success.
	if v, ok := status.ReplicasStates[spec.ReplicaStateSucceeded]; ok && int32(v) == *s.Spec.Replicas {
		status.State = spec.ReplicaStateSucceeded
		return status, nil
	}

	return status, nil
}

func (s *TFReplicaSet) jobName(index int32) string {
	return fmt.Sprintf("%v-%v-%v", strings.ToLower(string(s.Spec.TfReplicaType)), s.Job.job.Spec.RuntimeId, index)
}
