#!/bin/bash

# Rewrite some imports
files=`find ./ -name *.go`
for f in $files; do
  sed -i "s/mlkube.io\//github.com\/jlewi\/mlkube.io\//" ${f}
done
