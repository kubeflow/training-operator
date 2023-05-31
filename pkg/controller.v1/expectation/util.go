package expectation

import (
	"strings"
)

// GenExpectationPodsKey generates an expectation key for pods of a job
func GenExpectationPodsKey(jobKey string, replicaType string) string {
	return jobKey + "/" + strings.ToLower(replicaType) + "/pods"
}

// GenExpectationPodsKey generates an expectation key for services of a job
func GenExpectationServicesKey(jobKey string, replicaType string) string {
	return jobKey + "/" + strings.ToLower(replicaType) + "/services"
}
