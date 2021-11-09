# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import logging
import subprocess

logger = logging.getLogger(__name__)


def train(train_config_filepath: str):
    cmd = ["lightgbm", f"config={train_config_filepath}"]
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE)
    line = proc.stdout.readline()
    while line:
        logger.info((line.decode("utf-8").strip()))
        line = proc.stdout.readline()
