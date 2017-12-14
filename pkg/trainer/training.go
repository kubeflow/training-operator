// training is a package for managing TensorFlow training jobs.
package trainer

import (
	"fmt"

	"reflect"

	log "github.com/golang/glog"
	"github.com/tensorflow/k8s/pkg/spec"
	"github.com/tensorflow/k8s/pkg/util"
	"github.com/tensorflow/k8s/pkg/util/k8sutil"
	"github.com/tensorflow/k8s/pkg/util/retryutil"

	"math"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

var (
	reconcileInterval = 8 * time.Second
)

type jobEventType string

const (
	eventDeleteJob jobEventType = "Delete"
	eventModifyJob jobEventType = "Modify"
)

type jobEvent struct {
	typ jobEventType
	// TODO(jlewi): Rename cluster to job.
	cluster *spec.TfJob
}

// TODO(jlewi): We should switch a New pattern and make trainingJob private so we can
// ensure correctness on creation.
type TrainingJob struct {
	job *spec.TfJob

	KubeCli kubernetes.Interface

	Replicas []*TFReplicaSet

	TensorBoard *TBReplicaSet

	tfJobClient k8sutil.TfJobClient

	// in memory state of the job.
	// status is the source of truth after job struct is materialized. Changes to the status to be persisted
	// should be made here.
	status spec.TfJobStatus

	memberCounter int

	// eventCh is used to provide Kubernetes events for a particular cluster that need to be handled.
	eventCh chan *jobEvent

	// stopCh is a channel used to communicate that the cluster needs to be stopped.
	stopCh chan struct{}
}

// ClusterSpec represents a cluster TensorFlow specification.
// https://www.tensorflow.org/deploy/distributed#create_a_tftrainclusterspec_to_describe_the_cluster
// It is a map from job names to network addressess.
type ClusterSpec map[string][]string

type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

func initJob(kubeCli kubernetes.Interface, tfJobClient k8sutil.TfJobClient, job *spec.TfJob, stopC <-chan struct{}, wg *sync.WaitGroup) (*TrainingJob, error) {
	j := &TrainingJob{
		KubeCli:     kubeCli,
		tfJobClient: tfJobClient,
		Replicas:    make([]*TFReplicaSet, 0),
		TensorBoard: nil,
		job:         job,
		eventCh:     make(chan *jobEvent, 100),
		stopCh:      make(chan struct{}),
		status:      job.Status.Copy(),
	}

	return j, nil
}

func initTensorBoard(clientSet kubernetes.Interface, tj *TrainingJob) (*TBReplicaSet, error) {
	if tj.job.Spec.TensorBoard != nil {
		return NewTBReplicaSet(clientSet, *tj.job.Spec.TensorBoard, tj)
	}
	return nil, nil
}

func NewJob(kubeCli kubernetes.Interface, tfJobClient k8sutil.TfJobClient, job *spec.TfJob, stopC <-chan struct{}, wg *sync.WaitGroup, config *spec.ControllerConfig) (*TrainingJob, error) {
	j, err := initJob(kubeCli, tfJobClient, job, stopC, wg)
	if err != nil {
		return nil, err
	}
	// Increment the wait group which the controller uses to monitor the job processing.
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := j.setup(config); err != nil {
			log.Errorf("TfJob failed to setup: %v", err)
			if j.status.Phase != spec.TfJobPhaseFailed {
				j.status.SetReason(err.Error())
				j.status.SetPhase(spec.TfJobPhaseFailed)
				if err := j.updateTPRStatus(); err != nil {
					log.Errorf("failed to update cluster phase (%v): %v", spec.TfJobPhaseFailed, err)
				}
			}
			return
		}
		j.run(config, stopC)
	}()

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
func (j *TrainingJob) createResources(config *spec.ControllerConfig) error {
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

// TODO(jlewi): We can probably delete this.

//func replicaSetStatusToProto(r *TFReplicaSet, status *TFReplicaSetStatus) *tpb.TFReplicaSetStatus {
//
//	p := &tpb.TFReplicaSetStatus{
//		State: status.State.Enum(),
//		// Type: r.Spec.TfReplicaTypeProcess.Type,
//		ReplicaStates: make([]*tpb.TFReplicaSetStatus_ReplicaStates, 0),
//	}
//
//	for state, count := range status.ReplicasStates {
//		p.ReplicaStates = append(p.ReplicaStates, &tpb.TFReplicaSetStatus_ReplicaStates{
//			State: state.Enum(),
//			NumReplicas: proto.Int(count),
//		})
//	}
//	return p
//}

func (j *TrainingJob) GetStatus() (spec.State, []*spec.TfReplicaStatus, error) {
	state := spec.StateUnknown
	replicaStatuses := make([]*spec.TfReplicaStatus, 0)

	// The state for each replica.
	// TODO(jlewi): We will need to modify this code if we want to allow multiples of a given type of replica.
	replicaSetStates := make(map[spec.TfReplicaType]spec.ReplicaState)

	for _, r := range j.Replicas {
		rStatus, err := r.GetStatus()
		if err != nil {
			log.Errorf("GetStatus() for %v returned error; %v", r.Spec.TfReplicaType, err)
		}

		replicaSetStates[r.Spec.TfReplicaType] = rStatus.State

		replicaStatuses = append(replicaStatuses, &rStatus)

		// If any replicas are failed mark job as failed.
		if rStatus.State == spec.ReplicaStateFailed {
			state = spec.StateFailed
		}
	}

	if v, ok := replicaSetStates[spec.MASTER]; ok && v == spec.ReplicaStateSucceeded {
		state = spec.StateSucceeded
		return state, replicaStatuses, nil
	}

	if v, ok := replicaSetStates[spec.MASTER]; ok && v == spec.ReplicaStateFailed {
		state = spec.StateFailed
		return state, replicaStatuses, nil
	}

	state = spec.StateRunning
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
func (j *TrainingJob) setup(config *spec.ControllerConfig) error {
	if j.job == nil {
		return fmt.Errorf("job.Spec can't be nil")
	}

	err := j.job.Spec.SetDefaults(config.TfImage)
	if err != nil {
		return fmt.Errorf("there was a problem setting defaults for job spec: %v", err)
	}

	err = j.job.Spec.Validate()
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

	if err := j.job.Spec.ConfigureAccelerators(config.Accelerators); err != nil {
		return fmt.Errorf("ConfigureAccelerators(...) error; %v", err)
	}

	if j.job.Spec.RuntimeId == "" {
		j.job.Spec.RuntimeId = util.RandString(4)
	}

	var shouldCreateCluster bool
	switch j.status.Phase {
	case spec.TfJobPhaseNone:
		shouldCreateCluster = true
		//case spec.TfJobPhaseCreating:
		//	return errCreatedCluster
	case spec.TfJobPhaseRunning:
		shouldCreateCluster = false
	case spec.TfJobPhaseFailed:
		shouldCreateCluster = false
	default:
		return fmt.Errorf("unexpected TfJob phase: %s", j.status.Phase)
	}

	if shouldCreateCluster {
		return j.triggerCreatePhase()
	}
	return nil
}

// triggerCreatePhase sets the phase to TfJobPhaseCreating additional resource creation happens in TrainingJob.run
// TODO(jlewi): Need to reconcile this function copied from the etcd core operator OS code with the pattern
// for the TF job. What exactly do we want to do during the Create job phase? Right now the create method
// is called on each invocation of reconcile in run to ensure all the required resources exist. Maybe there's
// a better way?
func (j *TrainingJob) triggerCreatePhase() error {
	j.status.SetPhase(spec.TfJobPhaseCreating)

	if err := j.updateTPRStatus(); err != nil {
		return fmt.Errorf("cluster create: failed to update TfJob phase (%v): %v", spec.TfJobPhaseCreating, err)
	}
	log.Infof("Creating job: %v with Spec (%#v), Status (%#v)", j.job.Metadata.Name, j.job.Spec, j.job.Status)

	return nil
}

func (j *TrainingJob) Delete() {
	// Delete doesn't actually delete any resources. It just sends an event which will be processed by the run
	// method.
	j.send(&jobEvent{typ: eventDeleteJob})
}

// TODO(jlewi): This is sending a clusterEvent to the channel. I think these are events
// coming from the cluster code and not k8s events.
func (j *TrainingJob) send(ev *jobEvent) {
	select {
	case j.eventCh <- ev:
		l, ecap := len(j.eventCh), cap(j.eventCh)
		if l > int(float64(ecap)*0.8) {
			log.Warningf("eventCh buffer is almost full [%d/%d]", l, ecap)
		}
	case <-j.stopCh:
	}
}

// Update sends an update event for the job.
func (j *TrainingJob) Update(newJob *spec.TfJob) {
	j.send(&jobEvent{
		typ:     eventModifyJob,
		cluster: newJob,
	})
}

// updateTPRStatus updates the job status based on TraingingJob.status.
func (j *TrainingJob) updateTPRStatus() error {
	// If the status hasn't changed then there's no reason to update the TPR.
	if reflect.DeepEqual(j.job.Status, j.status) {
		return nil
	}

	newJob := j.job
	newJob.Status = j.status
	newJob, err := j.tfJobClient.Update(j.job.Metadata.Namespace, newJob)
	if err != nil {
		return err
	}

	j.job = newJob

	return nil
}

func (j *TrainingJob) run(config *spec.ControllerConfig, stopC <-chan struct{}) {
	// TODO(jlewi): What does the run function do?
	clusterFailed := false

	defer func() {
		if clusterFailed {
			j.reportFailedStatus()
		}

		close(j.stopCh)
	}()

	// Update the phase to running.
	j.status.SetPhase(spec.TfJobPhaseRunning)
	if err := j.updateTPRStatus(); err != nil {
		log.Warningf("failed to update TPR status: %v", err)
	}
	log.Infof("start running...")

	var rerr error
	for {
		select {
		case <-stopC:
			return
		case event := <-j.eventCh:
			switch event.typ {

			// TODO(jlewi): We need handle a modify event.
			//case eventModifyCluster:
			//	if isSpecEqual(event.cluster.Spec, j.job.Spec) {
			//		break
			//	}
			case eventDeleteJob:
				// TODO(jlewi): Delete is what should cause us to delete the Pods.
				// we shouldn't delete the pods when the jobs finish because leaving the pods
				// allows us to get the logs from the pods after the job finishes.
				//
				log.Infof("TfJob %v deleted by the user", j.fullname())
				// TODO(jlewi): This logic is probably insufficient.
				if j.job.Status.Phase != spec.TfJobPhaseCleanUp {
					j.status.SetPhase(spec.TfJobPhaseCleanUp)
				}

				// TODO(jlewi): Does it make sense to explicitly delete the resources? Should
				// we just rely on K8s garbage collection to delete the resources before
				// deleting TfJob?
				if cErr := j.deleteResources(); cErr != nil {
					log.Errorf("trainingJob.deleteResources() error; %v", cErr)
				}

				// Return from run because we want to stop reconciling the object.
				return
			}

		case <-time.After(reconcileInterval):
			// TODO(jlewi): Can we determine from the TPR status whether we should
			// Create the resources or not? We need to ensure the resources exist so for
			// now we always call Create.
			if j.job.Status.Phase == spec.TfJobPhaseRunning {
				// We call Create to make sure all the resources exist and are running.
				if cErr := j.createResources(config); cErr != nil {
					log.Errorf("trainingJobCreateReplicas() error; %v", cErr)
				}

				state, replicaStatuses, err := j.GetStatus()

				j.status.ReplicaStatuses = replicaStatuses
				if err != nil {
					log.Errorf("GetStatus() for job %v returned error: %v", j.job.Metadata.Name, err)
				}
				// TODO(jlewi): We should update the Phase if we detect the job is done.
				if state == spec.StateFailed {
					log.Errorf("Master failed Job: %v.", j.job.Metadata.Name)
					j.status.SetPhase(spec.TfJobPhaseDone)
					j.status.SetState(spec.StateFailed)
				} else if state == spec.StateSucceeded {
					log.Infof("Master succeeded Job: %v.", j.job.Metadata.Name)
					j.status.SetPhase(spec.TfJobPhaseDone)
					j.status.SetState(spec.StateSucceeded)
				} else {
					log.V(1).Infof("Job %v status=%v", j.job.Metadata.Name, util.Pformat(j.status))
				}
			}

			// If the phase changed we should update the TPR.
			if err := j.updateTPRStatus(); err != nil {
				log.Warningf("Job %v, failed to update TPR status error: %v", j.job.Metadata.Name, err)
			}

			if j.job.Status.Phase == spec.TfJobPhaseCleanUp {
				if cErr := j.deleteResources(); cErr != nil {
					log.Errorf("Job %v trainingJob.Delete() error; %v", j.job.Metadata.Name, cErr)
				}
				// j.status.SetPhase(spec.TfJobPhaseDone)
				// Return from run because we want to stop reconciling the object.
				return
			}

			if rerr != nil {
				log.Errorf("failed to reconcile job %v, error: %v", j.job.Metadata.Name, rerr)
				break
			}

			// updateTPRStatus will update the status of the TPR with c.Status if c.Status
			// doesn't match c.Cluster.status. So you can change c.Status in order to propogate
			// changes to the TPR status.
			if err := j.updateTPRStatus(); err != nil {
				log.Warningf("Job %v; failed to update TPR status error: %v", j.job.Metadata.Name, err)
			}
		}

		//if isFatalError(rerr) {
		//	clusterFailed = true
		//	j.status.SetReason(rerr.Error())
		//
		//	log.Errorf("cluster failed: %v", rerr)
		//	return
		//}
	}
}

//func isSpecEqual(s1, s2 spec.TfJobSpec) bool {
//	// TODO(jlewi): Need to implement this function.
//	return false
//	//if s1.Size != s2.Size || s1.Paused != s2.Paused || s1.Version != s2.Version {
//	//	return false
//	//}
//	//return isBackupPolicyEqual(s1.Backup, s2.Backup)
//}

// TODO(jlewi): We probably need to update this function.
func (j *TrainingJob) reportFailedStatus() {
	retryInterval := 5 * time.Second

	f := func() (bool, error) {
		j.status.SetPhase(spec.TfJobPhaseFailed)
		err := j.updateTPRStatus()
		if err == nil || k8sutil.IsKubernetesResourceNotFoundError(err) {
			return true, nil
		}

		if !apierrors.IsConflict(err) {
			log.Warningf("retry report status in %v: fail to update: %v", retryInterval, err)
			return false, nil
		}

		cl, err := j.tfJobClient.Get(j.job.Metadata.Namespace, j.job.Metadata.Name)
		if err != nil {
			// Update (PUT) will return conflict even if object is deleted since we have UID set in object.
			// Because it will check UID first and return something like:
			// "Precondition failed: UID in precondition: 0xc42712c0f0, UID in object meta: ".
			if k8sutil.IsKubernetesResourceNotFoundError(err) {
				return true, nil
			}
			log.Warningf("retry report status in %v: fail to get latest version: %v", retryInterval, err)
			return false, nil
		}
		j.job = cl
		return false, nil

	}

	retryutil.Retry(retryInterval, math.MaxInt64, f)
}

func (j *TrainingJob) name() string {
	return j.job.Metadata.GetName()
}

// fullname returns the namespace and name for the job.
func (j *TrainingJob) fullname() string {
	return j.job.Metadata.GetNamespace() + ":" + j.job.Metadata.GetName()
}
