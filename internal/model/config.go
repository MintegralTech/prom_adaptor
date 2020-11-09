package model

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	requestBuffSize         int // requestBuffSize = batchSize * 10 + 10
	batchSize               int //每批次处理的数据量
	queueSize               int // merge send queue的容量大小
	shard                   int
	mode                    string
	remoteUrl               string
	windows                 int //聚合窗口大小 单位：秒
	jobNames                []string
	workersNum              int   //队列的数量
	freshRequestQueuePeriod int64 //消费requestqueue队列中数据的时间周期， 单位秒
	cleanerTime             int   //自动清除clearTime 小时未更新的缓存数据

	//log
	logPath     string
	runLogLevel logrus.Level
	accLogLevel logrus.Level
	reqLogLevel logrus.Level
}

var Conf *Config

var (
	defaultConfigPath = "./conf"
	defaultConfigName = "adaptor"
	defaultConfigType = "yaml"
)

const (
	defaultRequestBuffSize         = 1000 //默认队列长度
	defaultWorkersNum              = 2    //默认队列个数
	defaultFreshRequestQueuePeriod = 20
	defaultCleanerTime             = 24 //默认 删除24小时内未更新的缓存数据, 保持的最短时间为3小时
	defaultBatchSize               = 10
)

func InitConfig() {
	Conf = NewConfig()
}

func NewConfig() *Config {
	config := &Config{}
	v := setViper(defaultConfigPath, defaultConfigName, defaultConfigType)
	config.shard = v.GetInt("runtime.shard")
	config.mode = v.GetString("runtime.mode")
	config.windows = v.GetInt("data.windows") * 1000 //转为毫秒
	config.jobNames = v.GetStringSlice("data.whitelist")
	config.remoteUrl = v.GetString("data.remoteUrl")
	config.workersNum = v.GetInt("runtime.workersNum")
	config.freshRequestQueuePeriod = v.GetInt64("runtime.freshRequestQueuePeriod")
	config.cleanerTime = v.GetInt("runtime.cleanerTime")
	config.queueSize = v.GetInt("runtime.queueSize")
	config.batchSize = v.GetInt("runtime.batchSize")
	config.requestBuffSize = 10*config.batchSize + 10
	//log
	config.logPath = v.GetString("log.logPath")
	config.runLogLevel = logrus.Level(v.GetInt("log.runLogLevel"))
	config.accLogLevel = logrus.Level(v.GetInt("log.accLogLevel"))
	config.reqLogLevel = logrus.Level(v.GetInt("log.reqLogLevel"))

	for i, v := range config.jobNames {
		config.jobNames[i] = strings.ToLower(v)
	}

	if config.requestBuffSize < defaultRequestBuffSize {
		config.requestBuffSize = defaultRequestBuffSize
	}

	if config.workersNum < defaultWorkersNum {
		config.workersNum = defaultWorkersNum
	}

	if config.freshRequestQueuePeriod < defaultFreshRequestQueuePeriod {
		config.freshRequestQueuePeriod = defaultFreshRequestQueuePeriod
	}

	if config.cleanerTime < 3 {
		config.cleanerTime = defaultCleanerTime
	}

	if config.batchSize < defaultBatchSize {
		config.batchSize = defaultBatchSize
	}
	requestBufferLengthGauge.With(prometheus.Labels{"type": "rqtLength"}).Set(float64(config.requestBuffSize))
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

func (c *Config) GetRequestBuffCapacity() int {
	return c.requestBuffSize
}

func (c *Config) GetShard() int {
	return c.shard
}

func (c *Config) GetWorkersNum() int {
	return c.workersNum
}

func (c *Config) GetFreshRequestQueuePeriod() int64 {
	return c.freshRequestQueuePeriod
}

func (c *Config) GetMode() string {
	return c.mode
}

func (c *Config) GetCleanerTime() int {
	return c.cleanerTime
}

func (c *Config) GetQueueCapacity() int {
	return c.queueSize
}

func (c *Config) GetBatchSize() int {
	return c.batchSize
}
