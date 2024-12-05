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

import "time"

// FakeWorkQueue implements RateLimitingInterface but actually does nothing.
type FakeWorkQueue[T any] struct{}

// Add WorkQueue Add method
func (f *FakeWorkQueue[T]) Add(item T) {}

// Len WorkQueue Len method
func (f *FakeWorkQueue[T]) Len() int { return 0 }

// Get WorkQueue Get method
func (f *FakeWorkQueue[T]) Get() (item T, shutdown bool) {
	var empty T
	return empty, false
}

// Done WorkQueue Done method
func (f *FakeWorkQueue[T]) Done(item T) {}

// ShutDown WorkQueue ShutDown method
func (f *FakeWorkQueue[T]) ShutDown() {}

// ShutDownWithDrain WorkQueue ShutDownWithDrain method
func (f *FakeWorkQueue[T]) ShutDownWithDrain() {}

// ShuttingDown WorkQueue ShuttingDown method
func (f *FakeWorkQueue[T]) ShuttingDown() bool { return true }

// AddAfter WorkQueue AddAfter method
func (f *FakeWorkQueue[T]) AddAfter(item T, duration time.Duration) {}

// AddRateLimited WorkQueue AddRateLimited method
func (f *FakeWorkQueue[T]) AddRateLimited(item T) {}

// Forget WorkQueue Forget method
func (f *FakeWorkQueue[T]) Forget(item T) {}

// NumRequeues WorkQueue NumRequeues method
func (f *FakeWorkQueue[T]) NumRequeues(item T) int { return 0 }
