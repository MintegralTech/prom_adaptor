package model

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	tsQueueLengthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "adapter",
			Subsystem: "aggregator",
			Name:      "queue_length",
			Help:      "timeSeries queue length",
		},
		[]string{"type"},
	)
	mergeMetricCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "adapter",
			Subsystem: "aggregator",
			Name:      "merge_count",
			Help:      "merge counter",
		},
		[]string{"jobname", "type"},
	)
	cacheDataLengthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "adapter",
			Subsystem: "aggregator",
			Name:      "cache_data_length",
			Help:      "cache data length",
		},
		[]string{"jobname", "type"},
	)
	sendRequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "adapter",
			Subsystem: "aggregator",
			Name:      "send_request_count",
			Help:      "send request counter",
		},
		[]string{"succ"},
	)
	metricsSizeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "adapter",
			Subsystem: "aggregator",
			Name:      "metrics_size",
			Help:      "data size",
		},
		[]string{"jobname"},
	)
)

func InitMonitor() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(tsQueueLengthGauge)
	prometheus.MustRegister(mergeMetricCounter)
	prometheus.MustRegister(cacheDataLengthGauge)
	prometheus.MustRegister(sendRequestCounter)
	prometheus.MustRegister(metricsSizeCounter)
}

func GaugeMonitor() {
	t := time.NewTicker(time.Second * time.Duration(15))
	for {
		<-t.C
		tsQueueLengthGauge.With(prometheus.Labels{"type": "request"}).Set(float64(TsQueue.RequestLength()))
		tsQueueLengthGauge.With(prometheus.Labels{"type": "merge"}).Set(float64(TsQueue.MergeLength()))
		for jobName, agg := range Collection.whiteJobName {
			cacheDataLengthGauge.With(prometheus.Labels{"jobname": jobName, "type": "prev"}).Set(float64(len(agg.prevCache.data)))
		}
	}
}
