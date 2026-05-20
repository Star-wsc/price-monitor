# price-monitor 移植开发计划

## 项目现状

| 项目 | 状态 |
|------|------|
| 数据库商品数 | 2个（Redmi Watch 6目标450元，小米15 Ultra目标3200元） |
| 爬虫方式 | Node.js + Playwright（外部脚本） |
| 通知 | 企业微信Webhook（未实现） |
| 部署 | NAS Linux，端口38472 |

---

## 核心问题

1. **crawler.js 路径硬编码**：`/home/wsc768043912/price-monitor/scripts/crawler.js`
2. **依赖外部Node.js**：需要单独安装Node.js + Playwright
3. **通知未实现**：Notifier.sendWeChatNotification 是空壳

---

## 移植方案

### 方案A：Go-playwright 直接方案（推荐）

用 Go 原生的 [go-playwright](https://github.com/playwright-community/playwright-go) 库替代外部 Node.js 脚本。

**优点**：
- 纯 Go，无需外部依赖
- 跨平台编译
- Docker 单容器搞定

**改动点**：
```go
// 原来：调用外部 node scripts/crawler.js
cmd := exec.Command("node", c.crawlerJS, url, "jd")

// 改成：直接用 Go-playwright
browser, _ := playwright.chromium.launch()
page, _ := browser.newPage()
page.goto(url)
```

---

### 方案B：MCP 桥接方案

利用用户已有的 mcporter (filesystem + playwright MCP) 或 miclaw (内置playwright skill)。

```
Go Backend → HTTP API → MCP/Skill → Playwright → 价格数据
```

**优点**：
- 复用现有工具链
- 可以利用云端算力

**缺点**：
- 架构复杂
- 依赖外部服务可用性

---

## 推荐架构（方案A改进版）

### Docker 单容器部署

```dockerfile
FROM alpine:3.19
# 包含：Go + Chromium + Playwright
# 暴露端口：38472
```

### 目录结构

```
price-monitor/
├── cmd/server/main.go       # 程序入口
├── internal/
│   ├── config/              # 配置（端口、数据库路径、Webhook）
│   ├── handler/             # REST API
│   ├── model/               # 数据模型
│   ├── repository/          # 数据库CRUD
│   ├── router/              # 路由
│   └── service/
│       ├── crawler.go       # 爬虫（Go-playwright）
│       ├── monitor.go       # 定时任务
│       └── notifier.go      # 通知（企业微信）
├── pkg/database/            # SQLite封装
├── web/                     # 前端
├── Dockerfile               # Docker构建
├── docker-compose.yml       # 部署配置
└── scripts/
    └── init.sql             # 数据库初始化
```

---

## 开发任务

### P0 - 必须完成

| 任务 | 说明 |
|------|------|
| 替换爬虫为go-playwright | 移除Node.js依赖 |
| 修复路径硬编码 | 改为相对路径或环境变量 |
| Docker化 | 单容器包含所有依赖 |
| 验证京东价格采集 | 测试能否获取正确价格 |

### P1 - 重要功能

| 任务 | 说明 |
|------|------|
| 实现企业微信通知 | 完成Webhook发送 |
| 前端优化 | 图表展示价格走势 |
| 配置管理 | 支持环境变量配置 |

### P2 - 改进功能

| 任务 | 说明 |
|------|------|
| 支持更多平台 | 淘宝/天猫/拼多多 |
| 价格预测 | 基于历史数据 |
| 浏览器指纹 | 绕过反爬 |

---

## 数据库现状

```sql
-- products 表
id: 2, 3
商品1: Redmi Watch 6 (jd) - 目标价450元
商品2: 小米15 Ultra (jd) - 目标价3200元
```

---

## 下一步

1. 确认方案（方案A go-playwright 还是方案B MCP桥接）
2. 搭建开发环境
3. 编写 Dockerfile
4. 迁移爬虫逻辑
5. 测试验证
