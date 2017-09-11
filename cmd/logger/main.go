// logger tails the stdout/stderr of the TensorFlow processes and applies various transformations to the log
// entries.
//
// The input is the json formatted file produced by Docker containing the stdout/stderr logs of the Docker container.
// These log entries are processed to attach metadata such as TfReplicaType and TfReplicaIndex.
package main

import (
  "flag"
  "fmt"
  "io/ioutil"
  "github.com/fsnotify/fsnotify"
  log "github.com/golang/glog"
  "github.com/hpcloud/tail"
  "github.com/jlewi/mlkube.io/pkg/logger"
  "github.com/jlewi/mlkube.io/pkg/util/k8sutil"
  meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "path/filepath"
  "os"
  "regexp"
  "strings"
  "github.com/jlewi/mlkube.io/pkg/util"
)

var (
  labelsStr = flag.String("labels", "", "Comma separated list of labels to attach to all log entryies; e.g. labels=key1=value1,key2=value2")
)

func parseLabels(l string) logger.LogLabels {
  labels := logger.LogLabels{}

  if l == "" {
    return labels
  }
  pairs := strings.Split(l, ",")
  for _, p := range pairs {
    kv := strings.Split(p, "=")
    if len(kv) == 0 {
      continue
    }
    if len(kv) != 2 {
      log.Errorf("label %v is not in the form key=value", kv)
      continue
    }
    labels[kv[0]] = kv[1]
  }

  return labels
}

// LogFileWatcher emits the name of every log file matching some expression everytime it appears
type LogFileWatcher struct {
  Paths chan string

  // Dir is the diectory to wathc.
  Dir string

  // NameRegex is the regex to match log files.
  NameRegex *regexp.Regexp

  // Names of files to process
  names chan string
  // List of files already seen.
  seen map[string] bool
}

// New creates a new LogFileWatcher.
func NewLogFileWatcher(dir string, re *regexp.Regexp) (*LogFileWatcher, error) {
  w := &LogFileWatcher{
    Paths: make(chan string, 5),
    Dir: dir,
    NameRegex: re,
    names: make(chan string, 10),
    seen: map[string]bool{},
  }

  files, err := ioutil.ReadDir(dir)
  if err != nil {
    return nil, fmt.Errorf("Could not read dir: %v, Error: %v", dir, err)
  }

  // Process all the current files
  go func(){
    for _, f := range files {
     w.names <- f.Name()
    }
  }()

  watcher, err := fsnotify.NewWatcher()

  if err != nil {
    log.Fatalf("Could not create a new watcher; Error: %v", err)
  }

  if err := watcher.Add(dir); err != nil {
    log.Fatalf("Could not watch %v; Error: %v", dir, err)
  }

  // Get notified about new files.
  go func() {
      for e := range watcher.Events {
        w.names <- e.Name
      }
  }()

  // Start processing events.
  go func() {
    w.process()
  }()
  return w, nil
}

// process a file named name.
func (w *LogFileWatcher) process() {
  for name := range w.names {
    if !w.NameRegex.MatchString(name) {
      continue
    }

    if _, ok := w.seen[name]; ok {
      continue
    }

    w.seen[name] = true

    w.Paths <- filepath.Join(w.Dir, name)
  }
}

// TODO(jlewi): We probably want to save position in the files so that if we restart we don't end up restreaming
// the same logs.
func main() {
  flag.Parse()

  // We need to know the UID of the pod because we will use that to locate the logs.
  //podUid := os.Getenv("MY_POD_UID")
  //if len(podUid) == 0 {
  //  log.Fatalf("must set env MY_POD_UID")
  //}

  podNamespace := os.Getenv("MY_POD_NAMESPACE")
  if len(podNamespace) == 0 {
    log.Fatalf("must set env MY_POD_NAMESPACE")
  }

  podName := os.Getenv("MY_POD_NAME")
  if len(podName) == 0 {
    log.Fatalf("must set env MY_POD_NAMES")
  }

  kubeCli := k8sutil.MustNewKubeClient()

  pod, err := kubeCli.CoreV1().Pods(podNamespace).Get(podName,  meta_v1.GetOptions{})
  if err != nil {
    log.Fatalf("Coud not get pod; namespace=%v, pod name=%v; Error: %v", podNamespace, podName, util.Pformat(err))
  }

  labels := parseLabels(*labelsStr)

  podLogDir := fmt.Sprintf("/var/log/pods/%v", pod.ObjectMeta.UID)

  log.Infof("logger is processing pod logs: %v", podLogDir)

  // Add pod labels.
  for k, v := range pod.Labels {
    labels[k] = v
  }

  watcher, err := NewLogFileWatcher(podLogDir, regexp.MustCompile("tensorflow_.*log"))

  if err != nil {
    log.Errorf("Could not create watcher for dir: %v; Error: %v", podLogDir, err)
  }

  for logFile := range watcher.Paths {
    log.Infof("logger is monitoring log file: %v", logFile)

    t, err := tail.TailFile(logFile, tail.Config{Follow: true, ReOpen: true, })

    if err != nil {
      // TODO(jlewi): Is crashing the write thing to do?
         log.Fatalf("There was a problem tailing file: %v; error: %v", logFile, err)
    }

    // The go function will exit when the channel is closed.
    // Tail will close the channel when the file is deleted.
    // The file should be deleted when Docker/Kubernetes garbage collects the logs.
    // In the event the container is in a crash loop; old log files will get deleted thus causing
    // the channel to be closed.
    go logger.TailLogs(t.Lines, labels, os.Stdout, os.Stderr)
  }
}
