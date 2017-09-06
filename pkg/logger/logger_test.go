package logger

import (
  "testing"
  "encoding/json"
  "reflect"
  "github.com/jlewi/mlkube.io/pkg/util"
)

func toJson(t *testing.T, d map[string] interface{}) []byte {
  b, err := json.Marshal(d)

  if err != nil {
    t.Fatalf("Couldn't serialize to json; %v", err)
  }

  return b
}

func TestParseJson(t *testing.T) {

  type testCase struct {
    in       []byte
    expected *JsonLogEntry
  }

  testCases := []testCase{
    {
       in: []byte("some non-json log line"),
       expected: nil,
    },
    {
      in: toJson(t, map[string]interface{}{
        "message": "some json message",
        "Created": "2017",
      }),
      expected: &JsonLogEntry{
        "message": "some json message",
        "Created": "2017",
      },
    },
  }

  for _, c := range testCases {
    e := ParseJson(c.in)

    if !reflect.DeepEqual(e, c.expected) {
      t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(e))
    }
  }
}
