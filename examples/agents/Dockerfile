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

FROM gcr.io/tensorflow/tensorflow:1.3.0

# For pulling agents via github
RUN apt-get update && apt-get install -y git

# For renders
RUN apt-get install -y libav-tools

# Agents and environment interaction
RUN pip install -e git://github.com/tensorflow/agents.git#egg=agents

# For environments
RUN pip install gym pybullet

ADD task.py /app/
WORKDIR /app/

ENTRYPOINT ["python", "task.py"]
