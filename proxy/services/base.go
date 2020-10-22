package services

import (
	"encoding/json"
	"fmt"
	cfg "seckill/proxy/configs"
	"strconv"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
)

const (
	ErrInvalidRequest      = 1001
	ErrNotFoundProductId   = 1002
	ErrUserCheckAuthFailed = 1003
	ErrUserServiceBusy     = 1004
	ErrActiveNotStart      = 1005
	ErrActiveAlreadyEnd    = 1006
	ErrActiveSaleOut       = 1007
	SecReqChanSize         = 10
)

var (
	logs                 = cfg.Logs
	blackRedisPool       *redis.Pool
	proxy2LayerRedisPool *redis.Pool
	ipBlackList          map[string]bool
	idBlackList          map[int]bool
	RWBlackLock          sync.RWMutex
	SecReqChan           chan *secRequest
)

func init() {
	if err := initSec(); err != nil {
		logs.Errorf("init sec handler failed! err:%v", err)
	}
}

func initSec() error {
	if err := loadBlackList(); err != nil {
		logs.Errorf("load black list failed! err:%v", err)
		return err
	}
	if err := initProxy2LayerRedis(); err != nil {
		logs.Errorf("load proxy2layer redis pool failed! err:%v", err)
		return err
	}
	if err := initRedisProcessFunc(); err != nil {
		logs.Errorf("load initRedisProcessFunc redis pool failed! err:%v", err)
		return err
	}
	SecReqChan = make(chan *secRequest, SecReqChanSize)
	logs.Debug("init sec handler success!")
	return nil
}

//
func initRedis(rCfgs *cfg.RedisBaseCfgs) (*redis.Pool, error) {
	// 连接池
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

//
func loadBlackList() (err error) { // 加载黑名单到内存
	var redisBlackCfgs = cfg.RedisCfgs
	blackRedisPool, err = initRedis(redisBlackCfgs)
	if err != nil {
		logs.Debugf("init redis failed! err:%v", err)
		return err
	}
	// 获取黑名单
	conn := blackRedisPool.Get()
	defer conn.Close()
	// id黑名单
	reply, err := conn.Do("hgetall", "idblacklist")
	idList, err := redis.Strings(reply, err)
	if err != nil {
		logs.Warnf("hget all black list failed! err:%v", err)
		return err
	}

	for _, v := range idList {
		id, err := strconv.Atoi(v)
		if err != nil {
			logs.Warnf("invalid user id [%v]", id)
			continue
		}
		idBlackList[id] = true
	}
	// ip黑名单
	reply, err = conn.Do("hgetall", "ipblacklist")
	ipList, err := redis.Strings(reply, err)
	if err != nil {
		logs.Warnf("hget all black list failed! err:%v", err)
		return err
	}

	for _, v := range ipList {
		ipBlackList[v] = true
	}

	go syncIpBlackList() // 同步黑名单
	go syncIdBlackList()
	return nil
}

func syncIpBlackList() {
	var (
		ipList   []string
		lastTime = time.Now().Unix()
	)
	for {
		conn := blackRedisPool.Get()
		if _, err := conn.Do("ping"); err != nil {
			logs.Errorf("ping redis failed, err:%v", err)
			fmt.Println("err:", err.Error())
			conn.Close()
			return
		}
		reply, err := conn.Do("BLPOP", "blackiplist", time.Second) // "blackiplist" 新增的黑名单列表
		ip, err := redis.String(reply, err)
		if err != nil {
			conn.Close()
			continue
		}
		curTime := time.Now().Unix()
		ipList = append(ipList, ip)
		// 若更改频率过高, 应优化为 批量更新
		if len(ipList) > 100 || curTime-lastTime > 5 { // 大于某个值 or 某个时间时, 批量更新
			RWBlackLock.Lock()
			for _, v := range ipList {
				ipBlackList[v] = true
			}
			RWBlackLock.Unlock()
			lastTime = curTime
			logs.Infof("sync ip list from redis succ, ip[%v]", ipList)
		}
		conn.Close()
	}
}

func syncIdBlackList() {
	var (
		idList   []int
		lastTime = time.Now().Unix()
	)
	for {
		conn := blackRedisPool.Get()
		reply, err := conn.Do("BLPOP", "blackidlist", time.Second) // "blackiplist" 新增的黑名单列表
		id, err := redis.Int(reply, err)
		if err != nil {
			conn.Close()
			continue
		}
		curTime := time.Now().Unix()
		idList = append(idList, id)
		// 若更改频率过高, 应优化为 批量更新
		if len(idList) > 100 || curTime-lastTime > 5 { // 大于某个值 or 某个时间时, 批量更新
			RWBlackLock.Lock()
			for _, v := range idList {
				idBlackList[v] = true
			}
			RWBlackLock.Unlock()
			lastTime = curTime
			logs.Infof("sync user id list from redis succ, id[%v]", idList)
		}
		conn.Close()
	}
}

//
func initProxy2LayerRedis() (err error) {
	proxy2LayerRedisPool, err = initRedis(cfg.RedisProxy2LayerCfgs)
	if err != nil {
		logs.Debugf("init redis failed! err:%v", err)
		return err
	}

	conn := proxy2LayerRedisPool.Get()
	defer conn.Close()

	return nil
}

func initRedisProcessFunc() error {
	// write
	for i := 0; i < cfg.WriteProxy2LayerGoroutineNum; i++ {
		go writeHandler()
	}
	// read
	for i := 0; i < cfg.ReadProxy2LayerGoroutineNum; i++ {
		go readHandler()
	}
	return nil
}

func writeHandler() { // 从channel取数据, 发送到redis队列
	for {
		conn := proxy2LayerRedisPool.Get()
		req := <-SecReqChan
		data, err := json.Marshal(req)
		if err != nil { // ?
			logs.Errorf("json Marshal failed! err:%v, req:%v", err, req)
			conn.Close()
			continue
		}
		_, err = conn.Do("LPUSH", "sec_queue", data)
		if err != nil {
			logs.Errorf("lpush failed! err:%v, req:%v", err, req)
			conn.Close()
			continue
		}
		conn.Close()
	}
}
func readHandler() {
	// for {
	// 	conn := proxy2LayerRedisPool.Get()
	// 	reply, err := conn.Do("RPOP", "recv_queue")
	// 	if err == redis.ErrNil { // reply为空时 => 比如key不存在
	// 		time.Sleep(time.Second)
	// 		conn.Close()
	// 		continue
	// 	}
	// 	if err != nil {
	// 		logs.Errorf("rpop failed! err:%v", err)
	// 		conn.Close()
	// 		continue
	// 	}

	// 	var result secRequest
	// 	err = json.Unmarshal([]byte(reply), &result)
	// 	if err != nil {
	// 		logs.Errorf("json unmarshal failed! err:%v", err)
	// 		conn.Close()
	// 		continue
	// 	}

	// 	conn.Close()
	// }
}
