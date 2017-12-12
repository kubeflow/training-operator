package helper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tfv1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	"fmt"
	"k8s.io/client-go/pkg/api/v1"
	"github.com/tensorflow/k8s/pkg/util"
	"time"
)

func AsOwner(tfJob *tfv1.TfJob) metav1.OwnerReference {
	trueVar := true
	// TODO: In 1.6 this is gonna be "k8s.io/kubernetes/pkg/apis/meta/v1"
	// Both api.OwnerReference and metatypes.OwnerReference are combined into that.
	return metav1.OwnerReference{
		APIVersion:         tfJob.APIVersion,
		Kind:               tfJob.Kind,
		Name:               tfJob.ObjectMeta.Name,
		UID:                tfJob.ObjectMeta.UID,
		Controller:         &trueVar,
		BlockOwnerDeletion: &trueVar,
	}
}

// ConfigureAcceleratorsForTfJobSpec adds any accelerator specific configuration to the pods.
func ConfigureAcceleratorsForTfJobSpec(c *tfv1.TfJobSpec, accelerators map[string]tfv1.AcceleratorConfig) error {
	for _, r := range c.ReplicaSpecs {
		if r.Template == nil {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}
		for i, c := range r.Template.Spec.Containers {
			if c.Name == string(tfv1.TENSORFLOW) {
				// Identify the accelerators attached to this container.
				a := map[string]tfv1.AcceleratorConfig{}

				lists := []v1.ResourceList{c.Resources.Limits, c.Resources.Requests}
				for _, resources := range lists {
					for name, _ := range resources {

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
func Cleanup(c *tfv1.TfJobSpec) {
	// TODO(jlewi): Add logic to cleanup user provided spec; e.g. by filling in defaults.
	// We should have default container images so user doesn't have to provide these.
}

func CRDName() string {
	return fmt.Sprintf("%s.%s", tfv1.CRDKindPlural, tfv1.CRDGroup)
}

// TODO(jlewi): Get rid of the append methods that we don't need
func AppendScalingDownCondition(cs *tfv1.TfJobStatus, from, to int) {
	c := tfv1.TfJobCondition{
		Type:           tfv1.TfJobConditionScalingDown,
		Reason:         scalingReason(from, to),
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	AppendCondition(cs, c)
}

func AppendRecoveringCondition(cs *tfv1.TfJobStatus) {
	c := tfv1.TfJobCondition{
		Type:           tfv1.TfJobConditionRecovering,
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	AppendCondition(cs, c)
}

func AppendUpgradingCondition(cs *tfv1.TfJobStatus, to string, member string) {
	reason := fmt.Sprintf("upgrading cluster member %s version to %v", member, to)

	c := tfv1.TfJobCondition{
		Type:           tfv1.TfJobConditionUpgrading,
		Reason:         reason,
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	AppendCondition(cs, c)
}

func AppendRemovingDeadMember(cs *tfv1.TfJobStatus, name string) {
	reason := fmt.Sprintf("removing dead member %s", name)

	c := tfv1.TfJobCondition{
		Type:           tfv1.TfJobConditionRemovingDeadMember,
		Reason:         reason,
		TransitionTime: time.Now().Format(time.RFC3339),
	}
	AppendCondition(cs, c)
}

func SetReadyCondition(cs *tfv1.TfJobStatus, ) {
	c := tfv1.TfJobCondition{
		Type:           tfv1.TfJobConditionReady,
		TransitionTime: time.Now().Format(time.RFC3339),
	}

	if len(cs.Conditions) == 0 {
		AppendCondition(cs, c)
		return
	}

	lastc := cs.Conditions[len(cs.Conditions)-1]
	if lastc.Type == tfv1.TfJobConditionReady {
		return
	}
	AppendCondition(cs, c)
}

func AppendCondition(cs *tfv1.TfJobStatus, c tfv1.TfJobCondition) {
	cs.Conditions = append(cs.Conditions, c)
	if len(cs.Conditions) > 10 {
		cs.Conditions = cs.Conditions[1:]
	}
}

func scalingReason(from, to int) string {
	return fmt.Sprintf("Current cluster size: %d, desired cluster size: %d", from, to)
}
