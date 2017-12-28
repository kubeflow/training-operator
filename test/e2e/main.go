// e2e provides an E2E test for TfJobs.
//
// The test creates TfJobs and runs various checks to ensure various operations work as intended.
// The test is intended to run as a helm test that ensures the TfJob operator is working correctly.
// Thus, the program returns non-zero exit status on error.
//
// TODO(jlewi): Do we need to make the test output conform to the TAP(https://testanything.org/)
// protocol so we can fit into the K8s dashboard
//
// TODO(https://github.com/tensorflow/k8s/issues/21) The E2E test should actually run distributed TensorFlow.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	"github.com/tensorflow/k8s/pkg/spec"
	"github.com/tensorflow/k8s/pkg/util"
	"github.com/tensorflow/k8s/pkg/util/k8sutil"
	"k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	Namespace = "default"
)

var (
	image   = flag.String("image", "", "The Docker image containing the TF program to run.")
	numJobs = flag.Int("num_jobs", 1, "The number of jobs to run.")
	timeout = flag.Duration("timeout", 5*time.Minute, "The timeout for the test")
)

func createAndWaitTfJob(tfJobClient k8sutil.TfJobClient, name string) (tfJob *spec.TfJob, err error) {
	// Wait for the job to complete for up to timeout.
	for endTime := time.Now().Add(*timeout); time.Now().Before(endTime); {
		tfJob, err = tfJobClient.Get(Namespace, name)
		if err != nil {
			log.Warningf("There was a problem getting TfJob: %v; error %v", name, err)
		}

		if tfJob.Status.State == spec.StateSucceeded || tfJob.Status.State == spec.StateFailed {
			log.Infof("job %v finished:\n%v", name, util.Pformat(tfJob))
			break
		}
		log.Infof("Waiting for job %v to finish:\n%v", name, util.Pformat(tfJob))
		time.Sleep(5 * time.Second)
	}

	if tfJob == nil {
		return nil, fmt.Errorf("failed to get TfJob %v", name)
	}

	if tfJob.Status.State != spec.StateSucceeded {
		// TODO(jlewi): Should we clean up the job.
		return nil, fmt.Errorf("TfJob %v did not succeed;\n %v", name, util.Pformat(tfJob))
	}

	if tfJob.Spec.RuntimeId == "" {
		return nil, fmt.Errorf("TfJob %v doesn't have a RuntimeId", name)
	}
	return tfJob, nil
}

func checkCreatedTensorBoard(kubeCli kubernetes.Interface, name string, original, created *spec.TfJob) (tbDeployName string, err error) {
	// Check that the TensorBoard deployment is present
	tbDeployName = fmt.Sprintf("%v-tensorboard-%v", fmt.Sprintf("%.40s", original.Metadata.Name), created.Spec.RuntimeId)
	_, err = kubeCli.ExtensionsV1beta1().Deployments(Namespace).Get(tbDeployName, metav1.GetOptions{})

	if err != nil {
		err = fmt.Errorf("TfJob %v did not create Deployment %v for TensorBoard", name, tbDeployName)
		return
	}
	// Check that the TensorBoard service is present
	_, err = kubeCli.CoreV1().Services(Namespace).Get(tbDeployName, metav1.GetOptions{})

	if err != nil {
		err = fmt.Errorf("TfJob %v did not create Service %v for TensorBoard", name, tbDeployName)
		return
	}
	return
}

func checkCreatedJobs(kubeCli kubernetes.Interface, name string, original, created *spec.TfJob) error {
	// Loop over each replica and make sure the expected resources were created.
	for _, r := range original.Spec.ReplicaSpecs {
		baseName := strings.ToLower(string(r.TfReplicaType))

		for i := 0; i < int(*r.Replicas); i += 1 {
			jobName := fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", original.Metadata.Name), baseName, created.Spec.RuntimeId, i)

			_, err := kubeCli.BatchV1().Jobs(Namespace).Get(jobName, metav1.GetOptions{})

			if err != nil {
				return fmt.Errorf("the Tfob %v did not create Job %v for ReplicaType %v Index %v", name, jobName, r.TfReplicaType, i)
			}
		}
	}
	return nil
}

func getJobNames(original, created *spec.TfJob) (jobs map[string]bool) {
	jobs = make(map[string]bool)
	// Loop over each replica and make sure the expected resources are being deleted.
	for _, r := range original.Spec.ReplicaSpecs {
		baseName := strings.ToLower(string(r.TfReplicaType))

		for i := 0; i < int(*r.Replicas); i += 1 {
			jobName := fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", original.Metadata.Name), baseName, created.Spec.RuntimeId, i)
			jobs[jobName] = true
		}
	}
	return
}

