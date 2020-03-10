package model

import (
    "github.com/prometheus/prometheus/prompb"
)


type timeSeriesQueue struct {
    queue chan *prompb.TimeSeries
}

var TsQueue *timeSeriesQueue

func init() {
    buffer := 100000
    if Conf.buffer >= buffer {
        buffer = Conf.buffer
    }
    TsQueue = NewTimeSeriesQueue(buffer)
}

func NewTimeSeriesQueue(buffer int) *timeSeriesQueue {
    return &timeSeriesQueue{
        queue : make(chan *prompb.TimeSeries, buffer),
    }
}

func (tsq *timeSeriesQueue) Producer(wreq *prompb.WriteRequest) {
    for _, ts := range wreq.Timeseries {
        tsq.queue <- ts
    }
}

//func (tsq *timeSeriesQueue) Consumer(){
//    
//}

func (tsq *timeSeriesQueue) Length() int {
    return len(tsq.queue)
}


