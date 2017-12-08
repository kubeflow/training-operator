# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#FROM gcr.io/dev01-181118-181500/agents-base:cpu
FROM gcr.io/dev01-181118-181500/agents-base:cpu-tf1.4.0

ADD trainer/* /app/trainer/
WORKDIR /app/

# TODO: Need better smoke testing of container
RUN python -m trainer.task --help

ENTRYPOINT ["python", "-m", "trainer.task"]
