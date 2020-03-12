package model

import (
    "io"
    "io/ioutil"
    "bytes"
    "net/http"

    "github.com/golang/snappy"
    "github.com/sirupsen/logrus"
    "github.com/gogo/protobuf/proto"
    "github.com/prometheus/prometheus/prompb"
)

type Client struct {
    url    string
    client *http.Client
}

var client *Client

func InitClient() {
    client = NewClient(Conf.remoteUrl)
}

func NewClient(url string) *Client {
    return &Client{
        url:    url,
        client: &http.Client{},
    }
}

func (c *Client) Write(samples []*prompb.TimeSeries) error {
    for _, ts := range samples {
        ReqLog.WithFields(logrus.Fields{"metric": GetMetric(ts)+GetSample(ts)}).Info("client send")
    }
    var buf []byte
    req, _, err := buildWriteRequest(samples, buf)
    httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(req))
    if err != nil {
        return err
    }
    httpReq.Header.Add("Content-Encoding", "snappy")
    httpReq.Header.Set("Content-Type", "application/x-protobuf")
    httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

    httpResp, err := c.client.Do(httpReq)
    if err != nil {
        return err
    }
    defer func() {
        io.Copy(ioutil.Discard, httpResp.Body)
        httpResp.Body.Close()
    }()
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

