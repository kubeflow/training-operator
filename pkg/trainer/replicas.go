package trainer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sErrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	tfv1alpha1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	"github.com/tensorflow/k8s/pkg/util/k8sutil"
	// TOOO(jlewi): Rename to apiErrors
	"github.com/tensorflow/k8s/pkg/apis/tensorflow/helper"
	"github.com/tensorflow/k8s/pkg/util"
)

const (
	SuccessfulCreateReason = "SuccessfulCreate"
	FailedCreateReason     = "FailedCreate"
)

// TFReplicaSet is a set of TF processes all acting as the same role (e.g. worker
type TFReplicaSet struct {
	ClientSet kubernetes.Interface
	recorder  record.EventRecorder
	// Job is a pointer to the TrainingJob to which this replica belongs.
	Job  *TrainingJob
	Spec tfv1alpha1.TFReplicaSpec
}

// TFReplicas is an interface for managing a set of replicas.
type TFReplicaSetInterface interface {
	Create() error
	Delete() error
	GetStatus() (tfv1alpha1.TFReplicaStatus, error)
}

// TFConfig is a struct representing the TensorFlow config. This struct is turned into an environment
// which is used by TensorFlow processes to configure themselves.
type TFConfig struct {
	// Cluster represents a TensorFlow ClusterSpec.
	// See: https://www.tensorflow.org/api_docs/python/tf/train/ClusterSpechttps://www.tensorflow.org/api_docs/python/tf/train/ClusterSpec
	Cluster ClusterSpec `json:"cluster"`
	Task    TaskSpec    `json:"task"`
	// Environment is used by tensorflow.contrib.learn.python.learn in versions <= 1.3
	// TODO(jlewi): I don't think it is used in versions TF >- 1.4. So we can eventually get rid of it.
	Environment string `json:"environment"`
}

func NewTFReplicaSet(clientSet kubernetes.Interface, recorder record.EventRecorder, tfReplicaSpec tfv1alpha1.TFReplicaSpec, job *TrainingJob) (*TFReplicaSet, error) {
	if tfReplicaSpec.TFReplicaType == tfv1alpha1.MASTER && *tfReplicaSpec.Replicas != 1 {
		return nil, errors.New("The MASTER must have Replicas = 1")
	}

	if tfReplicaSpec.TFPort == nil {
		return nil, errors.New("tfReplicaSpec.TFPort can't be nil.")
	}

	if tfReplicaSpec.Template == nil && tfReplicaSpec.TFReplicaType != tfv1alpha1.PS {
		return nil, fmt.Errorf("tfReplicatfv1alpha1.Template can't be nil for replica type %v.", tfReplicaSpec.TFReplicaType)
	}

	// Make sure the replica type is valid.
	validReplicaTypes := []tfv1alpha1.TFReplicaType{tfv1alpha1.MASTER, tfv1alpha1.PS, tfv1alpha1.WORKER}

	isValidReplicaType := false
	for _, t := range validReplicaTypes {
		if t == tfReplicaSpec.TFReplicaType {
			isValidReplicaType = true
			break
		}
	}

	if !isValidReplicaType {
		return nil, fmt.Errorf("tfReplicaSpec.TFReplicaType is %v but must be one of %v", tfReplicaSpec.TFReplicaType, validReplicaTypes)
	}

	return &TFReplicaSet{
		ClientSet: clientSet,
		recorder:  recorder,
		Job:       job,
		Spec:      tfReplicaSpec,
	}, nil
}

// Labels returns the labels for this replica set.
func (s *TFReplicaSet) Labels() KubernetesLabels {
	return KubernetesLabels(map[string]string{
		"kubeflow.org": "",
		"job_type":     string(s.Spec.TFReplicaType),
		// runtime_id is set by Job.setup, which is called after the TFReplicaSet is created.
		// this is why labels aren't a member variable.
		"runtime_id":  s.Job.job.Spec.RuntimeId,
		"tf_job_name": s.Job.job.ObjectMeta.Name})
}

