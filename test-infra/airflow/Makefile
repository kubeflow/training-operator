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

IMG = gcr.io/mlkube-testing/airflow
TAG = $(shell date +v%Y%m%d)-$(shell git describe --tags --always --dirty)-$(shell git diff | sha256sum | cut -c -6)


all: build

build:
	@echo {\"image\": \"$(IMG):$(TAG)\"} > version.json
	mkdir -p ./py
	cp ../../py/*.py ./py/
	docker build -t $(IMG):$(TAG) .
	docker tag $(IMG):$(TAG) $(IMG):latest
	@echo Built $(IMG):$(TAG) and tagged with latest
	rm -f version.json
	rm -rf ./py

push: build
	gcloud docker -- push $(IMG):$(TAG)
	gcloud docker -- push $(IMG):latest
	@echo Pushed $(IMG) with :latest and :$(TAG) tags

# TODO(jlewi): Don't commit this.
# This is a helper command to restart Airflow for local testing
debug: build
	docker rm -f airflow
	docker run -d -e POSTGRES_HOST=postgre -p 8080:8080 --name=airflow -v /home/jlewi/git_training/test-infra/airflow/dags:/usr/local/airflow/dags --network=airflow gcr.io/mlkube-testing/airflow:latest webserver

# Start the postgre container locally for debugging
postgre:
	docker rm -f postgre
	docker run -d --network=airflow --name=postgre -p 5432:5432 -e POSTGRES_USER=airflow -e POSTGRES_PASSWORD=airflow -e POSTGRES_DB=airflow postgres:9.6


