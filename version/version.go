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

	"github.com/qiniu/log"
)

var (
	Version = "0.3.0+git"
	// TODO(jlewi): Need to figure out how to bake in the git version.
	GitSHA = "Not provided."
)

// PrintVersion print version info
func PrintVersion(printVersion bool) {
	if printVersion {
		fmt.Println("tf_operator Version:", Version)
		fmt.Println("Git SHA:", GitSHA)
		fmt.Println("Go Version:", runtime.Version())
		fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	log.Infof("tf_operator Version: %v", Version)
	log.Infof("Git SHA: %s", GitSHA)
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
}
