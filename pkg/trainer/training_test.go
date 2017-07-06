package trainer

import (
  //"reflect"
  //"testing"
  //
  //
  //"k8s.io/client-go/pkg/api/v1"
  //"github.com/gogo/protobuf/proto"
)

//func TestIsRetryableTerminationState(t *testing.T) {
//	type TestCase struct {
//		State    v1.ContainerStateTerminated
//		Expected bool
//	}
//
//	cases := []TestCase{
//		{
//      // Since reason is empty we don't trust the exit code.
//			State: v1.ContainerStateTerminated{
//				ExitCode: 0,
//			},
//      Expected: true,
//		},
//    {
//      State: v1.ContainerStateTerminated{
//        ExitCode: 0,
//        Message: "some reason",
//      },
//      Expected: false,
//    },
//    {
//      State: v1.ContainerStateTerminated{
//        ExitCode: 1,
//        Message: "some reason",
//      },
//      Expected: false,
//    },
//    {
//      // Since Reason is empty we don't trust the exit code.
//      State: v1.ContainerStateTerminated{
//        ExitCode: 1,
//      },
//      Expected: true,
//    },
//    {
//      State: v1.ContainerStateTerminated{
//        ExitCode: 244,
//        Message: "some reason",
//      },
//      Expected: true,
//    },
//    {
//      State: v1.ContainerStateTerminated{
//        ExitCode: 244,
//        Reason: "OOMKilled",
//      },
//      Expected: false,
//    },
//	}
//
//  for _, c := range cases {
//    actual := isRetryableTerminationState(&c.State)
//		if actual != c.Expected {
//      t.Errorf("isRetryableTerminationState(%+v)=%v want %v", c.State, actual, c.Expected)
//    }
//  }
//}
//
//func TestClusterSpec(t *testing.T) {
//  type TestCase struct {
//    Processes []*tpb.TensorFlowProcess
//    Expected map[string][]string
//  }
//
//  cases := []TestCase{
//    {
//      Processes: []*tpb.TensorFlowProcess {
//        {
//          NumReplicas: proto.Int32(2),
//          Type: tpb.TensorFlowProcess_PS.Enum(),
//          TfPort: proto.Int32(22),
//        },
//        {
//          NumReplicas: proto.Int32(1),
//          Type: tpb.TensorFlowProcess_MASTER.Enum(),
//          TfPort: proto.Int32(44),
//        },
//      },
//      Expected: map[string][]string{
//        "ps": []string{"ps-runtime-0:22", "ps-runtime-1:22"},
//        "master": []string{"master-runtime-0:44"},
//      },
//    },
//  }
//
//  for _, c:= range cases {
//    actual := newClusterSpec(c.Processes, "runtime")
//
//    for k, v := range c.Expected {
//      actualV, ok := actual[k]
//      if !ok {
//        t.Errorf("Actual cluster spec is missing key: %v", k)
//        continue
//      }
//      if !reflect.DeepEqual(actualV, v) {
//        t.Errorf("Key %v got %v want %v", k, actualV, v)
//      }
//    }
//  }
//}