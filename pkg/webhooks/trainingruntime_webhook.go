/*
Copyright 2024 The Kubeflow Authors.

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

package webhooks

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	trainer "github.com/kubeflow/trainer/v2/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/v2/pkg/constants"
	trainingruntime "github.com/kubeflow/trainer/v2/pkg/util/trainingruntime"
)

const (
	rJobReplicasErrorMsg       = "always must be 1"
	rJobContainerNamesErrorMsg = "must contain the required container for the ancestor: %s"
	rJobContainerClaimErrorMsg = "references resourceClaim %q which is not defined in the Pod's resourceClaims"
	mpiGroupJobNameErrorMsg    = "must match the name of a replicatedJob in spec.template.spec.replicatedJobs"
	mpiGroupContainerErrorMsg  = "must match the name of a container in the replicatedJob %s"
	mpiGroupReplicasErrorMsg   = "must have 1 replica, since the replicatedJob %s is an MPI launcher or node group"
	mpiGroupAncestorErrorMsg   = "must not be set when mlPolicy.mpi.launcher is set, since the launcher is the trainer step"
	mpiGroupNumNodesErrorMsg   = "must not be set when mlPolicy.mpi.nodeGroups is set, set numNodes per node group instead"
)

var (
	expectedContainerNames = map[string]string{
		constants.AncestorTrainer:    constants.Node,
		constants.ModelInitializer:   constants.ModelInitializer,
		constants.DatasetInitializer: constants.DatasetInitializer,
	}
)

// +kubebuilder:webhook:path=/validate-trainer-kubeflow-org-v1alpha1-trainingruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=trainer.kubeflow.org,resources=trainingruntimes,verbs=create;update,versions=v1alpha1,name=validator.trainingruntime.trainer.kubeflow.org,admissionReviewVersions=v1

// TrainingRuntimeValidator validates TrainingRuntimes
type TrainingRuntimeValidator struct{}

var _ admission.Validator[*trainer.TrainingRuntime] = (*TrainingRuntimeValidator)(nil)

func setupWebhookForTrainingRuntime(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &trainer.TrainingRuntime{}).
		WithValidator(&TrainingRuntimeValidator{}).
		Complete()
}

func (w *TrainingRuntimeValidator) ValidateCreate(ctx context.Context, obj *trainer.TrainingRuntime) (admission.Warnings, error) {
	log := ctrl.LoggerFrom(ctx).WithName("trainingruntime-webhook")
	log.V(5).Info("Validating create", "trainingRuntime", klog.KObj(obj))
	var warnings admission.Warnings
	if trainingruntime.IsSupportDeprecated(obj.Labels) {
		warnings = append(warnings, fmt.Sprintf(
			"TrainingRuntime \"%s\" is deprecated and will be removed in a future release of Kubeflow Trainer. See runtime deprecation policy: %s",
			obj.Name,
			constants.RuntimeDeprecationPolicyURL,
		))
	}
	return warnings, validateRuntimeSpec(&obj.Spec).ToAggregate()
}

// validateRuntimeSpec validates the spec shared by TrainingRuntime and ClusterTrainingRuntime.
func validateRuntimeSpec(spec *trainer.TrainingRuntimeSpec) field.ErrorList {
	allErrs := validateReplicatedJobs(spec.Template.Spec.ReplicatedJobs)
	return append(allErrs, validateMPINodeGroups(spec)...)
}

// validateMPINodeGroups validates that the MPI launcher and node groups reference existing
// replicatedJobs and containers. The launcher is found by name, so the replicatedJobs must not
// carry the trainer ancestor label, and numNodes is set per node group.
func validateMPINodeGroups(spec *trainer.TrainingRuntimeSpec) field.ErrorList {
	if spec.MLPolicy == nil || spec.MLPolicy.MPI == nil || spec.MLPolicy.MPI.Launcher.JobName == "" {
		return nil
	}
	var allErrs field.ErrorList
	mpi := spec.MLPolicy.MPI
	mpiPath := field.NewPath("spec", "mlPolicy", "mpi")
	if spec.MLPolicy.NumNodes != nil {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "mlPolicy", "numNodes"), mpiGroupNumNodesErrorMsg))
	}
	rJobs := make(map[string]*jobsetv1alpha2.ReplicatedJob, len(spec.Template.Spec.ReplicatedJobs))
	for i := range spec.Template.Spec.ReplicatedJobs {
		rJobs[spec.Template.Spec.ReplicatedJobs[i].Name] = &spec.Template.Spec.ReplicatedJobs[i]
	}
	validateGroup := func(path *field.Path, jobName, containerName string) {
		rJob, ok := rJobs[jobName]
		if !ok {
			allErrs = append(allErrs, field.Invalid(path.Child("jobName"), jobName, mpiGroupJobNameErrorMsg))
			return
		}
		if rJob.Replicas != 1 {
			allErrs = append(allErrs, field.Invalid(path.Child("jobName"), jobName, fmt.Sprintf(mpiGroupReplicasErrorMsg, jobName)))
		}
		name := cmp.Or(containerName, constants.Node)
		if !slices.ContainsFunc(rJob.Template.Spec.Template.Spec.Containers, func(c corev1.Container) bool { return c.Name == name }) {
			allErrs = append(allErrs, field.Invalid(path.Child("containerName"), name, fmt.Sprintf(mpiGroupContainerErrorMsg, jobName)))
		}
	}
	validateGroup(mpiPath.Child("launcher"), mpi.Launcher.JobName, mpi.Launcher.ContainerName)
	for i, nodeGroup := range mpi.NodeGroups {
		validateGroup(mpiPath.Child("nodeGroups").Index(i), nodeGroup.JobName, nodeGroup.ContainerName)
	}
	rJobsPath := field.NewPath("spec", "template", "spec", "replicatedJobs")
	for idx, rJob := range spec.Template.Spec.ReplicatedJobs {
		if rJob.Template.Labels[constants.LabelTrainJobAncestor] == constants.AncestorTrainer {
			allErrs = append(allErrs, field.Forbidden(
				rJobsPath.Index(idx).Child("template", "metadata", "labels").Key(constants.LabelTrainJobAncestor), mpiGroupAncestorErrorMsg))
		}
	}
	return allErrs
}

func validateReplicatedJobs(rJobs []jobsetv1alpha2.ReplicatedJob) field.ErrorList {
	ancestors := sets.New(constants.AncestorTrainer, constants.ModelInitializer, constants.DatasetInitializer)
	rJobsPath := field.NewPath("spec").
		Child("template").
		Child("spec").
		Child("replicatedJobs")
	var allErrs field.ErrorList
	for idx, rJob := range rJobs {
		// Every container resources.claims entry must reference a Pod-level resourceClaims entry.
		// Otherwise every TrainJob would have to add the Pod-level claim itself to use the runtime.
		podSpec := &rJob.Template.Spec.Template.Spec
		podSpecPath := rJobsPath.Index(idx).Child("template", "spec", "template", "spec")
		podClaims := sets.New[string]()
		for _, pc := range podSpec.ResourceClaims {
			podClaims.Insert(pc.Name)
		}
		validateClaims := func(containersField string, containers []corev1.Container) {
			for k, c := range containers {
				for m, claim := range c.Resources.Claims {
					if !podClaims.Has(claim.Name) {
						allErrs = append(allErrs, field.Invalid(
							podSpecPath.Child(containersField).Index(k).Child("resources", "claims").Index(m),
							claim.Name, fmt.Sprintf(rJobContainerClaimErrorMsg, claim.Name)))
					}
				}
			}
		}
		validateClaims("initContainers", podSpec.InitContainers)
		validateClaims("containers", podSpec.Containers)

		if rJob.Template.Labels == nil {
			continue
		}

		if labelAncestor, ok := rJob.Template.Labels[constants.LabelTrainJobAncestor]; ok && ancestors.Has(labelAncestor) {
			if rJob.Replicas != 1 {
				allErrs = append(allErrs, field.Invalid(rJobsPath.Index(idx).Child("replicas"), rJob.Replicas, rJobReplicasErrorMsg))
			}

			// Validate replicated job contains the required containers.
			// Mapping of the ancestor labels to the containers:
			// 1. dataset-initializer - dataset-initializer
			// 2. model-initializer - model-initializer
			// 3. trainer - node
			hasRequiredContainer := false
			for _, container := range rJob.Template.Spec.Template.Spec.Containers {
				if container.Name == expectedContainerNames[labelAncestor] {
					hasRequiredContainer = true
					break
				}
			}
			if !hasRequiredContainer {
				allErrs = append(allErrs, field.Invalid(
					rJobsPath.Index(idx).Child("template").Child("spec").Child("template").Child("spec").Child("containers"),
					rJob.Template.Spec.Template.Spec.Containers,
					fmt.Sprintf(rJobContainerNamesErrorMsg, labelAncestor),
				))
			}
		}
	}
	return allErrs
}

func (w *TrainingRuntimeValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *trainer.TrainingRuntime) (admission.Warnings, error) {
	log := ctrl.LoggerFrom(ctx).WithName("trainingruntime-webhook")
	log.V(5).Info("Validating update", "trainingRuntime", klog.KObj(newObj))
	return nil, validateRuntimeSpec(&newObj.Spec).ToAggregate()
}

func (w *TrainingRuntimeValidator) ValidateDelete(ctx context.Context, obj *trainer.TrainingRuntime) (admission.Warnings, error) {
	return nil, nil
}
