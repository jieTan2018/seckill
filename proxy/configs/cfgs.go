package configs

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// system
const (
	CookieSecretKey = "lFvdjdEvX*1!b1OPBxup#^H%aDC29q5d"
	// time
	DayTime  = time.Hour * 24
	WeekTime = DayTime * 7
	// format
	DateFormatString = "2006-01-02"
	TimeFormatString = "2006-01-02 15:04:05"
)

// redis
type RedisBaseCfgs struct {
	Network     string
	Uri, User   string
	Password    string
	Database    string
	MaxIdle     int
	MaxActive   int
	IdleTimeout time.Duration // 值表示 ns => 纳秒
}

var (
	RedisCfgs = &RedisBaseCfgs{
		Network:     "tcp",
		Uri:         "your uri",
		User:        "",
		Password:    "your password",
		Database:    "1",
		MaxIdle:     64,
		MaxActive:   0,
		IdleTimeout: 300 * time.Second, // 300ns*1000 => 300s
	}
	RedisProxy2LayerCfgs = &RedisBaseCfgs{ // 业务层的redis配置
		Network:     "tcp",
		Uri:         "your uri",
		User:        "",
		Password:    "your password",
		Database:    "1",
		MaxIdle:     64,
		MaxActive:   0,
		IdleTimeout: 300 * time.Second, // 300ns*1000 => 300s
	}
	ReferWhiteList = [2]string{"127.0.0.1", "localhost"} // 访问白名单
	IpWhiteList    map[string]bool
	IdBlackList    map[int]bool
)

const (
	WriteProxy2LayerGoroutineNum = 16
	ReadProxy2LayerGoroutineNum  = 16
	RedisProxy2LayerQueueName    = "redis_poxy2layer_queue_name"
)

// ETCD
var EtcdCfgs = &struct {
	Endpoints   []string
	DialTimeout time.Duration
}{
	Endpoints:   []string{"your etcd uri"},
	DialTimeout: 5 * time.Second,
}

const (
	ProductStatusNormal   = iota
	ProductStatusSaleOut       // 正常卖光
	ProductStatusForceOut      // 强制卖光
	UserSecAccessLimit    = 1  // x次/秒 超过这个频率的访问用户将被加入黑名单
	IpSecAccessLimit      = 10 // 某ip下, 一秒最多有x个用户访问
	EtcdSecKeyPrefix      = "seckill"
	EtcdSecProductKey     = "products"
)

type SecInfoConf struct {
	ProductId int
	StartTime int64
	EndTime   int64
	Status    int
	Total     int
	Left      int
}

var (
	SecProductInfosMap = make(map[int]*SecInfoConf, 1024)
	RWSecProductLock   sync.RWMutex // 明确这个锁就是加载secProductInfosMap是使用的
)

// log
var Logs = logrus.New()

const (
	LogDirectoryPath = "./logs"
	LogFileName      = "seckill"
	LogFileSuffix    = ".log"
)
