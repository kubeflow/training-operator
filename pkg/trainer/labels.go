package trainer

import (
	"fmt"
	"strings"
)

// KubernetesLabels represents a set of labels to apply to a Kubernetes resources.
type KubernetesLabels map[string]string

// ToSelector converts the labels to a selector matching the labels.
func (l KubernetesLabels) ToSelector() (string, error) {
	pieces := make([]string, 0, len(l))
	for k, v := range l {
		pieces = append(pieces, fmt.Sprintf("%v=%v", k, v))
	}

	return strings.Join(pieces, ","), nil
}
