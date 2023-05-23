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

package util

import (
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func LoggerForReplica(job metav1.Object, rtype string) *log.Entry {
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// Its more common in K8s to use a period to indicate namespace.name. So that's what we use.
		"job":          job.GetNamespace() + "." + job.GetName(),
		"uid":          job.GetUID(),
		"replica-type": rtype,
	})
}

func LoggerForJob(job metav1.Object) *log.Entry {
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// Its more common in K8s to use a period to indicate namespace.name. So that's what we use.
		"job": job.GetNamespace() + "." + job.GetName(),
		"uid": job.GetUID(),
	})
}

func LoggerForPod(pod *v1.Pod, kind string) *log.Entry {
	job := ""
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		if controllerRef.Kind == kind {
			job = pod.Namespace + "." + controllerRef.Name
		}
	}
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job": job,
		"pod": pod.Namespace + "." + pod.Name,
		"uid": pod.ObjectMeta.UID,
	})
}

func LoggerForService(svc *v1.Service, kind string) *log.Entry {
	job := ""
	if controllerRef := metav1.GetControllerOf(svc); controllerRef != nil {
		if controllerRef.Kind == kind {
			job = svc.Namespace + "." + controllerRef.Name
		}
	}
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job":     job,
		"service": svc.Namespace + "." + svc.Name,
		"uid":     svc.ObjectMeta.UID,
	})
}

func LoggerForKey(key string) *log.Entry {
	return log.WithFields(log.Fields{
		// The key used by the workQueue should be namespace + "/" + name.
		// Its more common in K8s to use a period to indicate namespace.name. So that's what we use.
		"job": strings.Replace(key, "/", ".", -1),
	})
}

func LoggerForUnstructured(obj *metav1unstructured.Unstructured, kind string) *log.Entry {
	job := ""
	if obj.GetKind() == kind {
		job = obj.GetNamespace() + "." + obj.GetName()
	}
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job": job,
		"uid": obj.GetUID(),
	})
}
