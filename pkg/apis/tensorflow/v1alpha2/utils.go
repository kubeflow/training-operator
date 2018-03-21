package v1alpha2

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Pformat returns a pretty format output of any value that can be marshalled to JSON.
func Pformat(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	valueJSON, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Warningf("Couldn't pretty format %v, error: %v", value, err)
		return fmt.Sprintf("%v", value)
	}
	return string(valueJSON)
}
