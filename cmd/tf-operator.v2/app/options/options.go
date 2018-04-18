// Copyright 2018 The Kubeflow Authors
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

package options

import (
	"flag"
)

// ServerOption is the main context object for the controller manager.
type ServerOption struct {
	Kubeconfig    string
	MasterURL     string
	Threadiness   int
	PrintVersion  bool
	JSONLogFormat bool
}

// NewServerOption creates a new CMServer with a default config.
func NewServerOption() *ServerOption {
	s := ServerOption{}
	return &s
}

// AddFlags adds flags for a specific CMServer to the specified FlagSet.
func (s *ServerOption) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.MasterURL, "master", "",
		`The url of the Kubernetes API server,
		 will overrides any value in kubeconfig, only required if out-of-cluster.`)

	fs.IntVar(&s.Threadiness, "threadiness", 2,
		`How many threads to process the main logic`)

	fs.BoolVar(&s.PrintVersion, "version", false, "Show version and quit")

	fs.BoolVar(&s.JSONLogFormat, "json-log-format", false,
		"Set true to use json style log format. Set false to use plaintext style log format")
}
