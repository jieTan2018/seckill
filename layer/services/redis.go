package services

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	cfg "seckill/layer/configs"
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	proxy2LayerRedisPool *redis.Pool
	layer2ProxyRedisPool *redis.Pool
)

func initRedisPool(rCfgs *cfg.RedisBaseCfgs) (*redis.Pool, error) {
	pool := &redis.Pool{
		MaxIdle:     rCfgs.MaxIdle,
		MaxActive:   rCfgs.MaxActive,
		IdleTimeout: rCfgs.IdleTimeout, // ns
		Dial: func() (redis.Conn, error) {
			coon, err := redis.Dial(rCfgs.Network, rCfgs.Uri)
			if err != nil { // 连接失败时
				panic(err.Error())
				return nil, err
			}
			// 用户认证
			if _, err := coon.Do("auth", rCfgs.Password); err != nil {
				return nil, err
			}
			return coon, nil
		},
	}
	// redis操作
	coon := pool.Get()
	defer coon.Close()
	if _, err := coon.Do("ping"); err != nil {
		logs.Errorf("ping redis failed, err:%v", err)
		return pool, err
	}
	return pool, nil
}

func initRedis() (err error) {
	proxy2LayerRedisPool, err = initRedisPool(cfg.RedisProxy2LayerCfgs)
	if err != nil {
		logs.Errorf("init proxy2LayerRedisPool failed! err:%v", err)
		return
	}
	layer2ProxyRedisPool, err = initRedisPool(cfg.RedisLayer2ProxyCfgs)
	if err != nil {
		logs.Errorf("init layer2ProxyRedisPool failed! err:%v", err)
		return
	}

	logs.Info("init redis succ.")
	return nil
}

func runProcess() (err error) { // 读取 => 处理 => 写入
	// 读取
	for i := 0; i < cfg.ReadLayer2ProxyGoroutineNum; i++ {
		waitGroup.Add(1)
		go handlerReader()
	}
	// 写入
	for i := 0; i < cfg.WriteLayer2ProxyGoroutineNum; i++ {
		waitGroup.Add(1)
		go handlerWriter()
	}
	// 处理
	for i := 0; i < cfg.HandleUserGoroutineNum; i++ {
		waitGroup.Add(1)
		go handlerUser()
	}
	logs.Debug("all process goroutine start!")
	waitGroup.Wait()
	logs.Debug("wait all goroutine exited!")
	return nil
}

func handlerReader() {
	logs.Debug("read goroutine running.")
	for {
		conn := proxy2LayerRedisPool.Get() // 若写在最外层, close之后, 就不能在获取连接了!
		for {                              // 复用上层的conn, 不用每次都获取连接
			data, err := redis.String(
				conn.Do("blpop", cfg.RedisProxy2LayerQueueName, 0)) // 0代表没有请求时,一直阻塞。因为是goroutine调用, 所以就不会占cpu了
			if err != nil { // 异常时, 再从上层获取连接
				conn.Close()
				break
			}

			logs.Debugf("pop from queue, data: %s", data)
			var req secRequest
			err = json.Unmarshal([]byte(data), &req)
			if err != nil {
				logs.Errorf("unmarshal to secrequest failed! err:%v", err)
				continue
			}
			// 丢弃过期请求。若c端请求后,最大等待时间是10s, 那么从队列中取出的,在10s前的req便可以丢掉了!
			if time.Now().Unix()-req.AccessTime.Unix() > cfg.MaxRequestWaitTimeout {
				logs.Warnf("req[%v] is expire!", req)
				continue
			}
			// 管道满了,怎么处理?!secRequest
			timer := time.NewTicker(time.Duration(cfg.Send2HandleChanTimeout) * time.Millisecond)
			select {
			case Read2HandleChan <- &req:
			case <-timer.C:
				logs.Warnf("send to handle chan timeout, req:%v", req)
				break
			}
		}
		conn.Close()
	}
}

func handlerWriter() {
	logs.Debug("write goroutine running.")

	for resp := range Handle2WriteChan {
		err := sendToRedis(resp)
		if err != nil {
			logs.Errorf("send to redis failed! err:%v, res:%v", err, resp)
			continue
		}
	}
}

func sendToRedis(resp *secResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		logs.Errorf("marshal failed! err:%v", err)
		return err
	}

	conn := layer2ProxyRedisPool.Get()
	_, err = redis.String(
		conn.Do("rpush", cfg.RedisLayer2ProxyQueueName, string(data)))
	if err != nil {
		logs.Warnf("rpush to redis failed! err:%v", err)
		return err
	}
	return nil
}

func handlerUser() {
	logs.Debug("handle goroutine running.")
	for req := range Read2HandleChan {
		logs.Debugf("befin process request:%v", req)
		resp, err := handleSeckill(req)
		if err != nil {
			logs.Warnf("process request %v failed, err:%v", err)
			resp = &secResponse{
				Code: ErrServiceBusy,
			}
		}
		// 管道满了,怎么处理?!secRequest
		timer := time.NewTicker(time.Duration(cfg.Send2WriteChanTimeout) * time.Millisecond)
		select {
		case Handle2WriteChan <- resp:
		case <-timer.C:
			logs.Warnf("send to write chan timeout, req:%v", req)
			break
		}
	}
}

func handleSeckill(req *secRequest) (resp *secResponse, err error) {
	// 是否售罄
	RWSecProductLock.RLock()
	product, ok := SecProductInfosMap[req.ProductId]
	RWSecProductLock.RUnlock()
	if !ok {
		logs.Errorf("not found product!")
		resp.Code = ErrNotFoundProduct
		return
	}
	if product.Status == ProductStatusSaleOut {
		resp.Code = ErrSaleOut
		return
	}
	// 是否超速
	now := time.Now().Unix()
	alreadySoldCount := product.secLimit.Check(now)
	if alreadySoldCount >= product.soldMaxLimit {
		resp.Code = ErrRetry
		return
	}
	// 是否已购买 => 针对每一件商品、每一个人
	HistoryMapLock.Lock()
	userHistory, ok := HistoryMap[req.UserId]
	if !ok {
		userHistory = &UserBuyHistory{
			history: make(map[int]int, 16),
		}
		HistoryMap[req.UserId] = userHistory
	}
	historyCount := userHistory.GetProductBuyCount(req.ProductId)
	HistoryMapLock.Unlock()
	if historyCount >= product.onePersonBuyLimit {
		resp.Code = ErrAlreadyBuy
		return
	}
	// 总数是否超限 => "超限"则设置为"售罄"状态
	curSoldCount := productCountMgr.Count(req.ProductId)
	if curSoldCount >= product.Total {
		resp.Code = ErrSaleOut
		product.Status = ProductStatusSaleOut
		return
	}
	// 是否黑名单
	// 随机抽奖
	curRate := rand.Float64()
	if curRate > product.BuyRate { // 概率控制
		resp.Code = ErrRetry
		return
	}
	// 总数更新
	userHistory.Add(req.ProductId, 1)     // 该人买到的商品数+1
	productCountMgr.Add(req.ProductId, 1) // 卖出去的商品数+1
	resp.Code = ErrSecKillSucc
	// Token加密 => 用户id、商品id、当前时间、秘钥
	tmpToken := fmt.Sprintf("userId=%d&productId=%d&timestamp=%d&secrity=%s",
		req.UserId, req.ProductId, now, cfg.SecKillTokenPasswd)
	resp.Token = fmt.Sprintf("%x", md5.Sum([]byte(tmpToken)))
	resp.TokenTime = now
	return
}
