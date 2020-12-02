package model

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	_ "github.com/sirupsen/logrus"
)

type Client struct {
	urls   []string
	client *http.Client
}

type ErrorInto struct {
	Err error
}

var client *Client

func InitClient() {
	client = NewClient(Conf.remoteUrls)
}

func NewClient(urls []string) *Client {
	var transport *http.Transport
	transport = &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     time.Duration(10) * time.Second,
		MaxIdleConnsPerHost: 250, // 使用长连接，需要调高该值
	}
	return &Client{
		urls: urls,
		client: &http.Client{
			Transport: transport,
		},
	}
}

func (c *Client) Write(samples []*prompb.TimeSeries, index int) {
	var buf []byte
	samplesLen := len(samples)
	req, _, _ := buildWriteRequest(samples, buf)
	var wg sync.WaitGroup
	wg.Add(len(c.urls))
	for i, url := range c.urls {
		go func(urlIndex int, url string) {
			httpReq, err := http.NewRequest("POST", url, bytes.NewReader(req))
			if err != nil {
				sendMetricsNumCounter.With(prometheus.Labels{"status": err.Error(), "urlIndex": strconv.Itoa(urlIndex), "queueIndex": "queue-" + strconv.Itoa(index)}).Add(float64(samplesLen))
				sendRequestNumCounter.With(prometheus.Labels{"status": err.Error(), "urlIndex": strconv.Itoa(urlIndex), "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
				return
			}
			httpReq.Header.Add("Content-Encoding", "snappy")
			httpReq.Header.Set("Content-Type", "application/x-protobuf")
			httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

			begin := time.Now().UnixNano()
			httpResp, err := c.client.Do(httpReq)
			writeRequestTime.WithLabelValues("queue-"+strconv.Itoa(index), strconv.Itoa(urlIndex), strconv.Itoa(httpResp.StatusCode)).Observe(float64((time.Now().UnixNano() - begin) / 1e6))
			if err != nil {
				sendMetricsNumCounter.With(prometheus.Labels{"status": err.Error(), "urlIndex": strconv.Itoa(urlIndex), "queueIndex": "queue-" + strconv.Itoa(index)}).Add(float64(samplesLen))
				sendRequestNumCounter.With(prometheus.Labels{"status": err.Error(), "urlIndex": strconv.Itoa(urlIndex), "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
				return
			}
			defer func() {
				io.Copy(ioutil.Discard, httpResp.Body)
				httpResp.Body.Close()
				wg.Done()
			}()
			sendMetricsNumCounter.With(prometheus.Labels{"status": httpResp.Status, "urlIndex": strconv.Itoa(urlIndex), "queueIndex": "queue-" + strconv.Itoa(index)}).Add(float64(samplesLen))
			sendRequestNumCounter.With(prometheus.Labels{"status": httpResp.Status, "urlIndex": strconv.Itoa(urlIndex), "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
			return
		}(i, url)
	}
	wg.Wait()
}

func buildWriteRequest(samples []*prompb.TimeSeries, buf []byte) ([]byte, int64, error) {
	var highest int64
	for _, ts := range samples {
		// At the moment we only ever append a TimeSeries with a single sample in it.
		if ts.Samples[0].Timestamp > highest {
			highest = ts.Samples[0].Timestamp
		}
	}
	req := &prompb.WriteRequest{
		Timeseries: samples,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return nil, highest, err
	}

	// snappy uses len() to see if it needs to allocate a new slice. Make the
	// buffer as long as possible.
	if buf != nil {
		buf = buf[0:cap(buf)]
	}
	compressed := snappy.Encode(buf, data)
	return compressed, highest, nil
}

func (e *ErrorInto) Error() string {
	return fmt.Sprintf(e.Err.Error())
}
