// training is a package for managing TensorFlow training jobs.
package trainer

import (
	"fmt"

	"reflect"

	log "github.com/golang/glog"
	"github.com/tensorflow/k8s/pkg/util"

	"strings"

	tfv1alpha1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	"github.com/tensorflow/k8s/pkg/apis/tensorflow/validation"
	"github.com/tensorflow/k8s/pkg/client/clientset/versioned/scheme"
	tfjobclient "github.com/tensorflow/k8s/pkg/client/clientset/versioned"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/tensorflow/k8s/pkg/apis/tensorflow/helper"
)

// TODO(jlewi): We should switch a New pattern and make trainingJob private so we can
// ensure correctness on creation.
type TrainingJob struct {
	job *tfv1alpha1.TfJob

	KubeCli kubernetes.Interface

	Replicas []*TFReplicaSet

	TensorBoard *TBReplicaSet

	tfJobClient tfjobclient.Interface

	// in memory state of the job.
	// status is the source of truth after job struct is materialized. Changes to the status to be persisted
	// should be made here.
	status tfv1alpha1.TfJobStatus

	memberCounter int
}

// ClusterSpec represents a cluster TensorFlow specification.
// https://www.tensorflow.org/deploy/distributed#create_a_tftrainclusterspec_to_describe_the_cluster
// It is a map from job names to network addresses.
type ClusterSpec map[string][]string

type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

func initJob(kubeCli kubernetes.Interface, tfJobClient tfjobclient.Interface, job *tfv1alpha1.TfJob) (*TrainingJob, error) {
	j := &TrainingJob{
		KubeCli:     kubeCli,
		tfJobClient: tfJobClient,
		Replicas:    make([]*TFReplicaSet, 0),
		TensorBoard: nil,
		job:         job,
		status:      *job.Status.DeepCopy(),
	}

	return j, nil
}

func initTensorBoard(clientSet kubernetes.Interface, tj *TrainingJob) (*TBReplicaSet, error) {
	if tj.job.Spec.TensorBoard != nil {
		return NewTBReplicaSet(clientSet, *tj.job.Spec.TensorBoard, tj)
	}
	return nil, nil
}

