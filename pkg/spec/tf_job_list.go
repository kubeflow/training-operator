package spec

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TfJobList is a list of etcd clusters.
type TfJobList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	Metadata metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of third party objects
	Items []TfJob `json:"items"`
}

// There is known issue with TPR in client-go:
//   https://github.com/kubernetes/client-go/issues/8
// Workarounds:
// - We include `Metadata` field in object explicitly.
// - we have the code below to work around a known problem with third-party resources and ugorji.

type TfJobListCopy TfJobList
type TfJobCopy TfJob

func (c *TfJob) UnmarshalJSON(data []byte) error {
	tmp := TfJobCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := TfJob(tmp)
	*c = tmp2
	return nil
}

func (cl *TfJobList) UnmarshalJSON(data []byte) error {
	tmp := TfJobListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := TfJobList(tmp)
	*cl = tmp2
	return nil
}
