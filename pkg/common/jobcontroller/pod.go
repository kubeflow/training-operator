package jobcontroller

import (
	"fmt"
	"reflect"
	"strconv"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"

	jclogger "github.com/kubeflow/tf-operator/pkg/logger"
)

// When a pod is created, enqueue the job that manages it and update its expectations.
func (jc *JobController) AddPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	if pod.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		// tc.deletePod(pod)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		job := jc.resolveControllerRef(pod.Namespace, controllerRef)

		logger := jclogger.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)

		if job == nil {
			// If this is a TFJob pod
			if pod.Labels[jc.Controller.GetGroupNameLabelKey()] == jc.Controller.GetGroupNameLabelValue() {
				logger.Info("This pod's job does not exist")
			}
			return
		}

		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			logger.Infof("Failed to get the jobkey: %v", err)
			return
		}

		if _, ok := pod.Labels[jc.Controller.GetReplicaTypeLabelKey()]; !ok {
			logger.Infof("This pod maybe not created by %v", jc.Controller.ControllerName())
			return
		}

		rtype := pod.Labels[jc.Controller.GetReplicaTypeLabelKey()]
		expectationPodsKey := GenExpectationPodsKey(jobKey, rtype)

		jc.Expectations.CreationObserved(expectationPodsKey)
		// TODO: we may need add backoff here
		jc.WorkQueue.Add(jobKey)

		return
	}

}

// When a pod is updated, figure out what tfjob/s manage it and wake them up.
// If the labels of the pod have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Pod types.
func (jc *JobController) UpdatePod(old, cur interface{}) {
	curPod := cur.(*v1.Pod)
	oldPod := old.(*v1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	logger := jclogger.LoggerForPod(curPod, jc.Controller.GetAPIGroupVersionKind().Kind)
	curControllerRef := metav1.GetControllerOf(curPod)
	oldControllerRef := metav1.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if job := jc.resolveControllerRef(oldPod.Namespace, oldControllerRef); job != nil {
			logger.Infof("pod ControllerRef updated: %v, %v", curPod, oldPod)
			jobKey, err := controller.KeyFunc(job)
			if err != nil {
				return
			}
			// TODO: we may need add backoff here
			jc.WorkQueue.Add(jobKey)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		job := jc.resolveControllerRef(curPod.Namespace, curControllerRef)
		if job == nil {
			return
		}
		logger.Debugf("pod has a ControllerRef: %v, %v", curPod, oldPod)
		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			return
		}
		// TODO: we may need add backoff here
		jc.WorkQueue.Add(jobKey)
		return
	}
}

// When a pod is deleted, enqueue the job that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (jc *JobController) DeletePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)

	logger := jclogger.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new job will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		pod, ok = tombstone.Obj.(*v1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a pod %+v", obj))
			return
		}
	}

	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	job := jc.resolveControllerRef(pod.Namespace, controllerRef)
	if job == nil {
		return
	}
	jobKey, err := controller.KeyFunc(job)
	if err != nil {
		return
	}

	if _, ok := pod.Labels[jc.Controller.GetReplicaTypeLabelKey()]; !ok {
		logger.Infof("This pod maybe not created by %v", jc.Controller.ControllerName())
		return
	}

	rtype := pod.Labels[jc.Controller.GetReplicaTypeLabelKey()]
	expectationPodsKey := GenExpectationPodsKey(jobKey, rtype)

	jc.Expectations.DeletionObserved(expectationPodsKey)
	// TODO: we may need add backoff here
	jc.WorkQueue.Add(jobKey)
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

	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := jc.Controller.GetJobFromAPIClient(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := controller.NewPodControllerRefManager(jc.PodControl, job, selector, jc.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}

// FilterPodsForReplicaType returns pods belong to a replicaType.
func (jc *JobController) FilterPodsForReplicaType(pods []*v1.Pod, replicaType string) ([]*v1.Pod, error) {
	var result []*v1.Pod

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[jc.Controller.GetReplicaTypeLabelKey()] = replicaType

	for _, pod := range pods {
		selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
		if err != nil {
			return nil, err
		}
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		result = append(result, pod)
	}
	return result, nil
}

// getPodSlices returns a slice, which element is the slice of pod.
func (jc *JobController) GetPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	podSlices := make([][]*v1.Pod, replicas)
	for _, pod := range pods {
		if _, ok := pod.Labels[jc.Controller.GetReplicaIndexLabelKey()]; !ok {
			logger.Warning("The pod do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(pod.Labels[jc.Controller.GetReplicaIndexLabelKey()])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d", index)
		} else {
			podSlices[index] = append(podSlices[index], pod)
		}
	}
	return podSlices
}
