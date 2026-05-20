package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"price-monitor/internal/repository"
)

// PriceCrawler 价格爬虫服务
type PriceCrawler struct {
	httpClient  *http.Client
	productRepo *repository.ProductRepo
	scraperURL  string // scraper 服务地址
}

// scraperResponse scraper API 响应结构
type scraperResponse struct {
	Success       bool    `json:"success"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price"`
	ImageURL      string  `json:"image_url"`
	Source        string  `json:"source"`
	ProductURL    string  `json:"product_url"`
	Error         string  `json:"error,omitempty"`
}

// scrapeRequest scraper API 请求结构
type scrapeRequest struct {
	URL string `json:"url"`
}

func NewPriceCrawler() *PriceCrawler {
	return &PriceCrawler{
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
		productRepo: repository.NewProductRepo(),
		scraperURL:  "http://localhost:38473",
	}
}

// PriceInfo 价格信息
type PriceInfo struct {
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	ImageURL   string  `json:"image_url"`
	Source     string  `json:"source"`
	ProductURL string  `json:"product_url"`
}

// FetchPrice 获取商品价格
func (c *PriceCrawler) FetchPrice(url string) (*PriceInfo, error) {
	switch {
	case strings.Contains(url, "jd.com") || strings.Contains(url, "jd.com.cn"):
		return c.fetchJDPrice(url)
	case strings.Contains(url, "taobao.com") || strings.Contains(url, "tmall.com"):
		return c.fetchTaobaoPrice(url)
	case strings.Contains(url, "maotai") || strings.Contains(url, "i茅台"):
		return c.fetchMaotaiPrice(url)
	default:
		return c.fetchGenericPrice(url)
	}
}

// fetchJDPrice 获取京东价格（通过 scraper 服务）
func (c *PriceCrawler) fetchJDPrice(url string) (*PriceInfo, error) {
	reqBody := scrapeRequest{URL: url}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.scraperURL+"/scrape/jd",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("调用 scraper 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result scraperResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !result.Success {
		// scraper 失败时，回退到直接 HTTP 方式
		return c.scrapeJDPage(url)
	}

	return &PriceInfo{
		Name:       result.Name,
		Price:      result.Price,
		ImageURL:   result.ImageURL,
		Source:     "jd",
		ProductURL: result.ProductURL,
	}, nil
}

// fetchTaobaoPrice 获取淘宝/天猫价格（通过 scraper 服务）
func (c *PriceCrawler) fetchTaobaoPrice(url string) (*PriceInfo, error) {
	reqBody := scrapeRequest{URL: url}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.scraperURL+"/scrape/taobao",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("调用 scraper 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result scraperResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !result.Success {
		// scraper 失败时，回退到直接 HTTP 方式
		return c.scrapeTaobaoPage(url)
	}

	return &PriceInfo{
		Name:       result.Name,
		Price:      result.Price,
		ImageURL:   result.ImageURL,
		Source:     "taobao",
		ProductURL: result.ProductURL,
	}, nil
}

// fetchGenericPrice 通用价格获取（通过 scraper 服务）
func (c *PriceCrawler) fetchGenericPrice(url string) (*PriceInfo, error) {
	reqBody := scrapeRequest{URL: url}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.scraperURL+"/scrape",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("调用 scraper 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result scraperResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !result.Success {
		// scraper 失败时，回退到直接 HTTP 方式
		return c.scrapeGenericPage(url)
	}

	return &PriceInfo{
		Name:       result.Name,
		Price:      result.Price,
		ImageURL:   result.ImageURL,
		Source:     result.Source,
		ProductURL: result.ProductURL,
	}, nil
}

