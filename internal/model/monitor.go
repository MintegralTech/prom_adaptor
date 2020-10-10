package model

import (
    "strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus"
)

const(
    namespace = "adaptor"
    subsystem = "aggregator"
)

var (
    tsQueueLengthGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "queue_length",
            Help:      "timeSeries queue length",
        },
        []string{"type", "queueIndex"},
    )
    mergeMetricCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "merge_count",
            Help:      "merge counter",
        },
        []string{"jobname", "type", "queueIndex"},
    )
    cacheDataLengthGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "cache_data_length",
            Help:      "cache data length",
        },
        []string{"jobname", "type"},
    )
    sendMetricsNumCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "send_metrics_num_count",
            Help:      "send metrics num counter",
        },
        []string{"succ", "queueIndex"},
    )

    sendRequestNumCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "send_request_num_count",
            Help:      "send request num counter",
        },
        []string{"succ", "queueIndex"},
    )

    receiveMetricsNumCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "receive_metrics_num_count",
            Help:      "receive metrics num counter",
        },
        []string{"jobname", "queueIndex", "type"},
    )

    metricsSizeCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "metrics_size",
            Help:      "data size",
        },
        []string{"jobname"},
    )
    packStatusCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: subsystem,
            Name:      "pack_status_count",
            Help:      "pack status count",
        },
        []string{"jobname","flag"},
    )
)

func InitMonitor() {
    // Metrics have to be registered to be exposed:
    prometheus.MustRegister(tsQueueLengthGauge)
    prometheus.MustRegister(mergeMetricCounter)
    prometheus.MustRegister(cacheDataLengthGauge)
    prometheus.MustRegister(sendRequestNumCounter)
    prometheus.MustRegister(metricsSizeCounter)
    prometheus.MustRegister(packStatusCounter)
    prometheus.MustRegister(receiveMetricsNumCounter)
    prometheus.MustRegister(sendMetricsNumCounter)
}

func GaugeMonitor() {
    t := time.NewTicker(time.Second * time.Duration(15))
    for {
        <-t.C
        for i := 0; i < Conf.queuesNum; i++ {
            tsQueueLengthGauge.With(prometheus.Labels{"type": "request", "queueIndex": "queue-" + strconv.Itoa(i)}).Set(float64(TsQueue.RequestLength(i)))
            tsQueueLengthGauge.With(prometheus.Labels{"type": "merge", "queueIndex": "queue-" + strconv.Itoa(i)}).Set(float64(TsQueue.MergeLength(i)))
        }
        for jobName, agg := range Collection.whiteJobName {
            cacheDataLengthGauge.With(prometheus.Labels{"jobname": jobName, "type": "prev"}).Set(float64(len(agg.prevCache.data)))
        }

    }
}
