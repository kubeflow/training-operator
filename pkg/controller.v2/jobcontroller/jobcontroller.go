package jobcontroller

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeclientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/kubeflow/tf-operator/pkg/control"
)

// Common Interaface to be implemented by all operators
type ControllerInterface interface {

	// AdoptFunc used byControlRefManager to get the latest object if UID matches
	AdoptFunc(job metav1.Object) func() (metav1.Object, error)

	// Returns total replicas for a job. This is used for gang scheduling
	GetTotalReplicas(obj metav1.Object) int32

	// Returns the GrouoVersionKinf of the API
	GetAPIGroupVersionKind() schema.GroupVersionKind

	GetAPIGroupVersion() schema.GroupVersion

	GetGroupNameLabel() string

	GetJobNameLabel() string

	GetJobGroupName() string
}

// JobControllerConfiguration contains configuration of operator.
type JobControllerConfiguration struct {
	// ReconcilerSyncLoopPeriod is the amount of time the reconciler sync states loop
	// wait between two reconciler sync.
	// It is set to 15 sec by default.
	// TODO(cph): maybe we can let it grows by multiple in the future
	// and up to 5 minutes to reduce idle loop.
	// e.g. 15s, 30s, 60s, 120s...
	ReconcilerSyncLoopPeriod metav1.Duration

	// Enable gang scheduling by kube-arbitrator
	EnableGangScheduling bool
}

// JobController abstracts other operators to manage the lifecycle of Jobs.
type JobController struct {
	Controller ControllerInterface

	Config JobControllerConfiguration

	// podControl is used to add or delete pods.
	PodControl controller.PodControlInterface

	// serviceControl is used to add or delete services.
	ServiceControl control.ServiceControlInterface

	// kubeClientSet is a standard kubernetes clientset.
	KubeClientSet kubeclientset.Interface

	// podLister can list/get pods from the shared informer's store.
	PodLister corelisters.PodLister

	// serviceLister can list/get services from the shared informer's store.
	ServiceLister corelisters.ServiceLister

	// podInformerSynced returns true if the pod store has been synced at least once.
	PodInformerSynced cache.InformerSynced

	// serviceInformerSynced returns true if the service store has been synced at least once.
	ServiceInformerSynced cache.InformerSynced

	// A TTLCache of pod/services creates/deletes each job expects to see
	// We use Job namespace/name + ReplicaType + pods/services as an expectation key,
	// For example, there is a TFJob with namespace "tf-operator" and name "tfjob-abc":
	// {
	//     "PS": {
	//         "Replicas": 2,
	//     },
	//     "Worker": {
	//         "Replicas": 4,
	//     }
	// }
	// We will create 4 expectations:
	// - "tf-operator/tfjob-abc/ps/services", expects 2 adds.
	// - "tf-operator/tfjob-abc/ps/pods", expects 2 adds.
	// - "tf-operator/tfjob-abc/worker/services", expects 4 adds.
	// - "tf-operator/tfjob-abc/worker/pods", expects 4 adds.
	Expectations controller.ControllerExpectationsInterface

	// workQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	WorkQueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder
}

func (jc *JobController) GenOwnerReference(obj metav1.Object) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         jc.Controller.GetAPIGroupVersion().String(),
		Kind:               jc.Controller.GetAPIGroupVersionKind().Kind,
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func (jc *JobController) GenLabels(jobName string) map[string]string {
	labelGroupName := jc.Controller.GetGroupNameLabel()
	labelJobName := jc.Controller.GetJobNameLabel()
	groupName := jc.Controller.GetJobGroupName()
	return map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(jobName, "/", "-", -1),
	}
}

// getPodsForJob returns the set of pods that this job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned Pods are pointers into the cache.
func (jc *JobController) GetPodsForJob(job metav1.Object) ([]*v1.Pod, error) {
	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: jc.GenLabels(job.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	pods, err := jc.PodLister.Pods(job.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).

	canAdoptFunc := RecheckDeletionTimestamp(jc.Controller.AdoptFunc(job))
	cm := controller.NewPodControllerRefManager(jc.PodControl, job, selector, jc.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}

// getServicesForJob returns the set of services that this job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned services are pointers into the cache.
func (jc *JobController) GetServicesForJob(job metav1.Object) ([]*v1.Service, error) {
	// Create selector
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: jc.GenLabels(job.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all services to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	services, err := jc.ServiceLister.Services(job.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing services (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(jc.Controller.AdoptFunc(job))
	cm := control.NewServiceControllerRefManager(jc.ServiceControl, job, selector, jc.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimServices(services)
}

// SyncPdb will create a PDB for gang scheduling by kube-arbitrator.
func (jc *JobController) SyncPdb(job metav1.Object) (*v1beta1.PodDisruptionBudget, error) {
	labelJobName := jc.Controller.GetJobNameLabel()
	totalJobReplicas := jc.Controller.GetTotalReplicas(job)
	// Non-distributed training is not required gang scheduling
	if totalJobReplicas < 2 {
		return nil, nil
	}

	// Check the pdb exist or not
	pdb, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err == nil || !k8serrors.IsNotFound(err) {
		if err == nil {
			err = errors.New(string(metav1.StatusReasonAlreadyExists))
		}
		return pdb, err
	}

	// Create pdb for gang scheduling by kube-arbitrator
	minAvailable := intstr.FromInt(int(totalJobReplicas))
	createPdb := &v1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.GetName(),
			OwnerReferences: []metav1.OwnerReference{
				*jc.GenOwnerReference(job),
			},
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelJobName: job.GetName(),
				},
			},
		},
	}
	return jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Create(createPdb)
}

func (jc *JobController) DeletePdb(job metav1.Object) error {

	// Check the pdb exist or not
	_, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Get(job.GetName(), metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	msg := fmt.Sprintf("Deleting pdb %s", job.GetName())
	log.Info(msg)

	if err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Delete(job.GetName(), &metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("unable to delete pdb: %v", err)
	}
	return nil
}
