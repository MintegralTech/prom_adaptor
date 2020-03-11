package model

import (
	"time"
    "net/url"
    "net/http"

    "github.com/go-kit/kit/log"
    _ "github.com/go-kit/kit/log/level"
    "github.com/prometheus/common/model"
    _ "github.com/prometheus/prometheus/prompb"
)

type Client struct {
    url     string
    client *http.Client
}

func NewClient(logger log.Logger, url string, timeout time.Duration) *Client {
    return &Client{
        url:     url,
    }
}

func (c *Client) Write(wreq *prompb.WriteRequest) error {
    

}


