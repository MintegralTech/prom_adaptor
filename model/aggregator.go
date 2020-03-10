package model

import (
    "fmt"
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
    timestamp int64
    prevCache *cache
    sumCache  *cache
    pack      *block
    ready     *block
}

type Aggregators struct {
    jobNum      int
    whitelist   map[string]struct{}
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
        timestamp: 0,
        prevCache: &cache{data: make(map[int]*prompb.Sample)},
        sumCache:  &cache{data: make(map[int]*prompb.Sample)},
        pack:      &block{data: make(map[int]*prompb.TimeSeries)},
        ready:     &block{data: make(map[int]*prompb.TimeSeries)},
    }
}

func NewAggregators() *Aggregators {
    whitelist := make(map[string]struct{}, len(Conf.whitelist))
    for _, jobName := range Conf.whitelist {
        whitelist[jobName] = struct{}{}
    }
    aggs := &Aggregators{
        jobNum:      len(Conf.jobNames)+len(Conf.whitelist),
        whitelist:   whitelist,
        aggregators: make(map[string]*Aggregator),
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

func (collection *Aggregators) pack(jobName string, ts *prompb.TimeSeries, sumVal float64) {
    noInstTs := DeleteLable(ts, INSTANCE)
    metric := GetMetric(noInstTs)
    hc := hashcode.String(metric)
    pack := collection.aggregators[jobName].pack
    curTime := ts.Samples[0].Timestamp - (ts.Samples[0].Timestamp % int64(Conf.window))
    if curTime != collection.aggregators[jobName].timestamp {
        collection.aggregators[jobName].ready = collection.aggregators[jobName].pack
        collection.aggregators[jobName].pack = &block{make(map[int]*prompb.TimeSeries)}
        go collection.send(jobName)
        collection.aggregators[jobName].timestamp = curTime
    }
    if _, ok := pack.data[hc]; !ok {
        pack.data[hc] = noInstTs
    } else {
        pack.data[hc].Samples[0].Value += noInstTs.Samples[0].Value
    }
}

func (collection *Aggregators) packNoFilter(jobName string, ts *prompb.TimeSeries) {
    //todo  
    ReqLog.WithFields(logrus.Fields{"jobName": jobName}).Info("whitelist")
}

func (collection *Aggregators) send(jobName string) {
    var wreq *prompb.WriteRequest
    ready := collection.aggregators[jobName].ready
    for _, ts := range ready.data {
        tempTs := *ts
        wreq.Timeseries = append(wreq.Timeseries, &tempTs)
    }
    collection.aggregators[jobName].ready = nil
    ReqLog.Info(wreq)
}

func (collection *Aggregators) MergeMetric(ts *prompb.TimeSeries) error {
    metric := GetMetric(ts)
    ReqLog.WithFields(logrus.Fields{"metric": metric}).Info("metrics")
    fields := strings.Split(metric, "_")
    ReqLog.WithFields(logrus.Fields{"fields": fields}).Info("fields")
    if len(fields) < 1 {
        return errors.New("split metric name error")
    }
    jobName := strings.ToLower(fields[0] + "_" + fields[1])
    ReqLog.WithFields(logrus.Fields{"jobName": jobName}).Info("jobName")
    if _, ok := collection.whitelist[jobName]; ok {
        ReqLog.Info("without filter")
        collection.packNoFilter(jobName, ts)
        return nil
    }
    if _, ok := collection.aggregators[jobName]; !ok {
        ReqLog.Info("no this job")
        return errors.New("no this job")
    }
    if len(ts.Samples) < 1 {
        ReqLog.Info("no sample")
        return errors.New("no sample")
    }
    hc := hashcode.String(metric)
    incVal := collection.updatePrevCache(jobName, hc, &ts.Samples[0])
    sumVal := collection.updateSumCache(jobName, hc, &ts.Samples[0], incVal)
    collection.pack(jobName, ts, sumVal)
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
