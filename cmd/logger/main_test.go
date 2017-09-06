package main

import (
  "reflect"
  "testing"
  "github.com/jlewi/mlkube.io/pkg/logger"
  "github.com/jlewi/mlkube.io/pkg/util"
)

func TestParseJson(t *testing.T) {

  type testCase struct {
    in       string
    expected logger.LogLabels
  }

  testCases := []testCase{
    {
      in: "",
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
