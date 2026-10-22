/*
Copyright 2025 The Kubeflow Authors.

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

package mpi

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	batchv1ac "k8s.io/client-go/applyconfigurations/batch/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/klog/v2/ktesting"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	jobsetv1alpha2ac "sigs.k8s.io/jobset/client-go/applyconfiguration/jobset/v1alpha2"

	trainer "github.com/kubeflow/trainer/v2/pkg/apis/trainer/v1alpha1"
	"github.com/kubeflow/trainer/v2/pkg/constants"
	"github.com/kubeflow/trainer/v2/pkg/runtime"
	"github.com/kubeflow/trainer/v2/pkg/runtime/framework"
	utiltesting "github.com/kubeflow/trainer/v2/pkg/util/testing"
)

func TestMPI(t *testing.T) {
	objCmpOpts := []gocmp.Option{
		cmpopts.SortSlices(func(a, b apiruntime.Object) int {
			return cmp.Compare(a.GetObjectKind().GroupVersionKind().String(), b.GetObjectKind().GroupVersionKind().String())
		}),
		cmpopts.SortSlices(func(a, b corev1.EnvVar) int { return cmp.Compare(a.Name, b.Name) }),
		gocmp.Comparer(utiltesting.MPISecretDataComparer),
	}
	errorGetSSHAuthSecretFromAPI := errors.New("failed to get SSH Auth Secret from API during Build")

	cases := map[string]struct {
		info              *runtime.Info
		trainJob          *trainer.TrainJob
		objs              []client.Object
		wantInfo          *runtime.Info
		wantObjs          []apiruntime.Object
		wantMLPolicyError error
		wantBuildError    error
	}{
		"no action when info is nil": {},
		"no action when mlPolicySource is nil": {
			info: &runtime.Info{
				Labels: map[string]string{"key": "value"},
			},
			wantInfo: &runtime.Info{
				Labels: map[string]string{"key": "value"},
			},
		},
		"no action when mlPolicySource mpi is null": {
			info: &runtime.Info{
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().Obj(),
				},
			},
			wantInfo: &runtime.Info{
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().Obj(),
				},
			},
		},
		"trainJob numNodes is respected rather than mlPolicy one": {
			info: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
								yield("trainJob-node-1-1.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), nil).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{
					PodLabels: make(map[string]string),
				},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Trainer(
					utiltesting.MakeTrainJobTrainerWrapper().
						NumNodes(2).
						Obj()).
				Obj(),
			wantInfo: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
								*corev1ac.Volume().
									WithName(constants.MPIHostfileVolumeName).
									WithConfigMap(corev1ac.ConfigMapVolumeSource().
										WithName(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix)).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(constants.MPIHostfileName).
												WithPath(constants.MPIHostfileName).
												WithMode(0444),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](2),
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
								yield("trainJob-node-1-1.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), nil).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			wantObjs: []apiruntime.Object{
				utiltesting.MakeSecretWrapper(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix), metav1.NamespaceDefault).
					WithImmutable(true).
					WithType(corev1.SecretTypeSSHAuth).
					WithData(map[string][]byte{
						constants.MPISSHPublicKey: []byte("EXIST"),
						corev1.SSHAuthPrivateKey:  []byte("EXIST"),
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
				utiltesting.MakeConfigMapWrapper(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix), metav1.NamespaceDefault).
					WithData(map[string]string{
						constants.MPIHostfileName: `trainJob-node-1-0.trainJob slots=1
trainJob-node-1-1.trainJob slots=1
`,
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
			},
		},
		"trainJob numProcPerNode is respected rather than mlPolicy one": {
			info: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), nil).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{
					PodLabels: make(map[string]string),
				},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Trainer(
					utiltesting.MakeTrainJobTrainerWrapper().
						NumNodes(1).
						NumProcPerNode(2).
						Obj()).
				Obj(),
			wantInfo: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
								*corev1ac.Volume().
									WithName(constants.MPIHostfileVolumeName).
									WithConfigMap(corev1ac.ConfigMapVolumeSource().
										WithName(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix)).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(constants.MPIHostfileName).
												WithPath(constants.MPIHostfileName).
												WithMode(0444),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](1),
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](2), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), nil).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			wantObjs: []apiruntime.Object{
				utiltesting.MakeSecretWrapper(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix), metav1.NamespaceDefault).
					WithImmutable(true).
					WithType(corev1.SecretTypeSSHAuth).
					WithData(map[string][]byte{
						constants.MPISSHPublicKey: []byte("EXIST"),
						corev1.SSHAuthPrivateKey:  []byte("EXIST"),
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
				utiltesting.MakeConfigMapWrapper(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix), metav1.NamespaceDefault).
					WithData(map[string]string{
						constants.MPIHostfileName: `trainJob-node-1-0.trainJob slots=2
`,
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
			},
		},
		"numProcPerNode is set to number of GPUs in TrainJob": {
			info: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), nil).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{
					PodLabels: make(map[string]string),
				},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Trainer(
					utiltesting.MakeTrainJobTrainerWrapper().
						NumNodes(1).
						Container("test:trainjob", []string{"trainjob"}, []string{"trainjob"}, corev1.ResourceList{
							"custom.com/gpu": resource.MustParse("5"),
						}).
						Obj()).
				Obj(),
			wantInfo: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
								*corev1ac.Volume().
									WithName(constants.MPIHostfileVolumeName).
									WithConfigMap(corev1ac.ConfigMapVolumeSource().
										WithName(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix)).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(constants.MPIHostfileName).
												WithPath(constants.MPIHostfileName).
												WithMode(0444),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](1),
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](5), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), nil).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			wantObjs: []apiruntime.Object{
				utiltesting.MakeSecretWrapper(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix), metav1.NamespaceDefault).
					WithImmutable(true).
					WithType(corev1.SecretTypeSSHAuth).
					WithData(map[string][]byte{
						constants.MPISSHPublicKey: []byte("EXIST"),
						corev1.SSHAuthPrivateKey:  []byte("EXIST"),
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
				utiltesting.MakeConfigMapWrapper(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix), metav1.NamespaceDefault).
					WithData(map[string]string{
						constants.MPIHostfileName: `trainJob-node-1-0.trainJob slots=5
`,
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
			},
		},
		"runLauncherAsNode is true": {
			info: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(true)).
						Obj(),
				},
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
							Containers: []runtime.Container{{
								Name: constants.Node,
							}},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
							Containers: []runtime.Container{{
								Name: constants.Node,
							}},
						},
					},
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Trainer(
					utiltesting.MakeTrainJobTrainerWrapper().
						NumNodes(10).
						Obj()).
				Obj(),
			wantInfo: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(true)).
						Obj(),
				},
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Containers: []runtime.Container{{
								Name: constants.Node,
								VolumeMounts: []corev1ac.VolumeMountApplyConfiguration{
									*corev1ac.VolumeMount().
										WithName(constants.MPISSHAuthVolumeName).
										WithMountPath("/root/.ssh"),
									*corev1ac.VolumeMount().
										WithName(constants.MPIHostfileVolumeName).
										WithMountPath("/etc/mpi"),
								},
								Env: []corev1ac.EnvVarApplyConfiguration{
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvHostFileLocation).
										WithValue(fmt.Sprintf("%s/%s", constants.MPIHostfileDir, constants.MPIHostfileName)),
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvKeepFQDNHostNames).
										WithValue("true"),
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvDefaultSlots).
										WithValue("1"),
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvKeyRSHArgs).
										WithValue(constants.OpenMPIEnvDefaultValueRSHArgs),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvHostFileLocation).
										WithValue(fmt.Sprintf("%s/%s", constants.MPIHostfileDir, constants.MPIHostfileName)),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvKeepFQDNHostNames).
										WithValue("false"),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvDefaultSlots).
										WithValue("1"),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvKeySSHArgs).
										WithValue(constants.OpenMPIEnvDefaultValueRSHArgs),
								},
							}},
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
								*corev1ac.Volume().
									WithName(constants.MPIHostfileVolumeName).
									WithConfigMap(corev1ac.ConfigMapVolumeSource().
										WithName(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix)).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(constants.MPIHostfileName).
												WithPath(constants.MPIHostfileName).
												WithMode(0444),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](9),
							Containers: []runtime.Container{{
								Name: constants.Node,
								VolumeMounts: []corev1ac.VolumeMountApplyConfiguration{
									*corev1ac.VolumeMount().
										WithName(constants.MPISSHAuthVolumeName).
										WithMountPath("/root/.ssh"),
								},
							}},
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
						},
					},
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			wantObjs: []apiruntime.Object{
				utiltesting.MakeSecretWrapper(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix), metav1.NamespaceDefault).
					WithImmutable(true).
					WithType(corev1.SecretTypeSSHAuth).
					WithData(map[string][]byte{
						constants.MPISSHPublicKey: []byte("EXIST"),
						corev1.SSHAuthPrivateKey:  []byte("EXIST"),
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
				utiltesting.MakeConfigMapWrapper(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix), metav1.NamespaceDefault).
					WithData(map[string]string{
						constants.MPIHostfileName: `trainJob-launcher-0-0.trainJob slots=1
trainJob-node-1-0.trainJob slots=1
`,
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
			},
		},
		"sshAuthMountPath and mpiImplementation are nil and fall back to defaults": {
			info: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: func() *trainer.MLPolicySource {
						p := utiltesting.MakeMLPolicySourceWrapper().
							MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, ptr.To(true)).
							Obj()
						p.MPI.MPIImplementation = nil
						return p
					}(),
				},
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
							Containers: []runtime.Container{{
								Name: constants.Node,
							}},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](1),
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
							Containers: []runtime.Container{{
								Name: constants.Node,
							}},
						},
					},
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Trainer(
					utiltesting.MakeTrainJobTrainerWrapper().
						NumNodes(10).
						Obj()).
				Obj(),
			wantInfo: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: func() *trainer.MLPolicySource {
						p := utiltesting.MakeMLPolicySourceWrapper().
							MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, ptr.To(true)).
							Obj()
						p.MPI.MPIImplementation = nil
						return p
					}(),
				},
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name:  constants.Launcher,
							Count: ptr.To[int32](1),
							Containers: []runtime.Container{{
								Name: constants.Node,
								VolumeMounts: []corev1ac.VolumeMountApplyConfiguration{
									*corev1ac.VolumeMount().
										WithName(constants.MPISSHAuthVolumeName).
										WithMountPath(constants.MPISSHAuthDefaultMountPath),
									*corev1ac.VolumeMount().
										WithName(constants.MPIHostfileVolumeName).
										WithMountPath("/etc/mpi"),
								},
								Env: []corev1ac.EnvVarApplyConfiguration{
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvHostFileLocation).
										WithValue(fmt.Sprintf("%s/%s", constants.MPIHostfileDir, constants.MPIHostfileName)),
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvKeepFQDNHostNames).
										WithValue("true"),
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvDefaultSlots).
										WithValue("1"),
									*corev1ac.EnvVar().
										WithName(constants.OpenMPIEnvKeyRSHArgs).
										WithValue(constants.OpenMPIEnvDefaultValueRSHArgs),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvHostFileLocation).
										WithValue(fmt.Sprintf("%s/%s", constants.MPIHostfileDir, constants.MPIHostfileName)),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvKeepFQDNHostNames).
										WithValue("false"),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvDefaultSlots).
										WithValue("1"),
									*corev1ac.EnvVar().
										WithName(constants.PRTEEnvKeySSHArgs).
										WithValue(constants.OpenMPIEnvDefaultValueRSHArgs),
								},
							}},
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
								*corev1ac.Volume().
									WithName(constants.MPIHostfileVolumeName).
									WithConfigMap(corev1ac.ConfigMapVolumeSource().
										WithName(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix)).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(constants.MPIHostfileName).
												WithPath(constants.MPIHostfileName).
												WithMode(0444),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
						{
							Name:     constants.Node,
							Ancestor: ptr.To(constants.AncestorTrainer),
							Count:    ptr.To[int32](9),
							Containers: []runtime.Container{{
								Name: constants.Node,
								VolumeMounts: []corev1ac.VolumeMountApplyConfiguration{
									*corev1ac.VolumeMount().
										WithName(constants.MPISSHAuthVolumeName).
										WithMountPath(constants.MPISSHAuthDefaultMountPath),
								},
							}},
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-node-1-0.trainJob")
							},
						},
					},
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			wantObjs: []apiruntime.Object{
				utiltesting.MakeSecretWrapper(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix), metav1.NamespaceDefault).
					WithImmutable(true).
					WithType(corev1.SecretTypeSSHAuth).
					WithData(map[string][]byte{
						constants.MPISSHPublicKey: []byte("EXIST"),
						corev1.SSHAuthPrivateKey:  []byte("EXIST"),
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
				utiltesting.MakeConfigMapWrapper(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix), metav1.NamespaceDefault).
					WithData(map[string]string{
						constants.MPIHostfileName: `trainJob-launcher-0-0.trainJob slots=1
trainJob-node-1-0.trainJob slots=1
`,
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
			},
		},
		"sshAuth secret already has existed in the cluster": {
			objs: []client.Object{
				utiltesting.MakeSecretWrapper(sshAuthSecretName("trainJob"), metav1.NamespaceDefault).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					WithImmutable(true).
					Obj(),
			},
			info: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name: constants.Launcher,
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(true)).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Obj(),
			wantInfo: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name: constants.Launcher,
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
								*corev1ac.Volume().
									WithName(constants.MPIHostfileVolumeName).
									WithConfigMap(corev1ac.ConfigMapVolumeSource().
										WithName(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix)).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(constants.MPIHostfileName).
												WithPath(constants.MPIHostfileName).
												WithMode(0444),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(true)).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			wantObjs: []apiruntime.Object{
				utiltesting.MakeConfigMapWrapper(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix), metav1.NamespaceDefault).
					WithData(map[string]string{
						constants.MPIHostfileName: `trainJob-launcher-0-0.trainJob slots=1
`,
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
			},
		},
		"failed to get sshAuth secret due to API error": {
			info: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name: constants.Launcher,
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(true)).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Obj(),
			wantInfo: &runtime.Info{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				TemplateSpec: runtime.TemplateSpec{
					PodSets: []runtime.PodSet{
						{
							Name: constants.Launcher,
							Volumes: []corev1ac.VolumeApplyConfiguration{
								*corev1ac.Volume().
									WithName(constants.MPISSHAuthVolumeName).
									WithSecret(corev1ac.SecretVolumeSource().
										WithSecretName(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix)).
										WithDefaultMode(constants.MPISSHAuthDefaultMode).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(corev1.SSHAuthPrivateKey).
												WithPath(constants.MPISSHPrivateKeyFile).
												WithMode(constants.MPISSHPrivateKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHPublicKeyFile).
												WithMode(constants.MPISSHPublicKeyFileMode),
											corev1ac.KeyToPath().
												WithKey(constants.MPISSHPublicKey).
												WithPath(constants.MPISSHAuthorizedKeys).
												WithMode(constants.MPISSHPublicKeyFileMode),
										),
									),
								*corev1ac.Volume().
									WithName(constants.MPIHostfileVolumeName).
									WithConfigMap(corev1ac.ConfigMapVolumeSource().
										WithName(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix)).
										WithItems(
											corev1ac.KeyToPath().
												WithKey(constants.MPIHostfileName).
												WithPath(constants.MPIHostfileName).
												WithMode(0444),
										),
									),
							},
							Endpoints: func(yield func(string) bool) {
								yield("trainJob-launcher-0-0.trainJob")
							},
						},
					},
				},
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(true)).
						Obj(),
				},
				Scheduler: &runtime.Scheduler{PodLabels: make(map[string]string)},
			},
			wantBuildError: errorGetSSHAuthSecretFromAPI,
		},
		"nil NumProcPerNode in runtime and trainjob defaults safely without panic": {
			info: &runtime.Info{
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(nil, trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(false)).
						Obj(),
				},
			},
			trainJob: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "trainJob").
				UID("trainJob").
				Obj(),
			wantInfo: &runtime.Info{
				RuntimePolicy: runtime.RuntimePolicy{
					MLPolicySource: utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(nil, trainer.MPIImplementationOpenMPI, ptr.To("/root/.ssh"), ptr.To(false)).
						Obj(),
				},
			},
			wantObjs: []apiruntime.Object{
				utiltesting.MakeSecretWrapper(fmt.Sprintf("trainJob%s", constants.MPISSHAuthSecretSuffix), metav1.NamespaceDefault).
					WithImmutable(true).
					WithType(corev1.SecretTypeSSHAuth).
					WithData(map[string][]byte{
						constants.MPISSHPublicKey: []byte("EXIST"),
						corev1.SSHAuthPrivateKey:  []byte("EXIST"),
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
				utiltesting.MakeConfigMapWrapper(fmt.Sprintf("trainJob%s", constants.MPIHostfileConfigMapSuffix), metav1.NamespaceDefault).
					WithData(map[string]string{
						constants.MPIHostfileName: "",
					}).
					ControllerReference(trainer.SchemeGroupVersion.WithKind(trainer.TrainJobKind), "trainJob", "trainJob").
					Obj(),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			var cancel func()
			ctx, cancel = context.WithCancel(ctx)
			t.Cleanup(cancel)
			b := utiltesting.NewClientBuilder().WithObjects(tc.objs...)
			b.WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*corev1.Secret); ok && errors.Is(tc.wantBuildError, errorGetSSHAuthSecretFromAPI) {
						return errorGetSSHAuthSecretFromAPI
					}
					return client.Get(ctx, key, obj, opts...)
				},
			})
			cli := b.Build()

			p, err := New(ctx, cli, nil, nil)
			if err != nil {
				t.Fatalf("Failed to initialize MPI plugin: %v", err)
			}
			err = p.(framework.EnforceMLPolicyPlugin).EnforceMLPolicy(tc.info, tc.trainJob)
			if diff := gocmp.Diff(tc.wantMLPolicyError, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected error from EnforceMLPolicy (-want, +got): %s", diff)
			}
			if diff := gocmp.Diff(tc.wantInfo, tc.info,
				cmpopts.SortSlices(func(a, b string) bool { return a < b }),
				cmpopts.SortMaps(func(a, b int) bool { return a < b }),
				utiltesting.PodSetEndpointsCmpOpts,
			); len(diff) != 0 {
				t.Errorf("Unexpected info from EnforceMLPolicy (-want, +got): %s", diff)
			}
			var objs []apiruntime.ApplyConfiguration
			objs, err = p.(framework.ComponentBuilderPlugin).Build(ctx, tc.info, tc.trainJob)
			if diff := gocmp.Diff(tc.wantBuildError, err, cmpopts.EquateErrors()); len(diff) != 0 {
				t.Errorf("Unexpected error from Build (-want, +got): %s", diff)
			}
			var typedObjs []apiruntime.Object
			typedObjs, err = utiltesting.ToObject(cli.Scheme(), objs...)
			if err != nil {
				t.Errorf("Failed to convert object: %v", err)
			}
			if diff := gocmp.Diff(tc.wantObjs, typedObjs, objCmpOpts...); len(diff) != 0 {
				t.Errorf("Unexpected objects from Build (-want, +got): %s", diff)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	cases := map[string]struct {
		info         *runtime.Info
		oldObj       *trainer.TrainJob
		newObj       *trainer.TrainJob
		wantError    field.ErrorList
		wantWarnings admission.Warnings
	}{
		"no action when info is nil ": {},
		"no action when info does not have MLPolicySource": {
			info: runtime.NewInfo(),
			oldObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Obj(),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Obj(),
		},
		"info does not have MPIPolicySource": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().Obj()),
			),
			oldObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Obj(),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Obj(),
		},
		"numProcPerNode typed is int": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, nil).
						Obj(),
					).
					Obj(),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Trainer(utiltesting.MakeTrainJobTrainerWrapper().
					NumProcPerNode(8).
					Obj(),
				).
				Obj(),
		},
		"runtime does not have Launcher but TrainJob has only 1 numNodes": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, ptr.To(true)).
						Obj(),
					).
					Obj(),
				),
				runtime.WithPodSet(constants.Node, ptr.To(constants.AncestorTrainer), 1, corev1.PodSpec{}, corev1ac.PodSpec().
					WithContainers(corev1ac.Container().WithName(constants.Node)),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Trainer(&trainer.Trainer{NumNodes: ptr.To(int32(1))}).Obj(),
		},
		"runtime does not have Launcher even though MPI with runLauncherAsNode has 2 numNodes": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, ptr.To(true)).
						Obj(),
					).
					Obj(),
				),
				runtime.WithPodSet(constants.Node, nil, 1, corev1.PodSpec{}, corev1ac.PodSpec().
					WithContainers(corev1ac.Container().WithName(constants.Node)),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Trainer(&trainer.Trainer{NumNodes: ptr.To(int32(2))}).Obj(),
			wantError: field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("trainer", "numNodes"), ptr.To(int32(2)), "must have 1 when MPI trainingRuntime with enabled runLauncherAsNode does not have either launcher and node"),
			},
		},
		"runtime does not have Node even though MPI with runLauncherAsNode has 2 numNodes": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, ptr.To(true)).
						Obj(),
					).
					Obj(),
				),
				runtime.WithPodSet(constants.Launcher, nil, 1, corev1.PodSpec{}, corev1ac.PodSpec().
					WithContainers(corev1ac.Container().WithName(constants.Node)),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Trainer(&trainer.Trainer{NumNodes: ptr.To(int32(2))}).Obj(),
			wantError: field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("trainer", "numNodes"), ptr.To(int32(2)), "must have 1 when MPI trainingRuntime with enabled runLauncherAsNode does not have either launcher and node"),
			},
		},
		"runtime does not have Launcher and Node even though MPI with runLauncherAsNode has 2 numNodes": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, ptr.To(true)).
						Obj(),
					).
					Obj(),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").Trainer(&trainer.Trainer{NumNodes: ptr.To(int32(2))}).Obj(),
			wantError: field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("trainer", "numNodes"), ptr.To(int32(2)), "must have 1 when MPI trainingRuntime with enabled runLauncherAsNode does not have either launcher and node"),
			},
		},
		"trainer has reserved MPI env OMPI_MCA_orte_default_hostfile": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, nil).
						Obj(),
					).
					Obj(),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Trainer(utiltesting.MakeTrainJobTrainerWrapper().
					Env(corev1.EnvVar{Name: constants.OpenMPIEnvHostFileLocation, Value: "/custom/hostfile"}).
					Obj(),
				).
				Obj(),
			wantError: field.ErrorList{
				field.Invalid(
					field.NewPath("spec").Child("trainer").Child("env"),
					[]corev1.EnvVar{{Name: constants.OpenMPIEnvHostFileLocation, Value: "/custom/hostfile"}},
					fmt.Sprintf("must not have reserved envs, invalid envs configured: %v", []string{constants.OpenMPIEnvHostFileLocation}),
				),
			},
		},
		"trainer has multiple reserved MPI envs": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, nil).
						Obj(),
					).
					Obj(),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Trainer(utiltesting.MakeTrainJobTrainerWrapper().
					Env(
						corev1.EnvVar{Name: constants.OpenMPIEnvHostFileLocation, Value: "/custom/hostfile"},
						corev1.EnvVar{Name: constants.OpenMPIEnvKeyRSHArgs, Value: "custom-args"},
					).
					Obj(),
				).
				Obj(),
			wantError: field.ErrorList{
				field.Invalid(
					field.NewPath("spec").Child("trainer").Child("env"),
					[]corev1.EnvVar{
						{Name: constants.OpenMPIEnvHostFileLocation, Value: "/custom/hostfile"},
						{Name: constants.OpenMPIEnvKeyRSHArgs, Value: "custom-args"},
					},
					fmt.Sprintf("must not have reserved envs, invalid envs configured: %v", []string{constants.OpenMPIEnvHostFileLocation, constants.OpenMPIEnvKeyRSHArgs}),
				),
			},
		},
		"trainer has reserved PRTE env": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, nil).
						Obj(),
					).
					Obj(),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Trainer(utiltesting.MakeTrainJobTrainerWrapper().
					Env(corev1.EnvVar{Name: constants.PRTEEnvHostFileLocation, Value: "/custom/hostfile"}).
					Obj(),
				).
				Obj(),
			wantError: field.ErrorList{
				field.Invalid(
					field.NewPath("spec").Child("trainer").Child("env"),
					[]corev1.EnvVar{{Name: constants.PRTEEnvHostFileLocation, Value: "/custom/hostfile"}},
					fmt.Sprintf("must not have reserved envs, invalid envs configured: %v", []string{constants.PRTEEnvHostFileLocation}),
				),
			},
		},
		"trainer has non-reserved env is valid": {
			info: runtime.NewInfo(
				runtime.WithMLPolicySource(utiltesting.MakeMLPolicyWrapper().
					WithMLPolicySource(*utiltesting.MakeMLPolicySourceWrapper().
						MPIPolicy(ptr.To[int32](1), trainer.MPIImplementationOpenMPI, nil, nil).
						Obj(),
					).
					Obj(),
				),
			),
			newObj: utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").
				Trainer(utiltesting.MakeTrainJobTrainerWrapper().
					Env(corev1.EnvVar{Name: "CUSTOM_ENV", Value: "custom-value"}).
					Obj(),
				).
				Obj(),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			var cancel func()
			ctx, cancel = context.WithCancel(ctx)
			t.Cleanup(cancel)
			p, err := New(ctx, utiltesting.NewClientBuilder().Build(), nil, nil)
			if err != nil {
				t.Fatalf("Failed to initialize MPI plugin: %v", err)
			}
			warnings, errs := p.(framework.CustomValidationPlugin).Validate(ctx, tc.info, tc.oldObj, tc.newObj)
			if diff := gocmp.Diff(tc.wantError, errs); len(diff) != 0 {
				t.Errorf("Unexpected error from Validate (-want, +got): %s", diff)
			}
			if diff := gocmp.Diff(tc.wantWarnings, warnings); len(diff) != 0 {
				t.Errorf("Unexpected warnings from Validate (-want, +got): %s", diff)
			}
		})
	}
}

// nodeGroupsInfo builds the runtime Info of an MPI runtime with node groups, with one PodSet
// per replicatedJob, as newRuntimeInfo does once the disabled node groups are removed.
func nodeGroupsInfo(mpiPolicy *trainer.MPIMLPolicySource, rJobs ...*jobsetv1alpha2ac.ReplicatedJobApplyConfiguration) *runtime.Info {
	jobSetSpec := jobsetv1alpha2ac.JobSetSpec()
	var podSets []runtime.PodSet
	for _, rJob := range rJobs {
		jobSetSpec.WithReplicatedJobs(rJob)
		ps := runtime.PodSet{Name: *rJob.Name, Count: ptr.To[int32](1)}
		for _, c := range rJob.Template.Spec.Template.Spec.Containers {
			ps.Containers = append(ps.Containers, runtime.Container{Name: *c.Name})
		}
		if *rJob.Name == mpiPolicy.Launcher.JobName {
			ps.Ancestor = ptr.To(constants.AncestorTrainer)
		}
		podSets = append(podSets, ps)
	}
	return &runtime.Info{
		RuntimePolicy: runtime.RuntimePolicy{MLPolicySource: &trainer.MLPolicySource{MPI: mpiPolicy}},
		TemplateSpec:  runtime.TemplateSpec{ObjApply: jobSetSpec, PodSets: podSets},
		Scheduler:     &runtime.Scheduler{PodLabels: make(map[string]string)},
	}
}

func nodeGroupRJob(name string, containers ...*corev1ac.ContainerApplyConfiguration) *jobsetv1alpha2ac.ReplicatedJobApplyConfiguration {
	return jobsetv1alpha2ac.ReplicatedJob().WithName(name).WithTemplate(
		batchv1ac.JobTemplateSpec().WithSpec(batchv1ac.JobSpec().WithTemplate(
			corev1ac.PodTemplateSpec().WithSpec(corev1ac.PodSpec().WithContainers(containers...)))))
}

func TestEnforceMPIGroups(t *testing.T) {
	gpus := func(n string) *corev1ac.ResourceRequirementsApplyConfiguration {
		return corev1ac.ResourceRequirements().WithLimits(corev1.ResourceList{"nvidia.com/gpu": resource.MustParse(n)})
	}
	newPolicy := func() *trainer.MPIMLPolicySource {
		return &trainer.MPIMLPolicySource{
			NumProcPerNode:    ptr.To[int32](1),
			MPIImplementation: ptr.To(trainer.MPIImplementationOpenMPI),
			SSHAuthMountPath:  ptr.To("/home/mpiuser/.ssh"),
			Launcher:          trainer.MPILauncher{JobName: constants.Launcher, RunAsNode: ptr.To(true)},
			NodeGroups: []trainer.MPINodeGroup{
				{JobName: "cpu", NumNodes: ptr.To[int32](1), NumProcPerNode: ptr.To[int32](32)},
				{JobName: "l40s", ContainerName: "mpi", NumNodes: ptr.To[int32](0)},
				{JobName: "w7900", NumNodes: ptr.To[int32](0)},
			},
		}
	}
	// w7900 is disabled, so newRuntimeInfo already removed it and it has no PodSet.
	newRJobs := func() []*jobsetv1alpha2ac.ReplicatedJobApplyConfiguration {
		return []*jobsetv1alpha2ac.ReplicatedJobApplyConfiguration{
			nodeGroupRJob("cpu", corev1ac.Container().WithName(constants.Node)),
			nodeGroupRJob("l40s", corev1ac.Container().WithName("mpi").WithResources(gpus("2")), corev1ac.Container().WithName("log-shipper")),
			nodeGroupRJob(constants.Launcher, corev1ac.Container().WithName(constants.Node)),
		}
	}
	cases := map[string]struct {
		trainer      *trainer.Trainer
		wantCounts   map[string]int32
		wantHostfile string
	}{
		"node groups get their own counts and slots, and the GPU fallback applies per group": {
			trainer: &trainer.Trainer{NodeGroups: []trainer.MPINodeGroupOverride{{Name: "l40s", NumNodes: ptr.To[int32](2)}}},
			wantCounts: map[string]int32{
				"cpu": 1, "l40s": 2, constants.Launcher: 1,
			},
			wantHostfile: "test-cpu-0-0.test slots=32\n" +
				"test-l40s-0-0.test slots=2\n" +
				"test-l40s-0-1.test slots=2\n" +
				"test-launcher-0-0.test slots=1\n",
		},
		"TrainJob numProcPerNode disables the GPU fallback, and the launcher can stop running as a node": {
			trainer: &trainer.Trainer{
				Launcher: &trainer.MPILauncherOverride{RunAsNode: ptr.To(false)},
				NodeGroups: []trainer.MPINodeGroupOverride{{
					Name: "l40s", NumNodes: ptr.To[int32](1), MPIGroupOverride: trainer.MPIGroupOverride{NumProcPerNode: ptr.To[int32](1)},
				}},
			},
			wantCounts: map[string]int32{
				"cpu": 1, "l40s": 1, constants.Launcher: 1,
			},
			wantHostfile: "test-cpu-0-0.test slots=32\n" +
				"test-l40s-0-0.test slots=1\n",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			var cancel func()
			ctx, cancel = context.WithCancel(ctx)
			t.Cleanup(cancel)
			cli := utiltesting.NewClientBuilder().Build()
			p, err := New(ctx, cli, nil, nil)
			if err != nil {
				t.Fatalf("Failed to initialize MPI plugin: %v", err)
			}
			info := nodeGroupsInfo(newPolicy(), newRJobs()...)
			trainJob := utiltesting.MakeTrainJobWrapper(metav1.NamespaceDefault, "test").UID("test").Trainer(tc.trainer).Obj()
			if err = p.(framework.EnforceMLPolicyPlugin).EnforceMLPolicy(info, trainJob); err != nil {
				t.Fatalf("Unexpected error from EnforceMLPolicy: %v", err)
			}

			gotCounts := make(map[string]int32)
			for i, ps := range info.TemplateSpec.PodSets {
				gotCounts[ps.Name] = *ps.Count
				// The JobSet plugin computes the endpoints from the counts in PreBuildSync.
				info.TemplateSpec.PodSets[i].Endpoints = func(yield func(string) bool) {
					for podIdx := range *ps.Count {
						if !yield(fmt.Sprintf("test-%s-0-%d.test", ps.Name, podIdx)) {
							return
						}
					}
				}
			}
			if diff := gocmp.Diff(tc.wantCounts, gotCounts); len(diff) != 0 {
				t.Errorf("Unexpected PodSet counts (-want, +got): %s", diff)
			}

			for _, ps := range info.TemplateSpec.PodSets {
				volumes := make(map[string]bool)
				for _, v := range ps.Volumes {
					volumes[*v.Name] = true
				}
				launcher := ps.Name == constants.Launcher
				if !volumes[constants.MPISSHAuthVolumeName] || volumes[constants.MPIHostfileVolumeName] != launcher {
					t.Errorf("Unexpected volumes in PodSet %s: %v", ps.Name, volumes)
				}
				for _, c := range ps.Containers {
					mpiContainer := c.Name == constants.Node || (ps.Name == "l40s" && c.Name == "mpi")
					mounts := make(map[string]string)
					for _, m := range c.VolumeMounts {
						mounts[*m.Name] = *m.MountPath
					}
					if (mounts[constants.MPISSHAuthVolumeName] == "/home/mpiuser/.ssh") != mpiContainer {
						t.Errorf("Unexpected SSH mount in container %s of PodSet %s: %v", c.Name, ps.Name, mounts)
					}
					hasHostfileEnv := slices.ContainsFunc(c.Env, func(e corev1ac.EnvVarApplyConfiguration) bool {
						return *e.Name == constants.OpenMPIEnvHostFileLocation
					})
					if hasHostfileEnv != (launcher && mpiContainer) {
						t.Errorf("Unexpected hostfile env in container %s of PodSet %s", c.Name, ps.Name)
					}
				}
			}

			objs, err := p.(framework.ComponentBuilderPlugin).Build(ctx, info, trainJob)
			if err != nil {
				t.Fatalf("Unexpected error from Build: %v", err)
			}
			var gotHostfile string
			for _, obj := range objs {
				if cm, ok := obj.(*corev1ac.ConfigMapApplyConfiguration); ok {
					gotHostfile = cm.Data[constants.MPIHostfileName]
				}
			}
			if diff := gocmp.Diff(tc.wantHostfile, gotHostfile); len(diff) != 0 {
				t.Errorf("Unexpected hostfile (-want, +got): %s", diff)
			}
		})
	}
}

func TestValidateMPIGroups(t *testing.T) {
	newPolicy := func() *trainer.MPIMLPolicySource {
		return &trainer.MPIMLPolicySource{
			NumProcPerNode: ptr.To[int32](1),
			Launcher:       trainer.MPILauncher{JobName: constants.Launcher, RunAsNode: ptr.To(false)},
			NodeGroups: []trainer.MPINodeGroup{
				{JobName: "cpu", NumNodes: ptr.To[int32](0)},
				{JobName: "l40s", NumNodes: ptr.To[int32](0)},
			},
		}
	}
	trainerPath := field.NewPath("spec", "trainer")
	cases := map[string]struct {
		mpiPolicy *trainer.MPIMLPolicySource
		trainer   *trainer.Trainer
		wantError field.ErrorList
	}{
		"valid TrainJob enables a node group": {
			mpiPolicy: newPolicy(),
			trainer:   &trainer.Trainer{NodeGroups: []trainer.MPINodeGroupOverride{{Name: "cpu", NumNodes: ptr.To[int32](2)}}},
		},
		"flat trainer fields are accepted by runtimes without node groups": {
			mpiPolicy: &trainer.MPIMLPolicySource{NumProcPerNode: ptr.To[int32](1)},
			trainer:   &trainer.Trainer{NumNodes: ptr.To[int32](2), Image: ptr.To("image")},
		},
		"launcher and nodeGroups are rejected by MPI runtimes without node groups": {
			mpiPolicy: &trainer.MPIMLPolicySource{NumProcPerNode: ptr.To[int32](1)},
			trainer: &trainer.Trainer{
				Launcher:   &trainer.MPILauncherOverride{RunAsNode: ptr.To(true)},
				NodeGroups: []trainer.MPINodeGroupOverride{{Name: "cpu"}},
			},
			wantError: field.ErrorList{
				field.Forbidden(trainerPath.Child("launcher"), mpiNodeGroupsOnlyErrorMsg),
				field.Forbidden(trainerPath.Child("nodeGroups"), mpiNodeGroupsOnlyErrorMsg),
			},
		},
		"nodeGroups are rejected by runtimes without an MPI policy": {
			trainer: &trainer.Trainer{NodeGroups: []trainer.MPINodeGroupOverride{{Name: "cpu"}}},
			wantError: field.ErrorList{
				field.Forbidden(trainerPath.Child("nodeGroups"), mpiNodeGroupsOnlyErrorMsg),
			},
		},
		"flat trainer fields are rejected by runtimes with node groups": {
			mpiPolicy: newPolicy(),
			trainer: &trainer.Trainer{
				Image:    ptr.To("image"),
				NumNodes: ptr.To[int32](2),
				Launcher: &trainer.MPILauncherOverride{RunAsNode: ptr.To(true)},
			},
			wantError: field.ErrorList{
				field.Forbidden(trainerPath.Child("image"), mpiFlatFieldErrorMsg),
				field.Forbidden(trainerPath.Child("numNodes"), mpiFlatFieldErrorMsg),
			},
		},
		"node group names must match the runtime": {
			mpiPolicy: newPolicy(),
			trainer:   &trainer.Trainer{NodeGroups: []trainer.MPINodeGroupOverride{{Name: "h100", NumNodes: ptr.To[int32](1)}, {Name: "cpu", NumNodes: ptr.To[int32](1)}}},
			wantError: field.ErrorList{
				field.NotFound(trainerPath.Child("nodeGroups").Index(0).Child("name"), "h100"),
			},
		},
		"launcher numProcPerNode requires the launcher to run as a node": {
			mpiPolicy: newPolicy(),
			trainer: &trainer.Trainer{
				Launcher:   &trainer.MPILauncherOverride{MPIGroupOverride: trainer.MPIGroupOverride{NumProcPerNode: ptr.To[int32](2)}},
				NodeGroups: []trainer.MPINodeGroupOverride{{Name: "cpu", NumNodes: ptr.To[int32](1)}},
			},
			wantError: field.ErrorList{
				field.Invalid(trainerPath.Child("launcher", "numProcPerNode"), int32(2), "must not be set when the launcher does not run as a node"),
			},
		},
		"at least one MPI host must be enabled": {
			mpiPolicy: newPolicy(),
			wantError: field.ErrorList{
				field.Invalid(trainerPath.Child("nodeGroups"), int32(0),
					"must enable at least one MPI host: set numNodes of a node group or run the launcher as a node"),
			},
		},
		"the launcher running as a node is an MPI host": {
			mpiPolicy: newPolicy(),
			trainer:   &trainer.Trainer{Launcher: &trainer.MPILauncherOverride{RunAsNode: ptr.To(true)}},
		},
		"reserved MPI envs are rejected in node groups": {
			mpiPolicy: newPolicy(),
			trainer: &trainer.Trainer{NodeGroups: []trainer.MPINodeGroupOverride{{
				Name: "cpu", NumNodes: ptr.To[int32](1),
				MPIGroupOverride: trainer.MPIGroupOverride{Env: []corev1.EnvVar{{Name: constants.OpenMPIEnvHostFileLocation, Value: "/tmp/hostfile"}}},
			}}},
			wantError: field.ErrorList{
				field.Invalid(trainerPath.Child("nodeGroups").Index(0).Child("env"),
					[]corev1.EnvVar{{Name: constants.OpenMPIEnvHostFileLocation, Value: "/tmp/hostfile"}},
					fmt.Sprintf("must not have reserved envs, invalid envs configured: %v", []string{constants.OpenMPIEnvHostFileLocation})),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := gocmp.Diff(tc.wantError, validateMPIGroups(tc.mpiPolicy, tc.trainer)); len(diff) != 0 {
				t.Errorf("Unexpected error from validateMPIGroups (-want, +got): %s", diff)
			}
		})
	}
}
