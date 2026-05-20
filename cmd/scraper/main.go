package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/playwright-community/playwright-go"
)

// PriceResult 价格采集结果
type PriceResult struct {
	Success       bool    `json:"success"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price"`
	ImageURL      string  `json:"image_url"`
	Source        string  `json:"source"`
	ProductURL    string  `json:"product_url"`
	Error         string  `json:"error,omitempty"`
}

// ScrapeRequest 采集请求
type ScrapeRequest struct {
	URL    string `json:"url"`
	Cookie string `json:"cookie,omitempty"`
}

var pw *playwright.Playwright
var browser playwright.Browser

func main() {
	// 初始化 Playwright
	if err := initPlaywright(); err != nil {
		log.Fatalf("初始化Playwright失败: %v", err)
	}
	defer pw.Stop()

	// 启动 Gin 服务
	port := os.Getenv("SCRAPER_PORT")
	if port == "" {
		port = "38473"
	}

	r := gin.Default()

	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "scraper"})
	})

	// 采集接口
	r.POST("/scrape/jd", scrapeJD)
	r.POST("/scrape/taobao", scrapeTaobao)
	r.POST("/scrape", scrapeAny)

	log.Printf("Scraper 服务启动，监听端口 %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func initPlaywright() error {
	var err error

	// 启动 Playwright（自动下载 driver，跳过浏览器下载）
	pw, err = playwright.Run(&playwright.RunOptions{
		SkipInstallBrowsers: true,
	})
	if err != nil {
		return err
	}

	// 获取系统 Chromium 路径
	chromeBin := os.Getenv("CHROME_BIN")
	if chromeBin == "" {
		chromeBin = "/usr/bin/chromium"
	}

	// 使用系统 Chromium
	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(chromeBin),
		Headless:       playwright.Bool(true),
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--single-process",
		},
	})
	if err != nil {
		return err
	}

	log.Printf("Chromium 启动成功，路径: %s", chromeBin)
	return nil
}

