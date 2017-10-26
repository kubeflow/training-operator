#!/bin/bash

# Rewrite some imports
files=`find ./ -name *.go`
for f in $files; do
  sed -i "s/github.com\/jlewi\/mlkube.io\//github.com\/tensorflow\/k8s\//" ${f}
done
