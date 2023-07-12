package v1

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

func TestSetElasticPolicy(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	type args struct {
		job *PyTorchJob
	}
	type result struct {
		expectedMinReplicas *int32
		expectedMaxReplicas *int32
	}
	tests := []struct {
		name   string
		args   args
		result result
	}{
		{
			name: "minReplicas and maxReplicas to null",
			args: args{
				job: &PyTorchJob{
					Spec: PyTorchJobSpec{
						ElasticPolicy: &ElasticPolicy{},
						PyTorchReplicaSpecs: map[ReplicaType]*ReplicaSpec{
							PyTorchJobReplicaTypeWorker: {
								Replicas: pointer.Int32(1),
							},
						},
					},
				},
			},
			result: result{
				expectedMinReplicas: pointer.Int32(1),
				expectedMaxReplicas: pointer.Int32(1),
			},
		},
		{
			name: "minReplicas and maxReplicas to 1",
			args: args{
				job: &PyTorchJob{
					Spec: PyTorchJobSpec{
						ElasticPolicy: &ElasticPolicy{
							MaxReplicas: pointer.Int32(1),
							MinReplicas: pointer.Int32(1),
						},
						PyTorchReplicaSpecs: map[ReplicaType]*ReplicaSpec{
							PyTorchJobReplicaTypeWorker: {
								Replicas: pointer.Int32(1),
							},
						},
					},
				},
			},
			result: result{
				expectedMinReplicas: pointer.Int32(1),
				expectedMaxReplicas: pointer.Int32(1),
			},
		},
		{
			name: "minReplicas and maxReplicas to 1",
			args: args{
				job: &PyTorchJob{
					Spec: PyTorchJobSpec{
						ElasticPolicy: &ElasticPolicy{
							MaxReplicas: pointer.Int32(1),
							MinReplicas: pointer.Int32(1),
						},
						PyTorchReplicaSpecs: map[ReplicaType]*ReplicaSpec{
							PyTorchJobReplicaTypeWorker: {
								Replicas: pointer.Int32(1),
							},
						},
					},
				},
			},
			result: result{
				expectedMinReplicas: pointer.Int32(1),
				expectedMaxReplicas: pointer.Int32(1),
			},
		},
		{
			name: "minReplicas to null, maxRepliacs to 1",
			args: args{
				job: &PyTorchJob{
					Spec: PyTorchJobSpec{
						ElasticPolicy: &ElasticPolicy{
							MaxReplicas: pointer.Int32(1),
							MinReplicas: nil,
						},
						PyTorchReplicaSpecs: map[ReplicaType]*ReplicaSpec{
							PyTorchJobReplicaTypeWorker: {
								Replicas: pointer.Int32(1),
							},
						},
					},
				},
			},
			result: result{
				expectedMinReplicas: pointer.Int32(1),
				expectedMaxReplicas: pointer.Int32(1),
			},
		},
		{
			name: "maxRepliacs to null, minReplicas to 1",
			args: args{
				job: &PyTorchJob{
					Spec: PyTorchJobSpec{
						ElasticPolicy: &ElasticPolicy{
							MaxReplicas: nil,
							MinReplicas: pointer.Int32(1),
						},
						PyTorchReplicaSpecs: map[ReplicaType]*ReplicaSpec{
							PyTorchJobReplicaTypeWorker: {
								Replicas: pointer.Int32(1),
							},
						},
					},
				},
			},
			result: result{
				expectedMinReplicas: pointer.Int32(1),
				expectedMaxReplicas: pointer.Int32(1),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setElasticPolicy(test.args.job)
			if test.result.expectedMinReplicas != nil {
				gomega.Expect(test.args.job.Spec.ElasticPolicy.MinReplicas).
					To(gomega.Equal(test.result.expectedMinReplicas))
			} else {
				gomega.Expect(test.args.job.Spec.ElasticPolicy.MinReplicas).
					To(gomega.BeNil())
			}

			if test.result.expectedMaxReplicas != nil {
				gomega.Expect(test.args.job.Spec.ElasticPolicy.MaxReplicas).
					To(gomega.Equal(test.result.expectedMaxReplicas))
			} else {
				gomega.Expect(test.args.job.Spec.ElasticPolicy.MaxReplicas).
					To(gomega.BeNil())
			}
		})
	}
}

func TestSetDefaultNprocPerNode(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	t.Run("test default nproc per node", func(t *testing.T) {
		job := &PyTorchJob{
			Spec: PyTorchJobSpec{
				ElasticPolicy: &ElasticPolicy{
					NProcPerNode: nil,
				},
				PyTorchReplicaSpecs: map[ReplicaType]*ReplicaSpec{
					PyTorchJobReplicaTypeWorker: {
						Replicas: pointer.Int32(1),
					},
				},
			},
		}

		setDefaultNprocPerNode(job)
		gomega.Expect(job.Spec.NprocPerNode).
			To(gomega.Equal(&DefaultNprocPerNode))
	})
	t.Run("test default nproc per node", func(t *testing.T) {
		job := &PyTorchJob{
			Spec: PyTorchJobSpec{
				ElasticPolicy: nil,
				PyTorchReplicaSpecs: map[ReplicaType]*ReplicaSpec{
					PyTorchJobReplicaTypeWorker: {
						Replicas: pointer.Int32(1),
					},
				},
			},
		}

		setDefaultNprocPerNode(job)
		gomega.Expect(job.Spec.NprocPerNode).
			To(gomega.Equal(&DefaultNprocPerNode))
	})
}
