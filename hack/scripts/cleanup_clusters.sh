#!/bin/bash
#
# Helper script to remove all resources in a kubernetes cluster created by the CRD.

kubectl delete service --selector='kubeflow.org='
kubectl delete jobs --selector='kubeflow.org='
kubectl delete pods --selector='kubeflow.org='



