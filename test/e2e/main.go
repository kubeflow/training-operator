// e2e provides an E2E test for TfJobs.
//
// The test creates TfJobs and runs various checks to ensure various operations work as intended.
// The test is intended to run as a helm test that ensures the TfJob operator is working correctly.
// Thus, the program returns non-zero exit status on error.
//
// TODO(jlewi): Do we need to make the test output conform to the TAP(https://testanything.org/)
// protocol so we can fit into the K8s dashboard
//
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	"github.com/jlewi/mlkube.io/pkg/spec"
	"github.com/jlewi/mlkube.io/pkg/util"
	"github.com/jlewi/mlkube.io/pkg/util/k8sutil"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	Namespace = "default"
)

var (
	image = flag.String("image", "gcr.io/tf-on-k8s-dogfood/tf_sample:latest", "The Docker image to use with the TfJob.")
)

func run() error {
	kubeCli := k8sutil.MustNewKubeClient()
	tfJobClient, err := k8sutil.NewTfJobClient()
	if err != nil {
		return err
	}

	name := "e2e-test-job-" + util.RandString(4)

	original := &spec.TfJob{
		Metadata: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test.mlkube.io": "",
			},
		},
		Spec: spec.TfJobSpec{
			ReplicaSpecs: []*spec.TfReplicaSpec{
				{
					Replicas:      proto.Int32(1),
					TfPort:        proto.Int32(2222),
					TfReplicaType: spec.MASTER,
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "tensorflow",
									Image: *image,
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
				{
					Replicas:      proto.Int32(1),
					TfPort:        proto.Int32(2222),
					TfReplicaType: spec.PS,
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "tensorflow",
									Image: *image,
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
				{
					Replicas:      proto.Int32(1),
					TfPort:        proto.Int32(2222),
					TfReplicaType: spec.WORKER,
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "tensorflow",
									Image: *image,
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
			},
			TensorBoard: &spec.TensorBoardSpec{
				LogDir: "/tmp/tensorflow",
			},
		},
	}

	_, err = tfJobClient.Create(Namespace, original)

	if err != nil {
		return err
	}

	// Wait for the job to complete for up to 2 minutes.
	var tfJob *spec.TfJob
	for endTime := time.Now().Add(2 * time.Minute); time.Now().Before(endTime); {
		tfJob, err = tfJobClient.Get(Namespace, name)
		if err != nil {
			log.Warningf("There was a problem getting TfJob: %v; error %v", name, err)
		}

		if tfJob.Status.State == spec.StateSucceeded || tfJob.Status.State == spec.StateFailed {
			break
		}
		log.Infof("Waiting for job %v to finish", name)
		time.Sleep(5 * time.Second)
	}

	if tfJob == nil {
		return fmt.Errorf("Failed to get TfJob %v", name)
	}

	if tfJob.Status.State != spec.StateSucceeded {
		// TODO(jlewi): Should we clean up the job.
		return fmt.Errorf("TfJob %v did not succeed;\n %v", name, util.Pformat(tfJob))
	}

	if tfJob.Spec.RuntimeId == "" {
		return fmt.Errorf("TfJob %v doesn't have a RuntimeId", name)
	}

	// Loop over each replica and make sure the expected resources were created.
	for _, r := range original.Spec.ReplicaSpecs {
		baseName := strings.ToLower(string(r.TfReplicaType))

		for i := 0; i < int(*r.Replicas); i += 1 {
			jobName := fmt.Sprintf("%v-%v-%v", baseName, tfJob.Spec.RuntimeId, i)

			_, err := kubeCli.BatchV1().Jobs(Namespace).Get(jobName, metav1.GetOptions{})

			if err != nil {
				return fmt.Errorf("Tfob %v did not create Job %v for ReplicaType %v Index %v", name, jobName, r.TfReplicaType, i)
			}
		}
	}

	// Check that the TensorBoard deployment is present
	tbDeployName := fmt.Sprintf("tensorboard-%v", tfJob.Spec.RuntimeId)
	_, err = kubeCli.ExtensionsV1beta1().Deployments(Namespace).Get(tbDeployName, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("TfJob %v did not create Deployment %v for TensorBoard", name, tbDeployName)
	}

	// Check that the TensorBoard service is present
	_, err = kubeCli.CoreV1().Services(Namespace).Get(tbDeployName, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("TfJob %v did not create Service %v for TensorBoard", name, tbDeployName)
	}

	// Delete the job and make sure all subresources are properly garbage collected.
	if err := tfJobClient.Delete(Namespace, name); err != nil {
		log.Fatal("Failed to delete TfJob %v; error %v", name, err)
	}

	// Define sets to keep track of Job controllers corresponding to Replicas
	// that still exist.
	jobs := make(map[string]bool)
	isTBDeployDeleted := false

	// Loop over each replica and make sure the expected resources are being deleted.
	for _, r := range original.Spec.ReplicaSpecs {
		baseName := strings.ToLower(string(r.TfReplicaType))

		for i := 0; i < int(*r.Replicas); i += 1 {
			jobName := fmt.Sprintf("%v-%v-%v", baseName, tfJob.Spec.RuntimeId, i)

			jobs[jobName] = true
		}
	}

	// Wait for all jobs and deployment to be deleted.
	for endTime := time.Now().Add(5 * time.Minute); time.Now().Before(endTime) && (len(jobs) > 0 || !isTBDeployDeleted); {
		for k := range jobs {
			_, err := kubeCli.BatchV1().Jobs(Namespace).Get(k, metav1.GetOptions{})
			if k8s_errors.IsNotFound(err) {
				// Deleting map entry during loop is safe.
				// See: https://stackoverflow.com/questions/23229975/is-it-safe-to-remove-selected-keys-from-golang-map-within-a-range-loop
				delete(jobs, k)
			} else {
				log.Infof("Job %v still exists", k)
			}
		}

		if !isTBDeployDeleted {
			// Check that TensorBoard deployment is being deleted
			_, err = kubeCli.ExtensionsV1beta1().Deployments(Namespace).Get(tbDeployName, metav1.GetOptions{})
			if k8s_errors.IsNotFound(err) {
				isTBDeployDeleted = true
			} else {
				log.Infof("TensorBoard deployment still exists")
			}
		}

		if len(jobs) > 0 || !isTBDeployDeleted {
			time.Sleep(5 * time.Second)
		}
	}

	if len(jobs) > 0 {
		return fmt.Errorf("Not all Job controllers were successfully deleted.")
	}

	if !isTBDeployDeleted {
		return fmt.Errorf("TensorBoard deployment was not successfully deleted.")
	}

	return nil
}

func main() {
	flag.Parse()

	err := run()

	// Generate TAP (https://testanything.org/) output
	fmt.Println("1..1")
	if err == nil {
		fmt.Println("ok 1 - Successfully ran TfJob")
	} else {
		fmt.Printf("not ok 1 - Running TfJob failed %v \n", err)
		// Exit with non zero exit code for Helm tests.
		os.Exit(1)
	}
}