func scrapeJD(c *gin.Context) {
	var req ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PriceResult{Success: false, Error: "无效的请求"})
		return
	}

	result, err := scrapeJDPrice(req.URL)
	if err != nil {
		c.JSON(http.StatusOK, PriceResult{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func scrapeTaobao(c *gin.Context) {
	var req ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PriceResult{Success: false, Error: "无效的请求"})
		return
	}

	result, err := scrapeTaobaoPrice(req.URL)
	if err != nil {
		c.JSON(http.StatusOK, PriceResult{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func scrapeAny(c *gin.Context) {
	var req ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PriceResult{Success: false, Error: "无效的请求"})
		return
	}

	url := strings.ToLower(req.URL)
	var result PriceResult
	var err error

	switch {
	case strings.Contains(url, "jd.com") || strings.Contains(url, "jd.com.cn"):
		result, err = scrapeJDPrice(req.URL)
	case strings.Contains(url, "taobao.com") || strings.Contains(url, "tmall.com"):
		result, err = scrapeTaobaoPrice(req.URL)
	default:
		result, err = scrapeGenericPrice(req.URL)
	}

	if err != nil {
		c.JSON(http.StatusOK, PriceResult{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func scrapeJDPrice(url string) (PriceResult, error) {
	ctx, err := browser.NewContext()
	if err != nil {
		return PriceResult{}, err
	}
	defer ctx.Close()

	page, err := ctx.NewPage()
	if err != nil {
		return PriceResult{}, err
	}
	defer page.Close()

	log.Printf("访问京东商品: %s", url)

	waitState := playwright.WaitUntilState("domcontentloaded")
	_, err = page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30000),
		WaitUntil: &waitState,
	})
	if err != nil {
		log.Printf("页面加载失败: %v", err)
	}

	// 等待页面加载
	page.WaitForTimeout(3000)

	// 提取价格 - 尝试多个选择器
	priceStr := ""
	selectors := []string{".price J-price", "[class*='price']", "#price", ".p-price"}
	for _, sel := range selectors {
		price, err := page.Locator(sel).First().TextContent()
		if err == nil && price != "" {
			priceStr = price
			break
		}
	}

	// 提取商品名称
	productName := ""
	nameSelectors := []string{".sku-name", "[class*='name']", "title"}
	for _, sel := range nameSelectors {
		name, err := page.Locator(sel).First().TextContent()
		if err == nil && name != "" {
			productName = name
			break
		}
	}

	// 提取图片
	imgURL := ""
	img, err := page.Locator("#spec-img").GetAttribute("src")
	if err == nil {
		imgURL = img
	}

	finalPrice := parsePrice(priceStr)

	log.Printf("商品: %s, 价格: %s", productName, priceStr)

	return PriceResult{
		Success:       true,
		Name:          productName,
		Price:         finalPrice,
		OriginalPrice: finalPrice,
		ImageURL:      imgURL,
		Source:        "jd",
		ProductURL:    url,
	}, nil
}

func scrapeTaobaoPrice(url string) (PriceResult, error) {
	ctx, err := browser.NewContext()
	if err != nil {
		return PriceResult{}, err
	}
	defer ctx.Close()

	page, err := ctx.NewPage()
	if err != nil {
		return PriceResult{}, err
	}
	defer page.Close()

	log.Printf("访问淘宝商品: %s", url)

	waitState := playwright.WaitUntilState("domcontentloaded")
	_, err = page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30000),
		WaitUntil: &waitState,
	})
	if err != nil {
		log.Printf("页面加载失败: %v", err)
	}

	page.WaitForTimeout(3000)

	// 淘宝价格选择器
	priceStr := ""
	selectors := []string{"#price", "[class*='price']", ".tb-price"}
	for _, sel := range selectors {
		price, err := page.Locator(sel).First().TextContent()
		if err == nil && price != "" {
			priceStr = price
			break
		}
	}

	// 提取商品名称
	productName := ""
	nameSelectors := []string{".tb-main-title", "[class*='title']", "title"}
	for _, sel := range nameSelectors {
		name, err := page.Locator(sel).First().TextContent()
		if err == nil && name != "" {
			productName = name
			break
		}
	}

	finalPrice := parsePrice(priceStr)

	log.Printf("商品: %s, 价格: %s", productName, priceStr)

	return PriceResult{
		Success:       true,
		Name:          productName,
		Price:         finalPrice,
		OriginalPrice: finalPrice,
		Source:        "taobao",
		ProductURL:    url,
	}, nil
}

func scrapeGenericPrice(url string) (PriceResult, error) {
	ctx, err := browser.NewContext()
	if err != nil {
		return PriceResult{}, err
	}
	defer ctx.Close()

	page, err := ctx.NewPage()
	if err != nil {
		return PriceResult{}, err
	}
	defer page.Close()

	waitState := playwright.WaitUntilState("domcontentloaded")
	_, err = page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30000),
		WaitUntil: &waitState,
	})
	if err != nil {
		return PriceResult{}, err
	}

	page.WaitForTimeout(2000)

	// 通用价格选择器
	priceStr := ""
	price, _ := page.Locator("[class*='price'], #price, .price").First().TextContent()
	if price != "" {
		priceStr = price
	}

	title, _ := page.Title()

	finalPrice := parsePrice(priceStr)

	return PriceResult{
		Success:       true,
		Name:          title,
		Price:         finalPrice,
		OriginalPrice: finalPrice,
		Source:        "generic",
		ProductURL:    url,
	}, nil
}

// 解析价格字符串为 float
func parsePrice(priceStr string) float64 {
	if priceStr == "" {
		return 0
	}

	// 移除所有非数字和小数点的字符
	var result []byte
	for i := 0; i < len(priceStr); i++ {
		if (priceStr[i] >= '0' && priceStr[i] <= '9') || priceStr[i] == '.' {
			result = append(result, priceStr[i])
		}
	}

	if len(result) == 0 {
		return 0
	}

	price, err := strconv.ParseFloat(string(result), 64)
	if err != nil {
		return 0
	}

	return price
}
