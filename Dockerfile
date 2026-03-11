# 第一阶段：编译
FROM golang:1.24-alpine AS builder

# 安装 sqlite 编译需要的依赖（CGO 相关）
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

RUN go env -w GOPROXY=https://goproxy.cn,direct

COPY . .

# 编译 main.go。注意：因为用了 sqlite (CGO)，必须开启 CGO_ENABLED=1
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o gopherstore ./geecache/main/main.go

# 第二阶段：运行
FROM alpine:latest
RUN apk add --no-cache sqlite-libs

WORKDIR /app
RUN mkdir -p /app/data
COPY --from=builder /app/gopherstore .

# 暴露 RPC 端口和 Web 端口
EXPOSE 8001 9999

# 启动命令在 docker-compose 中定义
ENTRYPOINT ["./gopherstore"]