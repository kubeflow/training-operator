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

// Package that various helper routines for training.
package train

func IsRetryableExitCode(exitCode int32) bool {
	if exitCode == 1 || exitCode == 2 || exitCode == 126 ||
		exitCode == 127 || exitCode == 128 || exitCode == 139 {
		// Refers to http://tldp.org/LDP/abs/html/exitcodes.html, we identify the following exit codes
		// as permanent errors:
		//   1: General errors
		//   2: Misuse of shell builtins
		//   126: Command invoked cannot execute
		//   127: Command not found
		//   128: Invalid argument to exit
		//   139(128+11): terminated by SIGSEGV(Invalid memory reference)
		return false
	}

	if exitCode == 130 || exitCode == 137 || exitCode == 143 {
		// We think it's retryable error if the container exits due to the following sys signals
		// that are usually caused by transient issues(e.g. VM was rescheduled):
		//   130(128+2): Container terminated by Control-C
		//   137(128+9): Container received a SIGKILL
		//   143(128+15): Container received a SIGTERM
		// The exit code of container will be 128 + n for fatal error signals.
		// More info can be found in:
		//   http://tldp.org/LDP/abs/html/exitcodes.html,
		//   https://stackoverflow.com/questions/31297616/what-is-the-authoritative-list-of-docker-run-exit-codes
		return true
	}

	if exitCode == 138 {
		// We allow users to specify exit code for the cases that they think should retry.
		// We decide to take the exit code of SIGUSR1(138 = 128 + 10) for user defined retryable error.
		return true
	}

	// We make no guarantee for other exit status. Currently handling them same as permanent errors.
	return false
}
