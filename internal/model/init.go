package model

import (
    "fmt"
)

func init() {
    InitConfig()
    fmt.Println(Conf)
    InitLog()
    InitCollection()
    InitQueue()
    InitClient()
    InitMonitor()

    go TsQueue.RequestConsumer()
    go TsQueue.MergeConsumer()
    go Collection.PutIntoMergeQueue()
    go GaugeMonitor()
}
