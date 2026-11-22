package main

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	// 验证 Gorm 引入
	var _ *gorm.DB

	// 初始化 Gin 引擎
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 启动服务，默认监听 8080 端口
	r.Run(":8080")
}
