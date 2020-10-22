package main

import (
	"fmt"
	"seckill/web/middlewares"
	"seckill/web/models"
	"seckill/web/routers"

	"github.com/gin-gonic/gin"
)

// web apis
func startWeb() {
	r := gin.Default()
	if logger, err := middlewares.LoggerToFiles(); err == nil {
		r.Use(logger)
	}
	routers.RegisterRouter(r) // 封装的路由
	models.Init()
	r.Run(":9002")
}

func main() {
	fmt.Println("web run!")
	startWeb()
}
