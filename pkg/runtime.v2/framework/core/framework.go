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

package core

import (
	"context"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
	fwkplugins "github.com/kubeflow/training-operator/pkg/runtime.v2/framework/plugins"
)

type Framework struct {
	registry                     fwkplugins.Registry
	plugins                      map[string]framework.Plugin
	enforceMLPlugins             []framework.EnforceMLPolicyPlugin
	enforcePodGroupPolicyPlugins []framework.EnforcePodGroupPolicyPlugin
	customValidationPlugins      []framework.CustomValidationPlugin
	watchExtensionPlugins        []framework.WatchExtensionPlugin
	componentBuilderPlugins      []framework.ComponentBuilderPlugin
}

func New(ctx context.Context, c client.Client, r fwkplugins.Registry, indexer client.FieldIndexer) (*Framework, error) {
	f := &Framework{
		registry: r,
	}
	plugins := make(map[string]framework.Plugin, len(r))

	for name, factory := range r {
		plugin, err := factory(ctx, c, indexer)
		if err != nil {
			return nil, err
		}
		plugins[name] = plugin
		if p, ok := plugin.(framework.EnforceMLPolicyPlugin); ok {
			f.enforceMLPlugins = append(f.enforceMLPlugins, p)
		}
		if p, ok := plugin.(framework.EnforcePodGroupPolicyPlugin); ok {
			f.enforcePodGroupPolicyPlugins = append(f.enforcePodGroupPolicyPlugins, p)
		}
		if p, ok := plugin.(framework.CustomValidationPlugin); ok {
			f.customValidationPlugins = append(f.customValidationPlugins, p)
		}
		if p, ok := plugin.(framework.WatchExtensionPlugin); ok {
			f.watchExtensionPlugins = append(f.watchExtensionPlugins, p)
		}
		if p, ok := plugin.(framework.ComponentBuilderPlugin); ok {
			f.componentBuilderPlugins = append(f.componentBuilderPlugins, p)
		}
	}
	f.plugins = plugins
	return f, nil
}

func (f *Framework) RunEnforceMLPolicyPlugins(info *runtime.Info) error {
	for _, plugin := range f.enforceMLPlugins {
		if err := plugin.EnforceMLPolicy(info); err != nil {
			return err
		}
	}
	return nil
}

func (f *Framework) RunEnforcePodGroupPolicyPlugins(trainJob *kubeflowv2.TrainJob, info *runtime.Info) error {
	for _, plugin := range f.enforcePodGroupPolicyPlugins {
		if err := plugin.EnforcePodGroupPolicy(trainJob, info); err != nil {
			return err
		}
	}
	return nil
}

func (f *Framework) RunCustomValidationPlugins(oldObj, newObj *kubeflowv2.TrainJob,
	runtimeInfo *runtime.Info) (admission.Warnings, field.ErrorList) {
	var aggregatedWarnings admission.Warnings
	var aggregatedErrors field.ErrorList
	for _, plugin := range f.customValidationPlugins {
		warnings, errs := plugin.Validate(oldObj, newObj, runtimeInfo)
		if len(warnings) != 0 {
			aggregatedWarnings = append(aggregatedWarnings, warnings...)
		}
		if errs != nil {
			aggregatedErrors = append(aggregatedErrors, errs...)
		}
	}
	return aggregatedWarnings, aggregatedErrors
}

func (f *Framework) RunComponentBuilderPlugins(ctx context.Context, info *runtime.Info, trainJob *kubeflowv2.TrainJob) ([]client.Object, error) {
	var objs []client.Object
	for _, plugin := range f.componentBuilderPlugins {
		obj, err := plugin.Build(ctx, info, trainJob)
		if err != nil {
			return nil, err
		}
		if obj != nil {
			objs = append(objs, obj)
		}
	}
	return objs, nil
}

func (f *Framework) WatchExtensionPlugins() []framework.WatchExtensionPlugin {
	return f.watchExtensionPlugins
}
