# GopherStore

基于 Go 实现的后端基础组件集合，包含轻量 RPC、分布式缓存和简易 ORM，配套本地运行与 Docker Compose 运行方式。

## 已实现功能

- 分布式缓存：一致性哈希、节点路由、HTTP/RPC 两种节点通信方式、缓存回源
- 轻量 RPC：服务注册、请求/响应编解码、客户端/服务端、超时控制
- 简易 ORM：schema 映射、SQL 子句构造、CRUD、链式查询
- HTTP 网关：对外提供访问入口
- Docker 化：支持 Compose 一键拉起多节点

## 模块说明

- `geerpc`：轻量 RPC 框架（服务注册、codec、client/server）。
- `geecache`：分布式缓存（peer 选择、HTTP/RPC 远程拉取）。
- `geeorm`：简易 ORM（schema 映射、clause builder、session）。
- `gee`：简易 Web 框架（施工中）。

## 本地运行

启动两个缓存节点和一个 HTTP 网关：

```bash
# node1（RPC + API）
go run ./geecache/main/main.go -port=8001 -api=true -db=./data/common.db

# node2（RPC）
go run ./geecache/main/main.go -port=8002 -db=./data/common.db
```

访问：

```bash
curl "http://localhost:9999/api?key=Tom"
```

## Docker Compose

```bash
docker compose up --build
```

访问：

```bash
curl "http://localhost:9999/api?key=Tom"
```

## 目录结构

```text
geerpc/     轻量 RPC 框架
geecache/   分布式缓存
geeorm/     简易 ORM
gee/        简易 Web 框架（施工中）
```

## 测试

```bash
go test ./geeorm/session -run .
```
