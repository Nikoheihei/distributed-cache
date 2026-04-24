# GopherStore

一个学习向的 Go 后端小项目，把几个常见组件串起来：
- `geerpc`：轻量 RPC（注册/编解码/超时）
- `geecache`：分布式缓存（LRU + 一致性哈希 + peer 拉取）
- `geeorm`：迷你 ORM（schema/clause/session）
- 可观测性：Prometheus 指标 + Grafana 面板（Docker Compose）

## 快速开始（推荐：Docker Compose）

```bash
docker compose up --build
```

验证：

```bash
curl "http://localhost:9999/healthz"
curl "http://localhost:9999/api?key=Tom"
```

说明：
- 对外只暴露 `node1` 的 `9999`（HTTP 网关），`node2` 和 `db` 只在容器网络内通信。
- MySQL 初始化来自 `db/init.sql`。
- `db` 端口映射为 `3307:3306`（避免占用你本机已有的 `3306`）。

## 本地运行（两节点 + 网关）

默认用 sqlite（通过环境变量可切 MySQL）。

```bash
# node1（RPC + API 网关）
DB_TYPE=sqlite3 DB_DSN=./data/common.db PEERS=8001,8002 \
  go run ./cmd/gopherstore -port=8001 -api=true -metrics-port=9101

# node2（仅 RPC）
DB_TYPE=sqlite3 DB_DSN=./data/common.db PEERS=8001,8002 \
  go run ./cmd/gopherstore -port=8002 -metrics-port=9102
```

## 命令入口（cmd/）

为了避免 “一个目录里多个 main.go” 造成困惑，所有可执行入口统一放在 `cmd/`：
- `cmd/gopherstore`：节点进程（RPC + 可选 HTTP 网关 + metrics）
- `cmd/init-db`：初始化 sqlite/MySQL 表和少量演示数据
- `cmd/mock-data`：往 MySQL 灌入大量 `UserN` 测试数据（可选）
- `cmd/stress`：压测工具（支持 Zipf 热点分布）
- `cmd/smoke`：极简并发连通性测试

### 初始化数据（本地 sqlite 或 MySQL）

```bash
go run ./cmd/init-db
```

指定数据库：

```bash
DB_TYPE=sqlite3 DB_DSN=custom.db go run ./cmd/init-db
DB_TYPE=mysql   DB_DSN="root:root@tcp(127.0.0.1:3307)/gopherstore?parseTime=True" go run ./cmd/init-db
```

### 灌入大量数据（MySQL）

```bash
go run ./cmd/mock-data
```

也可以手动指定：

```bash
DB_TYPE=mysql DB_DSN="root:root@tcp(127.0.0.1:3307)/gopherstore?charset=utf8mb4&parseTime=True" go run ./cmd/mock-data
```

### 压测（Zipf 热点）

Zipf 用来模拟类似 80/20 的热点访问分布：

```bash
KEY_SPACE=10000 ZIPF_S=1.2 ZIPF_V=1 REQ_TIMEOUT=10s TOTAL_TIMEOUT=10m \
  go run ./cmd/stress
```

常用环境变量：
- `CONCURRENCY`：并发 worker 数（默认 50）
- `PER_WORKER`：每个 worker 请求数（默认 200）
- `BASE_URL`：目标地址（默认 `http://localhost:9999`）

## 缓存过期策略

当前项目同时支持：
- LRU 容量淘汰：超过 `maxBytes` 后淘汰最近最少使用的数据
- 可选 TTL 过期：到期后在读取时判定失效

启动节点时可以配置：

```bash
go run ./cmd/gopherstore -port=8001 -api=true -cache-ttl=5m
```

或者通过环境变量：

```bash
CACHE_TTL=5m go run ./cmd/gopherstore -port=8001 -api=true
```

## 监控

启动后可访问：
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`（默认 `admin/admin`）

如果想快速产生一些曲线数据（更偏功能验证），可以跑：

```bash
bash ./scripts/load_test.sh
```

压测/请求后可在 Prometheus 里看：

```promql
sum(rate(geecache_requests_total[1m]))
```
