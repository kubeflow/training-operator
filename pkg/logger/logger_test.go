package logger

import (
  "testing"
  "encoding/json"
  "github.com/hpcloud/tail"
  "bytes"
  log "github.com/golang/glog"
  "github.com/jlewi/mlkube.io/pkg/util"
)

func toJson(t *testing.T, d map[string] interface{}) []byte {
  b, err := json.Marshal(d)

  if err != nil {
    t.Fatalf("Couldn't serialize to json; %v", err)
  }

  return b
}

func checkLogEntries(expected JsonLogEntry, actual JsonLogEntry) bool {
  if len(expected) != len(actual) {
    return false
  }

  for k := range expected {
    _, ok := actual[k]
    if !ok {
      return false
    }

    if actual[k] != expected[k] {
      log.Errorf("Got %v=%v, Want %v=%v", k, actual[k], k, expected[k])
      return false
    }
  }
  return true
}

func TestTailLogs(t *testing.T) {

  type testCase struct {
    in       string
    expected *JsonLogEntry
  }

  labels := LogLabels{
    "label1": "val1",
    "label2": "val2",
  }

  testCases := []testCase{
    //{
    //   in: []byte("some non-json log line"),
    //   expected: nil,
    //},
    //{
    //  in: toJson(t, map[string]interface{}{
    //    "message": "some json message",
    //    "Created": "2017",
    //  }),
    //  expected: &JsonLogEntry{
    //    "message": "some json message",
    //    "Created": "2017",
    //  },
    //},
    {
      // Python entry corresponding to the default Python logging handler.
      in: `{"log":"INFO:root:Building server.\n","stream":"stderr","time":"2017-09-09T21:10:01.37275895Z"}`,
      expected: &JsonLogEntry{
        "created":  "0001-01-01T00:00:00Z",
        "label1": "val1",
        "label2": "val2",
        "level": "INFO",
        "message": `root:Building server.`,
      },
    },
//    {
//      // Message output by TensorFlow c++q.
//      in: `{"log":"2017-09-09 21:10:01.372998: W tensorflow/core/platform/cpu_feature_guard.cc:45] The TensorFlow library wasn't compiled to use SSE4.1 instructions, but these are availa
//ble on your machine and could speed up CPU computations.\n","stream":"stderr","time":"2017-09-09T21:10:01.373099692Z"}`
//    },
  }

  for _, c := range testCases {

    lChan := make(chan *tail.Line, 1)
    l := &tail.Line {
      Text: c.in,
    }
    lChan <- l
    close(lChan)


    stdout :=  &bytes.Buffer{}
    stderr :=  &bytes.Buffer{}

    TailLogs(lChan, labels, stdout, stderr)

    actualJson := stdout.Bytes()

    actual := &JsonLogEntry{}
    err := json.Unmarshal(actualJson, &actual)
    if err != nil {
      t.Fatalf("Couldn't unmarshal %v; Error: %v", string(actualJson), err)
    }

    //if !reflect.DeepEqual(e, c.expected) {
    //  t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(e))
    //}

    if  !checkLogEntries(*c.expected, *actual) {
      t.Errorf("Got %v; Want %v", util.Pformat(actual), util.Pformat(c.expected))
    }
  }
}
