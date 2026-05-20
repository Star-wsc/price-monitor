# price-monitor 构建脚本
# 用于编译 server 和 scraper 服务

$ErrorActionPreference = "Stop"

Write-Host "开始构建 price-monitor..." -ForegroundColor Cyan

# 创建 build 目录
$buildDir = "build"
if (-not (Test-Path $buildDir)) {
    New-Item -ItemType Directory -Path $buildDir | Out-Null
}

# 编译 server
Write-Host "`n编译 server..." -ForegroundColor Yellow
go build -o build/price-monitor-server ./cmd/server
if ($LASTEXITCODE -ne 0) {
    Write-Host "server 编译失败" -ForegroundColor Red
    exit 1
}
Write-Host "server 编译完成: build/price-monitor-server" -ForegroundColor Green

# 编译 scraper
Write-Host "`n编译 scraper..." -ForegroundColor Yellow
go build -o build/price-monitor-scraper ./cmd/scraper
if ($LASTEXITCODE -ne 0) {
    Write-Host "scraper 编译失败" -ForegroundColor Red
    exit 1
}
Write-Host "scraper 编译完成: build/price-monitor-scraper" -ForegroundColor Green

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "构建完成！" -ForegroundColor Green
Write-Host "输出文件:" -ForegroundColor White
Write-Host "  - build/price-monitor-server   (API 服务，端口 38472)" -ForegroundColor White
Write-Host "  - build/price-monitor-scraper  (爬虫服务，端口 38473)" -ForegroundColor White
Write-Host "========================================" -ForegroundColor Cyan