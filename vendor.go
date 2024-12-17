//go:build vendor

package main

// This file exists to trick "go mod vendor" to include "main" packages.
// It is not expected to build, the build tag above is only to prevent this
// file from being included in builds.

import (
	_ "k8s.io/code-generator"
)

func main() {}
