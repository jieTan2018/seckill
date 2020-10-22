package main

import (
	"fmt"
	"seckill/proxy/configs"
	"seckill/proxy/middlewares"
	"seckill/proxy/routers"

	"github.com/gin-gonic/gin"
)

// 接入层
func start() {
	r := gin.Default()
	if logger, err := middlewares.LoggerToFiles(); err == nil {
		r.Use(logger)
	}
	routers.RegisterRouter(r)
	configs.InitSeckill()
	r.Run(":9000")
}

func main() {
	fmt.Println("proxy run!")
	start()
}
