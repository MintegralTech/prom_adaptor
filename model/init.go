package model

import (
    "fmt"
)

func init() {
    InitConfig()
    fmt.Println(Conf)
    InitLog()
    fmt.Println("init log")
    InitCollection()
    fmt.Println("init collection")
    InitQueue()
    fmt.Println("init queue")
    InitClient()
    fmt.Println("init client")
    InitMonitor()
    fmt.Println("init monitor")

    go TsQueue.RequestConsumer()
    go TsQueue.MergeConsumer()
    go Collection.MonitorPack()
    go GaugeMonitor()
}
