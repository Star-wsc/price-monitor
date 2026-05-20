package repository

import (
	"database/sql"
	"fmt"
	"time"

	"price-monitor/internal/model"
	"price-monitor/pkg/database"
)

type ProductRepo struct{}

func NewProductRepo() *ProductRepo {
	return &ProductRepo{}
}

// Create 创建商品
func (r *ProductRepo) Create(p *model.Product) (int64, error) {
	result, err := database.DB.Exec(
		`INSERT INTO products (url, name, source, image_url, current_price, target_price, last_check) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.URL, p.Name, p.Source, p.ImageURL, p.CurrentPrice, p.TargetPrice, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetByURL 根据URL获取商品
func (r *ProductRepo) GetByURL(url string) (*model.Product, error) {
	p := &model.Product{}
	err := database.DB.QueryRow(
		`SELECT id, url, name, source, image_url, current_price, target_price, last_check, created_at 
		 FROM products WHERE url = ?`, url,
	).Scan(&p.ID, &p.URL, &p.Name, &p.Source, &p.ImageURL, &p.CurrentPrice, &p.TargetPrice, &p.LastCheck, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetByID 根据ID获取商品
func (r *ProductRepo) GetByID(id int64) (*model.Product, error) {
	p := &model.Product{}
	err := database.DB.QueryRow(
		`SELECT id, url, name, source, image_url, current_price, target_price, last_check, created_at 
		 FROM products WHERE id = ?`, id,
	).Scan(&p.ID, &p.URL, &p.Name, &p.Source, &p.ImageURL, &p.CurrentPrice, &p.TargetPrice, &p.LastCheck, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// List 获取所有商品
func (r *ProductRepo) List() ([]*model.Product, error) {
	rows, err := database.DB.Query(
		`SELECT id, url, name, source, image_url, current_price, target_price, last_check, created_at 
		 FROM products ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p := &model.Product{}
		if err := rows.Scan(&p.ID, &p.URL, &p.Name, &p.Source, &p.ImageURL, &p.CurrentPrice, &p.TargetPrice, &p.LastCheck, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

// Update 更新商品
func (r *ProductRepo) Update(p *model.Product) error {
	_, err := database.DB.Exec(
		`UPDATE products SET name = ?, image_url = ?, current_price = ?, target_price = ?, last_check = ?
		 WHERE id = ?`,
		p.Name, p.ImageURL, p.CurrentPrice, p.TargetPrice, time.Now(), p.ID,
	)
	return err
}

// UpdatePrice 更新价格
func (r *ProductRepo) UpdatePrice(id int64, price float64) error {
	_, err := database.DB.Exec(
		`UPDATE products SET current_price = ?, last_check = ? WHERE id = ?`,
		price, time.Now(), id,
	)
	return err
}

// Delete 删除商品
func (r *ProductRepo) Delete(id int64) error {
	_, err := database.DB.Exec(`DELETE FROM products WHERE id = ?`, id)
	return err
}

// AddPriceHistory 添加价格历史
func (r *ProductRepo) AddPriceHistory(productID int64, price float64) error {
	_, err := database.DB.Exec(
		`INSERT INTO price_history (product_id, price) VALUES (?, ?)`,
		productID, price,
	)
	return err
}

// GetPriceHistory 获取价格历史
func (r *ProductRepo) GetPriceHistory(productID int64, days int) ([]*model.PriceHistory, error) {
	rows, err := database.DB.Query(
		`SELECT id, product_id, price, checked_at FROM price_history 
		 WHERE product_id = ? AND checked_at > datetime('now', '-`+fmt.Sprintf("%d", days)+` days')
		 ORDER BY checked_at ASC`,
		productID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*model.PriceHistory
	for rows.Next() {
		h := &model.PriceHistory{}
		if err := rows.Scan(&h.ID, &h.ProductID, &h.Price, &h.CheckedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

// GetProductsNeedingCheck 获取需要检查的商品（超过1小时未检查的）
func (r *ProductRepo) GetProductsNeedingCheck() ([]*model.Product, error) {
	rows, err := database.DB.Query(
		`SELECT id, url, name, source, image_url, current_price, target_price, last_check, created_at 
		 FROM products 
		 WHERE last_check IS NULL OR last_check < datetime('now', '-1 hour')
		 ORDER BY last_check ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p := &model.Product{}
		if err := rows.Scan(&p.ID, &p.URL, &p.Name, &p.Source, &p.ImageURL, &p.CurrentPrice, &p.TargetPrice, &p.LastCheck, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}