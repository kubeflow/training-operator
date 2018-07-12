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

// Package controller provides a Kubernetes controller for a TFJob resource.
package controller

import (
	"time"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	log "github.com/sirupsen/logrus"
)

// cleanupBot cleans up the jobs that specifies a cleanup timeout after finished.
func (tc *TFJobController) cleanupBot(stopCh <-chan struct{}) {
	for {
		select {
		case ci := <-tc.cleanupInfoCh:
			go func() {
				time.Sleep(ci.timeout)
				tc.deleteTFJob(ci.tfJob)
			}()
		case <-stopCh:
			return
		}
	}
}

// deleteTFJob deletes all the pods and services in a tfJob.
func (tc *TFJobController) deleteTFJob(tfJob *tfv1alpha2.TFJob) {
	// delete all the pods
	pods, err := tc.getPodsForTFJob(tfJob)
	if err != nil {
		log.Infof("getPodsForTFJob error %v", err)
		return
	}
	for _, pod := range pods {
		if err :=
			tc.podControl.DeletePod(pod.Namespace, pod.Name, tfJob); err != nil {
			log.Infof("DeletePod error %v", err)
		}
	}
	// delete all the services
	services, err := tc.getServicesForTFJob(tfJob)
	if err != nil {
		log.Infof("getServicesForTFJob error %v", err)
		return
	}
	for _, service := range services {
		if err :=
			tc.serviceControl.DeleteService(service.Namespace, service.Name, tfJob); err != nil {
			log.Infof("DeleteService error %v", err)
		}
	}
}
