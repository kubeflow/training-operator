#!/bin/bash
python -m py.release postsubmit --build_info_path=gs://mlkube-testing-airflow/runs/tf_k8s_tests/manual__2017-11-21T15_26_20.721587/build_info.yaml --releases_path=gs://mlkube-testing-airflow/runs/tf_k8s_tests/manual__2017-11-21T15_26_20.721587 --project=mlkube-testing --src_dir=/tmp/tf_k8s_tests/manual__run
