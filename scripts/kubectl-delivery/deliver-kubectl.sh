#!/bin/bash

# Copyright 2018 The Kubeflow Authors.
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

# This shell script is used to build an image from our argo workflow

if ! which kubectl > /dev/null; then
  echo "kubectl needs to be installed"
  exit 1
fi

: ${TARGET_DIR:?"Need to set TARGET_DIR, e.g. /opt/kube"}

cp $(which kubectl) ${TARGET_DIR}

file="/etc/mpi/hostfile"
workers=()
i=0
IFS=$'\n'       # make newlines the only separator
set -f          # disable globbing
for line in $(cat < "$file"); do
      echo "$line"
      workers[i]=$line # Put it into the array
      i=$(($i + 1))
      echo $i
done

hello="hello"

for(( i=0;i<${#workers[@]};i++)) do
   worker=$(echo ${workers[i]} | awk '{print $1}')
   echo "worker name is ${worker}"
   while true
     do
        status=$(/opt/kube/kubectl exec -it ${worker} echo "hello")
        status="${status%%[[:cntrl:]]}"
        echo "hello is ${hello}"
        if [[ "a${status}" == "a${hello}" ]] ;then
            echo "${worker} is running.."
            break
        fi
        sleep 1
     done
done;
