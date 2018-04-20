// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trainer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sErrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	tfv1alpha1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha1"
	"github.com/kubeflow/tf-operator/pkg/util/k8sutil"
	// TOOO(jlewi): Rename to apiErrors
	"github.com/kubeflow/tf-operator/pkg/apis/tensorflow/helper"
	"github.com/kubeflow/tf-operator/pkg/util"
)

const (
	SuccessfulCreateReason = "SuccessfulCreate"
	FailedCreateReason     = "FailedCreate"
	indexField             = "replica"
)

// TFReplicaSet is a set of TF processes all acting as the same role (e.g. worker
type TFReplicaSet struct {
	ClientSet kubernetes.Interface
	recorder  record.EventRecorder
	// Job is a pointer to the TrainingJob to which this replica belongs.
	Job  *TrainingJob
	Spec tfv1alpha1.TFReplicaSpec

	// contextLogger is a logger to use for logging information about this replica.
	contextLogger *log.Entry
}

// TFReplicaSetInterface is an interface for managing a set of replicas.
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

// NewTFReplicaSet returns TFReplicaSet object for existing replica
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
		contextLogger: log.WithFields(log.Fields{
			"job_type":    string(tfReplicaSpec.TFReplicaType),
			"runtime_id":  job.job.Spec.RuntimeId,
			"tf_job_name": job.job.ObjectMeta.Name,
			// We use job to match the key used in controller.go
			// In controller.go we log the key used with the workqueue.
			"job": job.job.ObjectMeta.Namespace + "/" + job.job.ObjectMeta.Name,
		}),
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

// LabelsByIndex returns the labels for a pod in this replica set.
func (s *TFReplicaSet) LabelsByIndex(index int32) KubernetesLabels {
	labels := s.Labels()
	labels["task_index"] = fmt.Sprintf("%v", index)
	return labels
}

// CreateServiceWithIndex will create a new service with specify index
func (s *TFReplicaSet) CreateServiceWithIndex(index int32) (*v1.Service, error) {
	taskLabels := s.LabelsByIndex(index)

	// Create the service.
	service := &v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   s.genName(index),
			Labels: taskLabels,
			OwnerReferences: []meta_v1.OwnerReference{
				helper.AsOwner(s.Job.job),
			},
		},
		Spec: v1.ServiceSpec{
			Selector: taskLabels,
			// We use headless services here, because we don't need load balancing
			// since there is a single pod that is the backend for each service.
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name: "tf-port",
					Port: *s.Spec.TFPort,
				},
			},
		},
	}

	s.contextLogger.WithFields(log.Fields{
		indexField: index,
	}).Infof("Creating service: %v", service.ObjectMeta.Name)
	return s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Create(service)
}

