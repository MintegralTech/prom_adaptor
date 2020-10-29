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
	//首先要保证设置的requestbuffer的大小一定大于一次request发送过来的数据量（prometheus每次默认最大发送500条metrics）
	//然后 当当前缓存的余量不够存放一批request的数据时，启动requestconsumer开始消费requestbuffer。
	// TODO 添加metrics，记录每次的request发送过来的数据量
	batchSize := Conf.GetBatchSize()
	curSize := 0
	for _, ts := range wreq.Timeseries {
		if curSize >= batchSize {
			RequestBufferConsumerFlag <- FullCap
			curSize = 0
		}
		//对metrics名称hash，得到hashid 取余队列个数，按照其结果进行分发数据
		num, jobName, isMergeFlag, err := TsQueue.DistributeData(ts)
		if err != nil{
			receiveMetricsNumCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(num), "type": "fail"}).Inc()
			continue
		}
		receiveMetricsNumCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(num), "type": "succ"}).Inc()
		requestBufferLengthGauge.With(prometheus.Labels{"type": "realLength"}).Set(float64(len(RequestBuffer)))
		// 需要聚合的数据入缓存，其他直接入发送队列透传
		if isMergeFlag {
			RequestBuffer <- ts
			curSize++
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
		if flag == FullCap{
			triggerBufferConsumerCounter.With(prometheus.Labels{"type": "FullCap"}).Inc()

		}else{
			triggerBufferConsumerCounter.With(prometheus.Labels{"type": "TimeOut"}).Inc()
		}
		RwMtx.Lock()
		LastFlagTimeStamp = time.Now().Unix()
		RwMtx.Unlock()
		TsQueue.RequestBufferConsumer()
	}
}