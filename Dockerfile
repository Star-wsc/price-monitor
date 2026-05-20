# price-monitor Dockerfile
FROM golang:1.22 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 编译 server 和 scraper（CGO_ENABLED=1 因为 go-sqlite3 需要）
RUN CGO_ENABLED=1 GOOS=linux go build -o price-monitor-server ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -o price-monitor-scraper ./cmd/scraper

# 最终镜像 - Debian slim（Chromium 更可靠）
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    chromium \
    chromium-shell \
    && rm -rf /var/lib/apt/lists/*

ENV CHROME_BIN=/usr/bin/chromium
ENV SCRAPER_PORT=38473
ENV SERVER_PORT=38472

WORKDIR /app

COPY --from=builder /app/price-monitor-server .
COPY --from=builder /app/price-monitor-scraper .
COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh

EXPOSE 38472 38473

CMD ["/app/start.sh"]
