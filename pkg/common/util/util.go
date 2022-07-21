// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package util

import (
	"fmt"
	"time"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObjectFilterFunction func(obj metav1.Object) bool

// ConvertServiceList convert service list to service point list
func ConvertServiceList(list []corev1.Service) []*corev1.Service {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Service, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

// ConvertPodList convert pod list to pod pointer list
func ConvertPodList(list []corev1.Pod) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

// ConvertPodListWithFilter converts pod list to pod pointer list with ObjectFilterFunction
func ConvertPodListWithFilter(list []corev1.Pod, pass ObjectFilterFunction) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		obj := &list[i]
		if pass != nil {
			if pass(obj) {
				ret = append(ret, obj)
			}
		} else {
			ret = append(ret, obj)
		}
	}
	return ret
}

func GetReplicaTypes(specs map[commonv1.ReplicaType]*commonv1.ReplicaSpec) []commonv1.ReplicaType {
	keys := make([]commonv1.ReplicaType, 0, len(specs))
	for k := range specs {
		keys = append(keys, k)
	}
	return keys
}

// DurationUntilExpireTime returns the duration until job needs to be cleaned up, or -1 if it's infinite.
func DurationUntilExpireTime(runPolicy *commonv1.RunPolicy, jobStatus commonv1.JobStatus) (time.Duration, error) {
	if !commonutil.IsSucceeded(jobStatus) && !commonutil.IsFailed(jobStatus) {
		return -1, nil
	}
	currentTime := time.Now()
	ttl := runPolicy.TTLSecondsAfterFinished
	if ttl == nil {
		return -1, nil
	}
	duration := time.Second * time.Duration(*ttl)
	if jobStatus.CompletionTime == nil {
		return -1, fmt.Errorf("job completion time is nil, cannot cleanup")
	}
	finishTime := jobStatus.CompletionTime
	expireTime := finishTime.Add(duration)
	if currentTime.After(expireTime) {
		return 0, nil
	} else {
		return expireTime.Sub(currentTime), nil
	}
}
