/*
Copyright 2017 The Kubernetes Authors.

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

package options

import (
	"flag"
	"time"
)

// ServerOption is the main context object for the controller manager.
type ServerOption struct {
	ChaosLevel           int
	ControllerConfigFile string
	PrintVersion         bool
	GCInterval           time.Duration
}

// NewServerOption creates a new CMServer with a default config.
func NewServerOption() *ServerOption {
	s := ServerOption{}
	return &s
}

// AddFlags adds flags for a specific CMServer to the specified FlagSet
func (s *ServerOption) AddFlags(fs *flag.FlagSet) {
	// chaos level will be removed once we have a formal tool to inject failures.
	fs.IntVar(&s.ChaosLevel, "chaos-level", -1, "DO NOT USE IN PRODUCTION - level of chaos injected into the TfJob created by the operator.")
	fs.BoolVar(&s.PrintVersion, "version", false, "Show version and quit")
	fs.DurationVar(&s.GCInterval, "gc-interval", 10*time.Minute, "GC interval")
	fs.StringVar(&s.ControllerConfigFile, "controller-config-file", "", "Path to file containing the controller config.")
}
