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

    for i := 0; i < Conf.queuesNum; i++{
        go TsQueue.RequestConsumer(i)
        go TsQueue.MergeConsumer(i)
        go Collection.MonitorPack(i)
    }

    go GaugeMonitor()
    fmt.Println("init all done")
}
