package model

import (
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	_ "github.com/sirupsen/logrus"
)

type (
	cache struct {
		data map[string]*prompb.Sample
	}
	// 存放聚合后的数据
	block struct {
		data map[string]*prompb.TimeSeries
	}
	// 聚合器，与job绑定
	Aggregator struct {
		mtx       sync.Mutex
		prevCache *cache
		pack      *block
	}
	Aggregators struct {
		jobNum     int
		aggregator map[string]*Aggregator
	}
)

const (
	INSTANCE = "instance"
	//INSTANCE = "debug"
)

var Collection *Aggregators

func InitCollection() {
	Collection = NewAggregators()
}

func NewAggregator(jobName string) *Aggregator {
	return &Aggregator{
		prevCache: &cache{data: make(map[string]*prompb.Sample)},
		pack:      &block{data: make(map[string]*prompb.TimeSeries)},
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

func (collection *Aggregators) updatePrevCache(prevCache *cache, metric string, sample *prompb.Sample, jobName string, index int) float64 {
	incVal := sample.Value
	if prevSample, ok := prevCache.data[metric]; ok {
		curVal, prevVal := sample.Value, prevSample.Value
		if prevSample.Timestamp > sample.Timestamp {
			delayedMertricCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
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

func (collection *Aggregators) updatePack(jobName string, ts *prompb.TimeSeries, incVal float64) (string, bool) {
	isUpdate := false
	noInstTs := DeleteLable(ts, INSTANCE)
	metric, _ := GetMetric(noInstTs)
	collection.aggregator[jobName].mtx.Lock()
	defer collection.aggregator[jobName].mtx.Unlock()
	pack := collection.aggregator[jobName].pack
	if _, ok := pack.data[metric]; !ok {
		pack.data[metric] = noInstTs
	} else {
		if pack.data[metric].Samples[0].Value+incVal >= 0 {
			pack.data[metric].Samples[0].Value += incVal
		} else {
			pack.data[metric].Samples[0].Value = incVal
		}
		if pack.data[metric].Samples[0].Timestamp < ts.Samples[0].Timestamp {
			pack.data[metric].Samples[0].Timestamp = ts.Samples[0].Timestamp
			isUpdate = true
		}
	}
	cacheDataLengthGauge.With(prometheus.Labels{"jobname": jobName, "type": "pack"}).Set(float64(len(pack.data)))
	return metric, isUpdate
}

func (collection *Aggregators) PutMergedMetricsIntoSendQueue(index int) {
	for jobName, val := range TsQueue.mergedLists[index].list {
		for metric, _ := range val {
			collection.aggregator[jobName].mtx.Lock()
			tempTs := *(collection.aggregator[jobName].pack.data[metric])
			collection.aggregator[jobName].mtx.Unlock()
			TsQueue.SendProducer(&tempTs, index)
		}
		dataInMergedWinSizeGauge.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(index)}).Set(float64(len(val)))
	}
	TsQueue.mergedLists[index].ClearList() //清空记录
}

func (collection *Aggregators) MergeMetric(ts *prompb.TimeSeries, jobName string, metric string, index int) {
	res := TsQueue.mergeTimeWindow[index].CompareWithMergeTimeWindow(ts)
	if res == GreaterCurMergeWindow {
		collection.PutMergedMetricsIntoSendQueue(index)
		TsQueue.mergeTimeWindow[index].SetMergeTimeWindow(ts)
	} else if res == LessCurMergeWindow {
		// TODO 写metrics记录迟到的数据个数
		//delayedMertricCounter.With(prometheus.Labels{"jobname": jobName, "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
	}
	collection.aggregator[jobName].mtx.Lock()
	cache := collection.aggregator[jobName]
	incVal := collection.updatePrevCache(cache.prevCache, metric, &ts.Samples[0], jobName, index)
	cacheDataLengthGauge.With(prometheus.Labels{"jobname": jobName, "type": "prev"}).Set(float64(len(cache.prevCache.data)))
	collection.aggregator[jobName].mtx.Unlock()
	mergedMt, isUpdate := collection.updatePack(jobName, ts, incVal)
	if isUpdate {
		// 记录当前聚合窗口中已经聚合的指标
		if TsQueue.mergedLists[index].list[jobName] == nil {
			TsQueue.mergedLists[index].list[jobName] = make(map[string]bool)
		}
		TsQueue.mergedLists[index].list[jobName][mergedMt] = true
	}
}

//定时清除聚合缓存中大于配置时间的数据，避免内存持续增长
func AggregatorCacheCleaner() {
	t := time.NewTicker(time.Hour * time.Duration(Conf.GetCleanerTime()))
	keepTime := int64(Conf.GetCleanerTime() * 3600 * 1e9) //纳秒
	for {
		<-t.C
		curTime := int64(time.Now().Nanosecond())
		for jobName, agg := range Collection.aggregator {
			agg.mtx.Lock()
			ch := make(chan struct{})
			go func() {
				for mtStr, ts := range agg.prevCache.data {
					if curTime > ts.Timestamp && curTime-ts.Timestamp >= keepTime {
						delete(agg.prevCache.data, mtStr)
						cacheDeletedNumGauge.With(prometheus.Labels{"jobname": jobName, "type": "prev"}).Inc()
					}
				}
				ch <- struct{}{}
			}()

			for mtStr, ts := range agg.pack.data {
				if curTime > ts.Samples[0].Timestamp && curTime-ts.Samples[0].Timestamp >= keepTime {
					delete(agg.prevCache.data, mtStr)
					cacheDeletedNumGauge.With(prometheus.Labels{"jobname": jobName, "type": "pack"}).Inc()
				}
			}
			<-ch
			agg.mtx.Unlock()
		}
	}
}
