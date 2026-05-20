package service

import (
	"fmt"
	"log"
	"time"

	"price-monitor/internal/repository"
	"github.com/robfig/cron/v3"
)

type MonitorScheduler struct {
	crawler    *PriceCrawler
	notifier   *Notifier
	productRepo *repository.ProductRepo
	cron       *cron.Cron
}

func NewMonitorScheduler() *MonitorScheduler {
	return &MonitorScheduler{
		crawler:    NewPriceCrawler(),
		notifier:   NewNotifier(),
		productRepo: repository.NewProductRepo(),
		cron:       cron.New(),
	}
}

// Start 启动调度器
func (s *MonitorScheduler) Start() error {
	// 每小时执行一次价格检查
	_, err := s.cron.AddFunc("0 * * * *", func() {
		if err := s.checkAllProducts(); err != nil {
			log.Printf("价格检查失败: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("注册定时任务失败: %w", err)
	}

	s.cron.Start()
	log.Printf("价格监控调度器已启动，每小时检查一次")
	return nil
}

// Stop 停止调度器
func (s *MonitorScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Printf("价格监控调度器已停止")
}

// checkAllProducts 检查所有商品价格
func (s *MonitorScheduler) checkAllProducts() error {
	products, err := s.productRepo.GetProductsNeedingCheck()
	if err != nil {
		return err
	}

	log.Printf("开始检查 %d 个商品价格", len(products))

	for _, p := range products {
		// 获取最新价格
		info, err := s.crawler.FetchPrice(p.URL)
		if err != nil {
			log.Printf("获取商品 %s 价格失败: %v", p.URL, err)
			continue
		}

		oldPrice := p.CurrentPrice
		newPrice := info.Price

		// 更新价格
		if err := s.productRepo.UpdatePrice(p.ID, newPrice); err != nil {
			log.Printf("更新商品 %d 价格失败: %v", p.ID, err)
			continue
		}

		// 记录价格历史
		if err := s.productRepo.AddPriceHistory(p.ID, newPrice); err != nil {
			log.Printf("记录价格历史失败: %v", err)
		}

		// 检查是否触发通知
		if p.TargetPrice > 0 && newPrice <= p.TargetPrice {
			if err := s.notifier.NotifyPriceDrop(p, oldPrice, newPrice); err != nil {
				log.Printf("发送价格提醒失败: %v", err)
			}
		}

		// 休息一下，避免请求过快
		time.Sleep(2 * time.Second)
	}

	log.Printf("价格检查完成")
	return nil
}

// RunOnce 立即执行一次检查（用于测试）
func (s *MonitorScheduler) RunOnce() error {
	return s.checkAllProducts()
}