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
  "mlkube.io/pkg/util/k8sutil"
  log "github.com/golang/glog"
  "mlkube.io/pkg/spec"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "mlkube.io/pkg/util"
  "github.com/gogo/protobuf/proto"
  "k8s.io/client-go/pkg/api/v1"
  "flag"
  "time"
  "strings"
  "fmt"
  k8s_errors "k8s.io/apimachinery/pkg/api/errors"
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
      Labels: map[string] string {
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

    if tfJob.Status.State == spec.StateSucceeded  || tfJob.Status.State == spec.StateFailed {
      break
    }
    log.Infof("Waiting for job %v to finish", name)
    time.Sleep(5 * time.Second)
  }

  if tfJob == nil {
    log.Fatalf("Failed to get TfJob %v", name)
  }

  if tfJob.Status.State != spec.StateSucceeded {
    // TODO(jlewi): Should we clean up the job.
    log.Fatalf("TfJob %v did not succeed;\n %v", name, util.Pformat(tfJob))
  }

  if tfJob.Spec.RuntimeId == "" {
    log.Fatalf("TfJob %v doesn't have a RuntimeId", name)
  }

  // Loop over each replica and make sure the expected resources were created.
  for _, r := range original.Spec.ReplicaSpecs {
    baseName :=  strings.ToLower(string(r.TfReplicaType))

    for i := 0;  i < int(*r.Replicas); i += 1 {
      jobName := fmt.Sprintf("%v-%v-%v", baseName, tfJob.Spec.RuntimeId, i)

      _, err := kubeCli.BatchV1().Jobs(Namespace).Get(jobName, metav1.GetOptions{})

      if err != nil {
        log.Fatalf("Tfob %v did not create Job %v for ReplicaType %v Index %v", name, jobName, r.TfReplicaType, i)
      }
    }
  }

  // Delete the job and make sure all subresources are properly garbage collected.
  if err := tfJobClient.Delete(Namespace, name); err != nil {
    log.Fatal("Failed to delete TfJob %v; error %v", name, err)
  }

  // Define sets to keep track of Job controllers corresponding to Replicas
  // that still exist.
  jobs := make(map[string] bool)

  // Loop over each replica and make sure the expected resources are being deleted.
  for _, r := range original.Spec.ReplicaSpecs {
    baseName :=  strings.ToLower(string(r.TfReplicaType))

    for i := 0;  i < int(*r.Replicas); i += 1 {
      jobName := fmt.Sprintf("%v-%v-%v", baseName, tfJob.Spec.RuntimeId, i)

      jobs[jobName] = true
    }
  }

  // Wait for all jobs to be deleted.
  for endTime := time.Now().Add(5 * time.Minute); time.Now().Before(endTime) && len(jobs) >0; {
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

    if len(jobs) > 0 {
      time.Sleep(5 * time.Second)
    }
  }

  if len(jobs) > 0 {
    log.Fatalf("Not all Job controllers were successfully deleted.")
  }
  return nil
}

func main() {
  flag.Parse()

  if err := run(); err != nil {
    log.Fatalf("Test failed; %v", err)
  }
}