// CreatePodWithIndex will create a new pod with specify index
func (s *TFReplicaSet) CreatePodWithIndex(index int32) (*v1.Pod, error) {
	taskLabels := s.LabelsByIndex(index)

	pod := &v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        s.genPodName(index),
			Labels:      taskLabels,
			Annotations: map[string]string{},
			OwnerReferences: []meta_v1.OwnerReference{
				helper.AsOwner(s.Job.job),
			},
		},
		Spec: *s.Spec.Template.Spec.DeepCopy(),
	}

	pod.Spec.SchedulerName = s.Job.SchedulerName()

	// copy labels and annotations to pod from tfjob
	for k, v := range s.Spec.Template.Labels {
		if _, ok := pod.Labels[k]; !ok {
			pod.Labels[k] = v
		}
	}

	for k, v := range s.Spec.Template.Annotations {
		if _, ok := pod.Annotations[k]; !ok {
			pod.Annotations[k] = v
		}
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
		s.contextLogger.Errorf("Job: %v serializing tfConfig: %v return error; %v", s.Job.job.ObjectMeta.Name, util.Pformat(tfConfig), err)
		return nil, err
	}

	// Add TF_CONFIG environment variable.
	for i := range pod.Spec.Containers {
		// We can't get c in the loop variable because that would be by value so our modifications
		// wouldn't have any effect.
		c := &pod.Spec.Containers[i]
		if c.Name != tfv1alpha1.DefaultTFContainer {
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

	s.contextLogger.WithFields(log.Fields{
		indexField: index,
	}).Infof("Creating pod: %v", pod.ObjectMeta.Name)
	return s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).Create(pod)
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

	s.contextLogger.Infof("Deleting Jobs namespace=%v selector=%v", s.Job.job.ObjectMeta.Namespace, selector)
	err = s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		s.contextLogger.Errorf("There was a problem deleting the jobs; %v", err)
		failures = true
	}

	// We need to delete the completed pods.
	s.contextLogger.Infof("Deleting Pods namespace=%v selector=%v", s.Job.job.ObjectMeta.Namespace, selector)
	err = s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		s.contextLogger.Errorf("There was a problem deleting the pods; %v", err)
		failures = true
	}

	// Services doesn't support DeleteCollection so we delete them individually.
	// TODO(jlewi): We should check if this has changed with K8s 1.8 or other releases.
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		s.contextLogger.WithFields(log.Fields{
			indexField: index,
		}).Infof("Deleting Service %v:%v", s.Job.job.ObjectMeta.Namespace, s.genName((index)))
		err = s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Delete(s.genName(index), &meta_v1.DeleteOptions{})

		if err != nil {
			s.contextLogger.Errorf("Error deleting service %v; %v", s.genName(index), err)
			failures = true
		}
	}

	// If the ConfigMap for the default parameter server exists, we delete it
	s.contextLogger.Infof("Get ConfigMaps %v:%v", s.Job.job.ObjectMeta.Namespace, s.defaultPSConfigMapName())
	_, err = s.ClientSet.CoreV1().ConfigMaps(s.Job.job.ObjectMeta.Namespace).Get(s.defaultPSConfigMapName(), meta_v1.GetOptions{})
	if err != nil {
		if !k8sutil.IsKubernetesResourceNotFoundError(err) {
			s.contextLogger.Errorf("Error deleting ConfigMap %v; %v", s.defaultPSConfigMapName(), err)
			failures = true
		}
	} else {
		s.contextLogger.Infof("Delete ConfigMaps %v:%v", s.Job.job.ObjectMeta.Namespace, s.defaultPSConfigMapName())
		err = s.ClientSet.CoreV1().ConfigMaps(s.Job.job.ObjectMeta.Namespace).Delete(s.defaultPSConfigMapName(), &meta_v1.DeleteOptions{})
		if err != nil {
			s.contextLogger.Errorf("There was a problem deleting the ConfigMaps; %v", err)
			failures = true
		}
	}

	if failures {
		return errors.New("Some of the replicas resources could not be deleted")
	}
	return nil
}

// replicaStatusFromPodList returns a status from a list of pods for a job.
func replicaStatusFromPodList(l v1.PodList, name string) tfv1alpha1.ReplicaState {
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
		if i.Name != name {
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

		// TODO(wenzhel): Should consider to return more info about pod to users.
		return tfv1alpha1.ReplicaStateFailed
	}

	return tfv1alpha1.ReplicaStateUnknown
}

