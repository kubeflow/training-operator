package testutil

import (
	"time"
)

const (
	TestImageName = "test-image-for-kubeflow-tf-operator:latest"
	TestTFJobName = "test-tfjob"
	LabelWorker   = "worker"
	LabelPS       = "ps"

	SleepInterval = 500 * time.Millisecond
	ThreadCount   = 1
)

var (
	AlwaysReady = func() bool { return true }
)
