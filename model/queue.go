package model

import (
    "github.com/sirupsen/logrus"
    "github.com/prometheus/prometheus/prompb"
)


type TimeSeriesQueue struct {
    queue chan *prompb.TimeSeries
}

func NewTimeSeriesQueue(buffer int) *TimeSeriesQueue {
    return &TimeSeriesQueue{
        queue : make(chan *prompb.TimeSeries, buffer),
    }
}

func (tsq *TimeSeriesQueue) Producer(wreq *prompb.WriteRequest) {
    ct := 0
    for _, ts := range wreq.Timeseries {
        tsq.queue <- ts
        ct++
    }
    RunLog.WithFields(logrus.Fields{"queue length": tsq.Length(),"add metrics count:": ct}).Info("runtime")
}

func (tsq *TimeSeriesQueue) Consumer(){
    var ts *prompb.TimeSeries
    for {
        select {
        case ts = <-tsq.queue:
            Collection.MergeMetric(ts)
        }
    }
}

func (tsq *TimeSeriesQueue) Length() int {
    return len(tsq.queue)
}


