#!/bin/bash
set -ex

while :; do
echo starting build cycle
SRC_DIR=`mktemp -d /tmp/tfk8s.src.tmp.XXXXXX`

# We should be in the working directory where the copy of py is baked into
# the container
cd /opt/tf_k8s_releaser/
python -m py.release clone --src_dir=${SRC_DIR} lastgreen

GOPATH=${SRC_DIR}/go
mkdir -p ${GOPATH}

# Change to the directory we just cloned so that we pull the code from
# the code we just checkout.
# TODO(jlewi): Uncomment before submitting
cd ${SRC_DIR}
python -m py.release build_new_release \
  --src_dir=${SRC_DIR} \
  --registry=gcr.io/tf-on-k8s-dogfood \
  --project=tf-on-k8s-releasing \
  --releases_path=gs://tf-on-k8s-dogfood-releases

rm -rf ${SRC_DIR}

sleep 300
done