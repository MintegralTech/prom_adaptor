package model

import (
	"github.com/prometheus/prometheus/prompb"
	"github.com/sirupsen/logrus"
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
	RunLog.WithFields(logrus.Fields{"queue length": tsq.RequestLength(), "add metrics count:": len(wreq.Timeseries)}).Info("request producer")
	for _, ts := range wreq.Timeseries {
		tsq.requestQueue <- ts
		AccLog.WithFields(logrus.Fields{"metric": GetMetric(ts) + GetSample(ts)}).Info("request producer")
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
			if Conf.mode == "debug" {
				for i, l := range ts.Labels {
					if l.Name == "__name__" {
						ts.Labels[i].Value += "_aggregator"
						break
					}
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