func NewJob(kubeCli kubernetes.Interface, tfJobClient tfjobclient.Interface, job *tfv1alpha1.TfJob, config *tfv1alpha1.ControllerConfig) (*TrainingJob, error) {
	j, err := initJob(kubeCli, tfJobClient, job)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (j *TrainingJob) ClusterSpec() ClusterSpec {
	clusterSpec := make(ClusterSpec)

	for _, p := range j.Replicas {
		replicaNames := make([]string, 0, *p.Spec.Replicas)

		for i := int32(0); i < *p.Spec.Replicas; i++ {
			replicaNames = append(replicaNames, fmt.Sprintf("%v:%v", p.jobName(i), *p.Spec.TfPort))
		}

		clusterSpec[strings.ToLower(string(p.Spec.TfReplicaType))] = replicaNames
	}

	return clusterSpec
}

// createResources creates all the replicas and TensorBoard if requested
func (j *TrainingJob) createResources(config *tfv1alpha1.ControllerConfig) error {
	for _, r := range j.Replicas {
		if err := r.Create(config); err != nil {
			return err
		}
	}

	if j.TensorBoard != nil {
		if err := j.TensorBoard.Create(); err != nil {
			return err
		}
	}

	return nil
}

// deleteResources deletes the replicas and TensorBoard it it was created
func (j *TrainingJob) deleteResources() error {
	for _, r := range j.Replicas {
		if err := r.Delete(); err != nil {
			return err
		}
	}

	if j.TensorBoard != nil {
		if err := j.TensorBoard.Delete(); err != nil {
			return err
		}
	}
	return nil
}

func (j *TrainingJob) GetStatus() (tfv1alpha1.State, []*tfv1alpha1.TfReplicaStatus, error) {
	state := tfv1alpha1.StateUnknown
	replicaStatuses := make([]*tfv1alpha1.TfReplicaStatus, 0)

	// The state for each replica.
	// TODO(jlewi): We will need to modify this code if we want to allow multiples of a given type of replica.
	replicaSetStates := make(map[tfv1alpha1.TfReplicaType]tfv1alpha1.ReplicaState)

	for _, r := range j.Replicas {
		rStatus, err := r.GetStatus()
		if err != nil {
			log.Errorf("GetStatus() for %v returned error; %v", r.Spec.TfReplicaType, err)
		}

		replicaSetStates[r.Spec.TfReplicaType] = rStatus.State

		replicaStatuses = append(replicaStatuses, &rStatus)

		// If any replicas are failed mark job as failed.
		if rStatus.State == tfv1alpha1.ReplicaStateFailed {
			state = tfv1alpha1.StateFailed
		}
	}

	if v, ok := replicaSetStates[tfv1alpha1.MASTER]; ok && v == tfv1alpha1.ReplicaStateSucceeded {
		state = tfv1alpha1.StateSucceeded
		return state, replicaStatuses, nil
	}

	if v, ok := replicaSetStates[tfv1alpha1.MASTER]; ok && v == tfv1alpha1.ReplicaStateFailed {
		state = tfv1alpha1.StateFailed
		return state, replicaStatuses, nil
	}

	state = tfv1alpha1.StateRunning
	return state, replicaStatuses, nil
}

// isRetryableTerminationState returns true if a container terminated in a state
// that we consider retryable.
func isRetryableTerminationState(s *v1.ContainerStateTerminated) bool {
	// TODO(jlewi): Need to match logic in
	// https://cs.corp.google.com/piper///depot/google3/cloud/ml/beta/job/training_job_state_util.cc?l=88
	if s.Reason == "OOMKilled" {
		// If the user's process causes an OOM and Docker kills the container,
		// the termination reason of ContainerState will be specified to
		// 'OOMKilled'. In this case, we can't assume this to be a retryable error.
		//
		// This check should happen before checking the termination log, since
		// if the container terminated with an OOM, the termination log may not
		// be written.
		return false
	}

	// TODO(jlewi): Should we use the exit code reported in the termination
	// log message and not the ExitCode reported by the container.

	if s.ExitCode >= 0 && s.ExitCode <= 127 {
		// For the exit_code in [0, 127]:
		//   0 means success,
		//   1 - 127 corresponds to permanent user errors.
		// We don't want to retry for both cases.
		// More info about exit status can be found in:
		// https://www.gnu.org/software/bash/manual/html_node/Exit-Status.html
		return false
	}

	// For the remaining cases that exit_code from workers that doesn't
	// fall into [0, 127]. They can be:
	//   137 corresponds to SIGKILL,
	//   143 corresponds to SIGTERM,
	//   other values that have undefined behavior.
	// We treat them as internal errors for now and all the internal errors
	// will be retired.
	return true
}

func (j *TrainingJob) masterName() string {
	return fmt.Sprintf("master-%v-0", j.job.Spec.RuntimeId)
}

// setup the training job.
func (j *TrainingJob) setup(config *tfv1alpha1.ControllerConfig) {
	if j.job == nil {
		j.status.Reason = "Internal error; setup failed; job is missing spec."
		j.status.Phase = tfv1alpha1.TfJobPhaseFailed
		j.status.State = tfv1alpha1.StateFailed
	}

	err := func() error {
		// If the job has already started we shouldn't set it up again.
		if j.status.Phase != tfv1alpha1.TfJobPhaseNone {
			log.Warningf("Job %v has already been setup.", j.name())
			return nil
		}

		// Set defaults.
		scheme.Scheme.Default(j.job)

		err := validation.ValidateTfJobSpec(&j.job.Spec)
		if err != nil {
			return fmt.Errorf("invalid job spec: %v", err)
		}

		for _, t := range j.job.Spec.ReplicaSpecs {
			r, err := NewTFReplicaSet(j.KubeCli, *t, j)
			if err != nil {
				return err
			}
			j.Replicas = append(j.Replicas, r)
		}

		tb, err := initTensorBoard(j.KubeCli, j)
		if err != nil {
			return err
		}
		j.TensorBoard = tb

		if err := helper.ConfigureAcceleratorsForTfJobSpec(&j.job.Spec, config.Accelerators); err != nil {
			return fmt.Errorf("ConfigureAccelerators(...) error; %v", err)
		}

		if j.job.Spec.RuntimeId == "" {
			j.job.Spec.RuntimeId = util.RandString(4)
		}
		return nil
	}()

	if err != nil {
		j.status.Reason = err.Error()
		j.status.Phase = tfv1alpha1.TfJobPhaseFailed
		j.status.State = tfv1alpha1.StateFailed
	} else {
		j.status.Phase = tfv1alpha1.TfJobPhaseCreating
		j.status.State = tfv1alpha1.StateRunning
	}
}

func (j *TrainingJob) Delete() {
	// TODO(jlewi): Delete is what should cause us to delete the Pods.
	// we shouldn't delete the pods when the jobs finish because leaving the pods
	// allows us to get the logs from the pods after the job finishes.
	//
	log.Infof("TfJob %v deleted by the user", j.fullname())
	// TODO(jlewi): This logic is probably insufficient.
	if j.job.Status.Phase != tfv1alpha1.TfJobPhaseCleanUp {
		j.status.Phase = tfv1alpha1.TfJobPhaseCleanUp
	}

	// TODO(jlewi): Does it make sense to explicitly delete the resources? Should
	// we just rely on K8s garbage collection to delete the resources before
	// deleting TfJob?
	if cErr := j.deleteResources(); cErr != nil {
		log.Errorf("trainingJob.deleteResources() error; %v", cErr)
	}
}

// updateTPRStatus updates the job status based on TraingingJob.status.
func (j *TrainingJob) updateTPRStatus() error {
	// If the status hasn't changed then there's no reason to update the TPR.
	if reflect.DeepEqual(j.job.Status, j.status) {
		return nil
	}

	newJob := j.job
	newJob.Status = j.status
	newJob, err := j.tfJobClient.TensorflowV1alpha1().TfJobs(j.job.ObjectMeta.Namespace).Update(newJob)
	if err != nil {
		return err
	}

	j.job = newJob

	return nil
}

// reconcile tries to get the job into the desired state.
func (j *TrainingJob) Reconcile(config *tfv1alpha1.ControllerConfig) error {
	if j.status.Phase == tfv1alpha1.TfJobPhaseNone {
		// The job hasn't been setup.
		j.setup(config)

		if err := j.updateTPRStatus(); err != nil {
			log.Warningf("failed to update TPR status: %v", err)
			return err
		}
	}

	// TODO(jlewi): Can we determine from the CRD status whether we should
	// Create the resources or not? We need to ensure the resources exist so for
	// now we always call Create.
	if j.job.Status.Phase == tfv1alpha1.TfJobPhaseCreating || j.job.Status.Phase == tfv1alpha1.TfJobPhaseRunning {
		// We call Create to make sure all the resources exist and are running.
		if cErr := j.createResources(config); cErr != nil {
			log.Errorf("trainingJobCreateReplicas() error; %v", cErr)
			return cErr
		}

		state, replicaStatuses, err := j.GetStatus()

		j.status.ReplicaStatuses = replicaStatuses
		if err != nil {
			log.Errorf("GetStatus() for job %v returned error: %v", j.job.ObjectMeta.Name, err)
			return err
		}
		// TODO(jlewi): We should update the Phase if we detect the job is done.
		if state == tfv1alpha1.StateFailed {
			log.Errorf("Master failed Job: %v.", j.job.ObjectMeta.Name)
			j.status.Phase = tfv1alpha1.TfJobPhaseDone
			j.status.State = tfv1alpha1.StateFailed
		} else if state == tfv1alpha1.StateSucceeded {
			log.Infof("Master succeeded Job: %v.", j.job.ObjectMeta.Name)
			j.status.Phase = tfv1alpha1.TfJobPhaseDone
			j.status.State = tfv1alpha1.StateSucceeded
		} else {
			log.V(1).Infof("Job %v status=%v", j.job.ObjectMeta.Name, util.Pformat(j.status))
		}
	}

	// If the phase changed we should update the TPR.
	if err := j.updateTPRStatus(); err != nil {
		log.Warningf("Job %v, failed to update TPR status error: %v", j.job.ObjectMeta.Name, err)
		return err
	}

	if j.job.Status.Phase == tfv1alpha1.TfJobPhaseCleanUp {
		if cErr := j.deleteResources(); cErr != nil {
			log.Errorf("Job %v trainingJob.Delete() error; %v", j.job.ObjectMeta.Name, cErr)
		}
		// j.status.SetPhase(spec.TfJobPhaseDone)
		// Return from run because we want to stop reconciling the object.
		return nil
	}

	// updateTPRStatus will update the status of the TPR with c.Status if c.Status
	// doesn't match c.Cluster.status. So you can change c.Status in order to propagate
	// changes to the TPR status.
	if err := j.updateTPRStatus(); err != nil {
		log.Warningf("Job %v; failed to update TPR status error: %v", j.job.ObjectMeta.Name, err)
		return err
	}

	return nil
}

func (j *TrainingJob) name() string {
	return j.job.ObjectMeta.GetName()
}

// fullname returns the namespace and name for the job.
func (j *TrainingJob) fullname() string {
	return j.job.ObjectMeta.GetNamespace() + ":" + j.job.ObjectMeta.GetName()
}
