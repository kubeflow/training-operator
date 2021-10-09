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
// limitations under the License.

package controller_v1

import (
	"fmt"
	"strings"

	mxnetv1 "github.com/kubeflow/training-operator/pkg/apis/mxnet/v1"
	pytorchv1 "github.com/kubeflow/training-operator/pkg/apis/pytorch/v1"
	tensorflowv1 "github.com/kubeflow/training-operator/pkg/apis/tensorflow/v1"
	xgboostv1 "github.com/kubeflow/training-operator/pkg/apis/xgboost/v1"
	mxnetcontroller "github.com/kubeflow/training-operator/pkg/controller.v1/mxnet"
	pytorchcontroller "github.com/kubeflow/training-operator/pkg/controller.v1/pytorch"
	tensorflowcontroller "github.com/kubeflow/training-operator/pkg/controller.v1/tensorflow"
	xgboostcontroller "github.com/kubeflow/training-operator/pkg/controller.v1/xgboost"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const ErrTemplateSchemeNotSupported = "scheme %s is not supported yet"

type ReconcilerSetupFunc func(manager manager.Manager, enableGangScheduling bool) error

var SupportedSchemeReconciler = map[string]ReconcilerSetupFunc{
	tensorflowv1.Kind: func(mgr manager.Manager, enableGangScheduling bool) error {
		return tensorflowcontroller.NewReconciler(mgr, enableGangScheduling).SetupWithManager(mgr)
	},
	pytorchv1.Kind: func(mgr manager.Manager, enableGangScheduling bool) error {
		return pytorchcontroller.NewReconciler(mgr, enableGangScheduling).SetupWithManager(mgr)
	},
	mxnetv1.Kind: func(mgr manager.Manager, enableGangScheduling bool) error {
		return mxnetcontroller.NewReconciler(mgr, enableGangScheduling).SetupWithManager(mgr)
	},
	xgboostv1.Kind: func(mgr manager.Manager, enableGangScheduling bool) error {
		return xgboostcontroller.NewReconciler(mgr, enableGangScheduling).SetupWithManager(mgr)
	},
}

type EnabledSchemes []string

func (es *EnabledSchemes) String() string {
	return strings.Join(*es, ",")
}

func (es *EnabledSchemes) Set(kind string) error {
	kind = strings.ToLower(kind)
	for supportedKind := range SupportedSchemeReconciler {
		if strings.ToLower(supportedKind) == kind {
			*es = append(*es, supportedKind)
			return nil
		}
	}
	return fmt.Errorf(ErrTemplateSchemeNotSupported, kind)
}

func (es *EnabledSchemes) FillAll() {
	for supportedKind := range SupportedSchemeReconciler {
		*es = append(*es, supportedKind)
	}
}

func (es *EnabledSchemes) Empty() bool {
	return len(*es) == 0
}
