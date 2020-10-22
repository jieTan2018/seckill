package services

import (
	"fmt"
	"net/http"
	cfg "seckill/proxy/configs"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	results map[string]interface{}
)

type secRequest struct {
	ProductId    int       `form:"product_id" binding:"required"`
	Source       string    `form:"source" binding:"required"`
	AuthCode     string    `form:"auth_code" binding:"required"`
	SecTime      string    `form:"sec_time" binding:"required"`
	Nance        int       `form:"nance" binding:"required"`   // 随机数
	UserId       int       `form:"user_id" binding:"required"` // 暂时来源于query params
	UserAuthSign string    `form:"-"`
	AccessTime   time.Time `form:"-"`
	ClientAddr   string    `form:"-"` // 防止某个ip攻击
	ClientRefer  string    `form:"-"` // 访问的来源, 甄别是不是自己网站的请求
	// CloseNotify  <-chan bool      `form:"-" json:"-"`
	// ResultChan   chan *secRequest `form:"-" json:"-"`
}

func Seckill(c *gin.Context) { // GET
	results = map[string]interface{}{}
	secReq := secRequest{
		AccessTime:  time.Now(),
		ClientAddr:  c.ClientIP(),
		ClientRefer: c.Request.Referer(),
	}
	if err := c.ShouldBind(&secReq); err != nil {
		results["code"] = http.StatusBadRequest
		results["message"] = "missing query parameters!"
		c.JSON(http.StatusBadRequest, results)
		fmt.Println("get query err:", err.Error())
		return
	}
	// // cookie 用户校验
	// userId, _ := c.Cookie("user_id")
	// userAuthSign, _ := c.Cookie("user_auth_sign")
	// fmt.Println("userId:", userId, "userAuthSign:", userAuthSign)
	// if err := userCheck(&secReq); err != nil {
	// 	results["code"] = http.StatusUnauthorized
	// 	results["message"] = err.Error()
	// 	c.JSON(http.StatusUnauthorized, results)
	// 	return
	// }
	// 频率控制 => 比如一秒过来几千次请求的
	if err := antiSpqm(&secReq); err != nil {
		results["code"] = http.StatusConflict
		results["message"] = err.Error()
		c.JSON(http.StatusConflict, results)
		return
	}
	//
	cfg.RWSecProductLock.RLock()
	v, ok := cfg.SecProductInfosMap[secReq.ProductId]
	cfg.RWSecProductLock.RUnlock()
	if !ok {
		results["code"] = http.StatusBadRequest
		results["message"] = "could not find this product: " + strconv.Itoa(secReq.ProductId)
		c.JSON(http.StatusBadRequest, results)
		return
	}
	datas := getSecStatus(v)
	if datas["code"] != 0 {
		logs.Warnf("useId[%d] get product failed! code[%d] req[%v]", secReq.UserId, datas["code"], secReq)
		c.JSON(http.StatusBadRequest, datas)
		return
	}
	// 将可抢购的商品放进redis队列
	c.JSON(http.StatusOK, secReq)
}

func SecInfo(c *gin.Context) { // GET
	pId := c.Param("pid")
	proId, err := strconv.Atoi(pId)
	results = map[string]interface{}{"datas": map[string]interface{}{}}
	if err != nil {
		results["code"] = http.StatusBadRequest
		results["message"] = "product id expect a int value!"
		c.JSON(http.StatusBadRequest, results)
		return
	}
	//
	cfg.RWSecProductLock.RLock()
	v, ok := cfg.SecProductInfosMap[proId]
	cfg.RWSecProductLock.RUnlock()
	if !ok {
		results["code"] = http.StatusBadRequest
		results["message"] = "could not find this product: " + pId
		c.JSON(http.StatusBadRequest, results)
		return
	}
	datas := getSecStatus(v)
	results["datas"] = datas["infos"]
	results["code"] = datas["code"]
	results["message"] = "get info success!"
	c.JSON(http.StatusOK, results)
}

func SecInfosList(c *gin.Context) { // GET
	datas := []map[string]interface{}{}
	results = map[string]interface{}{"code": http.StatusOK, "message": "get info success!", "datas": datas}
	cfg.RWSecProductLock.RLock()
	for _, v := range cfg.SecProductInfosMap {
		info := getSecStatus(v)
		datas = append(datas, info["infos"].(map[string]interface{}))
	}
	cfg.RWSecProductLock.RUnlock()
	results["datas"] = datas
	c.JSON(http.StatusOK, results)
}

func getSecStatus(v *cfg.SecInfoConf) map[string]interface{} {
	var start, end bool
	var status string = "success"
	var code int
	now := time.Now().Unix()
	if now < v.StartTime {
		status = "sec kill is not start!"
		code = ErrActiveNotStart
	} else {
		start = true
		if now > v.EndTime {
			start = false
			end = true
			status = "sec kill is already end!"
			code = ErrActiveAlreadyEnd
		}
	}
	if v.Status == cfg.ProductStatusForceOut || v.Status == cfg.ProductStatusSaleOut {
		status = "product is sale out!"
		code = ErrActiveSaleOut
	}
	return map[string]interface{}{
		"infos": map[string]interface{}{
			"product_id": v.ProductId,
			"start":      start, // v.StartTime 不接传时间, 可保证所有client都已sever为准
			"end":        end,   // v.EndTime
			"status":     status,
		},
		"code": code,
	}
}
