package model

import "time"

// Product 商品
type Product struct {
	ID          int64     `json:"id"`
	URL         string    `json:"url"`
	Name        string    `json:"name"`
	Source      string    `json:"source"` // jd, taobao, maotai
	ImageURL    string    `json:"image_url"`
	CurrentPrice float64   `json:"current_price"`
	TargetPrice  float64   `json:"target_price"` // 用户期望价格
	LastCheck   time.Time `json:"last_check"`
	CreatedAt   time.Time `json:"created_at"`
}

// PriceHistory 价格历史
type PriceHistory struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	Price     float64   `json:"price"`
	CheckedAt time.Time `json:"checked_at"`
}

// Notification 通知记录
type Notification struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	Type      string    `json:"type"` // wechat, email
	Content   string    `json:"content"`
	SentAt    time.Time `json:"sent_at"`
}

// User 用户设置
type User struct {
	ID       int64  `json:"id"`
	OpenID   string `json:"open_id"` // 微信openid
	Settings string `json:"settings"` // JSON设置
}