package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

const (
	ActivityStatusNormal  = 0
	ActivityStatusDisable = 1 // 禁止
	ActivityStatusExpire  = 2 // 过期

)

var ActivityStatusMap = map[int]string{
	ActivityStatusNormal:  "正常",
	ActivityStatusDisable: "已禁用",
	ActivityStatusExpire:  "已结束",
}

type SecProductInfoConf struct {
	ProductId         int
	StartTime         int64
	EndTime           int64
	Status            int
	Total             int
	Left              int
	onePersonBuyLimit int // 个人购买限制
	BuyRate           float64
	soldMaxLimit      int // 配到etcd, 每秒最多卖多少个
}

type Activity struct {
	Id           int    `json:"id" form:"id"`
	Name         string `json:"name" form:"name" binding:"required"`
	ProductId    int    `json:"product_id" gorm:"column:product_id" form:"productId"`
	StartTime    int64  `json:"-" gorm:"column:start_time" form:"startTime"`
	EndTime      int64  `json:"-" gorm:"column:end_time" form:"endTime"`
	Total        int    `json:"total" form:"total"`
	Status       int    `json:"-" form:"status"`
	StartTimeStr string `json:"start_time" gorm:"-" form:"-"`
	EndTimeStr   string `json:"end_time" gorm:"-" form:"-"`
	StausStr     string `json:"status" gorm:"-" form:"-"`
	Speed        int    `json:"speed" gorm:"column:sec_speed" form:"speed"`
	BuyLimit     int    `json:"buy_limit" gorm:"column:buy_limit" form:"buyLimit"`
}

func NewActivity() *Activity {
	return &Activity{}
}

func (a *Activity) TableName() string {
	return "activities"
}

func (a *Activity) GetActivityList() (list *[]*Activity, err error) {
	list = &[]*Activity{}
	err = orm.Order("id desc").Find(list).Error // 倒序 => 将最新的排倒前面
	if err != nil {
		logs.Errorf("get activity list failed! err:%v", err)
		return
	}
	for _, v := range *list {
		// 部分显示格式化
		partActivityFormat(v)
	}
	return
}

func (a *Activity) ActivityWithId(id string) (acti *Activity, err error) {
	acti = &Activity{}
	err = orm.First(acti, id).Error
	if err != nil {
		logs.Errorf("get activity by id failed! err:%v", err)
		return
	}
	// 部分显示格式化
	partActivityFormat(acti)
	return
}

func (a *Activity) ProductValid(productId, total int) (valid bool, err error) {
	prod := NewProduct()
	prod, err = prod.ProductWithId(strconv.Itoa(productId))
	if err != nil {
		logs.Error(err)
		return
	}
	if total > prod.Total {
		logs.Errorf("product[%d]'s counts invalid!", productId)
		return
	}

	valid = true
	return
}

func (a *Activity) AddActivity(acti *Activity) (err error) {
	// 判断商品是否存在、并且不能超卖
	valid, err := a.ProductValid(acti.ProductId, acti.Total) // 验证total, 防止超卖
	if err != nil {
		logs.Errorf("product valid func failed! err:%v", err)
		return
	}
	if !valid {
		err = fmt.Errorf("product id[%d] is invalid!", acti.ProductId)
		logs.Error(err)
		return
	}
	// 检测活动开始、结束时间是否正确
	if acti.StartTime <= 0 || acti.EndTime <= 0 {
		err = fmt.Errorf("invalid start[%d]|end[%d] time!",
			acti.StartTime, acti.EndTime)
		logs.Error(err)
		return
	}
	if acti.EndTime <= acti.StartTime {
		err = fmt.Errorf("start[%d] is greate then end[%d] time!",
			acti.StartTime, acti.EndTime)
		logs.Error(err)
		return
	}
	now := time.Now().Unix()
	if acti.EndTime <= now || acti.StartTime <= now {
		err = fmt.Errorf("start[%d]|end[%d] time is less then now[%d]!",
			acti.StartTime, acti.EndTime, now)
		logs.Error(err)
		return
	}
	err = orm.Create(acti).Error
	if err != nil {
		logs.Errorf("create activity failed! err:%v", err)
		return
	}
	logs.Debug("insert database record succ!")
	// to etcd
	err = a.SyncToEtcd(acti)
	if err != nil {
		logs.Warnf("sync to etcd failed! err:%v, data:%v", err, acti)
	}
	logs.Debugf("sync to etcd succ! data:%v", acti)
	return
}

