// logger package provides a logger that can be run as a sidecar.
package logger

import (
  "encoding/json"
  log "github.com/golang/glog"
  "github.com/hpcloud/tail"
  "fmt"
  "regexp"
  "io"
)

// JsonLogEntry represents a log entry in json format.
type JsonLogEntry map[string] interface{}

// Enum defining standard fields in a log message.
type FieldName string

func (e *JsonLogEntry) SetField(f FieldName, v interface{}) {
  (*e)[string(f)] = v
}

func (e * JsonLogEntry) Write(f io.Writer) {

  b, err := json.Marshal(e)
  if err != nil {
    log.Warningf("Couldn't marshal value %v, error: %v", e, err)
  }
  f.Write(b)
  f.Write([]byte("\n"))
}

const (
  Created FieldName = "created"
  Level FieldName = "level"
  Message FieldName = "message"

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

// DockerLogEntry is an entry in the json formatted Docker container logs.
type DockerLogEntry struct {
  Log string `json:"log"`
  Stream string `json: "stream"`
  Time string `json: "time"`
}

var defPyRegex *regexp.Regexp

func init() {
  // Regex matching default Python logger output.
  var err error
  defPyRegex, err  = regexp.Compile("(INFO|WARNING|ERROR|CRITICAL|FATAL):([^:]*:.*)")

  if err != nil {
    log.Fatalf("Could not compile regex to match default python logger; %v", err)
  }
}

// TailLogs reads log lines from in, processes them and rewrites them to the sink.
// The function keeps reading until the channel is closed.
//
// TODO(jlewi): We should rewrite this to make it easier to test. We probably want to change the input
// so that the function can terminate after processing all the input entries rather than running forever.
func TailLogs(in chan *tail.Line, labels LogLabels, stdOut io.Writer, stdErr io.Writer) {

  // TO

  // TODO(jlewi): Need to handle multi-line non Json.
  for line := range in {
    entry := &DockerLogEntry{}

    outEntry := &JsonLogEntry{}

    // Add labels.
    for k, v := range labels {
      (*outEntry)[k] = v
    }

    if err := json.Unmarshal([]byte(line.Text), entry); err != nil {
      // Output a log message indicating an error parsing the message.

      outEntry.SetField(Message, fmt.Sprintf("There was a problem parsing log entry; %v, Error; %v", line.Text, err))
      outEntry.SetField(Created,line.Time)
      outEntry.Write(stdErr)
      continue
    }

    // Try matching the entry.
    m := defPyRegex.FindStringSubmatch(entry.Log)
    if m != nil {
      outEntry.SetField(Level, m[1])
      outEntry.SetField(Message, m[2])
    }

    outEntry.SetField(Created,line.Time)

    outEntry.Write(stdOut)

    //e := ParseJson([]byte(line.Text))
    //if e == nil {
    //  // Message isn't json.
    //  e = &JsonLogEntry{
    //    string(Message): line.Text,
    //  }
    //}


  }
}
