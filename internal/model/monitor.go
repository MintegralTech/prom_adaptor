package model

import (
    "github.com/prometheus/client_golang/prometheus"
)


var (
    requestBufferLengthGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "requestBufferLength",
            Help:      "timeSeries queue length (Gauge)",
        },
        []string{"type"},
    )
    triggerBufferConsumerCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "triggerBufferConsumer_counter",
            Help:      "trigger requestBuffSize consumer (Counter)",
        },
        []string{"type"},
    )
    //延迟发过来的数据 -----------
    delayedMertricCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "delayedMetric_counter",
            Help:      "delayed metric (Counter)",
        },
        []string{"jobname", "queueIndex"},
    )

    dataInMergedWinSizeGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "dataInMergedWinSize",
            Help:      "date in merged windows size (gauge)",
        },
        []string{"jobname", "queueIndex"},
    )


    tsQueueLengthGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "queueSize",
            Help:      "timeSeries queue length",
        },
        []string{"type", "queueIndex"},
    )
    originMetricCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "originMetric_counter",
            Help:      "merge counter",
        },
        []string{"jobname", "type", "queueIndex"},
    )
    cacheDataLengthGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "cacheDataLength",
            Help:      "cache data length",
        },
        []string{"jobname", "type"},
    )
    sendMetricsNumCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "sendMetricsNum_counter",
            Help:      "send metrics num counter",
        },
        []string{"succ", "queueIndex"},
    )
    sendRequestNumCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "sendRequestNum_counter",
            Help:      "send request num counter",
        },
        []string{"succ", "queueIndex"},
    )
    receiveMetricsNumCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "receiveMetricsNum_counter",
            Help:      "receive metrics num counter",
        },
        []string{"jobname", "queueIndex", "type"},
    )
    metricsTotalSizeCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "metricsTotalSize_counter",
            Help:      "data total size",
        },
        []string{"jobname"},
    )
    cacheDeletedNumGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "adaptor",
            Subsystem: "aggregator",
            Name:      "cacheDeletedNum",
            Help:      "cache map deleted metrics nums",
        },
        []string{"jobname", "type"},
    )
)

func InitMonitor() {
    // Metrics have to be registered to be exposed:
    // request相关
    prometheus.MustRegister(requestBufferLengthGauge)
    prometheus.MustRegister(delayedMertricCounter)
    prometheus.MustRegister(receiveMetricsNumCounter)
    prometheus.MustRegister(originMetricCounter)

    // 缓存队列相关
    prometheus.MustRegister(tsQueueLengthGauge)

    // 聚合相关
    prometheus.MustRegister(triggerBufferConsumerCounter)
    prometheus.MustRegister(dataInMergedWinSizeGauge)
    prometheus.MustRegister(cacheDataLengthGauge)
    prometheus.MustRegister(cacheDeletedNumGauge)

    // send相关
    prometheus.MustRegister(sendRequestNumCounter)
    prometheus.MustRegister(sendMetricsNumCounter)
    prometheus.MustRegister(metricsTotalSizeCounter)

}