func (s *TFReplicaSet) Create(config *tfv1alpha1.ControllerConfig) error {
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		taskLabels := s.Labels()
		taskLabels["task_index"] = fmt.Sprintf("%v", index)

		// Create the service.
		service := &v1.Service{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   s.jobName(index),
				Labels: taskLabels,
				OwnerReferences: []meta_v1.OwnerReference{
					helper.AsOwner(s.Job.job),
				},
			},
			Spec: v1.ServiceSpec{
				Selector: taskLabels,
				Ports: []v1.ServicePort{
					{
						Name: "tf-port",
						Port: *s.Spec.TFPort,
					},
				},
			},
		}

		log.Infof("Creating Service: %v", service.ObjectMeta.Name)
		createdService, err := s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Create(service)

		// If the job already exists do nothing.
		if err != nil {
			if k8s_errors.IsAlreadyExists(err) {
				log.Infof("Service %v already exists.", s.jobName(index))
			} else {
				s.recorder.Eventf(s.Job.job, v1.EventTypeWarning, FailedCreateReason, "Error creating: %v", err)
				return k8sErrors.NewAggregate([]error{fmt.Errorf("Creating service %v returned error.", createdService.ObjectMeta.Name), err})
			}
		} else {
			s.recorder.Eventf(s.Job.job, v1.EventTypeNormal, SuccessfulCreateReason, "Created service: %v", createdService.Name)
		}

		// Configure the TFCONFIG environment variable.
		tfConfig := TFConfig{
			Cluster: s.Job.ClusterSpec(),
			Task: TaskSpec{
				Type:  strings.ToLower(string(s.Spec.TFReplicaType)),
				Index: int(index),
			},
			// We need to set environment to cloud  otherwise it will default to local which isn't what we want.
			Environment: "cloud",
		}

		tfConfigJson, err := json.Marshal(tfConfig)
		if err != nil {
			log.Errorf("Job: %v serializing tfConfig: %v return error; %v", s.Job.job.ObjectMeta.Name, util.Pformat(tfConfig), err)
			return err
		}

		// Make a copy of the template because we will modify it below. .
		newPodSpecTemplate := s.Spec.Template.DeepCopy()

		newJ := &batch.Job{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   s.jobName(index),
				Labels: taskLabels,
				OwnerReferences: []meta_v1.OwnerReference{
					helper.AsOwner(s.Job.job),
				},
			},
			Spec: batch.JobSpec{
				Completions: proto.Int32(1),
				Parallelism: proto.Int32(1),
				Template:    *newPodSpecTemplate,
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
			if tfv1alpha1.ContainerName(c.Name) != tfv1alpha1.TENSORFLOW {
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
		createdJob, err := s.ClientSet.BatchV1().Jobs(s.Job.job.ObjectMeta.Namespace).Create(newJ)

		// If the job already exists do nothing.
		if err != nil {
			if k8s_errors.IsAlreadyExists(err) {
				log.Infof("%v already exists.", s.jobName(index))

			} else {
				s.recorder.Eventf(s.Job.job, v1.EventTypeWarning, FailedCreateReason, "Error creating: %v", err)
				return k8sErrors.NewAggregate([]error{fmt.Errorf("Creating Job %v returned error.", createdJob.ObjectMeta.Name), err})
			}
		} else {
			s.recorder.Eventf(s.Job.job, v1.EventTypeNormal, SuccessfulCreateReason, "Created job: %v", createdJob.Name)
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

	log.V(1).Infof("Deleting Jobs namespace=%v selector=%v", s.Job.job.ObjectMeta.Namespace, selector)
	err = s.ClientSet.BatchV1().Jobs(s.Job.job.ObjectMeta.Namespace).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		log.Errorf("There was a problem deleting the jobs; %v", err)
		failures = true
	}

	// We need to delete the completed pods.
	log.V(1).Infof("Deleting Pods namespace=%v selector=%v", s.Job.job.ObjectMeta.Namespace, selector)
	err = s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		log.Errorf("There was a problem deleting the pods; %v", err)
		failures = true
	}

	// Services doesn't support DeleteCollection so we delete them individually.
	// TODO(jlewi): We should check if this has changed with K8s 1.8 or other releases.
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		log.V(1).Infof("Deleting Service %v:%v", s.Job.job.ObjectMeta.Namespace, s.jobName((index)))
		err = s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Delete(s.jobName(index), &meta_v1.DeleteOptions{})

		if err != nil {
			log.Errorf("Error deleting service %v; %v", s.jobName(index), err)
			failures = true
		}
	}

	// If the ConfigMap for the default parameter server exists, we delete it
	log.V(1).Infof("Get ConfigMaps %v:%v", s.Job.job.ObjectMeta.Namespace, s.defaultPSConfigMapName())
	_, err = s.ClientSet.CoreV1().ConfigMaps(s.Job.job.ObjectMeta.Namespace).Get(s.defaultPSConfigMapName(), meta_v1.GetOptions{})
	if err != nil {
		if !k8sutil.IsKubernetesResourceNotFoundError(err) {
			log.Errorf("Error deleting ConfigMap %v; %v", s.defaultPSConfigMapName(), err)
			failures = true
		}
	} else {
		log.V(1).Infof("Delete ConfigMaps %v:%v", s.Job.job.ObjectMeta.Namespace, s.defaultPSConfigMapName())
		err = s.ClientSet.CoreV1().ConfigMaps(s.Job.job.ObjectMeta.Namespace).Delete(s.defaultPSConfigMapName(), &meta_v1.DeleteOptions{})
		if err != nil {
			log.Errorf("There was a problem deleting the ConfigMaps; %v", err)
			failures = true
		}
	}

	if failures {
		return errors.New("Some of the replicas resources could not be deleted")
	}
	return nil
}

