# ---- 构建阶段 ----
FROM golang:1.23-alpine AS builder

WORKDIR /src

# 先复制依赖清单，利用 Docker 层缓存加速构建
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译为静态二进制
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/mdns-scan .

# ---- 运行阶段 ----
# 用 alpine 保留基本工具（便于排查），如需极致精简可换 scratch。
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/mdns-scan /usr/local/bin/mdns-scan

# mDNS 依赖 UDP 组播 5353，运行时需 --network host 或 --cap-add=NET_ADMIN
EXPOSE 5353/udp

ENTRYPOINT ["mdns-scan"]
