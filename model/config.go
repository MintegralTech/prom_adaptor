package model

import (
    "os"
    "path/filepath"

    "github.com/spf13/viper"
    "github.com/sirupsen/logrus"
)

type Config struct {
    remoteUrl   string
    buffer      int
    window      int
    jobNames    []string
    whitelist   []string

    //log
    logPath     string
    runLogLevel logrus.Level
    accLogLevel logrus.Level
    reqLogLevel logrus.Level

}

var Conf *Config

var (
    defaultConfigPath = "./conf"
    defaultConfigName = "adapter"
    defaultConfigType = "yaml"
)

var (
    RunLog *logrus.Logger
    AccLog *logrus.Logger
    ReqLog *logrus.Logger
)

var Collection *Aggregators

var TsQueue *TimeSeriesQueue

func init() {
    Conf = NewConfig()
    initLog()
    initQueue()
    initCollection()
}

func initCollection() {
	Collection = NewAggregators(Conf.jobNames)
}

func initQueue() {
    buffer := 100000
    if Conf.buffer >= buffer {
        buffer = Conf.buffer
    }
    TsQueue = NewTimeSeriesQueue(buffer)
}

func initLog() {
    home, _ := os.Getwd()
    RunLog = NewLog(filepath.Join(home, Conf.logPath, "runtime"), Conf.runLogLevel, false)
    AccLog = NewLog(filepath.Join(home, Conf.logPath, "access"), Conf.accLogLevel, false)
    ReqLog = NewLog(filepath.Join(home, Conf.logPath, "request"), Conf.reqLogLevel, false)
}

func NewConfig() *Config {
    config := &Config{}
    v := setViper(defaultConfigPath, defaultConfigName, defaultConfigType)
    config.buffer = v.GetInt("runtime.buffer")
    config.window = v.GetInt("runtime.window")
    config.remoteUrl = v.GetString("data.remoteUrl")
    config.jobNames = v.GetStringSlice("data.jobNames")

    //log
    config.logPath = v.GetString("log.logPath")
    config.runLogLevel = logrus.Level(v.GetInt("log.runLogLevel"))
    config.accLogLevel = logrus.Level(v.GetInt("log.accLogLevel"))
    config.reqLogLevel = logrus.Level(v.GetInt("log.reqLogLevel"))

    config.whitelist = v.GetStringSlice("filter.whitelist")
    return config
}

func setViper(cfgPath, cfgName, cfgType string) *viper.Viper {
    v := viper.New()
    v.AddConfigPath(cfgPath)
    v.SetConfigName(cfgName)
    v.SetConfigType(cfgType)
    if err := v.ReadInConfig(); err != nil {
        panic(err)
    }
    return v
}

