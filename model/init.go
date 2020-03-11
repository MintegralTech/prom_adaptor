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

    go TsQueue.RequestConsumer()
    go TsQueue.MergeConsumer()
    go Collection.MonitorPack()
}
