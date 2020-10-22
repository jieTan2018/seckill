package main

import (
	"fmt"
	"seckill/layer/middlewares"
	"seckill/layer/services"

	"github.com/gin-gonic/gin"
)

// 逻辑层
func startLayer() {
	r := gin.Default()
	if logger, err := middlewares.LoggerToFiles(); err == nil {
		r.Use(logger)
	}
	services.InitLayer()
	r.Run(":9001")
}

func main() {
	fmt.Println("layer run!")
	startLayer()
}
