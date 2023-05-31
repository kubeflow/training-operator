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
	"context"
	"errors"
	"fmt"

	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	policyapi "k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FillPodGroupSpecFunc func(object metav1.Object) error

func (jc *JobController) SyncPodGroup(job metav1.Object, specFunc FillPodGroupSpecFunc) (metav1.Object, error) {
	pgctl := jc.PodGroupControl

	// Check whether podGroup exists or not
	podGroup, err := pgctl.GetPodGroup(job.GetNamespace(), job.GetName())
	if err == nil {
		// update podGroup for gang scheduling
		oldPodGroup := &podGroup
		if err = specFunc(podGroup); err != nil {
			return nil, fmt.Errorf("unable to fill the spec of PodGroup, '%v': %v", klog.KObj(podGroup), err)
		}
		if diff := cmp.Diff(oldPodGroup, podGroup); len(diff) != 0 {
			return podGroup, pgctl.UpdatePodGroup(podGroup.(client.Object))
		}
		return podGroup, nil
	} else if client.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf("unable to get a PodGroup: %v", err)
	} else {
		// create podGroup for gang scheduling
		newPodGroup := pgctl.NewEmptyPodGroup()
		newPodGroup.SetName(job.GetName())
		newPodGroup.SetNamespace(job.GetNamespace())
		newPodGroup.SetAnnotations(job.GetAnnotations())
		newPodGroup.SetOwnerReferences([]metav1.OwnerReference{*jc.GenOwnerReference(job)})
		if err = specFunc(newPodGroup); err != nil {
			return nil, fmt.Errorf("unable to fill the spec of PodGroup, '%v': %v", klog.KObj(newPodGroup), err)
		}

		err = pgctl.CreatePodGroup(newPodGroup)
		if err != nil {
			return podGroup, fmt.Errorf("unable to create PodGroup: %v", err)
		}
		createdPodGroupsCount.Inc()
	}

	createdPodGroup, err := pgctl.GetPodGroup(job.GetNamespace(), job.GetName())
	if err != nil {
		return nil, fmt.Errorf("unable to get PodGroup after success creation: %v", err)
	}

	return createdPodGroup, nil
}

// SyncPdb will create a PDB for gang scheduling.
func (jc *JobController) SyncPdb(job metav1.Object, minAvailableReplicas int32) (*policyapi.PodDisruptionBudget, error) {
	// Check the pdb exist or not
	pdb, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Get(context.TODO(), job.GetName(), metav1.GetOptions{})
	if err == nil || !k8serrors.IsNotFound(err) {
		if err == nil {
			err = errors.New(string(metav1.StatusReasonAlreadyExists))
		}
		return pdb, err
	}

	// Create pdb for gang scheduling
	minAvailable := intstr.FromInt(int(minAvailableReplicas))
	createPdb := &policyapi.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.GetName(),
			OwnerReferences: []metav1.OwnerReference{
				*jc.GenOwnerReference(job),
			},
		},
		Spec: policyapi.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					apiv1.JobNameLabel: job.GetName(),
				},
			},
		},
	}
	createdPdb, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Create(context.TODO(), createPdb, metav1.CreateOptions{})
	if err != nil {
		return createdPdb, fmt.Errorf("unable to create pdb: %v", err)
	}
	createdPDBCount.Inc()
	return createdPdb, nil
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

func (jc *JobController) DeletePdb(job metav1.Object) error {
	// Check whether pdb exists or not
	_, err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Get(context.TODO(), job.GetName(), metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	msg := fmt.Sprintf("Deleting pdb %s", job.GetName())
	log.Info(msg)

	if err := jc.KubeClientSet.PolicyV1beta1().PodDisruptionBudgets(job.GetNamespace()).Delete(context.TODO(), job.GetName(), metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("unable to delete pdb: %v", err)
	}
	deletedPDBCount.Inc()
	return nil
}
