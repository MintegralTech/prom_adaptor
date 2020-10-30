package model

import (
    "fmt"
)

func init() {
    InitMonitor()
    fmt.Println("init monitor")
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


    InitRequestController()
    fmt.Println("init request controller")
    go MonitorRequestBuffer()
    fmt.Println("init MonitorRequestBuffer")
    go MonitorRequestBufferTimer()
    fmt.Println("init MonitorRequestBufferTimer")


    for i := 0; i < Conf.workersNum; i++{
        go TsQueue.MergeQueueConsumer(i)
        go TsQueue.SendConsumer(i)
    }

    go AggregatorCacheCleaner()
    fmt.Println("init AggregatorCacheCleaner")
    go GaugeMonitor()
    fmt.Println("init all done")
}
