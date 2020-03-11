package model

import (
    "fmt"
)

func init() {
    InitConfig()
    fmt.Println(Conf)
    InitLog()
    fmt.Println("initlog")
    InitCollection()
    fmt.Println("initcollection")
    InitQueue()
    fmt.Println("initqueue")
    InitClient()
    fmt.Println("initclient")

    go TsQueue.RequestConsumer()
    go TsQueue.MergeConsumer()
    go Collection.MonitorPack()
}
