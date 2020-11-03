package model

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"strconv"
	"sync"
	"time"
)
var (
	RequestBuffer chan *prompb.TimeSeries
	RequestBufferConsumerFlag chan int
	LastFlagTimeStamp int64 //上一次flag chan中的时间戳
	RwMtx sync.RWMutex
)


const (
	requestBufferConsumerFlagLength = 10
	FullCap = 1 //标识request buffer满了
	TimeOut = 2 //标识定时器时间到
)

func InitRequestController(){
	RequestBuffer = make(chan *prompb.TimeSeries, Conf.GetRequestBuffCapacity())
	RequestBufferConsumerFlag = make(chan int, requestBufferConsumerFlagLength)
	LastFlagTimeStamp = 0
}


func RequestIntoBuffer(wreq *prompb.WriteRequest) {

	// TODO 添加metrics，记录每次的request发送过来的数据量
	batchSize := Conf.GetBatchSize()
	curSize := 0
	for _, ts := range wreq.Timeseries {
		curSize = len(RequestBuffer)
		if curSize >= batchSize {
			RequestBufferConsumerFlag <- FullCap
		}
		//对metrics名称hash，得到hashid 取余队列个数，按照其结果进行分发数据
		num, jobName, isMergeFlag, err := TsQueue.DistributeData(ts)
		if err != nil{
			receiveMetricsNumCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(num), "type": "fail"}).Inc()
			continue
		}
		receiveMetricsNumCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(num), "type": "succ"}).Inc()
		requestBufferLengthGauge.With(prometheus.Labels{"type": "realLength"}).Set(float64(curSize))
		// 需要聚合的数据入缓存，其他直接入发送队列透传
		if isMergeFlag {
			RequestBuffer <- ts
			originMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "need-aggregate", "queueIndex":"queue-" + strconv.Itoa(num)}).Inc()
		}else{
			TsQueue.SendProducer(ts, num)
			originMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "not-need-aggregate", "queueIndex":"queue-" + strconv.Itoa(num)}).Inc()
		}
	}
}


func MonitorRequestBufferTimer(){
	//每隔N秒，读走一次request buffer中的数据
	fmt.Println("d=", Conf.GetFreshRequestQueuePeriod())
	t := time.NewTicker(time.Second * time.Duration(Conf.GetFreshRequestQueuePeriod()))
	for{
		<-t.C
		//还有数据没有消费完
		curTime := time.Now().Unix()
		RwMtx.RLock()
		period := curTime - LastFlagTimeStamp
		RwMtx.RUnlock()
		if  period < Conf.GetFreshRequestQueuePeriod() || len(RequestBufferConsumerFlag) > 0 {
			continue
		}
		RequestBufferConsumerFlag <- TimeOut

	}
}


func MonitorRequestBuffer(){
	for{
		flag := <-RequestBufferConsumerFlag
		if flag == FullCap && len(RequestBuffer) >= Conf.GetBatchSize(){
			triggerBufferConsumerCounter.With(prometheus.Labels{"type": "FullCap"}).Inc()

		}else if flag == TimeOut {
			triggerBufferConsumerCounter.With(prometheus.Labels{"type": "TimeOut"}).Inc()
		}else{
			continue
		}
		RwMtx.Lock()
		LastFlagTimeStamp = time.Now().Unix()
		RwMtx.Unlock()
		TsQueue.RequestBufferConsumer()
	}
}