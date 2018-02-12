# Copyright 2017 The Kubeflow Authors.
#
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

# A Dockerfile suitable for building a docker container suitable for running
# the GKE example notebook.

FROM  gcr.io/cloud-datalab/datalab:local-20171023
LABEL authors="Jeremy Lewi <jeremy@lewi.us>"

# Install Python K8s client library
RUN pip install kubernetes

# Upgrade some of the GCP client librarys
RUN pip install --upgrade google-cloud-logging==1.3.0

# Install docker
# Docker is used when running locally to build the images.
# Note: 1.11+ changes the tarball format
# TODO(jlewi): I think it should be fine to upgrade to newer version of docker.
RUN curl -L "https://get.docker.com/builds/Linux/x86_64/docker-1.9.1.tgz" \
    | tar -C /usr/bin -xvzf- --strip-components=3 usr/local/bin/docker

# Install kubectl
RUN gcloud components install -q kubectl

# Install Helm
RUN wget -O /tmp/get_helm.sh \
    https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get && \
    chmod 700 /tmp/get_helm.sh && \
    /tmp/get_helm.sh && \
    rm /tmp/get_helm.sh
