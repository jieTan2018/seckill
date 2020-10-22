package services

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	cfg "seckill/layer/configs"
	"sync"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
)

var (
	EtcdCfgs           = cfg.EtcdCfgs
	key                = path.Join(cfg.EtcdSecKeyPrefix, cfg.EtcdSecProductKey)
	RWSecProductLock   = cfg.RWSecProductLock
	SecProductInfosMap = make(map[int]*SecInfoConf, 1024)
	HistoryMap         = make(map[int]*UserBuyHistory, 100000)
	productCountMgr    = NewProductCountMgr()
	HistoryMapLock     sync.Mutex
	etcdClient         *clientv3.Client
)

type SecInfoConf struct {
	cfg.SecInfoConf
	soldMaxLimit      int       // 配到etcd, 每秒最多卖多少个
	secLimit          *SecLimit // 限速控制
	onePersonBuyLimit int       // 个人购买限制
	BuyRate           float64
}

func initEtcd() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   EtcdCfgs.Endpoints,
		DialTimeout: EtcdCfgs.DialTimeout,
	})
	if err != nil {
		fmt.Println("etcd err:", err.Error())
		return err
	}

	etcdClient = cli
	return nil
}

func loadProductFromEtcd() error {
	ctx, cancle := context.WithTimeout(context.Background(), time.Second*cfg.EtcdConnTimeout)
	defer cancle()
	res, err := etcdClient.Get(ctx, key)
	if err != nil {
		logs.Error("get sec conf failed! err:", err.Error())
		return err
	}

	var secProductInfoConf []SecInfoConf
	for _, v := range res.Kvs {
		err = json.Unmarshal(v.Value, &secProductInfoConf)
		if err != nil {
			logs.Error("Unmarshal sec products info failed, err: ", err)
		}
		logs.Debug("sec info conf is: ", secProductInfoConf)
	}

	updateSecProductInfo(secProductInfoConf)

	initSecProductWatcher()
	return nil
}

func initSecProductWatcher() {
	go watchSecProductKey()
}

func watchSecProductKey() {
	rch := etcdClient.Watch(context.Background(), key)
	var secProductInfoConf []SecInfoConf
	var getConfSucc = true
	for wresp := range rch {
		for _, ev := range wresp.Events {
			// fmt.Printf("Type: %s Key:%s Value:%s\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			if ev.Type == mvccpb.DELETE {
				logs.Warnf("key[%s] 's config deleted.", key)
				continue
			}
			if ev.Type == mvccpb.PUT && string(ev.Kv.Key) == key {
				// fmt.Printf("etcd val:%s\n", ev.Kv.Value)
				err := json.Unmarshal(ev.Kv.Value, &secProductInfoConf)
				if err != nil {
					logs.Error("key[%s], Unmarshal[%s], err: %v", err)
					getConfSucc = false
					continue
				}
			}
			logs.Debugf("get config from etcd, %s %q : %q \n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}

		if getConfSucc {
			logs.Debugf("get config from etcd succ! %v", secProductInfoConf)
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
		product.secLimit = &SecLimit{} // 初始化限速
		tmpMap[product.ProductId] = &product
	}
	RWSecProductLock.Lock()
	SecProductInfosMap = tmpMap // 思考: 使用sync.Map是不是就能不用rwSecProductLock了?
	RWSecProductLock.Unlock()
}
