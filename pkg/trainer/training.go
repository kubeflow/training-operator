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
	tfjobclient "github.com/tensorflow/k8s/pkg/client/clientset/versioned"
	"github.com/tensorflow/k8s/pkg/client/clientset/versioned/scheme"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	"github.com/tensorflow/k8s/pkg/apis/tensorflow/helper"
)

// TODO(jlewi): We should switch a New pattern and make trainingJob private so we can
// ensure correctness on creation.
type TrainingJob struct {
	job *tfv1alpha1.TFJob

	KubeCli kubernetes.Interface

	recorder record.EventRecorder

	Replicas []*TFReplicaSet

	TensorBoard *TBReplicaSet

	tfJobClient tfjobclient.Interface

	// in memory state of the job.
	// status is the source of truth after job struct is materialized. Changes to the status to be persisted
	// should be made here.
	status tfv1alpha1.TFJobStatus

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

func initJob(kubeCli kubernetes.Interface, tfJobClient tfjobclient.Interface, recorder record.EventRecorder, job *tfv1alpha1.TFJob) (*TrainingJob, error) {
	j := &TrainingJob{
		KubeCli:     kubeCli,
		tfJobClient: tfJobClient,
		recorder:    recorder,
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

func NewJob(kubeCli kubernetes.Interface, tfJobClient tfjobclient.Interface, recorder record.EventRecorder, job *tfv1alpha1.TFJob, config *tfv1alpha1.ControllerConfig) (*TrainingJob, error) {
	j, err := initJob(kubeCli, tfJobClient, recorder, job)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (j *TrainingJob) UID() types.UID {
	return j.job.ObjectMeta.UID
}

func (j *TrainingJob) ClusterSpec() ClusterSpec {
	clusterSpec := make(ClusterSpec)

	for _, p := range j.Replicas {
		replicaNames := make([]string, 0, *p.Spec.Replicas)

		for i := int32(0); i < *p.Spec.Replicas; i++ {
			replicaNames = append(replicaNames, fmt.Sprintf("%v:%v", p.jobName(i), *p.Spec.TFPort))
		}

		clusterSpec[strings.ToLower(string(p.Spec.TFReplicaType))] = replicaNames
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

func (j *TrainingJob) GetStatus() (tfv1alpha1.State, []*tfv1alpha1.TFReplicaStatus, error) {
	chief := j.job.Spec.TerminationPolicy.Chief
	chiefState := tfv1alpha1.ReplicaStateUnknown

	state := tfv1alpha1.StateUnknown
	replicaStatuses := make([]*tfv1alpha1.TFReplicaStatus, 0)

	// The state for each replica.
	// TODO(jlewi): We will need to modify this code if we want to allow multiples of a given type of replica.
	replicaSetStates := make(map[tfv1alpha1.TFReplicaType]tfv1alpha1.ReplicaState)

	for _, r := range j.Replicas {
		rStatus, err := r.GetStatus()
		if err != nil {
			log.Errorf("GetStatus() for %v returned error; %v", r.Spec.TFReplicaType, err)
		}

		replicaSetStates[r.Spec.TFReplicaType] = rStatus.State

		replicaStatuses = append(replicaStatuses, &rStatus)

		if string(r.Spec.TFReplicaType) == chief.ReplicaName {
			chiefState = r.GetSingleReplicaStatus(int32(chief.ReplicaIndex))
		}
	}

	if chiefState == tfv1alpha1.ReplicaStateRunning {
		state = tfv1alpha1.StateRunning
	} else if chiefState == tfv1alpha1.ReplicaStateFailed {
		state = tfv1alpha1.StateFailed
	} else if chiefState == tfv1alpha1.ReplicaStateSucceeded {
		state = tfv1alpha1.StateSucceeded
	}

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
		j.status.Phase = tfv1alpha1.TFJobPhaseFailed
		j.status.State = tfv1alpha1.StateFailed
	}

	err := func() error {
		// If the job has already started we shouldn't set it up again.
		if j.status.Phase != tfv1alpha1.TFJobPhaseNone {
			log.Warningf("Job %v has already been setup.", j.name())
			return nil
		}

		// Set defaults.
		scheme.Scheme.Default(j.job)

		err := validation.ValidateTFJobSpec(&j.job.Spec)
		if err != nil {
			return fmt.Errorf("invalid job spec: %v", err)
		}

		if err := helper.ConfigureAcceleratorsForTFJobSpec(&j.job.Spec, config.Accelerators); err != nil {
			return fmt.Errorf("ConfigureAccelerators(...) error; %v", err)
		}

		if j.job.Spec.RuntimeId == "" {
			j.job.Spec.RuntimeId = util.RandString(4)
		}
		return nil
	}()

	if err != nil {
		j.status.Reason = err.Error()
		j.status.Phase = tfv1alpha1.TFJobPhaseFailed
		j.status.State = tfv1alpha1.StateFailed
	} else {
		j.status.Phase = tfv1alpha1.TFJobPhaseCreating
		j.status.State = tfv1alpha1.StateRunning
	}
}

// setup Replicas. This creates in memory data structures corresponding to the replicas.
func (j *TrainingJob) setupReplicas() error {

	if len(j.Replicas) != len(j.job.Spec.ReplicaSpecs) {
		j.Replicas = make([]*TFReplicaSet, 0, len(j.job.Spec.ReplicaSpecs))
		for _, t := range j.job.Spec.ReplicaSpecs {
			r, err := NewTFReplicaSet(j.KubeCli, j.recorder, *t, j)
			if err != nil {
				return err
			}
			j.Replicas = append(j.Replicas, r)
		}
	}

	if j.TensorBoard == nil {
		tb, err := initTensorBoard(j.KubeCli, j)
		if err != nil {
			return err
		}
		j.TensorBoard = tb
	}

	return nil
}

func (j *TrainingJob) Delete() {
	// TODO(jlewi): Delete is what should cause us to delete the Pods.
	// we shouldn't delete the pods when the jobs finish because leaving the pods
	// allows us to get the logs from the pods after the job finishes.
	//
	log.Infof("TFJob %v deleted by the user", j.fullname())
	// TODO(jlewi): This logic is probably insufficient.
	if j.job.Status.Phase != tfv1alpha1.TFJobPhaseCleanUp {
		j.status.Phase = tfv1alpha1.TFJobPhaseCleanUp
	}

	// TODO(jlewi): Does it make sense to explicitly delete the resources? Should
	// we just rely on K8s garbage collection to delete the resources before
	// deleting TFJob?
	if cErr := j.deleteResources(); cErr != nil {
		log.Errorf("trainingJob.deleteResources() error; %v", cErr)
	}
}

// updateCRDStatus updates the job status based on TraingingJob.status.
func (j *TrainingJob) updateCRDStatus() error {
	// If the status hasn't changed then there's no reason to update the CRD.
	if reflect.DeepEqual(j.job.Status, j.status) {
		return nil
	}

	newJob := j.job
	newJob.Status = j.status
	newJob, err := j.tfJobClient.KubeflowV1alpha1().TFJobs(j.job.ObjectMeta.Namespace).Update(newJob)
	if err != nil {
		return err
	}

	j.job = newJob

	return nil
}

// reconcile tries to get the job into the desired state.
func (j *TrainingJob) Reconcile(config *tfv1alpha1.ControllerConfig) error {
	if j.job.Status.Phase == tfv1alpha1.TFJobPhaseNone {
		// The job hasn't been setup.
		j.setup(config)

		if err := j.updateCRDStatus(); err != nil {
			log.Warningf("failed to update CRD status: %v", err)
			return err
		}
	}

	// setupreplicas initializes data structures inside TrainingJob representing the replicas.
	// These are go-lang structures which aren't preserved in the APIServer. So we always need to call setupReplicas
	// unlike setup which only needs to be called once during the lifecycle of the job.
	if err := j.setupReplicas(); err != nil {
		log.Errorf("failed to create replicas: %v", err)
		j.status.Reason = fmt.Sprintf("Could not create in memory datastructures; %v", err)
		if uErr := j.updateCRDStatus(); err != nil {
			log.Warningf("Job %v; failed to update status error: %v", j.job.ObjectMeta.Name, uErr)
		}
		return err
	}

	// TODO(jlewi): Can we determine from the CRD status whether we should
	// Create the resources or not? We need to ensure the resources exist so for
	// now we always call Create.
	if j.job.Status.Phase == tfv1alpha1.TFJobPhaseCreating || j.job.Status.Phase == tfv1alpha1.TFJobPhaseRunning {
		// We call Create to make sure all the resources exist and are running.
		if cErr := j.createResources(config); cErr != nil {
			// TODO(jlewi): Should we eventually give up and mark the job as failed if we can't create the resources?
			j.status.Reason = fmt.Sprintf("Could not create job resources; %v", cErr)
			if err := j.updateCRDStatus(); err != nil {
				log.Warningf("Job %v; failed to update status error: %v", j.job.ObjectMeta.Name, err)
				return err
			}
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
			j.status.Phase = tfv1alpha1.TFJobPhaseDone
			j.status.State = tfv1alpha1.StateFailed
		} else if state == tfv1alpha1.StateSucceeded {
			log.Infof("Master succeeded Job: %v.", j.job.ObjectMeta.Name)
			j.status.Phase = tfv1alpha1.TFJobPhaseDone
			j.status.State = tfv1alpha1.StateSucceeded
		} else {
			log.V(1).Infof("Job %v status=%v", j.job.ObjectMeta.Name, util.Pformat(j.status))
		}
	}

	// If the phase changed we should update the CRD.
	if err := j.updateCRDStatus(); err != nil {
		log.Warningf("Job %v, failed to update CRD status error: %v", j.job.ObjectMeta.Name, err)
		return err
	}

	if j.job.Status.Phase == tfv1alpha1.TFJobPhaseCleanUp {
		if cErr := j.deleteResources(); cErr != nil {
			log.Errorf("Job %v trainingJob.Delete() error; %v", j.job.ObjectMeta.Name, cErr)
		}
		// j.status.SetPhase(spec.TFJobPhaseDone)
		// Return from run because we want to stop reconciling the object.
		return nil
	}

	// updateCRDStatus will update the status of the CRD with c.Status if c.Status
	// doesn't match c.Cluster.status. So you can change c.Status in order to propagate
	// changes to the CRD status.
	if err := j.updateCRDStatus(); err != nil {
		log.Warningf("Job %v; failed to update CRD status error: %v", j.job.ObjectMeta.Name, err)
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
