# price-monitor Dockerfile
# 构建包含 server 和 scraper 的 Docker 镜像

FROM golang:1.22-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译 server
RUN CGO_ENABLED=0 GOOS=linux go build -o price-monitor-server ./cmd/server

# 编译 scraper（需要 Playwright）
RUN apk add --no-cache python3 npm nodejs && \
    npm install -g playwright && \
    playwright install chromium --with-deps

RUN CGO_ENABLED=0 GOOS=linux go build -o price-monitor-scraper ./cmd/scraper

# 最终镜像
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata chromium-browser

# 设置环境变量
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV SCRAPER_PORT=38473
ENV SERVER_PORT=38472

# 创建应用目录
WORKDIR /app

# 复制编译产物
COPY --from=builder /app/price-monitor-server .
COPY --from=builder /app/price-monitor-scraper .

# 暴露端口
EXPOSE 38472 38473

# 启动脚本
COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh

CMD ["/app/start.sh"]