package configs

import (
	"time"

	"github.com/sirupsen/logrus"
)

// general time
const (
	//
	DayTime  = time.Hour * 24
	WeekTime = DayTime * 7
	// format
	TimeFormatString = "2006-01-02 15:04:05"
)

// database
type DBConfigs struct {
	Driver   string
	Database string
	Host     string
	Port     string
	User     string
	Password string
}

var DBInfos *DBConfigs = &DBConfigs{
	Driver:   "mysql",
	Database: "seckill",
	Host:     "your database server uri",
	Port:     "3306",
	User:     "tq",
	Password: "your password",
}

// time
const (
	DateTimeFormatStr = "2006-01-02 15:04:05"
	DateFormatStr     = "2006-01-02"
	TimeFormatStr     = "15:04:05"
)

// etcd
var EtcdCfgs = &struct {
	Endpoints   []string
	DialTimeout time.Duration
}{
	Endpoints:   []string{"your etcd uri"},
	DialTimeout: 5 * time.Second,
}

const (
	EtcdSecKeyPrefix  = "seckill"
	EtcdSecProductKey = "products"
)

// log
var Logs = logrus.New()

const (
	LogDirectoryPath = "./logs"
	LogFileName      = "seckill"
	LogFileSuffix    = ".log"
)
