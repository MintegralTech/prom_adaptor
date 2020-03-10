package controller

import (
    "io/ioutil"
    "prom_adpator/model"
    "github.com/gin-gonic/gin"
    "github.com/prometheus/prometheus/prompb"
    "github.com/golang/snappy"
    "github.com/gogo/protobuf/proto"
)

func Receive(c *gin.Context) error {
    compressed, err := ioutil.ReadAll(c.Request.Body)
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
    model.TsQueue.Producer(&req)
    return nil
}
