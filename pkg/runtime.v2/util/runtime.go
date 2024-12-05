package runtime

import (
	"errors"
	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

var ErrorUnsupportedRuntime = errors.New("the specified runtime is not supported")

func RuntimeRefToGroupKind(runtimeRef kubeflowv2.RuntimeRef) schema.GroupKind {
	return schema.GroupKind{
		Group: ptr.Deref(runtimeRef.APIGroup, ""),
		Kind:  ptr.Deref(runtimeRef.Kind, ""),
	}
}
