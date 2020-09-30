package model

import (
    "bytes"
    "github.com/sirupsen/logrus"
    "io"
    "io/ioutil"
    "net/http"
    "strconv"
    "time"

    "github.com/gogo/protobuf/proto"
    "github.com/golang/snappy"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/prometheus/prompb"
)

type Client struct {
    url       string
    client    *http.Client
}

var client *Client

func InitClient() {
    client = NewClient(Conf.remoteUrl)
}

func NewClient(url string) *Client {
    var transport *http.Transport
    transport = &http.Transport{
        MaxIdleConns:        10,
        IdleConnTimeout:     time.Duration(10) * time.Second,
        MaxIdleConnsPerHost: 250, // 使用长连接，需要调高该值
    }
    return &Client{
        url:    url,
        client: &http.Client{
            Transport: transport,
        },
    }
}

func (c *Client) Write(samples []*prompb.TimeSeries, index int) error {
    if Conf.mode == "debug"{
        for _, ts := range samples {
            ReqLog.WithFields(logrus.Fields{"metric": GetMetric(ts) + GetSample(ts)}).Info("client send")
        }
    }

    var buf []byte
    req, _, err := buildWriteRequest(samples, buf)
    httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(req))
    if err != nil {
        sendMetricsNumCounter.With(prometheus.Labels{"succ": "false", "queueIndex": "queue-" + strconv.Itoa(index)}).Add(float64(len(samples)))
        sendRequestNumCounter.With(prometheus.Labels{"succ": "false", "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
        return err
    }
    httpReq.Header.Add("Content-Encoding", "snappy")
    httpReq.Header.Set("Content-Type", "application/x-protobuf")
    httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

    httpResp, err := c.client.Do(httpReq)
    if err != nil {
        sendMetricsNumCounter.With(prometheus.Labels{"succ": "false", "queueIndex": "queue-" + strconv.Itoa(index)}).Add(float64(len(samples)))
        sendRequestNumCounter.With(prometheus.Labels{"succ": "false", "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
        return err
    }
    defer func() {
        io.Copy(ioutil.Discard, httpResp.Body)
        httpResp.Body.Close()
    }()
    sendMetricsNumCounter.With(prometheus.Labels{"succ": "true", "queueIndex": "queue-" + strconv.Itoa(index)}).Add(float64(len(samples)))
    sendRequestNumCounter.With(prometheus.Labels{"succ": "true", "queueIndex": "queue-" + strconv.Itoa(index)}).Inc()
    return nil
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
