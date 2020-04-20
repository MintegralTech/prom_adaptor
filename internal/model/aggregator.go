package model

import (
    "errors"
    "fmt"
    "math"
    "strings"
    "strconv"
    "sync"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/common/model"
    "github.com/prometheus/prometheus/prompb"
    "github.com/sirupsen/logrus"
)

// 聚合后的数据带上一个flag 用来标记本次自然时间窗口是否有聚合操作
type TimeSeries struct {
    ts   *prompb.TimeSeries
    flag bool
}

// 存放上一采集的数据
type cache struct {
    data map[string]*prompb.Sample
}

// 存放聚合后的数据
type block struct {
    data map[string]*TimeSeries
}

// 聚合器，与job绑定
type Aggregator struct {
    jobName string
    mtx     sync.Mutex

    prevCache *cache
    pack      *block
}


type Aggregators struct {
    jobNum     int
    aggregator map[string]*Aggregator
}

const (
    INSTANCE = "instance"
)

var Collection *Aggregators

func InitCollection() {
    Collection = NewAggregators()
}

func NewAggregator(jobName string) *Aggregator {
    return &Aggregator{
        jobName:   jobName,
        prevCache: &cache{data: make(map[string]*prompb.Sample)},
        pack:      &block{data: make(map[string]*TimeSeries)},
    }
}

func NewAggregators() *Aggregators {
    aggs := &Aggregators{
        jobNum:       len(Conf.jobNames),
        aggregator: make(map[string]*Aggregator, len(Conf.jobNames)),
    }
    for _, jobName := range Conf.jobNames {
        aggs.aggregator[jobName] = NewAggregator(jobName)
    }
    return aggs
}

func (collection *Aggregators) updatePrevCache(prevCache *cache, metric string, sample *prompb.Sample) float64 {
    incVal := sample.Value
    if prevSample, ok := prevCache.data[metric]; ok {
        curVal, prevVal := sample.Value, prevSample.Value
        if prevSample.Timestamp > sample.Timestamp {
            return 0
        }
        if curVal >= prevVal {
            incVal = curVal - prevVal
        }
    }
    tempSample := *sample
    prevCache.data[metric] = &tempSample
    return incVal
}

func (collection *Aggregators) updatePack(jobName string, ts *prompb.TimeSeries, incVal float64) {
    noInstTs := DeleteLable(ts, INSTANCE)
    metric := GetMetric(noInstTs)
    collection.aggregator[jobName].mtx.Lock()
    defer collection.aggregator[jobName].mtx.Unlock()
    pack := collection.aggregator[jobName].pack
    if _, ok := pack.data[metric]; !ok {
        pack.data[metric] = &TimeSeries {
             ts:   noInstTs,
             flag: true,
        }
    } else {
        if pack.data[metric].ts.Samples[0].Value + incVal >= 0 {
            pack.data[metric].ts.Samples[0].Value += incVal
        } else {
            pack.data[metric].ts.Samples[0].Value = incVal
        }
        if pack.data[metric].ts.Samples[0].Timestamp < ts.Samples[0].Timestamp {
            pack.data[metric].ts.Samples[0].Timestamp = ts.Samples[0].Timestamp
        }
        pack.data[metric].flag = true
    }
}

func (collection *Aggregators) PutIntoMergeQueue() {
    ch := make(chan struct{}, collection.jobNum)
    for i := 0; i < collection.jobNum; i++ {
        ch <- struct{}{}
    }
    for jobIndex := 0; ; jobIndex++ {
        <-ch
        jobName := Conf.jobNames[jobIndex]
        window := Conf.windows[jobIndex]
        go func() {
            t := time.NewTicker(time.Second * time.Duration(window))
            for {
                <-t.C
                collection.aggregator[jobName].mtx.Lock()
                pack := collection.aggregator[jobName].pack
                //pack.Print()
                for _, val := range pack.data {
                    packStatusCounter.With(prometheus.Labels{"jobname": jobName, "flag": strconv.FormatBool(val.flag)}).Add(1)
                    if val.flag {
                        val.flag = false
                        tempTs := *(val.ts)
                        TsQueue.MergeProducer(&tempTs)
                    }
                }
                collection.aggregator[jobName].mtx.Unlock()
            }
        }()
    }
}

func (collection *Aggregators) MergeMetric(ts *prompb.TimeSeries) error {
    metric := GetMetric(ts)
    jobName, err := GetJobName(metric)
    if err != nil {
        return err
    }
    //RunLog.WithFields(logrus.Fields{"jobName": jobName}).Info("jobName")
    if len(ts.Samples) != 1 {
        return errors.New(fmt.Sprintf("error sample size[%d]", len(ts.Samples)))
    }
    //带ip的label直接放到请求队列, 不做聚合
    for _, l := range ts.Labels {
        if l.Name == "ip" {
            tempTs := *ts
            TsQueue.MergeProducer(&tempTs)
            mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "withou-aggregate"}).Add(1)
            return nil
        }
    }
    if math.IsNaN(ts.Samples[0].Value) {
        //RunLog.WithFields(logrus.Fields{"ts": metric}).Info("NaN")
        ts.Samples[0].Value = 0
    }
    if cache, ok := collection.aggregator[jobName]; ok {
        incVal := collection.updatePrevCache(cache.prevCache, metric, &ts.Samples[0])
        collection.updatePack(jobName, ts, incVal)
        mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "aggregate"}).Add(1)
    } else {
        tempTs := *ts
        TsQueue.MergeProducer(&tempTs)
        mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "without-aggregate"}).Add(1)
    }
    return nil
}

func DeleteLable(ts *prompb.TimeSeries, labelName string) *prompb.TimeSeries {
    res := *ts
    for i, l := range res.Labels {
        if l.Name == labelName {
            res.Labels = append(res.Labels[:i], res.Labels[i+1:]...)
            break
        }
    }
    return &res
}

func GetMetric(ts *prompb.TimeSeries) string {
    m := make(model.Metric, len(ts.Labels))
    for _, l := range ts.Labels {
        m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
    }
    metric := fmt.Sprint(m)
    return metric
}

func (c *cache) Print() {
    for k, v := range c.data {
        fmt.Println(k, v.Value)
    }
}

func (b *block) Print() {
    for k, v := range b.data {
        fmt.Println(k, GetMetric(v.ts)+GetSample(v.ts), v.flag)
    }
}

func GetSample(ts *prompb.TimeSeries) string {
    var sample string
    for _, s := range ts.Samples {
        sample = fmt.Sprintf("  %f %d", s.Value, s.Timestamp)
    }
    return sample
}

func GetJobName(metric string) (string, error) {
    var jobName string
    temp := strings.Split(metric, "{")
    if len(temp) < 1 {
        return "", errors.New("get metricsName error")
    }
    metricName := temp[0]
    fields := strings.Split(metricName, "_")
    if len(fields) <= 0 {
        return "", errors.New("get jobName error")
    }
    if len(fields) == 1 {
        jobName = strings.ToLower(fields[0])
    } else {
        jobName = strings.ToLower(fields[0] + "_" + fields[1])
    }
    return jobName, nil
}

