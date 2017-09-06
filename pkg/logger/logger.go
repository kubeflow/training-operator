// logger package provides a logger that can be run as a sidecar.
package logger

import (
  "encoding/json"
  "os"
  log "github.com/golang/glog"
  "github.com/hpcloud/tail"
)

// JsonLogEntry represents a log entry in json format.
type JsonLogEntry map[string] interface{}

// Enum defining standard fields in a log message.
type FieldName string

const (
  Message FieldName = "message"
  Created FieldName = "created"
)

type LogLabels map[string]string

// ParseJson a log entry as a JsonLogEntry. Returns nil if the log entry can't be processed as json.
func ParseJson(b []byte) *JsonLogEntry {
  e := &JsonLogEntry{}
  err := json.Unmarshal(b, e)

  if err != nil {
    return nil
  }
  return e
}

// TailLogs reads logs from the source, processes them and rewrites them to the sink.
//
// TODO(jlewi): We should rewrite this to make it easier to test. We probably want to change the input
// so that the function can terminate after processing all the input entries rather than running forever.
func TailLogs(in string, labels LogLabels, out *os.File) {
  log.Infof("Reading from; %v", in)
  t, err := tail.TailFile(in, tail.Config{Follow: true, ReOpen: true, })

  if err != nil {
    log.Fatalf("There was a problem tailing file: %v; error: %v", in, err)
  }

  // TODO(jlewi): Need to handle multi-line non Json.
  for line := range t.Lines {
    e := ParseJson([]byte(line.Text))
    if e == nil {
      // Message isn't json.
      e = &JsonLogEntry{
        string(Message): line.Text,
      }
    }

    // Add labels.
    for k, v := range labels {
      (*e)[k] = v
    }

    b, err := json.Marshal(e)
    if err != nil {
      log.Warningf("Couldn't marshal value %v, error: %v", e, err)
      continue
    }
    out.Write(b)
    out.Write([]byte("\n"))
  }
}
