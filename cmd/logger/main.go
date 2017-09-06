// logger tails files containing stdout/stderr logs and reformats entries as json attaching metadata.
package main

import (
  "flag"
  log "github.com/golang/glog"
  "github.com/jlewi/mlkube.io/pkg/logger"
  "os"
  "strings"
)

var (
  stdout    = flag.String("stdout", "", "The path to the file containing stdout.")
  stderr    = flag.String("stderr", "", "The path to the file containing stderr.")
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

// TODO(jlewi): We probably want to save position in the files so that if we restart we don't end up restreaming
// the same logs.
func main() {
  flag.Parse()

  if *stdout == "" && *stderr == "" {
    log.Fatalf("At least one of --stdout and --stderr must be provided.")
  }

  labels := parseLabels(*labelsStr)

  if *stdout != "" {
    go logger.TailLogs(*stdout, labels, os.Stdout)
  }

  if *stderr != "" {
    go logger.TailLogs(*stderr, labels, os.Stderr)
  }

  // Run forever.
  select {}
}
