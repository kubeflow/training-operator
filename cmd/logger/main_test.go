package main

import (
  "io/ioutil"
  "reflect"
  "testing"
  log "github.com/golang/glog"
  "github.com/jlewi/mlkube.io/pkg/logger"
  "github.com/jlewi/mlkube.io/pkg/util"
  "os"
  "path/filepath"
  "fmt"
  "regexp"
  "time"
  "sort"
)

func TestParseLabels(t *testing.T) {

  type testCase struct {
    in       string
    expected logger.LogLabels
  }

  testCases := []testCase{
    {
      in:       "",
      expected: logger.LogLabels{},
    },
    {
      in: "key1=value1,key2=value2",
      expected: logger.LogLabels{
        "key1": "value1",
        "key2": "value2",
      },
    },
  }

  for _, c := range testCases {
    l := parseLabels(c.in)

    if !reflect.DeepEqual(l, c.expected) {
      t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(l))
    }
  }
}

func checkLists(expected []string, actual [] string) bool {
  if len(expected) != len(actual) {
    return false
  }

  for i, v := range expected {
    if v != actual[i] {
      log.Errorf("Got: %v, Want: %v", actual[i], v)
      return false
    }
  }
  return true
}

func TestLogFileWatcher(t *testing.T) {
  dir, err := ioutil.TempDir("", "logFileWatcherTest")
  if err != nil {
    t.Fatalf("Could not create temporary directory; Error: %v", err)
  }
  defer os.RemoveAll(dir) // clean up

  // Create some files in the directory.
  for i := 0; i < 2; i++ {
    tmpfn := filepath.Join(dir, fmt.Sprintf("tensorflow_%v.log", i))
    if err := ioutil.WriteFile(tmpfn, []byte("hello world."), 0666); err != nil {
      t.Fatalf("Couldn't create temp file; %v; Error: %v", tmpfn, err)
    }
  }

  w, err := NewLogFileWatcher(dir, regexp.MustCompile("tensorflow_.*log"))

  if err != nil {
    t.Fatalf("Could not create log file watcher; Error: %v", err)
  }

  // Expect two files
  actual := []string{}
  timer := time.NewTimer(time.Second * 10)
  for i := 0; i < 2; i++ {
    select {
    case p := <-w.Paths:
      actual = append(actual, p)
    case <-timer.C:
      t.Fatalf("Timeout waiting for the watcher to detect two files.")
    }
  }

  sort.Strings(actual)

  expected := []string{filepath.Join(dir, "tensorflow_0.log"), filepath.Join(dir, "tensorflow_1.log")}

  if !checkLists(expected, actual) {
    t.Fatalf("Got %v Want %v", actual, expected)
  }

  // Create some more files.
  for i := 2; i < 4; i++ {
    tmpfn := filepath.Join(dir, fmt.Sprintf("tensorflow_%v.log", i))
    if err := ioutil.WriteFile(tmpfn, []byte("hello world."), 0666); err != nil {
      t.Fatalf("Couldn't create temp file; %v; Error: %v", tmpfn, err)
    }
  }

  // Expect two files
  timer = time.NewTimer(time.Second * 10)
  for i := 0; i < 2; i++ {
    select {
    case p := <-w.Paths:
      actual = append(actual, p)
    case <-timer.C:
      t.Fatalf("Timeout waiting for the watcher to detect two files.")
    }
  }

  sort.Strings(actual)

  expected = append(expected, filepath.Join(dir, "tensorflow_2.log"))
  expected = append(expected, filepath.Join(dir, "tensorflow_3.log"))

  if !checkLists(expected, actual) {
    t.Fatalf("Got %v Want %v", actual, expected)
  }
}
