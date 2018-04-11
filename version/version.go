// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"fmt"
	"os"
	"runtime"
)

var (
	Version = "v0.1.0-alpha"
	GitSHA  = "Not provided."
)

// PrintVersionAndExit prints versions from the array returned by Info() and exit
func PrintVersionAndExit(apiVersion string) {
	for _, i := range Info(apiVersion) {
		fmt.Printf("%v\n", i)
	}
	os.Exit(0)
}

// Info returns an array of various service versions
func Info(apiVersion string) []string {
	return []string{
		fmt.Sprintf("API Version: %s", apiVersion),
		fmt.Sprintf("Version: %s", Version),
		fmt.Sprintf("Git SHA: %s", GitSHA),
		fmt.Sprintf("Go Version: %s", runtime.Version()),
		fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
