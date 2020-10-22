package configs

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// // ** 逻辑层的配置 **
// system
const (
	SecKillTokenPasswd = "hu6CSWrf08^akrWo@UyDrONw69LcW4sqQm$^OhHK!J2U!WB@WnKFHCeYwUnnZl%Q" // 抢购成功后,加购物车的秘钥 => 优化: 可为每个商品添加单独的秘钥
	//
	DayTime  = time.Hour * 24
	WeekTime = DayTime * 7
	// format
	DateFormatString = "2006-01-02"
	TimeFormatString = "2006-01-02 15:04:05"
)

//etcd
const (
	ProductStatusNormal   = iota
	ProductStatusSaleOut  // 正常卖光
	ProductStatusForceOut // 强制卖光
	EtcdSecKeyPrefix      = "seckill"
	EtcdSecProductKey     = "products"
	EtcdConnTimeout       = 5
)

type SecInfoConf struct {
	ProductId int
	StartTime int64
	EndTime   int64
	Status    int
	Total     int
	Left      int
}

var EtcdCfgs = &struct {
	Endpoints   []string
	DialTimeout time.Duration
}{
	Endpoints:   []string{"your etcd uri"},
	DialTimeout: 5 * time.Second,
}

var (
	// secProductInfos    []SecInfoConf // 秒杀的商品配置
	SecProductInfosMap = make(map[int]*SecInfoConf, 1024)
	RWSecProductLock   sync.RWMutex // 明确这个锁就是加载secProductInfosMap是使用的
)

//redis => 后面组好放到etcd中, 保证一致性
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
	RedisProxy2LayerCfgs = &RedisBaseCfgs{ // 业务层的redis配置
		Network:     "tcp",
		Uri:         "your redis server uri",
		User:        "",
		Password:    "your password",
		Database:    "1",
		MaxIdle:     64,
		MaxActive:   0,
		IdleTimeout: 300 * time.Second, // 300ns*1000 => 300s
	}
	RedisLayer2ProxyCfgs = &RedisBaseCfgs{
		Network:     "tcp",
		Uri:         "your redis server uri",
		User:        "",
		Password:    "your password",
		Database:    "1",
		MaxIdle:     64,
		MaxActive:   0,
		IdleTimeout: 300 * time.Second, // 300ns*1000 => 300s
	}
)

const (
	WriteLayer2ProxyGoroutineNum       = 16
	ReadLayer2ProxyGoroutineNum        = 16
	HandleUserGoroutineNum             = 16
	Read2HandleChanSize                = 100000
	MaxRequestWaitTimeout        int64 = 30
	Handle2WriteSize                   = 100000
	Send2WriteChanTimeout              = 100 // 毫秒
	Send2HandleChanTimeout             = 100 //毫秒
	RedisLayer2ProxyQueueName          = "sec_queue"
)

const (
	WriteProxy2LayerGoroutineNum = 16
	ReadProxy2LayerGoroutineNum  = 16
	RedisProxy2LayerQueueName    = "redis_poxy2layer_queue_name"
)

// log
var Logs = logrus.New()

const (
	LogDirectoryPath = "./logs"
	LogFileName      = "seckill"
	LogFileSuffix    = ".log"
)
