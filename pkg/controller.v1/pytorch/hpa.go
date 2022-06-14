// Copyright 2021 The Kubeflow Authors
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
// limitations under the License

package pytorch

import (
	"context"

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func (r *PyTorchJobReconciler) ReconcileHPA(pytorchJob *kubeflowv1.PyTorchJob) error {
	logger := r.Log.WithValues(kubeflowv1.PytorchJobSingular, pytorchJob.Name)

	if pytorchJob.Spec.ElasticPolicy == nil || pytorchJob.Spec.ElasticPolicy.Metrics == nil {
		logger.V(1).Info(
			"No ElasicPolicy or Metric is specified, skipping HPA reconciling process")
		return nil
	}

	current := &autoscalingv2beta2.HorizontalPodAutoscaler{}

	// Get the exepected HPA.
	expected, err := desiredHPA(pytorchJob, r.Scheme)
	if err != nil {
		return err
	}

	if err := r.Get(context.TODO(), types.NamespacedName{
		Name:      pytorchJob.Name,
		Namespace: pytorchJob.Namespace,
	}, current); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// Create the new HPA.
		logger.V(1).Info("Creating HPA", "namespace", expected.Namespace, "name", expected.Name)
		err = r.Create(context.TODO(), expected)
		if err != nil {
			return err
		}
		return nil
	}

	if !equality.Semantic.DeepEqual(expected.Spec, current.Spec) {
		logger.V(1).Info("Updating HPA", "namespace", current.Namespace, "name", current.Name)
		expected.ResourceVersion = current.ResourceVersion
		err = r.Update(context.TODO(), expected)
		if err != nil {
			return err
		}
	}
	return nil
}

func desiredHPA(pytorchJob *kubeflowv1.PyTorchJob, scheme *runtime.Scheme) (
	*autoscalingv2beta2.HorizontalPodAutoscaler, error) {
	hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pytorchJob.Name,
			Namespace: pytorchJob.Namespace,
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				Kind: pytorchJob.Kind,
				Name: pytorchJob.Name,
			},
			MinReplicas: pytorchJob.Spec.ElasticPolicy.MinReplicas,
			MaxReplicas: *pytorchJob.Spec.ElasticPolicy.MaxReplicas,
			Metrics:     pytorchJob.Spec.ElasticPolicy.Metrics,
		},
	}
	if err := controllerruntime.SetControllerReference(pytorchJob, hpa, scheme); err != nil {
		return nil, err
	}
	return hpa, nil
}
