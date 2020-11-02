package model

import (
	"errors"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"strings"
)

func DeleteLable(ts *prompb.TimeSeries, labelName string) *prompb.TimeSeries {
	res := *ts
	for i, l := range res.Labels {
		if l.Name == labelName {
			res.Labels = append(res.Labels[:i], res.Labels[i+1:]...)
			break
		}
	}
	return &res
}

func GetMetric(ts *prompb.TimeSeries) string {
	m := make(model.Metric, len(ts.Labels))
	for _, l := range ts.Labels {
		m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
	}
	metric := fmt.Sprint(m)
	return metric
}

func (c *cache) Print() {
	for k, v := range c.data {
		fmt.Println(k, v.Value)
	}
	fmt.Println("======================")
}

func (b *block) Print() {
	for k, v := range b.data {
		fmt.Println(k, GetMetric(v.ts)+GetSample(v.ts), v.flag)
	}
	fmt.Println("----------------------")
}

func GetSample(ts *prompb.TimeSeries) string {
	var sample string
	for _, s := range ts.Samples {
		sample = fmt.Sprintf("  %f %d", s.Value, s.Timestamp)
	}
	return sample
}

func GetJobName(metric string) (string, string, error) {
	var jobName string
	temp := strings.Split(metric, "{")
	if len(temp) < 1 {
		return "", "", errors.New("get metricsName error")
	}
	metricName := temp[0]
	fields := strings.Split(metricName, "_")
	if len(fields) <= 0 {
		return "", "", errors.New("get jobName error")
	}
	if len(fields) == 1 {
		jobName = strings.ToLower(fields[0])
	} else {
		jobName = strings.ToLower(fields[0] + "_" + fields[1])
	}
	return jobName, metricName, nil
}

func GetMetricsName(originMetricName string) (string, error) {
	var metricsName string

	fields := strings.Split(originMetricName, "_")
	if len(fields) <= 0 {
		return "", errors.New("get jobName error")
	}
	l := len(fields)
	if l == 1 {
		metricsName = strings.ToLower(fields[0])
	} else if fields[l-1] == "sum" || fields[l-1] == "bucket" || fields[l-1] == "count" {
		metricsName = strings.ToLower(strings.TrimLeft(originMetricName, fields[l-1]))
	} else {
		metricsName = originMetricName
	}
	return metricsName, nil
}