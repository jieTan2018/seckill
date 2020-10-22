package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	cfg "seckill/proxy/configs"
	"strconv"
	"sync"
)

var secLimitMgr = &SecLimitMgr{
	UserLimitMap: make(map[int]*SecLimit, 10000),
	IpLimitMap:   make(map[string]*SecLimit, 10000),
}

type SecLimitMgr struct { // 对访问的用户计数
	sync.Mutex   // 匿名属性
	UserLimitMap map[int]*SecLimit
	IpLimitMap   map[string]*SecLimit
}

type SecLimit struct {
	count   int
	curTime int64
}

func userCheck(req *secRequest) error { // 用户auth检查
	// 验证请求是否在白名单
	inWhite := false
	for _, v := range cfg.ReferWhiteList {
		if v == req.ClientRefer {
			inWhite = true
			break
		}
	}
	if !inWhite {
		cfg.Logs.Warnf("user[%d] is reject by refer, req[%v]", req.UserId, req)
		return fmt.Errorf("invalid request!")
	}
	// 验证用户cookie auth
	authData := strconv.Itoa(req.UserId) + ":" + cfg.CookieSecretKey
	sum := md5.Sum([]byte(authData))
	authSign := hex.EncodeToString(sum[:])
	if req.UserAuthSign != authSign {
		return fmt.Errorf("invalid user cookie auth!")
	}
	return nil
}

func antiSpqm(req *secRequest) error {
	secLimitMgr.Lock()
	// user维度的限制
	userLimit, ok := secLimitMgr.UserLimitMap[req.UserId]
	if !ok { // 第一次访问的用户
		userLimit = &SecLimit{}
		secLimitMgr.UserLimitMap[req.UserId] = userLimit
	}
	uCount := userLimit.Counter(req.AccessTime.Unix())
	// Ip维度的限制
	ipLimit, ok := secLimitMgr.IpLimitMap[req.ClientAddr]
	if !ok {
		ipLimit = &SecLimit{}
		secLimitMgr.IpLimitMap[req.ClientAddr] = ipLimit
	}
	iCount := ipLimit.Counter(req.AccessTime.Unix())
	secLimitMgr.Unlock()

	if uCount > cfg.UserSecAccessLimit { // user访问频率超过限制
		return fmt.Errorf("invalid request!")
	}
	if iCount > cfg.IpSecAccessLimit { // ip访问频率超过限制
		return fmt.Errorf("invalid request!")
	}
	return nil
}

func (s *SecLimit) Counter(nowTime int64) int { // 计算访问次数
	if s.curTime != nowTime {
		s.count = 1
		s.curTime = nowTime
		return s.count
	}
	s.count++
	return s.count
}

func (s *SecLimit) Check(nowTime int64) int { // 获取访问次数
	if s.curTime != nowTime {
		return 0
	}

	return s.count
}
