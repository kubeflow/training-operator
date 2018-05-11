// e2e provides an E2E test for TFJobs.
//
// The test creates TFJobs and runs various checks to ensure various operations work as intended.
// The test is intended to run as a helm test that ensures the TFJob operator is working correctly.
// Thus, the program returns non-zero exit status on error.
//
// TODO(jlewi): Do we need to make the test output conform to the TAP(https://testanything.org/)
// protocol so we can fit into the K8s dashboard
//
// TODO(https://github.com/kubeflow/tf-operator/issues/21) The E2E test should actually run distributed TensorFlow.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	"k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	tfv1alpha1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha1"
	tfjobclient "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/util"
)

var (
	image     = flag.String("image", "", "The Docker image containing the TF program to run.")
	name      = flag.String("name", "", "The name for the TFJob to create..")
	namespace = flag.String("namespace", "default", "The namespace to create the test job in.")
	numJobs   = flag.Int("num_jobs", 1, "The number of jobs to run.")
	timeout   = flag.Duration("timeout", 5*time.Minute, "The timeout for the test")
)

type tfReplicaType tfv1alpha1.TFReplicaType

func (tfrt tfReplicaType) toSpec() *tfv1alpha1.TFReplicaSpec {
	return &tfv1alpha1.TFReplicaSpec{
		Replicas:      proto.Int32(1),
		TFPort:        proto.Int32(2222),
		TFReplicaType: tfv1alpha1.TFReplicaType(tfrt),
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
	}
}

func run() (string, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	kubeCli, err := clientset.NewForConfig(config)
	if err != nil {
		return "", err
	}

	tfJobClient, err := tfjobclient.NewForConfig(config)
	if err != nil {
		return "", err
	}

	if *name == "" {
		name = proto.String("e2e-test-job-" + util.RandString(4))
	}

	original := &tfv1alpha1.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: *name,
			Labels: map[string]string{
				"test.mlkube.io": "",
			},
		},
		Spec: tfv1alpha1.TFJobSpec{
			ReplicaSpecs: []*tfv1alpha1.TFReplicaSpec{
				tfReplicaType(tfv1alpha1.MASTER).toSpec(),
				tfReplicaType(tfv1alpha1.PS).toSpec(),
				tfReplicaType(tfv1alpha1.WORKER).toSpec(),
			},
		},
	}

	_, err = tfJobClient.KubeflowV1alpha1().TFJobs(*namespace).Create(original)

	if err != nil {
		log.Errorf("Creating the job failed; %v", err)
		return *name, err
	}

	// Wait for the job to complete for up to timeout.
	var tfJob *tfv1alpha1.TFJob
	for endTime := time.Now().Add(*timeout); time.Now().Before(endTime); {
		tfJob, err = tfJobClient.KubeflowV1alpha1().TFJobs(*namespace).Get(*name, metav1.GetOptions{})
		if err != nil {
			log.Warningf("There was a problem getting TFJob: %v; error %v", *name, err)
		}

		if tfJob.Status.State == tfv1alpha1.StateSucceeded || tfJob.Status.State == tfv1alpha1.StateFailed {
			log.Infof("job %v finished:\n%v", *name, util.Pformat(tfJob))
			break
		}
		log.Infof("Waiting for job %v to finish:\n%v", *name, util.Pformat(tfJob))
		time.Sleep(5 * time.Second)
	}

	if tfJob == nil {
		return *name, fmt.Errorf("Failed to get TFJob %v", *name)
	}

	if tfJob.Status.State != tfv1alpha1.StateSucceeded {
		// TODO(jlewi): Should we clean up the job.
		return *name, fmt.Errorf("TFJob %v did not succeed;\n %v", *name, util.Pformat(tfJob))
	}

	if tfJob.Spec.RuntimeId == "" {
		return *name, fmt.Errorf("TFJob %v doesn't have a RuntimeId", *name)
	}

	// Loop over each replica and make sure the expected resources were created.
	for _, r := range original.Spec.ReplicaSpecs {
		baseName := strings.ToLower(string(r.TFReplicaType))

		for i := 0; i < int(*r.Replicas); i += 1 {
			jobName := fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", original.ObjectMeta.Name), baseName, tfJob.Spec.RuntimeId, i)

			_, err := kubeCli.BatchV1().Jobs(*namespace).Get(jobName, metav1.GetOptions{})

			if err != nil {
				return *name, fmt.Errorf("TFob %v did not create Job %v for ReplicaType %v Index %v", *name, jobName, r.TFReplicaType, i)
			}
		}
	}

	// Delete the job and make sure all subresources are properly garbage collected.
	if err := tfJobClient.KubeflowV1alpha1().TFJobs(*namespace).Delete(*name, &metav1.DeleteOptions{}); err != nil {
		log.Fatalf("Failed to delete TFJob %v; error %v", *name, err)
	}

	// Define sets to keep track of Job controllers corresponding to Replicas
	// that still exist.
	jobs := make(map[string]bool)

	// Loop over each replica and make sure the expected resources are being deleted.
	for _, r := range original.Spec.ReplicaSpecs {
		baseName := strings.ToLower(string(r.TFReplicaType))

		for i := 0; i < int(*r.Replicas); i += 1 {
			jobName := fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", original.ObjectMeta.Name), baseName, tfJob.Spec.RuntimeId, i)

			jobs[jobName] = true
		}
	}

	// Wait for all jobs and deployment to be deleted.
	for endTime := time.Now().Add(*timeout); time.Now().Before(endTime) && len(jobs) > 0; {
		for k := range jobs {
			_, err := kubeCli.BatchV1().Jobs(*namespace).Get(k, metav1.GetOptions{})
			if k8s_errors.IsNotFound(err) {
				// Deleting map entry during loop is safe.
				// See: https://stackoverflow.com/questions/23229975/is-it-safe-to-remove-selected-keys-from-golang-map-within-a-range-loop
				delete(jobs, k)
			} else {
				log.Infof("Job %v still exists", k)
			}
		}

		if len(jobs) > 0 {
			time.Sleep(5 * time.Second)
		}
	}

	if len(jobs) > 0 {
		return *name, fmt.Errorf("Not all Job controllers were successfully deleted for TFJob %v.", *name)
	}

	return *name, nil
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
				log.Errorf("TFJob %v didn't run successfully; %v", name, err)
			} else {
				log.Infof("TFJob %v ran successfully", name)
			}
			c <- Result{
				Name:  name,
				Error: err,
			}
		}()
	}

	numSucceded := 0
	numFailed := 0

	for endTime := time.Now().Add(*timeout); numSucceded+numFailed < *numJobs && time.Now().Before(endTime); {
		select {
		case res := <-c:
			if res.Error == nil {
				numSucceded += 1
			} else {
				numFailed += 1
			}
		case <-time.After(time.Until(endTime)):
			log.Errorf("Timeout waiting for TFJob to finish.")
			fmt.Println("timeout 2")
		}
	}

	if numSucceded+numFailed < *numJobs {
		log.Errorf("Timeout waiting for jobs to finish; only %v of %v TFJobs completed.", numSucceded+numFailed, *numJobs)
	}

	// Generate TAP (https://testanything.org/) output
	fmt.Println("1..1")
	if numSucceded == *numJobs {
		fmt.Println("ok 1 - Successfully ran TFJob")
	} else {
		fmt.Printf("not ok 1 - Running TFJobs failed \n")
		// Exit with non zero exit code for Helm tests.
		os.Exit(1)
	}
}
