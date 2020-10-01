package model

import (
    "errors"
    "fmt"
    "math"
    "strconv"
    "sync"
    "time"

    "github.com/hashicorp/terraform/helper/hashcode"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/prometheus/prompb"
)

type TimeSeries struct {
    ts   *prompb.TimeSeries
    flag bool
}

type block struct {
    data map[int]*TimeSeries
}

type cache struct {
    data map[int]*prompb.Sample
}

type Aggregator struct {
    jobName string
    mtx     sync.Mutex

    prevCache *cache
    pack      *block
}

type Aggregators struct {
    jobNum       int
    whiteJobName map[string]*Aggregator
}

const (
    INSTANCE = "instance"
    //INSTANCE = "ip"
)

var Collection *Aggregators

func InitCollection() {
    Collection = NewAggregators()
}

func NewAggregator(jobName string) *Aggregator {
    return &Aggregator{
        jobName:   jobName,
        prevCache: &cache{data: make(map[int]*prompb.Sample)},
        pack:      &block{data: make(map[int]*TimeSeries)},
    }
}

func NewAggregators() *Aggregators {
    aggs := &Aggregators{
        jobNum:       len(Conf.jobNames),
        whiteJobName: make(map[string]*Aggregator, len(Conf.jobNames)),
    }
    for _, jobName := range Conf.jobNames {
        aggs.whiteJobName[jobName] = NewAggregator(jobName)
    }
    return aggs
}

func (collection *Aggregators) updatePrevCache(prevCache *cache, hc int, sample *prompb.Sample) float64 {
    incVal := sample.Value
    if prevSample, ok := prevCache.data[hc]; ok {
        //fmt.Println(prevSample.Timestamp, sample.Timestamp)
        if prevSample.Timestamp > sample.Timestamp {
            //RunLog.WithFields(logrus.Fields{"prev timestamp":prevSample.Timestamp, "cur timestamp": sample.Timestamp}).Info("delay")
            incVal = 0
            return incVal
        } else {
            curVal, prevVal := sample.Value, prevSample.Value
            if curVal >= prevVal {
                incVal = curVal - prevVal
            }
        }
    }
    tempSample := *sample
    prevCache.data[hc] = &tempSample
    return incVal
}

func (collection *Aggregators) updatePack(jobName string, ts *prompb.TimeSeries, incVal float64) {
    noInstTs := DeleteLable(ts, INSTANCE)
    metric := GetMetric(noInstTs)
    hc := hashcode.String(metric)
    collection.whiteJobName[jobName].mtx.Lock()
    defer collection.whiteJobName[jobName].mtx.Unlock()
    pack := collection.whiteJobName[jobName].pack
    if _, ok := pack.data[hc]; !ok {
        pack.data[hc] = &TimeSeries {
             ts:   noInstTs,
             flag: true,
        }
    } else {
        pack.data[hc].ts.Samples[0].Value += incVal
        if pack.data[hc].ts.Samples[0].Timestamp < ts.Samples[0].Timestamp {
            pack.data[hc].ts.Samples[0].Timestamp = ts.Samples[0].Timestamp
        }
        pack.data[hc].flag = true
    }
}

func (collection *Aggregators) MonitorPack(index int) {
    ch := make(chan struct{}, collection.jobNum)
    for i := 0; i < collection.jobNum; i++ {
        ch <- struct{}{}
    }
    jobIndex := 0
    for {
        <-ch
        jobName := Conf.jobNames[jobIndex]
        window := Conf.windows[jobIndex]
        go func() {
            t := time.NewTicker(time.Second * time.Duration(window))
            for {
                <-t.C
                //fmt.Println(jobName," ",window," ",time.Now())
                collection.whiteJobName[jobName].mtx.Lock()
                pack := collection.whiteJobName[jobName].pack
               // pack.Print()
                for _, val := range pack.data {
                    packStatusCounter.With(prometheus.Labels{"jobname": jobName, "flag": strconv.FormatBool(val.flag)}).Add(1)
                    if val.flag {
                        val.flag = false
                        tempTs := *(val.ts)
                        TsQueue.MergeProducer(&tempTs, index)
                    }
                }
                collection.whiteJobName[jobName].mtx.Unlock()
            }
        }()
        jobIndex++
    }
}

func (collection *Aggregators) MergeMetric(ts *prompb.TimeSeries, index int) error {
    metric := GetMetric(ts)
    jobName, err := GetJobName(metric)
    if err != nil {
        return err
    }
    //RunLog.WithFields(logrus.Fields{"jobName": jobName}).Info("jobName")
    if len(ts.Samples) != 1 {
        return errors.New(fmt.Sprintf("error sample size[%d]", len(ts.Samples)))
    }
    for _, l := range ts.Labels {
        if l.Name == "ip" {
            tempTs := *ts
            TsQueue.MergeProducer(&tempTs, index)
            mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "without-aggregate", "queueIndex":"queue-" + strconv.Itoa(index)}).Inc()
            return nil
        }
    }
    if math.IsNaN(ts.Samples[0].Value) {
        //RunLog.WithFields(logrus.Fields{"ts": metric}).Info("NaN")
        ts.Samples[0].Value = 0
    }
    hc := hashcode.String(metric)

    if cache, ok := collection.whiteJobName[jobName]; ok {
        collection.whiteJobName[jobName].mtx.Lock()
        incVal := collection.updatePrevCache(cache.prevCache, hc, &ts.Samples[0])
        collection.whiteJobName[jobName].mtx.Unlock()
        //cache.prevCache.Print()
        collection.updatePack(jobName, ts, incVal)

        mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "aggregate", "queueIndex":"queue-" + strconv.Itoa(index)}).Inc()

    } else {
        tempTs := *ts
        TsQueue.MergeProducer(&tempTs, index)
        mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "without-aggregate", "queueIndex":"queue-" + strconv.Itoa(index)}).Inc()
    }

    return nil
}