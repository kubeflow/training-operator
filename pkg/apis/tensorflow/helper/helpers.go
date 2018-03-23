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

package helper

import (
	"fmt"

	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha1"
	"github.com/kubeflow/tf-operator/pkg/util"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	groupVersionKind = schema.GroupVersionKind{
		Group:   tfv1.GroupName,
		Version: tfv1.GroupVersion,
		Kind:    tfv1.TFJobResourceKind,
	}
)

// AsOwner make OwnerReference according to the parameter
func AsOwner(tfJob *tfv1.TFJob) metav1.OwnerReference {
	trueVar := true
	// Both api.OwnerReference and metatypes.OwnerReference are combined into that.
	return metav1.OwnerReference{
		APIVersion:         groupVersionKind.GroupVersion().String(),
		Kind:               groupVersionKind.Kind,
		Name:               tfJob.ObjectMeta.Name,
		UID:                tfJob.ObjectMeta.UID,
		Controller:         &trueVar,
		BlockOwnerDeletion: &trueVar,
	}
}

// ConfigureAcceleratorsForTFJobSpec adds any accelerator specific configuration to the pods.
func ConfigureAcceleratorsForTFJobSpec(c *tfv1.TFJobSpec, accelerators map[string]tfv1.AcceleratorConfig) error {
	for _, r := range c.ReplicaSpecs {
		if r.Template == nil {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}
		for i, c := range r.Template.Spec.Containers {
			if c.Name == tfv1.DefaultTFContainer {
				// Identify the accelerators attached to this container.
				a := map[string]tfv1.AcceleratorConfig{}

				lists := []v1.ResourceList{c.Resources.Limits, c.Resources.Requests}
				for _, resources := range lists {
					for name := range resources {

						if _, ok := accelerators[string(name)]; !ok {
							continue
						}

						// Add the expected mounts to the pods.
						a[string(name)] = accelerators[string(name)]
					}
				}

				// Add accelerator information to the pod.
				for _, config := range a {
					for _, v := range config.Volumes {
						r.Template.Spec.Volumes = append(r.Template.Spec.Volumes,
							v1.Volume{
								Name: v.Name,
								VolumeSource: v1.VolumeSource{
									HostPath: &v1.HostPathVolumeSource{
										Path: v.HostPath,
									},
								},
							})
						c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
							Name:      v.Name,
							MountPath: v.MountPath,
						})
					}

					for _, envVar := range config.EnvVars {
						c.Env = append(c.Env, v1.EnvVar{
							Name:  envVar.Name,
							Value: envVar.Value,
						})
					}
				}
				r.Template.Spec.Containers[i] = c
				break
			}
		}
	}
	return nil
}

// Cleanup cleans up user passed spec, e.g. defaulting, transforming fields.
// TODO: move this to admission controller
func Cleanup(c *tfv1.TFJobSpec) {
	// TODO(jlewi): Add logic to cleanup user provided spec; e.g. by filling in defaults.
	// We should have default container images so user doesn't have to provide these.
}

// CRDName returns the custom resource definition name which is combination of kind and group
func CRDName() string {
	return fmt.Sprintf("%s.%s", tfv1.CRDKindPlural, tfv1.CRDGroup)
}

// scalingReason returns the reason for scaling the cluster size
func scalingReason(from, to int) string {
	return fmt.Sprintf("Current cluster size: %d, desired cluster size: %d", from, to)
}
