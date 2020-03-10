package model

import (
    "time"

    rotate "github.com/lestrrat-go/file-rotatelogs"
    "github.com/sirupsen/logrus"
)

//NewLog generate logger
func NewLog(file string, level logrus.Level, enableCaller bool) *logrus.Logger {
    log := logrus.New()
    logf, err := rotate.New(
        file+".%Y-%m-%d-%H",
        rotate.WithMaxAge(5*24*time.Hour),
        rotate.WithRotationTime(time.Hour),
    )
    if err != nil {
        panic(err)
    }
    // Log as JSON instead of the default ASCII formatter.
    log.SetFormatter(&logrus.JSONFormatter{})

    // Output to stdout instead of the default stderr
    // Can be any io.Writer, see below for File example
    log.SetOutput(logf)

    // Only log the warning severity or above.
    log.SetLevel(level)

    //func info control
    log.SetReportCaller(enableCaller)

    return log
}
