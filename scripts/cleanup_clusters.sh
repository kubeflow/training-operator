#!/bin/bash
#
# Helper script to remove all resources in a kubernetes cluster created by the TPR.

kubectl delete service -l cloud_ml=""
kubectl delete jobs -l cloud_ml=""
kubectl delete pods -l cloud_ml=""


