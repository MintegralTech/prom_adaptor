package main

import (
    "github.com/MintegralTech/prom_adaptor/internal/controller"
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

func main() {
    router := gin.New()
    router.Use(gin.Recovery())
    router.NoMethod(controller.HandleNotFound)
    router.NoRoute(controller.HandleNotFound)
    router.GET("/helloworld", helloworld)
    router.GET("/metrics", gin.WrapH(promhttp.Handler()))
    router.POST("/receive", controller.Wrapper(controller.Receive))
    router.Run(":1234")
}

func helloworld(c *gin.Context) {
    c.String(http.StatusOK, "hello world")
    //AccLog.WithFields(logrus.Fields{"request": c.Request.PostForm, "url": c.Request.URL}).Info("access")
}
