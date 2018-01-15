#!/bin/bash
#
# Helper script to remove all resources in a kubernetes cluster created by the CRD.

kubectl delete service --selector='tensorflow.org='
kubectl delete jobs --selector='tensorflow.org='
kubectl delete pods --selector='tensorflow.org='



