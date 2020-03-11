package model

import (
    "github.com/sirupsen/logrus"
    "github.com/prometheus/prometheus/prompb"
)


type TimeSeriesQueue struct {
    requestQueue chan *prompb.TimeSeries
    mergeQueue   chan *prompb.TimeSeries
}

var TsQueue *TimeSeriesQueue

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
    ct := 0
    for _, ts := range wreq.Timeseries {
        tsq.requestQueue <- ts
        ct++
    }
    RunLog.WithFields(logrus.Fields{"queue length": tsq.RequestLength(),"add metrics count:": ct}).Info("request producer")
}

func (tsq *TimeSeriesQueue) RequestConsumer() {
    var err error
    var ts *prompb.TimeSeries
    for {
        select {
        case ts = <-tsq.requestQueue:
            err = Collection.MergeMetric(ts)
            if err != nil {
                RunLog.Error("get job name error")
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
            tsSlice = append(tsSlice, ts)
            if len(tsSlice) == Conf.shard || len(tsq.mergeQueue) == 0 {
                client.Write(tsSlice)
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

