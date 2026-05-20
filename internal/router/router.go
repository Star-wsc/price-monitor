package router

import (
	"os"

	"github.com/gin-gonic/gin"

	"price-monitor/internal/handler"
)

func Setup() *gin.Engine {
	r := gin.Default()

	// 允许跨域
	r.Use(corsMiddleware())

	h := handler.NewAPIHandler()

	// 健康检查
	r.GET("/ping", h.Ping)

	// API路由
	api := r.Group("/api")
	{
		// 商品管理
		api.POST("/products", h.AddProduct)
		api.GET("/products", h.ListProducts)
		api.GET("/products/:id", h.GetProduct)
		api.PUT("/products/:id", h.UpdateProduct)
		api.DELETE("/products/:id", h.DeleteProduct)
		api.POST("/products/:id/refresh", h.RefreshPrice)

		// 价格历史
		api.GET("/products/:id/history", h.GetPriceHistory)
	}

	// 前端页面
	// 获取可执行文件所在目录，用于静态文件路径
	execPath := os.Getenv("EXEC_PATH")
	if execPath == "" {
		execPath = "/app"
	}

	r.Static("/static", execPath+"/web/static")
	r.GET("/", func(c *gin.Context) {
		c.File(execPath + "/web/index.html")
	})

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}