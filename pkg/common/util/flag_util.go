package util

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GvkListFlag is the custom flag to parse GroupVersionKind list for trial resources.
type GvkListFlag []schema.GroupVersionKind

// Set is the method to convert gvk to string value
func (flag *GvkListFlag) String() string {
	gvkStrings := []string{}
	for _, x := range []schema.GroupVersionKind(*flag) {
		gvkStrings = append(gvkStrings, x.String())
	}
	return strings.Join(gvkStrings, ",")
}

// Set is the method to set gvk from string flag value
func (flag *GvkListFlag) Set(value string) error {
	gvk, _ := schema.ParseKindArg(value)
	if gvk == nil {
		return fmt.Errorf("Invalid GroupVersionKind: %v", value)
	}
	*flag = append(*flag, *gvk)
	return nil
}