// replicaStatusFromPodList returns a status from a list of pods for a job.
func replicaStatusFromPodList(l v1.PodList, name tfv1alpha1.ContainerName) tfv1alpha1.ReplicaState {
	log.V(1).Infof("Get replicaStatus from PodList: %v", util.Pformat(l))
	var latest *v1.Pod
	for _, i := range l.Items {
		if latest == nil {
			latest = &i
			continue
		}
		if latest.Status.StartTime.Before(i.Status.StartTime) {
			latest = &i
		}
	}

	if latest == nil {
		return tfv1alpha1.ReplicaStateRunning
	}

	var tfState v1.ContainerState

	for _, i := range latest.Status.ContainerStatuses {
		if i.Name != string(name) {
			continue
		}

		// We need to decide whether to use the current state or the previous termination state.
		tfState = i.State

		// If the container previously terminated we will look at the termination to decide whether it is a retryable
		// or permanenent error.
		if i.LastTerminationState.Terminated != nil {
			tfState = i.LastTerminationState
		}
	}

	if tfState.Running != nil || tfState.Waiting != nil {
		return tfv1alpha1.ReplicaStateRunning
	}

	if tfState.Terminated != nil {
		if tfState.Terminated.ExitCode == 0 {
			return tfv1alpha1.ReplicaStateSucceeded
		}

		if isRetryableTerminationState(tfState.Terminated) {
			// Since its a retryable error just return RUNNING.
			// We can just let Kubernetes restart the container to retry.
			return tfv1alpha1.ReplicaStateRunning
		}

		return tfv1alpha1.ReplicaStateFailed
	}

	return tfv1alpha1.ReplicaStateUnknown
}

func (s *TFReplicaSet) GetSingleReplicaStatus(index int32) tfv1alpha1.ReplicaState {
	j, err := s.ClientSet.BatchV1().Jobs(s.Job.job.ObjectMeta.Namespace).Get(s.jobName(index), meta_v1.GetOptions{})

	if err != nil {
		return tfv1alpha1.ReplicaStateUnknown
	}

	if j.Status.Succeeded >= 1 {
		return tfv1alpha1.ReplicaStateSucceeded
	}

	labels := s.Labels()
	labels["task_index"] = fmt.Sprintf("%v", index)
	selector, err := labels.ToSelector()
	if err != nil {
		log.Errorf("labels.ToSelector() error; %v", err)
		return tfv1alpha1.ReplicaStateFailed
	}

	// TODO(jlewi): Handle errors. We need to get the pod and looking at recent container exits.
	l, err := s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).List(meta_v1.ListOptions{
		// TODO(jlewi): Why isn't the label selector working?
		LabelSelector: selector,
	})

	if err != nil {
		// TODO(jlewi): Are there errors that should be treated as retryable errors?
		return tfv1alpha1.ReplicaStateFailed
	}

	status := replicaStatusFromPodList(*l, tfv1alpha1.TENSORFLOW)
	return status
}

// Status returns the status of the replica set.
func (s *TFReplicaSet) GetStatus() (tfv1alpha1.TFReplicaStatus, error) {
	status := tfv1alpha1.TFReplicaStatus{
		TFReplicaType:  s.Spec.TFReplicaType,
		State:          tfv1alpha1.ReplicaStateUnknown,
		ReplicasStates: make(map[tfv1alpha1.ReplicaState]int),
	}

	increment := func(state tfv1alpha1.ReplicaState) {
		v, ok := status.ReplicasStates[state]
		if ok {
			status.ReplicasStates[state] = v + 1
		} else {
			status.ReplicasStates[state] = 1
		}
	}

	for index := int32(0); index < *s.Spec.Replicas; index++ {
		increment(s.GetSingleReplicaStatus(index))
	}

	// Determine the overall status for the replica set based on the status of the individual
	// replicas.
	// If any of the replicas failed mark the set as failed.
	if _, ok := status.ReplicasStates[tfv1alpha1.ReplicaStateFailed]; ok {
		status.State = tfv1alpha1.ReplicaStateFailed
		return status, nil
	}

	// If any replicas are RUNNING mark it as RUNNING.
	if _, ok := status.ReplicasStates[tfv1alpha1.ReplicaStateRunning]; ok {
		status.State = tfv1alpha1.ReplicaStateRunning
		return status, nil
	}

	// If all of the replicas succeeded consider it success.
	if v, ok := status.ReplicasStates[tfv1alpha1.ReplicaStateSucceeded]; ok && int32(v) == *s.Spec.Replicas {
		status.State = tfv1alpha1.ReplicaStateSucceeded
		return status, nil
	}

	return status, nil
}

func (s *TFReplicaSet) jobName(index int32) string {
	// Truncate tfjob name to 40 characters
	// The whole job name should be compliant with the DNS_LABEL spec, up to a max length of 63 characters
	// Thus jobname(40 chars)-replicaType(6 chars)-runtimeId(4 chars)-index(4 chars), also leaving some spaces
	// See https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
	return fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", s.Job.job.ObjectMeta.Name), strings.ToLower(string(s.Spec.TFReplicaType)), s.Job.job.Spec.RuntimeId, index)
}

func (s *TFReplicaSet) defaultPSConfigMapName() string {
	return fmt.Sprintf("cm-ps-%v", s.Job.job.Spec.RuntimeId)
}
