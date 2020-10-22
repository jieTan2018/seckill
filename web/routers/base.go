package routers

import (
	cfg "seckill/web/configs"
	"seckill/web/models"

	"github.com/gin-gonic/gin"
)

var (
	r        *gin.Engine
	product  *models.Product
	activity *models.Activity
	logs     = cfg.Logs
)

const (
	ErrSuccess        = 0
	ErrNotFound       = 1001
	ErrDBCreateFailed = 1002
	ErrParamsInvalid  = 1003
	ErrOthers         = 1004 // 该错误属于开发错误, 是可以避免的
	DBOpertorErrInfo  = "database operation failed!"
)

func RegisterRouter(router *gin.Engine) { // 注册路由
	r = router
	registerR(productUrls, activityUrls)
}

func registerR(routerFuncs ...func()) { // 注册具体model的路由
	for _, routerFunc := range routerFuncs {
		routerFunc()
	}
}
