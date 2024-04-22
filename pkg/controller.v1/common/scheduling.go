/*
Copyright 2023 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"fmt"

	volcanov1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	log "github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FillPodGroupSpecFunc func(object metav1.Object) (metav1.Object, error)

func (jc *JobController) SyncPodGroup(job metav1.Object, specFunc FillPodGroupSpecFunc) (metav1.Object, error) {
	pgctl := jc.PodGroupControl

	// Check whether podGroup exists or not
	podGroup, err := pgctl.GetPodGroup(job.GetNamespace(), job.GetName())
	if err == nil {
		// update podGroup for gang scheduling
		updatedSpecPodGroup, err := specFunc(podGroup)
		if err != nil {
			return nil, fmt.Errorf("unable to fill the spec of PodGroup, '%v': %v", klog.KObj(podGroup), err)
		}

		existVolcanoPodGroup := podGroup.(*volcanov1beta1.PodGroup)
		updatedSpecVolcanoPodGroup := updatedSpecPodGroup.(*volcanov1beta1.PodGroup)
		// The hpa-controller may update the num of replicas
		// https://github.com/kubeflow/common/pull/207
		if existVolcanoPodGroup.Spec.MinMember != updatedSpecVolcanoPodGroup.Spec.MinMember {
			// The queue name should not be changed after the pg is created
			updatedSpecVolcanoPodGroup.Spec.Queue = existVolcanoPodGroup.Spec.Queue
			return updatedSpecPodGroup, pgctl.UpdatePodGroup(updatedSpecPodGroup.(client.Object))
		}
	} else if client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf("unable to get a PodGroup: %v", err)
	} else {
		// create podGroup for gang scheduling
		newPodGroup := pgctl.NewEmptyPodGroup()
		newPodGroup.SetName(job.GetName())
		newPodGroup.SetNamespace(job.GetNamespace())
		newPodGroup.SetAnnotations(job.GetAnnotations())
		newPodGroup.SetOwnerReferences([]metav1.OwnerReference{*jc.GenOwnerReference(job)})
		updatedSpecPodGroup, err := specFunc(newPodGroup)
		if err != nil {
			return nil, fmt.Errorf("unable to fill the spec of PodGroup, '%v': %v", klog.KObj(newPodGroup), err)
		}

		err = pgctl.CreatePodGroup(updatedSpecPodGroup.(client.Object))
		if err != nil {
			return updatedSpecPodGroup, fmt.Errorf("unable to create PodGroup: %v", err)
		}
		createdPodGroupsCount.Inc()
	}

	createdPodGroup, err := pgctl.GetPodGroup(job.GetNamespace(), job.GetName())
	if err != nil {
		return nil, fmt.Errorf("unable to get PodGroup after success creation: %v", err)
	}

	return createdPodGroup, nil
}

func (jc *JobController) DeletePodGroup(job metav1.Object) error {
	pgctl := jc.PodGroupControl

	// Check whether podGroup exists or not
	_, err := pgctl.GetPodGroup(job.GetNamespace(), job.GetName())
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	log.Infof("Deleting PodGroup %s", job.GetName())

	// Delete podGroup
	err = pgctl.DeletePodGroup(job.GetNamespace(), job.GetName())
	if err != nil {
		return fmt.Errorf("unable to delete PodGroup: %v", err)
	}
	deletedPodGroupsCount.Inc()
	return nil
}
