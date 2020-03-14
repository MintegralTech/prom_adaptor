package model

import (
    "github.com/sirupsen/logrus"
    "github.com/prometheus/prometheus/prompb"
    "github.com/prometheus/client_golang/prometheus"
)

type TimeSeriesQueue struct {
    requestQueue chan *prompb.TimeSeries
    mergeQueue   chan *prompb.TimeSeries
}

var TsQueue *TimeSeriesQueue
var SUFFIX string = "_aggregator"

func InitQueue() {
    buffer := 100000
    if Conf.buffer >= buffer {
        buffer = Conf.buffer
    }
    TsQueue = NewTimeSeriesQueue(buffer)
}

func NewTimeSeriesQueue(buffer int) *TimeSeriesQueue {
    return &TimeSeriesQueue{
        requestQueue: make(chan *prompb.TimeSeries, buffer),
        mergeQueue:   make(chan *prompb.TimeSeries, buffer),
    }
}

func (tsq *TimeSeriesQueue) RequestProducer(wreq *prompb.WriteRequest) {
    RunLog.WithFields(logrus.Fields{"queue length": tsq.RequestLength(), "add metrics count:": len(wreq.Timeseries)}).Info("request producer")
    for _, ts := range wreq.Timeseries {
        tsq.requestQueue <- ts
        //AccLog.WithFields(logrus.Fields{"metric": GetMetric(ts) + GetSample(ts)}).Info("request producer")
    }
}

func (tsq *TimeSeriesQueue) RequestConsumer() {
    var err error
    var ts *prompb.TimeSeries
    for {
        select {
        case ts = <-tsq.requestQueue:
            err = Collection.MergeMetric(ts)
            if err != nil {
                RunLog.Error(err)
            }
        }
    }
}

func (tsq *TimeSeriesQueue) MergeProducer(ts *prompb.TimeSeries) {
    tsq.mergeQueue <- ts
}

func (tsq *TimeSeriesQueue) MergeConsumer() {
    var ts *prompb.TimeSeries
    var tsSlice []*prompb.TimeSeries
    for {
        select {
        case ts = <-tsq.mergeQueue:
            // debug
            if Conf.mode == "debug" {
                for i, l := range ts.Labels {
                    if l.Name == "__name__" {
                        if len(l.Value) <= len(SUFFIX) || l.Value[len(l.Value)-len(SUFFIX):] != SUFFIX {
                            ts.Labels[i].Value += SUFFIX
                        }
                        break
                    }
                }
            }
            for _, l := range ts.Labels {
                if l.Name == "job" {
                    metricsSizeCounter.With(prometheus.Labels{"jobname": l.Value}).Inc()
                    break
                }
            }
            tsSlice = append(tsSlice, ts)
            if len(tsSlice) == Conf.shard || len(tsq.mergeQueue) == 0 {
                if err := client.Write(tsSlice); err != nil {
                    ReqLog.Error(err)
                }
                tsSlice = []*prompb.TimeSeries{}
            }
        }
    }
}

func (tsq *TimeSeriesQueue) RequestLength() int {
    return len(tsq.requestQueue)
}

func (tsq *TimeSeriesQueue) MergeLength() int {
    return len(tsq.mergeQueue)
}
