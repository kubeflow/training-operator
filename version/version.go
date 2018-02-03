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
	"runtime"
)

var (
	Version = "0.3.0+git"
	GitSHA  = "Not provided."
)

// get version info
func Info() string{
	return fmt.Sprintf("Version: %v, Git SHA: %s, Go Version: %s, Go OS/Arch: %s/%s", Version, GitSHA, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
