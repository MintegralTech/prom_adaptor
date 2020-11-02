package model

import (
    "errors"
    "fmt"
    "github.com/hashicorp/terraform/helper/hashcode"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/prometheus/prompb"
    _ "github.com/sirupsen/logrus"
    "math"
    "sort"
    "strconv"
)

type (
	MergeTimeWindow struct {
        leftTime        int64
        rightTime       int64
        timeWindowSize  int64
    }

    //在一个窗口的聚合过的job 名单
	MergedJobNameList struct {
	    list map[string][]string //key=JobName, value=[metrics, m2]
    }

    TimeSeriesQueue struct {
        sendQueue []chan *prompb.TimeSeries
        mergeQueue   []chan *prompb.TimeSeries

        workersNum      int                 //队列个数， send和merge的队列数相同
        buffer          int                 //requestQueue和mergeQueue的容量
        mergeTimeWindow []MergeTimeWindow   //聚合时间窗口信息
        mergedLists     []MergedJobNameList //索引为对应的任务序号
    }
)

var (
	TsQueue *TimeSeriesQueue
    SUFFIX string = "_aggregator"
    MergeQueueConsumerLength []chan int
)

const (
    GreaterCurMergeWindow   = 1
    LessCurMergeWindow      = 2
    InCurMergeWindow        = 3
)

func InitQueue() {
    TsQueue = NewTimeSeriesQueue(Conf.GetRequestBuffCapacity(), Conf.GetWorkersNum())
    MergeQueueConsumerLength = make([]chan int, Conf.GetWorkersNum())
    for i := 0; i < Conf.GetWorkersNum(); i++ {
        MergeQueueConsumerLength[i] = make(chan int, 1)
    }
}

func NewTimeSeriesQueue(buffer int, workersNum int) *TimeSeriesQueue {
    tmpTimeSeriesQueue := &TimeSeriesQueue{
        sendQueue:       make([]chan *prompb.TimeSeries, workersNum),
        mergeQueue:      make([]chan *prompb.TimeSeries, workersNum),
        mergeTimeWindow: make([]MergeTimeWindow, workersNum),
        mergedLists:     make([]MergedJobNameList, workersNum),
        workersNum:      workersNum,
        buffer:          buffer,
    }

    for i := 0; i < workersNum; i++{
    	tmpTimeSeriesQueue.sendQueue[i] = make(chan *prompb.TimeSeries, Conf.GetQueueCapacity())
    	tmpTimeSeriesQueue.mergeQueue[i] = make(chan *prompb.TimeSeries, Conf.GetQueueCapacity())
    	tmpTimeSeriesQueue.mergeTimeWindow[i].timeWindowSize = Conf.GetWindows()
    	tmpTimeSeriesQueue.mergedLists[i].list = make(map[string][]string)
	}

	return tmpTimeSeriesQueue
}



func (tsq *TimeSeriesQueue) RequestBufferConsumer() {
    //RunLog.WithFields(logrus.Fields{"queue length": tsq.RequestLength(), "add metrics count:": len(wreq.Timeseries)}).Info("request producer")
    batchSize := Conf.GetBatchSize()
    mergeQueueLen := make([]int, Conf.GetWorkersNum())

    for i := 0; i < batchSize; i++ {
        ts := <- RequestBuffer
        //if Conf.GetMode() == "debug"{
        //    ReqLog.Println(ts)
        //}

        var err error
        //对metrics名称hash，得到hashid 取余队列个数，按照其结果进行分发数据
        // TODO 可以考虑优化，  在入requestbuffer的时候得到的num值一并保存起来，这里就无需重复distributeData了
        num, _, _, err := tsq.DistributeData(ts)
        if err != nil{
            continue
        }

        tsq.mergeQueue[num] <- ts
        mergeQueueLen[num] ++
    }
    for i := 0; i < len(mergeQueueLen); i++ {
        MergeQueueConsumerLength[i] <- mergeQueueLen[i]
    }
}
func (tsq *TimeSeriesQueue) DistributeData(ts *prompb.TimeSeries) (int, string, bool, error){
    var err error
    isMergeFlag := false
    metric, isExistIpLabel := GetMetric(ts)
    jobName, originMetricName, err := GetJobName(metric)
    if err != nil {
        return 0, "", false, err
    }
    //去除metrics名称最后的bucket,sum, count等字段，确保histogram指标（bucket，sum, count)在同一个队列中，否则会导致histogram不准确
    metricName, err := GetMetricsName(originMetricName)
    if err != nil {
        return 0, "", false, err
    }
    hashId := hashcode.String(metricName)
    remainder := hashId % tsq.workersNum
    if 0 > remainder || tsq.workersNum <= remainder{
        err = errors.New("distribute data: hash id illegal, hashId=" + strconv.Itoa(hashId) + ", remainder=" + strconv.Itoa(remainder))
        return 0, jobName, false, err
    }
    // 在聚合白名单且不含IP 标签的metric进行聚合
    if _, ok := Collection.aggregator[jobName]; ok && !isExistIpLabel {
        isMergeFlag = true
    }

    return remainder, jobName, isMergeFlag, nil
}


