package model

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	_ "github.com/sirupsen/logrus"
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
		jobNum:     len(Conf.jobNames),
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
		pack.data[metric] = &TimeSeries{
			ts:   noInstTs,
			flag: true,
		}
	} else {
		if pack.data[metric].ts.Samples[0].Value+incVal >= 0 {
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

func (collection *Aggregators) PutIntoMergeQueue(index int) {
	ch := make(chan struct{}, collection.jobNum)
	for i := 0; i < collection.jobNum; i++ {
		ch <- struct{}{}
	}
	for jobIndex := 0; ; jobIndex++ {
		<-ch
		jobName := Conf.jobNames[jobIndex]
		window := Conf.windows[jobIndex]
		go func() {
			//TODO 目前是基于当前时间进行聚合，存在聚合服务重启时，存在突刺的问题，后续优化为基于第一个指标时间为基准，计算每个windows中的指标。
			t := time.NewTicker(time.Second * time.Duration(window))
			for {
				<-t.C
				collection.aggregator[jobName].mtx.Lock()
				pack := collection.aggregator[jobName].pack
				for _, val := range pack.data {
					packStatusCounter.With(prometheus.Labels{"jobname": jobName, "flag": strconv.FormatBool(val.flag)}).Add(1)
					if val.flag {
						val.flag = false
						tempTs := prompb.TimeSeries{}
						DeepCopy(val.ts, &tempTs)
						TsQueue.MergeProducer(&tempTs, index)
					}
				}
				collection.aggregator[jobName].mtx.Unlock()
			}
		}()
	}
}

func (collection *Aggregators) MergeMetric(ts *prompb.TimeSeries, index int) error {
	metric := GetMetric(ts)
	jobName, _, err := GetJobName(metric)
	if err != nil {
		return err
	}
	//RunLog.WithFields(logrus.Fields{"jobName": jobName}).Info("jobName")
	if len(ts.Samples) != 1 {
		return errors.New(fmt.Sprintf("error sample size[%d]", len(ts.Samples)))
	}
	//带ip的label直接放到请求队列, 不做聚合
	for _, l := range ts.Labels {
		if strings.ToLower(l.Name) == "ip" {
			tempTs := prompb.TimeSeries{}
			DeepCopy(ts, &tempTs)
			TsQueue.MergeProducer(&tempTs, index)
			mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "without-aggregate", "queueIndex": "queue-" + strconv.Itoa(index)}).Add(1)
			return nil
		}
	}
	if math.IsNaN(ts.Samples[0].Value) {
		//RunLog.WithFields(logrus.Fields{"ts": metric}).Info("NaN")
		ts.Samples[0].Value = 0
	}
	if cache, ok := collection.aggregator[jobName]; ok {
		collection.aggregator[jobName].mtx.Lock()
		incVal := collection.updatePrevCache(cache.prevCache, metric, &ts.Samples[0])
		collection.aggregator[jobName].mtx.Unlock()
		collection.updatePack(jobName, ts, incVal)
		mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "aggregate", "queueIndex": "queue-" + strconv.Itoa(index)}).Add(1)
	} else {
		tempTs := prompb.TimeSeries{}
		DeepCopy(ts, &tempTs)
		TsQueue.MergeProducer(&tempTs, index)
		mergeMetricCounter.With(prometheus.Labels{"jobname": jobName, "type": "without-aggregate", "queueIndex": "queue-" + strconv.Itoa(index)}).Add(1)
	}
	return nil
}

func DeepCopy(src *prompb.TimeSeries, dest *prompb.TimeSeries) {
	dest.Samples = make([]prompb.Sample, len(src.Samples))
	dest.Labels = make([]*prompb.Label, len(src.Labels))
	copy(dest.Samples, src.Samples)
	for i, _ := range src.Labels {
		temp := *(src.Labels[i])
		dest.Labels[i] = &temp
	}
}
