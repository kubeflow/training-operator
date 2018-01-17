# Copyright 2017 The Kubernetes Authors.
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
#
# Requirements:
#   https://github.com/mattrobenolt/jinja2-cli
#   pip install jinja2-clie
IMG = gcr.io/mlkube-testing/airflow
# PROW_IMG is the image used by our Prow jobs to launch our tests.
PROW_IMG = gcr.io/mlkube-testing/builder
TAG := $(shell date +v%Y%m%d)-$(shell git describe --tags --always --dirty)-$(shell git diff | sha256sum | cut -c -6)
DIR := ${CURDIR}

all: build

# To build without the cache set the environment variable
# export DOCKER_BUILD_OPTS=--no-cache
build:
	@echo {\"image\": \"$(IMG):$(TAG)\"} > version.json
	mkdir -p ./py
	cp ../../../py/*.py ./py/
	docker build ${DOCKER_BUILD_OPTS} -t $(IMG):$(TAG) .
	docker tag $(IMG):$(TAG) $(IMG):latest
	@echo Built $(IMG):$(TAG) and tagged with latest
	# Build the image to be used by prow.
	jinja2 Dockerfile.prow.template --format=yaml -D base_image=$(IMG):$(TAG) > Dockerfile.prow
	docker build ${DOCKER_BUILD_OPTS} -t $(PROW_IMG):$(TAG) -f  Dockerfile.prow .
	docker tag $(PROW_IMG):$(TAG) $(PROW_IMG):latest
	@echo Built $(PROW_IMG):$(TAG) and tagged with latest
	rm -f version.json
	rm -rf ./py
	jinja2 deployment.yaml.template --format=yaml -D image=$(IMG):$(TAG) > deployment.yaml


push: build
	gcloud docker -- push $(IMG):$(TAG)
	gcloud docker -- push $(IMG):latest
	@echo Pushed $(IMG) with :latest and :$(TAG) tags
	gcloud docker -- push $(PROW_IMG):$(TAG)
	gcloud docker -- push $(PROW_IMG):latest
	@echo Pushed $(PROW_IMG) with :latest and :$(TAG) tags

# Update the Airflow deployment
update: push
	kubectl apply -f deployment.yaml

# Start Airflow locally for debugging/development and mount the DAGs as a
# volume mount.
# TODO(jlewi): Should we also mount the python code used by each step into the container?
# That might work if we changed the user id of Airflow to match the userid of the personing
# running the container.
run_airflow: build
	./start_airflow_local.py

# Start the postgre container locally for debugging
run_postgre:
	-docker rm -f postgres
	docker run -d --network=airflow --name=postgres -p 5432:5432 -e POSTGRES_USER=airflow -e POSTGRES_PASSWORD=airflow -e POSTGRES_DB=airflow  postgres:10.1-alpine


