package model

import (
    "github.com/spf13/viper"
    "github.com/sirupsen/logrus"
)

type Config struct {
    remoteUrl   string
    buffer      int
    windows     []int
    jobNames    []string

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

func InitConfig() {
    Conf = NewConfig()
}

func NewConfig() *Config {
    config := &Config{}
    v := setViper(defaultConfigPath, defaultConfigName, defaultConfigType)
    config.buffer = v.GetInt("runtime.buffer")
    config.windows = v.GetIntSlice("data.windows")
    config.jobNames = v.GetStringSlice("data.whitelist")
    config.remoteUrl = v.GetString("data.remoteUrl")

    //log
    config.logPath = v.GetString("log.logPath")
    config.runLogLevel = logrus.Level(v.GetInt("log.runLogLevel"))
    config.accLogLevel = logrus.Level(v.GetInt("log.accLogLevel"))
    config.reqLogLevel = logrus.Level(v.GetInt("log.reqLogLevel"))

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

