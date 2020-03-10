package main

import (
    "net/http"
    _ "prom_adaptor/model"
    "prom_adaptor/controller"
    _ "github.com/sirupsen/logrus"
    "github.com/gin-gonic/gin"
)


func main() {
    router := gin.New()
    router.Use(gin.Recovery())
    router.NoMethod(controller.HandleNotFound)
    router.NoRoute(controller.HandleNotFound)
    router.GET("/helloworld", helloworld)
    router.POST("/receive", controller.Wrapper(controller.Receive))
    //metrics := router.Group("/metrics")
    //{
    //    receive.POST("/aggregate", controller.Wrapper(controller.Receive))
    //}
    router.Run(":1234")
}

func helloworld(c *gin.Context) {
    c.String(http.StatusOK, "hello world")
    //AccLog.WithFields(logrus.Fields{"request": c.Request.PostForm, "url": c.Request.URL}).Info("access")
}
