package util

import "time"

// FakeWorkQueue implements RateLimitingInterface but actually does nothing.
type FakeWorkQueue struct{}

// Add WorkQueue Add method
func (f *FakeWorkQueue) Add(item interface{}) {}

// Len WorkQueue Len method
func (f *FakeWorkQueue) Len() int { return 0 }

// Get WorkQueue Get method
func (f *FakeWorkQueue) Get() (item interface{}, shutdown bool) { return nil, false }

// Done WorkQueue Done method
func (f *FakeWorkQueue) Done(item interface{}) {}

// ShutDown WorkQueue ShutDown method
func (f *FakeWorkQueue) ShutDown() {}

// ShuttingDown WorkQueue ShuttingDown method
func (f *FakeWorkQueue) ShuttingDown() bool { return true }

// AddAfter WorkQueue AddAfter method
func (f *FakeWorkQueue) AddAfter(item interface{}, duration time.Duration) {}

// AddRateLimited WorkQueue AddRateLimited method
func (f *FakeWorkQueue) AddRateLimited(item interface{}) {}

// Forget WorkQueue Forget method
func (f *FakeWorkQueue) Forget(item interface{}) {}

// NumRequeues WorkQueue NumRequeues method
func (f *FakeWorkQueue) NumRequeues(item interface{}) int { return 0 }