func (a *Activity) UpdateActivity(id string, params map[string]interface{}) (err error) {
	ret := orm.Model(Activity{}).Where("id=?", id).Updates(params)
	if ret.RowsAffected == 0 {
		logs.Error("no rows are affected!")
		return gorm.ErrRecordNotFound
	}
	return ret.Error
}

func (a *Activity) DeleteActivity(id string) (err error) {
	ret := orm.Unscoped().Where("id=?", id).Delete(Activity{})
	if ret.RowsAffected == 0 {
		logs.Error("no rows are affected!")
		return gorm.ErrRecordNotFound
	}
	return ret.Error
}

func (a *Activity) SyncToEtcd(activity *Activity) (err error) {
	secProductInfoList, err := loadProductFromEtcd(etcdKey)

	var secProductInfo SecProductInfoConf
	secProductInfo.EndTime = activity.EndTime
	secProductInfo.onePersonBuyLimit = activity.BuyLimit
	secProductInfo.ProductId = activity.ProductId
	secProductInfo.soldMaxLimit = activity.Speed
	secProductInfo.StartTime = activity.StartTime
	secProductInfo.Status = activity.Status
	secProductInfo.Total = activity.Total
	// 存放所有的etcd配置 => 后面应让不同的商品对应不同的etcdKey, 从而在多线程时避免修改其他商品的配置!
	secProductInfoList = append(secProductInfoList, secProductInfo)
	// json化配置,并写入到etcd
	data, err := json.Marshal(secProductInfoList)
	if err != nil {
		logs.Errorf("json marshal failed! err:%v", err)
		return
	}
	ctx, cancle := context.WithTimeout(context.Background(), time.Second*etcdConnTimeout)
	defer cancle()
	_, err = etcdClient.Put(ctx, etcdKey, string(data))
	if err != nil {
		logs.Errorf("put to etcd failed! err:%v, data[%v]", err, string(data))
		return
	}

	return
}

func loadProductFromEtcd(key string) (secProductInfo []SecProductInfoConf, err error) {
	ctx, cancle := context.WithTimeout(context.Background(), time.Second*etcdConnTimeout)
	defer cancle()
	res, err := etcdClient.Get(ctx, key)
	if err != nil {
		logs.Error("get sec conf failed! err:", err.Error())
		return
	}

	// var secProductInfo []SecProductInfoConf
	for _, v := range res.Kvs {
		err = json.Unmarshal(v.Value, &secProductInfo)
		if err != nil {
			logs.Error("Unmarshal sec products info failed, err: ", err)
		}
		logs.Debug("sec info conf is: ", secProductInfo)
	}

	return
}

//

type activityUpdateValid struct {
	Id        int    `json:"id" form:"id" binding:"required"`
	Name      string `json:"name" form:"name"`
	ProductId int    `json:"product_id" gorm:"column:product_id" form:"productId"`
	StartTime int    `json:"start_time" gorm:"column:start_time" form:"startTime"`
	EndTime   int    `json:"end_time" gorm:"column:end_time" form:"endTime"`
	Total     int    `json:"total" form:"total"`
	Status    int    `json:"status" form:"status"`
}

func NewActivityUpdateValid() *activityUpdateValid {
	return &activityUpdateValid{}
}

func partActivityFormat(activity *Activity) {
	// 日期格式化
	activity.StartTimeStr = time.Unix(activity.StartTime, 0).Format(DateTimeFormatStr)
	activity.EndTimeStr = time.Unix(activity.EndTime, 0).Format(DateTimeFormatStr)
	// 商品状态
	now := time.Now().Unix()
	if now > activity.EndTime {
		activity.Status = ActivityStatusExpire
		activity.StausStr = ActivityStatusMap[ActivityStatusExpire]
		return
	}
	if activity.Status == ActivityStatusNormal {
		activity.StausStr = ActivityStatusMap[ActivityStatusNormal]
	} else if activity.Status == ActivityStatusDisable {
		activity.StausStr = ActivityStatusMap[ActivityStatusDisable]
	}
}
