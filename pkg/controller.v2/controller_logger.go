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

package controller

import (
	log "github.com/sirupsen/logrus"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

func loggerForReplica(tfjob *tfv1alpha2.TFJob, rtype string) *log.Entry {
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job":          tfjob.ObjectMeta.Namespace + "/" + tfjob.ObjectMeta.Name,
		"uid":          tfjob.ObjectMeta.UID,
		"replica-type": rtype,
	})
}

func loggerForTFJob(tfjob *tfv1alpha2.TFJob) *log.Entry {
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job": tfjob.ObjectMeta.Namespace + "/" + tfjob.ObjectMeta.Name,
		"uid": tfjob.ObjectMeta.UID,
	})
}
