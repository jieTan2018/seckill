package configs

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/garyburd/redigo/redis"
	"go.etcd.io/etcd/clientv3"
)

var (
	pool       *redis.Pool // 声明一个redis pool
	etcdClient *clientv3.Client
	etcdKey    = path.Join(EtcdSecKeyPrefix, EtcdSecProductKey)
	logs       = Logs
)

func InitSeckill() {
	if err := initRedis(); err != nil {
		logs.Errorf("init redis failed! err:%v", err)
		panic(err.Error())
	}
	logs.Debug("init redis succ!")
	if err := initEtcd(); err != nil {
		logs.Errorf("init etcd failed! err:%v", err)
		panic(err.Error())
	}
	logs.Debug("init etcd succ!")
	// 监控secConf的变化
	secProductWatcher()
}

func initRedis() error {
	pool = &redis.Pool{
		MaxIdle:     RedisCfgs.MaxIdle,
		MaxActive:   RedisCfgs.MaxActive,
		IdleTimeout: RedisCfgs.IdleTimeout, // ns
		Dial: func() (redis.Conn, error) {
			coon, err := redis.Dial(RedisCfgs.Network, RedisCfgs.Uri)
			if err != nil { // 连接失败时
				panic(err.Error())
				return nil, err
			}
			// 用户认证
			if _, err := coon.Do("auth", RedisCfgs.Password); err != nil {
				return nil, err
			}
			return coon, nil
		},
	}
	coon := pool.Get()
	defer coon.Close()
	if coon.Err() != nil {
		logs.Errorf("conn redis failed! err:%v", coon.Err())
		return fmt.Errorf("redis err: %v", coon.Err())
	}
	logs.Debug("conn redis succ!")
	return nil
}

func initEtcd() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   EtcdCfgs.Endpoints,
		DialTimeout: EtcdCfgs.DialTimeout,
	})
	if err != nil {
		logs.Errorf("conn etcd failed! err:%v", err.Error())
		return err
	}
	logs.Debug("etcd conn succ!")
	etcdClient = cli
	// 更新secConf (也可是 初始化secConf)
	err = modifySecConf(etcdTestDatas())
	if err != nil {
		return err
	}
	// 加载secConf
	err = loadSecConf(etcdKey)
	if err != nil {
		Logs.Error("load sec conf failed, err:", err.Error())
		return err
	}
	logs.Debug("load sec conf succ!")
	return nil
}

func loadSecConf(key string) error { // 配置是json
	// get
	res, err := etcdClient.Get(context.Background(), key)
	if err != nil {
		Logs.Errorf("get sec conf failed! err:%v", err.Error())
		return err
	}

	var secProductInfoConf []SecInfoConf
	for _, v := range res.Kvs {
		err = json.Unmarshal(v.Value, &secProductInfoConf)
		if err != nil {
			Logs.Errorf("Unmarshal sec products info failed! err:%v", err)
		}
		Logs.Debug("sec info conf is: ", secProductInfoConf)
	}

	updateSecProductInfo(secProductInfoConf)
	return nil
}

func modifySecConf(confs *[]SecInfoConf) error {
	// put
	datas, err := json.Marshal(confs)
	if err != nil {
		Logs.Errorf("parse etcd datas faile! err:%v", err)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err = etcdClient.Put(ctx, etcdKey, string(datas))
	cancel()
	if err != nil {
		Logs.Errorf("The instraction put value to etcd was faild! err:%v", err)
		return err
	}
	Logs.Debugf("modifySecConf is finish! %s", string(datas))
	return nil
}

func secProductWatcher() {
	logs.Debug("sec product watcer running!")
	go watchSecProductKey()
	modifySecConf(etcdTestDatas2()) // 模拟数据更新
}

func watchSecProductKey() { // 监控secProduct信息的变化
	rch := etcdClient.Watch(context.Background(), etcdKey)
	var secProductInfoConf []SecInfoConf
	var getConfSucc = true
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if ev.Type == mvccpb.DELETE {
				Logs.Warnf("key[%s] 's config deleted.", etcdKey)
				continue
			}
			if ev.Type == mvccpb.PUT && string(ev.Kv.Key) == etcdKey {
				err := json.Unmarshal(ev.Kv.Value, &secProductInfoConf)
				if err != nil {
					Logs.Errorf("key[%s], Unmarshal[%s], err: %v", err)
					getConfSucc = false
					continue
				}
			}
			Logs.Debugf("get config from etcd, %s %q : %q \n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}

		if getConfSucc {
			Logs.Debugf("get config from etcd succ, %v", secProductInfoConf)
			updateSecProductInfo(secProductInfoConf)
		}
	}
	return
}

func updateSecProductInfo(secProductInfoConf []SecInfoConf) {
	// 使用tmpMap优化, 因数据量过大而导致加锁时间过长的问题
	tmpMap := make(map[int]*SecInfoConf, 1024)
	for i := 0; i < len(secProductInfoConf); i++ { // 此处不能用 _, v:=range ... 的方式, 详情见"PROBLEMS.md" => "### 作用域问题"
		product := secProductInfoConf[i]
		tmpMap[product.ProductId] = &product
	}
	RWSecProductLock.Lock()
	SecProductInfosMap = tmpMap // 思考: 使用sync.Map是不是就能不用rwSecProductLock了?
	RWSecProductLock.Unlock()
}
