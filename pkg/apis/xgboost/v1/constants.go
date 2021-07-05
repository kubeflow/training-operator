/*

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

package v1

const (
	// GroupName is the group name use in this package.
	GroupName = "kubeflow.org"
	// Kind is the kind name.
	Kind = "XGBoostJob"
	// GroupVersion is the version.
	GroupVersion = "v1"
	// Plural is the Plural for XGBoostJob.
	Plural = "xgboostjobs"
	// Singular is the singular for XGBoostJob.
	Singular = "xgboostjob"
	// XGBOOSTCRD is the CRD name for XGBoostJob.
	XGBoostCRD = "xgboostjobs.kubeflow.org"

	DefaultContainerName     = "xgboostjob"
	DefaultContainerPortName = "xgboostjob-port"
	DefaultPort              = 9999
)
