package model

import (
    "fmt"
    "sync"
    "time"
    "errors"
    "strings"

    "github.com/sirupsen/logrus"
    "github.com/prometheus/common/model"
    "github.com/prometheus/prometheus/prompb"
    "github.com/hashicorp/terraform/helper/hashcode"
)

type block struct {
    data map[int]*prompb.TimeSeries
}

type cache struct {
    data map[int]*prompb.Sample
}

type Aggregator struct {
    jobName   string
    mtx       sync.Mutex

    prevCache *cache
    sumCache  *cache
    pack      *block
}

type Aggregators struct {
    jobNum      int
    aggregators map[string]*Aggregator
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
        prevCache: &cache{data: make(map[int]*prompb.Sample)},
        sumCache:  &cache{data: make(map[int]*prompb.Sample)},
        pack:      &block{data: make(map[int]*prompb.TimeSeries)},
    }
}

func NewAggregators() *Aggregators {
    aggs := &Aggregators{
        jobNum:      len(Conf.jobNames),
        aggregators: make(map[string]*Aggregator, len(Conf.jobNames)),
    }
    for _, jobName := range Conf.jobNames {
        aggs.aggregators[jobName] = NewAggregator(jobName)
    }
    return aggs
}

func (collection *Aggregators) updatePrevCache(jobName string, hc int, sample *prompb.Sample) float64 {
    incVal := sample.Value
    prevCache := collection.aggregators[jobName].prevCache
    if prevSample, ok := prevCache.data[hc]; ok {
        curVal, prevVal := sample.Value, prevSample.Value
        if curVal > prevVal {
            incVal = curVal - prevVal
        }
    }
    tempSample := *sample
    prevCache.data[hc] = &tempSample
    return incVal
}

func (collection *Aggregators) updateSumCache(jobName string, hc int, sample *prompb.Sample, incVal float64) float64 {
    sumVal := incVal
    sumCache := collection.aggregators[jobName].sumCache
    if _, ok := sumCache.data[hc]; ok {
        sumCache.data[hc].Value += incVal
        sumCache.data[hc].Timestamp = sample.Timestamp
        sumVal = sumCache.data[hc].Value
    } else {
        tempSample := *sample
        sumCache.data[hc] = &tempSample
    }
    return sumVal
}

func (collection *Aggregators) updatePack(jobName string, ts *prompb.TimeSeries, sumVal float64) {
    noInstTs := DeleteLable(ts, INSTANCE)
    metric := GetMetric(noInstTs)
    hc := hashcode.String(metric)
    collection.aggregators[jobName].mtx.Lock()
    defer collection.aggregators[jobName].mtx.Unlock()
    pack := collection.aggregators[jobName].pack
    if _, ok := pack.data[hc]; !ok {
        pack.data[hc] = noInstTs
    } else {
        pack.data[hc].Samples[0].Value += noInstTs.Samples[0].Value
    }
}

func (collection *Aggregators) MonitorPack() {
    ch := make(chan struct{}, collection.jobNum)
    for i := 0; i < collection.jobNum; i++ {
        ch <- struct{}{}
    }
    jobIndex := 0
    for {
        <-ch
        jobName := Conf.jobNames[jobIndex]
        window :=  Conf.windows[jobIndex]
        go func() {
            t := time.NewTicker(time.Second * time.Duration(window))
            for {
                <-t.C
                collection.aggregators[jobName].mtx.Lock()
                pack := collection.aggregators[jobName].pack
                for _, ts := range pack.data {
                    tempTs := *ts
                    TsQueue.MergeProducer(&tempTs)
                }
                collection.aggregators[jobName].pack = &block{make(map[int]*prompb.TimeSeries)}
                collection.aggregators[jobName].mtx.Unlock()
            }
        }()
        jobIndex++
    }
}

func (collection *Aggregators) MergeMetric(ts *prompb.TimeSeries) error {
    metric := GetMetric(ts)
    RunLog.WithFields(logrus.Fields{"metric": metric}).Info("metrics")
    fields := strings.Split(metric, "_")
    RunLog.WithFields(logrus.Fields{"fields": fields}).Info("fields")
    if len(fields) < 1 {
        return errors.New("get job name error")
    }
    jobName := strings.ToLower(fields[0] + "_" + fields[1])
    RunLog.WithFields(logrus.Fields{"jobName": jobName}).Info("jobName")
    if len(ts.Samples) < 1 {
        return errors.New("no sample")
    }
    if _, ok := collection.aggregators[jobName]; !ok {
        RunLog.Info("without filter")
        tempTs := *ts
        TsQueue.MergeProducer(&tempTs)
        return nil
    }
    hc := hashcode.String(metric)
    incVal := collection.updatePrevCache(jobName, hc, &ts.Samples[0])
    sumVal := collection.updateSumCache(jobName, hc, &ts.Samples[0], incVal)
    collection.updatePack(jobName, ts, sumVal)
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
