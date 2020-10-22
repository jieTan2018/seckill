package services

import (
	cfg "seckill/layer/configs"
	"sync"
	"time"
)

var (
	logs             = cfg.Logs
	Proxy2LayerRedis = cfg.RedisProxy2LayerCfgs
	Layer2ProxyRedis = cfg.RedisLayer2ProxyCfgs
	Read2HandleChan  chan *secRequest
	Handle2WriteChan chan *secResponse
	waitGroup        sync.WaitGroup
)

const (
	ErrServiceBusy       = 1001
	ErrSecKillSucc       = 1002
	ErrNotFoundProduct   = 1003
	ErrSaleOut           = 1004
	ErrRetry             = 1005
	ErrAlreadyBuy        = 1006
	ProductStatusSaleOut = 2001
)

type secRequest struct {
	ProductId    int       `form:"product_id" binding:"required"`
	Source       string    `form:"source" binding:"required"`
	AuthCode     string    `form:"auth_code" binding:"required"`
	SecTime      string    `form:"sec_time" binding:"required"`
	Nance        int       `form:"nance" binding:"required"` // 随机数
	UserId       int       `form:"-"`
	UserAuthSign string    `form:"-"`
	AccessTime   time.Time `form:"-"`
	ClientAddr   string    `form:"-"` // 防止某个ip攻击
	ClientRefer  string    `form:"-"` // 访问的来源, 甄别是不是自己网站的请求
}

type secResponse struct {
	ProductId int
	UserId    int
	Token     string
	TokenTime int64
	Code      int
}

func InitLayer() error {
	if err := initSecLayer(); err != nil {
		logs.Errorf("init seckill failed! err:%v", err)
		return err
	}
	logs.Debug("init sec layer succ!")
	if err := serviceRun(); err != nil {
		logs.Errorf("layer service run return! err:%v", err)
		return err
	}
	logs.Debug("init service run succ!")
	logs.Info("layer service run exited.")
	return nil
}

// 初始化秒杀逻辑
func initSecLayer() error {
	// 要用到redis
	if err := initRedis(); err != nil {
		logs.Errorf("init redis failed! err:%v", err)
		return err
	}
	// 要用到etcd
	if err := initEtcd(); err != nil {
		logs.Errorf("init etcd failed! err:%v", err)
		return err
	}
	// 从etcd加载商品
	if err := loadProductFromEtcd(); err != nil {
		logs.Errorf("load product from etcd failed! err:%v", err)
		return err
	}
	// 初始化channel, 存放待处理的请求
	Read2HandleChan = make(chan *secRequest, cfg.Read2HandleChanSize)
	Handle2WriteChan = make(chan *secResponse, cfg.Read2HandleChanSize)
	return nil
}

// // 运行逻辑业务 => 放到service.go了
// func serviceRun() error {
// 	return nil
// }