// scrapeJDPage 爬取京东商品页面获取更多信息（回退方案）
func (c *PriceCrawler) scrapeJDPage(url string) (*PriceInfo, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("JD页面请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取JD页面失败: %w", err)
	}

	content := string(body)

	// 提取价格
	priceRe := regexp.MustCompile(`"jdPrice":"([^"]+)"`)
	priceMatch := priceRe.FindStringSubmatch(content)

	// 提取名称
	nameRe := regexp.MustCompile(`<title>([^<]+)</title>`)
	nameMatch := nameRe.FindStringSubmatch(content)

	// 提取图片
	imgRe := regexp.MustCompile(`src="(https://img\d+\.360buyimg\.com/[^"]+)"`)
	imgMatch := imgRe.FindStringSubmatch(content)

	price := 0.0
	if len(priceMatch) > 1 {
		price, _ = strconv.ParseFloat(priceMatch[1], 64)
	}

	name := ""
	if len(nameMatch) > 1 {
		name = nameMatch[1]
	}

	imgURL := ""
	if len(imgMatch) > 1 {
		imgURL = imgMatch[1]
	}

	return &PriceInfo{
		Name:       name,
		Price:      price,
		ImageURL:   imgURL,
		Source:     "jd",
		ProductURL: url,
	}, nil
}

// scrapeTaobaoPage 爬取淘宝/天猫商品页面（回退方案）
func (c *PriceCrawler) scrapeTaobaoPage(url string) (*PriceInfo, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", "t=abccc123; cookie2=xyz") // 需要真实cookie

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("淘宝页面请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取淘宝页面失败: %w", err)
	}

	content := string(body)

	// 提取价格
	priceRe := regexp.MustCompile(`"price"\s*:\s*"([^"]+)"`)
	priceMatch := priceRe.FindStringSubmatch(content)

	// 提取标题
	titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
	titleMatch := titleRe.FindStringSubmatch(content)

	price := 0.0
	if len(priceMatch) > 1 {
		price, _ = strconv.ParseFloat(priceMatch[1], 64)
	}

	name := ""
	if len(titleMatch) > 1 {
		name = titleMatch[1]
	}

	return &PriceInfo{
		Name:       name,
		Price:      price,
		ImageURL:   "",
		Source:     "taobao",
		ProductURL: url,
	}, nil
}

// scrapeGenericPage 通用页面爬取（回退方案）
func (c *PriceCrawler) scrapeGenericPage(url string) (*PriceInfo, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取页面失败: %w", err)
	}

	content := string(body)

	// 尝试提取价格
	priceRe := regexp.MustCompile(`[¥$]?\s*(\d+\.?\d*)\s*(元|块)?`)
	priceMatch := priceRe.FindAllStringSubmatch(content, -1)

	var price float64
	for _, m := range priceMatch {
		if len(m) > 1 {
			if p, err := strconv.ParseFloat(m[1], 64); err == nil && p > 0 && p < 1000000 {
				price = p
				break
			}
		}
	}

	// 提取标题
	titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
	titleMatch := titleRe.FindStringSubmatch(content)

	return &PriceInfo{
		Name:       func() string { if len(titleMatch) > 1 { return titleMatch[1] }; return "" }(),
		Price:      price,
		ImageURL:   "",
		Source:     "generic",
		ProductURL: url,
	}, nil
}

// fetchMaotaiPrice 获取茅台价格（简化版）
func (c *PriceCrawler) fetchMaotaiPrice(url string) (*PriceInfo, error) {
	return &PriceInfo{
		Name:       "53度飞天茅台",
		Price:      1499, // 官方指导价，实际市场价格更高
		ImageURL:   "",
		Source:     "maotai",
		ProductURL: url,
	}, nil
}

// UpdateProductPrice 更新商品价格
func (c *PriceCrawler) UpdateProductPrice(productID int64, url string) error {
	info, err := c.FetchPrice(url)
	if err != nil {
		return err
	}

	if err := c.productRepo.UpdatePrice(productID, info.Price); err != nil {
		return fmt.Errorf("更新价格失败: %w", err)
	}

	if err := c.productRepo.AddPriceHistory(productID, info.Price); err != nil {
		return fmt.Errorf("记录价格历史失败: %w", err)
	}

	return nil
}

// MonitorProducts 监控所有商品
func (c *PriceCrawler) MonitorProducts() error {
	products, err := c.productRepo.GetProductsNeedingCheck()
	if err != nil {
		return fmt.Errorf("获取监控商品失败: %w", err)
	}

	notified := 0
	for _, p := range products {
		info, err := c.FetchPrice(p.URL)
		if err != nil {
			continue
		}

		c.productRepo.UpdatePrice(p.ID, info.Price)
		c.productRepo.AddPriceHistory(p.ID, info.Price)

		if p.TargetPrice > 0 && info.Price <= p.TargetPrice {
			notified++
		}
	}

	return nil
}