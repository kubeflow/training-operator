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

package common

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ReconcilerName = "common-reconciler"

// GetReconcilerName returns the name of this reconciler, which is "common-reconciler"
func (r *ReconcilerUtil) GetReconcilerName() string {
	return ReconcilerName
}

// ReconcilerUtil defines a reconciler with utility features
type ReconcilerUtil struct {
	Recorder record.EventRecorder
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

// BareUtilReconciler returns a pointer of ReconcilerUtil with default implementation
func BareUtilReconciler(
	recorder record.EventRecorder,
	log logr.Logger,
	scheme *runtime.Scheme) *ReconcilerUtil {
	return &ReconcilerUtil{
		Recorder: recorder,
		Log:      log,
		Scheme:   scheme,
	}
}

// GetRecorder returns a record.EventRecorder
func (r *ReconcilerUtil) GetRecorder() record.EventRecorder {
	return r.Recorder
}

// GetLogger returns a logr.Logger
func (r *ReconcilerUtil) GetLogger(job client.Object) logr.Logger {
	return r.Log.WithValues(
		job.GetObjectKind().GroupVersionKind().Kind,
		types.NamespacedName{Name: job.GetName(), Namespace: job.GetNamespace()}.String())
}

// GetScheme returns the pointer of runtime.Schemes that is used in this reconciler
func (r *ReconcilerUtil) GetScheme() *runtime.Scheme {
	return r.Scheme
}