func (tsq *TimeSeriesQueue) MergeQueueConsumer(index int) {
    for {
        select {
        case queueLength := <-MergeQueueConsumerLength[index]:
            sortedQueue := make([]*prompb.TimeSeries, queueLength)
            l := len(tsq.mergeQueue[index])
            tsQueueLengthGauge.With(prometheus.Labels{"type": "merge", "queueIndex": "queue-" + strconv.Itoa(index)}).Set(float64(l))
            if l <= 0 {
                continue
            }

            // 读queueLength个数据，并校验数据格式正确性
            for i := 0; i < queueLength; i++ {
                ts := <- tsq.mergeQueue[index]
                if len(ts.Samples) != 1 {
                    RunLog.Error(errors.New(fmt.Sprintf("error sample size[%d]", len(ts.Samples))))
                    continue
                }
                if math.IsNaN(ts.Samples[0].Value) {
                    ts.Samples[0].Value = 0
                }
                sortedQueue[i] = ts
            }

            //升序
            sort.Slice(sortedQueue, func(i, j int) bool {
                return sortedQueue[i].Samples[0].Timestamp < sortedQueue[j].Samples[0].Timestamp
            })
            //使用当前时间戳最小的metric初始化聚合时间窗口
            tsq.mergeTimeWindow[index].SetMergeTimeWindow(sortedQueue[0])

            for _, v := range sortedQueue {
                metric, _ := GetMetric(v)
                jobName, _, err := GetJobName(metric)
                if err != nil {
                    continue
                }
                Collection.MergeMetric(v, jobName, metric, index)
            }
            // 最后一个窗口的数据 强制入send队列（因为最后一个窗口没有新数据自动触发该动作）
            Collection.PutMergedMetricsIntoSendQueue(index)
        }
    }
}



func (tsq *TimeSeriesQueue) SendProducer(ts *prompb.TimeSeries, index int) {
    tsq.sendQueue[index] <- ts
}

func (tsq *TimeSeriesQueue) SendConsumer(index int) {
    var ts *prompb.TimeSeries
    var tsSlice []*prompb.TimeSeries
    for {
        select {
        case ts = <-tsq.sendQueue[index]:
            // debug
            if Conf.GetMode() == "debug" {
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
                    metricsTotalSizeCounter.With(prometheus.Labels{"jobname": l.Value}).Inc()
                    break
                }
            }
            tsSlice = append(tsSlice, ts)
            if len(tsSlice) == Conf.GetShard() || (len(tsq.mergeQueue[index]) == 0 && len(RequestBuffer) == 0) {
                if err := client.Write(tsSlice, index); err != nil {
                    ReqLog.Error(err)
                }
                tsSlice = []*prompb.TimeSeries{}
            }
        }
    }
}


func (m *MergeTimeWindow)CompareWithMergeTimeWindow(ts *prompb.TimeSeries) int {
    tStamp := ts.Samples[0].Timestamp
    if tStamp > m.rightTime{
        return GreaterCurMergeWindow
    }else if tStamp < m.leftTime {
        return LessCurMergeWindow
    }
    return InCurMergeWindow
}

func (m *MergeTimeWindow)SetMergeTimeWindow(ts *prompb.TimeSeries) {
    m.leftTime = ts.Samples[0].Timestamp
    m.rightTime = m.leftTime + m.timeWindowSize
}

func (m *MergedJobNameList)ClearList(){
    m.list = make(map[string][]string)
}