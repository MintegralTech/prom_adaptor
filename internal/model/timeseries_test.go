package model

import (
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/tsdb/testutil"
	"testing"
)

func TestGetMetric(t *testing.T) {
	type target struct {
		metric         string
		isExistIpLabel bool
	}
	type test struct {
		in  prompb.TimeSeries
		out target
	}

	testData := make([]test, 2)

	testData[0].in.Labels = []*prompb.Label{
		{Name: "a", Value: "12"},
		{Name: "b", Value: "14"},
		{Name: "c", Value: "18"},
		{Name: "d", Value: "19"},
	}
	testData[0].in.Samples = []prompb.Sample{
		{Value: 100, Timestamp: 123445556},
	}
	testData[0].out.metric = ""
	testData[0].out.isExistIpLabel = false

	testData[1].in.Labels = []*prompb.Label{
		{Name: "a", Value: "12"},
		{Name: "b", Value: "14"},
		{Name: "ip", Value: "18.11.22.123"},
		{Name: "A", Value: "19"},
	}
	testData[1].in.Samples = []prompb.Sample{
		{Value: 1000, Timestamp: 123445556},
	}
	testData[1].out.metric = ""
	testData[1].out.isExistIpLabel = true

	for _, v := range testData {
		metric, isExistIpLabel := GetMetric(&v.in)
		testutil.Equals(t, v.out.isExistIpLabel, isExistIpLabel)
		fmt.Println(metric)
	}

}

func TestGetJobName(t *testing.T) {
	type target struct {
		jobName    string
		metricName string
		err        error
	}
	type test struct {
		in  prompb.TimeSeries
		out target
	}

	testData := make([]test, 2)

	testData[0].in.Labels = []*prompb.Label{
		{Name: "__name__", Value: "dsp_dsp_drs"},
		{Name: "b", Value: "14"},
		{Name: "c", Value: "18"},
		{Name: "d", Value: "19"},
	}
	testData[0].in.Samples = []prompb.Sample{
		{Value: 100, Timestamp: 123445556},
	}
	testData[0].out.metricName = ""
	testData[0].out.jobName = ""

	testData[1].in.Labels = []*prompb.Label{
		{Name: "__name__", Value: "dsp_dsp_drs"},
		{Name: "b", Value: "14"},
		{Name: "ip", Value: "18.11.22.123"},
		{Name: "A", Value: "19"},
	}
	testData[1].in.Samples = []prompb.Sample{
		{Value: 1000, Timestamp: 123445556},
	}
	testData[0].out.metricName = ""
	testData[0].out.jobName = ""

	for _, v := range testData {
		metric, _ := GetMetric(&v.in)
		jobName, _, err := GetJobName(metric)
		testutil.Ok(t, err)
		//testutil.Equals(t, v.out.isExistIpLabel, isExistIpLabel)
		fmt.Println(jobName, metric)
	}
}
