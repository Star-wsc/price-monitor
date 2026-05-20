#!/bin/sh
# 启动 scraper 服务（后台）
./price-monitor-scraper &
SCRAPER_PID=$!

# 等待 scraper 启动
sleep 2

# 启动 server 服务
./price-monitor-server

# 清理
kill $SCRAPER_PID 2>/dev/null
