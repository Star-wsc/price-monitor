package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"price-monitor/internal/config"
	"price-monitor/internal/router"
	"price-monitor/internal/service"
	"price-monitor/pkg/database"
)

func main() {
	_ = flag.String("config", "", "配置文件路径") // reserved for future use
	flag.Parse()

	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	if err := database.Init(cfg.DatabasePath); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 初始化路由
	r := router.Setup()

	// 启动定时监控
	scheduler := service.NewMonitorScheduler()
	if err := scheduler.Start(); err != nil {
		log.Printf("启动监控调度器失败: %v", err)
	}
	defer scheduler.Stop()

	// 优雅退出
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("收到退出信号，正在关闭...")
		database.Close()
		os.Exit(0)
	}()

	// 启动HTTP服务
	log.Printf("服务启动，监听端口 %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("启动HTTP服务失败: %v", err)
	}
}