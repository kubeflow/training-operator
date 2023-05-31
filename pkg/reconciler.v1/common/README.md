## Reconciler.v1

This is package providing most functionalities in `pkg/controller.v1` in the form of [reconciler](https://book.kubebuilder.io/cronjob-tutorial/controller-overview.html).

To use the reconciler, following methods must be overridden according to the APIs the reconciler handles.

```go
// GetJob returns the job that matches the request
func (r *JobReconciler) GetJob(ctx context.Context, req ctrl.Request) (client.Object, error)

// ExtractReplicasSpec extracts the ReplicasSpec map from this job
func (r *JobReconciler) ExtractReplicasSpec(job client.Object) (map[commonv1.ReplicaType]*commonv1.ReplicaSpec, error)

// ExtractRunPolicy extracts the RunPolicy from this job
func (r *JobReconciler) ExtractRunPolicy(job client.Object) (*commonv1.RunPolicy, error)

// ExtractJobStatus extracts the JobStatus from this job
func (r *JobReconciler) ExtractJobStatus(job client.Object) (*commonv1.JobStatus, error)

// IsMasterRole checks if Pod is the master Pod
func (r *JobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool
```

A simple example can be found at `test_job/reconciler.v1/test_job/test_job_reconciler.go`.