// GetSingleReplicaStatus returns status for a single replica
func (s *TFReplicaSet) GetSingleReplicaStatus(index int32) tfv1alpha1.ReplicaState {
	labels := s.LabelsByIndex(index)
	selector, err := labels.ToSelector()
	if err != nil {
		s.contextLogger.Errorf("labels.ToSelector() error; %v", err)
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

	status := replicaStatusFromPodList(*l, tfv1alpha1.DefaultTFContainer)
	return status
}

// GetStatus returns the status of the replica set.
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

// SyncPods will try to check current pods for this TFReplicaSet and try to make it as desired.
func (s *TFReplicaSet) SyncPods() error {
	for index := int32(0); index < *s.Spec.Replicas; index++ {

		// Label to get all pods of this TFReplicaType + index
		labels := s.LabelsByIndex(index)

		labelSelector, err := labels.ToSelector()
		if err != nil {
			return err
		}

		// Filter the unactive pods
		fieldSelector := fmt.Sprintf("status.phase!=%s", string(v1.PodFailed))

		options := meta_v1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		}

		// List to get pods
		pl, err := s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).List(options)
		if err != nil {
			return err
		}

		if len(pl.Items) == 0 {
			s.contextLogger.Infof("Job %v missing pod for replica %v index %v, creating a new one.", s.Job.name(), string(s.Spec.TFReplicaType), index)
			// Create the pod
			createdPod, err := s.CreatePodWithIndex(index)

			// If the pod already exists do nothing.
			if err != nil {
				if k8s_errors.IsAlreadyExists(err) {
					s.contextLogger.Infof("Pod: %v already exists.", createdPod.ObjectMeta.Name)
					continue
				}
				s.recorder.Eventf(s.Job.job, v1.EventTypeWarning, FailedCreateReason, "Error creating: %v", err)
				return k8sErrors.NewAggregate([]error{fmt.Errorf("Creating pod %v returned error.", createdPod.ObjectMeta.Name), err})
			}

			s.recorder.Eventf(s.Job.job, v1.EventTypeNormal, SuccessfulCreateReason, "Created pod: %v", createdPod.Name)
			continue
		}

		if err != nil {
			// TODO: handing this error
			continue
		}
	}

	return nil
}

// SyncServices will try to check current services for this TFReplicaSet and try to make it as desired.
func (s *TFReplicaSet) SyncServices() error {
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		_, err := s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Get(s.genName(index), meta_v1.GetOptions{})
		if err != nil && k8s_errors.IsNotFound(err) {
			s.contextLogger.Infof("Service: %v not found, create new one.", s.genName(index))
			// Create the service
			createdService, err := s.CreateServiceWithIndex(index)

			// If the service already exists do nothing.
			if err != nil {
				if k8s_errors.IsAlreadyExists(err) {
					s.contextLogger.Infof("Service: %v already exists.", s.genName(index))
					continue
				}
				s.recorder.Eventf(s.Job.job, v1.EventTypeWarning, FailedCreateReason, "Error creating: %v", err)
				return k8sErrors.NewAggregate([]error{fmt.Errorf("Creating Service %v returned error.", createdService.ObjectMeta.Name), err})
			}

			s.recorder.Eventf(s.Job.job, v1.EventTypeNormal, SuccessfulCreateReason, "Created Service: %v", createdService.Name)
			continue
		}

		if err != nil {
			// TODO: handing this error
			continue
		}
	}

	return nil
}

// genName generates the name which is concatenation of jabName, TFReplicaType, job RunId and index
func (s *TFReplicaSet) genName(index int32) string {
	// Truncate tfjob name to 40 characters
	// The whole job name should be compliant with the DNS_LABEL spec, up to a max length of 63 characters
	// Thus genName(40 chars)-replicaType(6 chars)-runtimeId(4 chars)-index(4 chars), also leaving some spaces
	// See https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
	return fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", s.Job.job.ObjectMeta.Name), strings.ToLower(string(s.Spec.TFReplicaType)), s.Job.job.Spec.RuntimeId, index)
}

// genPodName generate a new pod name with random string
func (s *TFReplicaSet) genPodName(index int32) string {
	return s.genName(index) + "-" + util.RandString(5)
}

//  defaultPSConfigMapName returns the map default PS configuration map name using job's runtimeId
func (s *TFReplicaSet) defaultPSConfigMapName() string {
	return fmt.Sprintf("cm-ps-%v", s.Job.job.Spec.RuntimeId)
}