func updateJobNames(kubeCli kubernetes.Interface, jobs map[string]bool) {
	for k := range jobs {
		_, err := kubeCli.BatchV1().Jobs(Namespace).Get(k, metav1.GetOptions{})
		if k8sErrors.IsNotFound(err) {
			// Deleting map entry during loop is safe.
			// See: https://stackoverflow.com/questions/23229975/is-it-safe-to-remove-selected-keys-from-golang-map-within-a-range-loop
			delete(jobs, k)
		} else {
			log.Infof("Job %v still exists", k)
		}
	}
}

func waitForDeletion(kubeCli kubernetes.Interface, name, tbDeployName string, original, created *spec.TfJob) (err error) {
	// Define sets to keep track of Job controllers corresponding to Replicas
	// that still exist.
	jobs := getJobNames(original, created)
	isTBDeployDeleted := false

	// Wait for all jobs and deployment to be deleted.
	for endTime := time.Now().Add(*timeout); time.Now().Before(endTime) && (len(jobs) > 0 || !isTBDeployDeleted); {
		updateJobNames(kubeCli, jobs)

		if !isTBDeployDeleted {
			// Check that TensorBoard deployment is being deleted
			_, err = kubeCli.ExtensionsV1beta1().Deployments(Namespace).Get(tbDeployName, metav1.GetOptions{})
			if k8sErrors.IsNotFound(err) {
				isTBDeployDeleted = true
			} else {
				log.Infof("TensorBoard deployment %v still exists for TfJob %v", tbDeployName, name)
			}
		}

		if len(jobs) > 0 || !isTBDeployDeleted {
			time.Sleep(5 * time.Second)
		}
	}

	if len(jobs) > 0 {
		err = fmt.Errorf("Not all Job controllers were successfully deleted for TfJob %v.", name)
		return
	}

	if !isTBDeployDeleted {
		err = fmt.Errorf("TensorBoard deployment %v was not successfully deleted for TfJob %v.", tbDeployName, name)
		return
	}
	return
}

func run() (string, error) {
	kubeCli := k8sutil.MustNewKubeClient()
	tfJobClient, err := k8sutil.NewTfJobClient()
	if err != nil {
		return "", err
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
		log.Errorf("Creating the job failed; %v", err)
		return name, err
	}

	tfJob, err := createAndWaitTfJob(tfJobClient, name)
	if err != nil {
		return name, err
	}

	if err := checkCreatedJobs(kubeCli, name, original, tfJob); err != nil {
		return name, err
	}

	tbDeployName, err := checkCreatedTensorBoard(kubeCli, name, original, tfJob)
	if err != nil {
		return name, err
	}

	// Delete the job and make sure all subresources are properly garbage collected.
	if err := tfJobClient.Delete(Namespace, name); err != nil {
		log.Fatalf("Failed to delete TfJob %v; error %v", name, err)
	}

	if err = waitForDeletion(kubeCli, name, tbDeployName, original, tfJob); err != nil {
		return name, err
	}

	return name, nil
}

func main() {
	flag.Parse()

	if *image == "" {
		log.Fatalf("--image must be provided.")
	}

	type Result struct {
		Error error
		Name  string
	}
	c := make(chan Result)

	for i := 0; i < *numJobs; i += 1 {
		go func() {
			name, err := run()
			if err != nil {
				log.Errorf("TfJob %v didn't run successfully; %v", name, err)
			} else {
				log.Infof("TfJob %v ran successfully", name)
			}
			c <- Result{
				Name:  name,
				Error: err,
			}
		}()
	}

	numSucceeded := 0
	numFailed := 0

	for endTime := time.Now().Add(*timeout); numSucceeded+numFailed < *numJobs && time.Now().Before(endTime); {
		select {
		case res := <-c:
			if res.Error == nil {
				numSucceeded += 1
			} else {
				numFailed += 1
			}
		case <-time.After(endTime.Sub(time.Now())):
			log.Errorf("Timeout waiting for TfJob to finish.")
			fmt.Println("timeout 2")
		}
	}

	if numSucceeded+numFailed < *numJobs {
		log.Errorf("Timeout waiting for jobs to finish; only %v of %v TfJobs completed.", numSucceeded+numFailed, *numJobs)
	}

	// Generate TAP (https://testanything.org/) output
	fmt.Println("1..1")
	if numSucceeded == *numJobs {
		fmt.Println("ok 1 - Successfully ran TfJob")
	} else {
		fmt.Printf("not ok 1 - Running TfJobs failed \n")
		// Exit with non zero exit code for Helm tests.
		os.Exit(1)
	}
}
