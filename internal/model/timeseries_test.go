package model

import (
	"fmt"
	"testing"
)

func TestGetJobName(t *testing.T) {
	metric := "Retarget_Dsp_request_counter{hello: 1}"
	jobName, metricName, err := GetJobName(metric)
	fmt.Printf("jobName=%v, metricName=%v, err=%v\n", jobName, metricName, err)
}
