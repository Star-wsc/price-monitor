package service

import (
	"fmt"
	"log"

	"price-monitor/internal/model"
)

// Notifier 通知服务
type Notifier struct {
	// 微信通知配置
	weChatWebhook string
}

func NewNotifier() *Notifier {
	return &Notifier{
		weChatWebhook: "", // 需要配置企业微信Webhook地址
	}
}

// NotifyPriceDrop 发送价格下降通知
func (n *Notifier) NotifyPriceDrop(product *model.Product, oldPrice, newPrice float64) error {
	// 计算降幅
	dropPercent := (oldPrice - newPrice) / oldPrice * 100
	
	// 构造消息
	title := "📉 价格监控提醒"
	content := fmt.Sprintf(`
商品：%s
来源：%s

原价：¥%.2f
现价：¥%.2f
降幅：%.1f%%

目标价：¥%.2f

立即查看：
%s
`, product.Name, getSourceName(product.Source), oldPrice, newPrice, dropPercent, product.TargetPrice, product.URL)

	// 发送微信通知
	if n.weChatWebhook != "" {
		return n.sendWeChatNotification(title, content)
	}

	// 如果没有配置Webhook，只打印日志
	log.Printf("价格提醒: %s 从 ¥%.2f 降至 ¥%.2f", product.Name, oldPrice, newPrice)
	return nil
}

// NotifyPriceRise 发送价格上涨通知
func (n *Notifier) NotifyPriceRise(product *model.Product, oldPrice, newPrice float64) error {
	risePercent := (newPrice - oldPrice) / oldPrice * 100
	
	content := fmt.Sprintf(`
商品：%s
来源：%s

原价：¥%.2f
现价：¥%.2f
涨幅：%.1f%%

立即查看：
%s
`, product.Name, getSourceName(product.Source), oldPrice, newPrice, risePercent, product.URL)

	if n.weChatWebhook != "" {
		return n.sendWeChatNotification("📈 价格监控提醒", content)
	}

	log.Printf("价格提醒: %s 从 ¥%.2f 涨至 ¥%.2f", product.Name, oldPrice, newPrice)
	return nil
}

// sendWeChatNotification 发送企业微信通知
func (n *Notifier) sendWeChatNotification(title, content string) error {
	// 企业微信Webhook机器人
	// 参考: https://developer.work.weixin.qq.com/document/path/91770
	log.Printf("发送微信通知: %s", content)
	return nil
}

func getSourceName(source string) string {
	switch source {
	case "jd":
		return "京东"
	case "taobao":
		return "淘宝/天猫"
	case "maotai":
		return "茅台"
	case "pinduoduo":
		return "拼多多"
	default:
		return source
	}
}

// FormatProductCard 格式化商品卡片
func FormatProductCard(p *model.Product) string {
	return fmt.Sprintf(`
【%s】
💰 现价：¥%.2f
🎯 目标价：¥%.2f
📦 来源：%s
🔗 %s
`, p.Name, p.CurrentPrice, p.TargetPrice, getSourceName(p.Source), p.URL)
}