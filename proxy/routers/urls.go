package routers

import (
	S "seckill/proxy/services"

	"github.com/gin-gonic/gin"
)

func RegisterRouter(r *gin.Engine) {
	sec := r.Group("sec")
	{
		sec.GET("/kill", S.Seckill)      // sec/kill?pid=1029&source=android&authcode=xxx&time=xxx&nance=xx
		sec.GET("/info/:pid", S.SecInfo) //
		sec.GET("/info", S.SecInfosList)
	}
}
