package controller

import (
    . "github.com/MintegralTech/prom_adaptor/model"
    "github.com/gin-gonic/gin"
    "github.com/gogo/protobuf/proto"
    "github.com/golang/snappy"
    "github.com/sirupsen/logrus"
    "github.com/prometheus/prometheus/prompb"
    "io/ioutil"
)

func Receive(c *gin.Context) error {
    compressed, err := ioutil.ReadAll(c.Request.Body)
    AccLog.WithFields(logrus.Fields{"request": c.Request.PostForm, "url": c.Request.URL}).Info("access")
    if err != nil {
        return ServerError()
    }
    reqBuf, err := snappy.Decode(nil, compressed)
    if err != nil {
        return BadRequest()
    }
    var req prompb.WriteRequest
    if err := proto.Unmarshal(reqBuf, &req); err != nil {
        return BadRequest()
    }
    RunLog.WithFields(logrus.Fields{"wr length": len(req.Timeseries)}).Info(req)
    TsQueue.Producer(&req)
    return nil
}
