package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"price-monitor/internal/model"
	"price-monitor/internal/repository"
	"price-monitor/internal/service"
)

type APIHandler struct {
	productRepo *repository.ProductRepo
	crawler     *service.PriceCrawler
}

func NewAPIHandler() *APIHandler {
	return &APIHandler{
		productRepo: repository.NewProductRepo(),
		crawler:     service.NewPriceCrawler(),
	}
}

// Response 统一响应格式
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// Ping 健康检查
func (h *APIHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "pong"})
}

// AddProduct 添加商品
func (h *APIHandler) AddProduct(c *gin.Context) {
	url := c.PostForm("url")
	targetPriceStr := c.PostForm("target_price")

	if url == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "商品链接不能为空"})
		return
	}

	// 检查商品是否已存在
	existing, err := h.productRepo.GetByURL(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "查询商品失败"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusOK, Response{Code: 0, Msg: "商品已存在", Data: existing})
		return
	}

	// 获取价格信息
	priceInfo, err := h.crawler.FetchPrice(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "获取价格失败: " + err.Error()})
		return
	}

	// 解析目标价
	targetPrice := 0.0
	if targetPriceStr != "" {
		targetPrice, _ = strconv.ParseFloat(targetPriceStr, 64)
	}

	// 创建商品
	product := &model.Product{
		URL:          url,
		Name:         priceInfo.Name,
		Source:       priceInfo.Source,
		ImageURL:     priceInfo.ImageURL,
		CurrentPrice: priceInfo.Price,
		TargetPrice:  targetPrice,
	}

	id, err := h.productRepo.Create(product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "添加商品失败"})
		return
	}

	product.ID = id
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "添加成功", Data: product})
}

// ListProducts 获取商品列表
func (h *APIHandler) ListProducts(c *gin.Context) {
	products, err := h.productRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "查询商品列表失败"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 0, Msg: "success", Data: products})
}

// GetProduct 获取单个商品
func (h *APIHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无效的商品ID"})
		return
	}

	product, err := h.productRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "查询商品失败"})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Msg: "商品不存在"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 0, Msg: "success", Data: product})
}

// UpdateProduct 更新商品
func (h *APIHandler) UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无效的商品ID"})
		return
	}

	product, err := h.productRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "查询商品失败"})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Msg: "商品不存在"})
		return
	}

	// 更新字段
	if name := c.PostForm("name"); name != "" {
		product.Name = name
	}
	if targetPrice := c.PostForm("target_price"); targetPrice != "" {
		if tp, err := strconv.ParseFloat(targetPrice, 64); err == nil {
			product.TargetPrice = tp
		}
	}

	if err := h.productRepo.Update(product); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "更新商品失败"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 0, Msg: "更新成功", Data: product})
}

// DeleteProduct 删除商品
func (h *APIHandler) DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无效的商品ID"})
		return
	}

	if err := h.productRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "删除商品失败"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 0, Msg: "删除成功"})
}

// GetPriceHistory 获取价格历史
func (h *APIHandler) GetPriceHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无效的商品ID"})
		return
	}

	days := 30 // 默认30天
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	history, err := h.productRepo.GetPriceHistory(id, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "查询价格历史失败"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 0, Msg: "success", Data: history})
}

// RefreshPrice 刷新单个商品价格
func (h *APIHandler) RefreshPrice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "无效的商品ID"})
		return
	}

	product, err := h.productRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "查询商品失败"})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Msg: "商品不存在"})
		return
	}

	// 获取最新价格
	priceInfo, err := h.crawler.FetchPrice(product.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "获取价格失败"})
		return
	}

	// 更新价格
	if err := h.productRepo.UpdatePrice(id, priceInfo.Price); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "更新价格失败"})
		return
	}

	// 记录历史
	h.productRepo.AddPriceHistory(id, priceInfo.Price)

	// 返回最新商品信息
	product.CurrentPrice = priceInfo.Price
	product.LastCheck = product.LastCheck
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "刷新成功", Data: product})
}