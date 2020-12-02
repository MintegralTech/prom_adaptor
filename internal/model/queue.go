package model

import (
	"errors"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	_ "github.com/sirupsen/logrus"
	"strconv"
)

type TimeSeriesQueue struct {
	requestQueue []chan *prompb.TimeSeries
	mergeQueue   []chan *prompb.TimeSeries
	queuesNum    int //队列个数， request和merge的队列数相同
}

var TsQueue *TimeSeriesQueue
var SUFFIX string = "_aggregator"

const (
	defaultBuffer    = 100000 //默认队列长度
	defaultQueuesNum = 2      //默认队列个数
)

func InitQueue() {
	buffer := defaultBuffer
	if Conf.buffer >= buffer {
		buffer = Conf.buffer
	}
	queuesNum := defaultQueuesNum
	if Conf.queuesNum > queuesNum {
		queuesNum = Conf.queuesNum
	}
	TsQueue = NewTimeSeriesQueue(buffer, queuesNum)
}

func NewTimeSeriesQueue(buffer int, queuesNum int) *TimeSeriesQueue {
	tmpTimeSeriesQueue := &TimeSeriesQueue{
		requestQueue: make([]chan *prompb.TimeSeries, queuesNum),
		mergeQueue:   make([]chan *prompb.TimeSeries, queuesNum),
		queuesNum:    queuesNum,
	}
	for i := 0; i < queuesNum; i++ {
		tmpTimeSeriesQueue.requestQueue[i] = make(chan *prompb.TimeSeries, buffer)
		tmpTimeSeriesQueue.mergeQueue[i] = make(chan *prompb.TimeSeries, buffer)
	}
	return tmpTimeSeriesQueue
}

func (tsq *TimeSeriesQueue) RequestProducer(wreq *prompb.WriteRequest) {
	//RunLog.WithFields(logrus.Fields{"queue length": tsq.RequestLength(), "add metrics count:": len(wreq.Timeseries)}).Info("request producer")

	for _, ts := range wreq.Timeseries {
		if Conf.mode == "debug" {
			ReqLog.Println(ts)
		}

		var err error
		//对metrics名称hash，得到hashid 取余队列个数，按照其结果进行分发数据
		num, jobName, err := tsq.distributeData(ts)

		if err != nil {
			receiveMetricsNumCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(num), "type": "fail"}).Inc()
			continue
		}
		receiveMetricsNumCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(num), "type": "succ"}).Inc()
		tsq.requestQueue[num] <- ts
		//AccLog.WithFields(logrus.Fields{"metric": GetMetric(ts) + GetSample(ts)}).Info("request producer")
	}
}
func (tsq *TimeSeriesQueue) distributeData(ts *prompb.TimeSeries) (int, string, error) {
	var err error
	metric := GetMetric(ts)
	jobName, originMetricName, err := GetJobName(metric)
	if err != nil {
		return 0, "", err
	}
	//去除metrics名称最后的bucket,sum, count等字段，确保histogram指标（bucket，sum, count)在同一个队列中，否则会导致histogram不准确
	metricName, err := GetMetricsName(originMetricName)
	if err != nil {
		return 0, "", err
	}
	hashId := hashcode.String(metricName)
	remainder := hashId % tsq.queuesNum
	if 0 > remainder || tsq.queuesNum <= remainder {
		err = errors.New("distribute data: hash id illegal, hashId=" + strconv.Itoa(hashId) + ", remainder=" + strconv.Itoa(remainder))
		return 0, jobName, err
	}

	return remainder, jobName, nil
}

func (tsq *TimeSeriesQueue) RequestConsumer(index int) {
	var err error
	var ts *prompb.TimeSeries
	for {
		select {
		case ts = <-tsq.requestQueue[index]:
			err = Collection.MergeMetric(ts, index)
			if err != nil {
				RunLog.Error(err)
			}
		}
	}
}

func (tsq *TimeSeriesQueue) MergeProducer(ts *prompb.TimeSeries, index int) {
	tsq.mergeQueue[index] <- ts
}

func (tsq *TimeSeriesQueue) MergeConsumer(index int) {
	var ts *prompb.TimeSeries
	var tsSlice []*prompb.TimeSeries
	for {
		select {
		case ts = <-tsq.mergeQueue[index]:
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
				client.Write(tsSlice, index)
				tsSlice = []*prompb.TimeSeries{}
			}
		}
	}
}

func (tsq *TimeSeriesQueue) RequestLength(index int) int {
	return len(tsq.requestQueue[index])
}

func (tsq *TimeSeriesQueue) MergeLength(index int) int {
	return len(tsq.mergeQueue[index])
}
