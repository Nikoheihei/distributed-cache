# GopherStore

A learning-oriented Go backend project that connects several common infrastructure building blocks:

- `geerpc`: a lightweight RPC implementation with registration, encoding/decoding, and timeout control.
- `geecache`: a distributed cache with LRU, consistent hashing, peer fetching, singleflight, and cache penetration/avalanche protection.
- `geeorm`: a small ORM with schema, clause, and session layers.
- Observability: Prometheus metrics and Grafana dashboards through Docker Compose.

## Architecture Overview

```
Client HTTP GET /api?key=Tom
    |
    v
HTTP gateway (node1:9999) -> group.Get("Tom")
    |
    +- LRU cache hit -> return directly
    |
    `- cache miss -> Singleflight coalesces source loads
         |
         +- consistent hash selects a remote node -> RPC peer call (CacheService.Get)
         |   +- peer local cache hit -> return
         |   `- peer local cache miss -> load from DB -> write peer cache -> return
         |
         `- no remote peer / remote failure -> local source load
              +- DB hit -> write local cache -> return
              `- DB miss -> cache empty value for 30s -> return error
```

## Quick Start with Docker Compose

```bash
docker compose up --build
```

Verify the service:

```bash
curl "http://localhost:9999/healthz"
curl "http://localhost:9999/api?key=Tom"
```

Notes:

- Only `node1:9999` is exposed as the external HTTP gateway.
- `node2` and `db` communicate inside the container network.
- MySQL is initialized from `db/init.sql`.
- The database is mapped as `3307:3306` to avoid conflicts with a local MySQL on `3306`.

## Local Run: Two Nodes plus Gateway

SQLite is used by default. MySQL can be selected through environment variables.

```bash
# node1: RPC plus API gateway
DB_TYPE=sqlite3 DB_DSN=./data/common.db PEERS=8001,8002 \
  go run ./cmd/gopherstore -port=8001 -api=true -metrics-port=9101

# node2: RPC only
DB_TYPE=sqlite3 DB_DSN=./data/common.db PEERS=8001,8002 \
  go run ./cmd/gopherstore -port=8002 -metrics-port=9102
```

## Command Entrypoints

- `cmd/gopherstore`: node process with RPC, optional HTTP gateway, and metrics.
- `cmd/init-db`: initializes sqlite/MySQL tables and demo data.
- `cmd/mock-data`: inserts large MySQL test data sets.
- `cmd/stress`: stress testing tool with Zipf hot-key distribution support.
- `cmd/smoke`: minimal concurrent connectivity smoke test.

### Initialize Data

```bash
go run ./cmd/init-db
```

Use a specific database:

```bash
DB_TYPE=sqlite3 DB_DSN=custom.db go run ./cmd/init-db
DB_TYPE=mysql   DB_DSN="root:root@tcp(127.0.0.1:3307)/gopherstore?parseTime=True" go run ./cmd/init-db
```

### Load Mock Data into MySQL

```bash
go run ./cmd/mock-data
```

Or specify the database manually:

```bash
DB_TYPE=mysql DB_DSN="root:root@tcp(127.0.0.1:3307)/gopherstore?charset=utf8mb4&parseTime=True" go run ./cmd/mock-data
```

### Stress Testing with Zipf Distribution

```bash
KEY_SPACE=10000 ZIPF_S=1.2 ZIPF_V=1 REQ_TIMEOUT=10s TOTAL_TIMEOUT=10m \
  go run ./cmd/stress
```

Common environment variables:

- `CONCURRENCY`: worker count, default `50`.
- `PER_WORKER`: requests per worker, default `200`.
- `BASE_URL`: target URL, default `http://localhost:9999`.

## Expiration and Eviction

| Mechanism | Description | Solves |
|-----------|-------------|--------|
| LRU eviction | Evicts least-recently-used entries after `maxBytes` is exceeded | Memory limits |
| Lazy TTL expiration | Checks expiration during reads, then deletes and reloads | Data freshness |
| Active TTL sweep | Background goroutine scans every TTL/4 | Memory leaks from lazy deletion |

Configure cache TTL with a flag:

```bash
go run ./cmd/gopherstore -port=8001 -api=true -cache-ttl=5m
```

Or with an environment variable:

```bash
CACHE_TTL=5m go run ./cmd/gopherstore -port=8001 -api=true
```

## Cache Protection

### Avalanche Protection: TTL Jitter

`cache.go` adds +/-10% random jitter to expiration time so keys written at the same time do not all expire at once:

```
expireAt = now + TTL + random(-TTL*10%, +TTL*10%)
```

### Penetration Protection: Empty-Value Cache

When the database has no value for a requested key, `geecache.go` caches an empty value with a short 30-second TTL to prevent repeated source loads.

### Breakdown Protection: Singleflight

`singleflight` merges concurrent requests for the same hot key into one source load, and all waiting requests share the result.

## Protocols

| Scenario | Protocol | Implementation |
|----------|----------|----------------|
| External client to gateway | HTTP | `http.go` HTTPPool + httpGetter |
| Internal node-to-node calls | Custom RPC over TCP | `rpc.go` RPCPool + CacheService |
| Message schema | Protobuf | `geecachepb/geecachepb.proto` |

HTTP uses a shared `http.Client` connection pool, and RPC clients are reused through `getClient()` to avoid creating a new connection per request.

## Monitoring

After startup, open:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` with the default account `admin/admin`

Generate sample traffic:

```bash
bash ./scripts/load_test.sh
```

Key metrics:

| Metric | Meaning |
|--------|---------|
| `gopherstore_cache_requests_total` | Total cache requests |
| `gopherstore_cache_hits_total` | Cache hits |
| `gopherstore_cache_misses_total` | Cache misses |
| `gopherstore_cache_source_requests_total` | Source-load requests |
| `gopherstore_cache_penetrations_total` | Requests where DB also misses |
| `gopherstore_cache_evictions_total` | Expiration/capacity evictions |
| `gopherstore_peer_requests_total` | Peer RPC requests |
| `gopherstore_peer_errors_total` | Peer RPC errors |
| `gopherstore_cache_load_seconds` | Source-load latency histogram |

Useful PromQL:

```promql
# Hit rate
sum(rate(gopherstore_cache_hits_total[1m])) / sum(rate(gopherstore_cache_requests_total[1m]))

# Penetration rate
sum(rate(gopherstore_cache_penetrations_total[1m])) / sum(rate(gopherstore_cache_requests_total[1m]))

# Source-load P95 latency
histogram_quantile(0.95, rate(gopherstore_cache_load_seconds_bucket[5m]))
```

---

# 中文版

# GopherStore

一个学习向的 Go 后端小项目，把几个常见组件串起来：
- `geerpc`：轻量 RPC（注册/编解码/超时）
- `geecache`：分布式缓存（LRU + 一致性哈希 + peer 拉取 + 防雪崩/穿透）
- `geeorm`：迷你 ORM（schema/clause/session）
- 可观测性：Prometheus 指标 + Grafana 面板（Docker Compose）

## 架构总览

```
客户端 HTTP GET /api?key=Tom
    │
    ▼
HTTP 网关 (node1:9999) → group.Get("Tom")
    │
    ├─ LRU 缓存命中 → 直接返回
    │
    └─ 缓存未命中 → Singleflight 合并回源
         │
         ├─ 一致性哈希选远程节点 → RPC 调用 peer (CacheService.Get)
         │   └─ peer 本地缓存命中 → 返回
         │   └─ peer 本地未命中 → getter 回源查 DB → 写入 peer 缓存 → 返回
         │
         └─ 无远程/远程失败 → 本地 getter 回源查 DB
              ├─ DB 有数据 → 写入本地缓存 → 返回
              └─ DB 无数据 → 缓存空值(30s短TTL) → 返回错误
```

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

为了避免 "一个目录里多个 main.go" 造成困惑，所有可执行入口统一放在 `cmd/`：
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

当前项目支持三层过期/淘汰机制：

| 机制 | 说明 | 解决问题 |
|------|------|---------|
| LRU 容量淘汰 | 超过 `maxBytes` 后淘汰最近最少使用的数据 | 内存容量限制 |
| TTL 过期（惰性删除） | 读取时检查过期时间，过期则删除并回源 | 数据时效性 |
| TTL 过期（主动扫描） | 后台 goroutine 每 TTL/4 扫描，清除已过期项 | 惰性删除导致的内存泄漏 |

启动节点时可以配置：

```bash
go run ./cmd/gopherstore -port=8001 -api=true -cache-ttl=5m
```

或者通过环境变量：

```bash
CACHE_TTL=5m go run ./cmd/gopherstore -port=8001 -api=true
```

## 缓存防护机制

### 缓存雪崩 → TTL Jitter

大量 key 同时过期会导致瞬间请求全部穿透到 DB。`cache.go` 在计算过期时间时加入 ±10% 的随机抖动（jitter），使同一时刻写入的 key 不会同时失效：

```
expireAt = now + TTL + random(-TTL*10%, +TTL*10%)
```

### 缓存穿透 → 空值缓存

当 DB 也查不到数据时，同一个 key 会反复穿透到 DB。`geecache.go` 的 `getlocally()` 在 DB 查无结果时，缓存一个空值并设 30s 短 TTL，防止短时间内重复穿透。

### 缓存击穿 → Singleflight

热点 key 过期瞬间，大量并发请求同时回源。`singleflight` 将同一 key 的并发请求合并为一次回源，其余请求共享结果。

## 通信协议

| 场景 | 协议 | 实现 |
|------|------|------|
| 外网（客户端→网关） | HTTP | `http.go` HTTPPool + httpGetter |
| 内网（节点间通信） | 自研 RPC (TCP) | `rpc.go` RPCPool + CacheService |
| 消息定义 | Protobuf | `geecachepb/geecachepb.proto` |

HTTP 连接使用共享 `http.Client` 连接池（MaxIdleConns=100），RPC 连接通过 `getClient()` 复用，避免每次请求创建新连接。

## 监控

启动后可访问：
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`（默认 `admin/admin`）

如果想快速产生一些曲线数据（更偏功能验证），可以跑：

```bash
bash ./scripts/load_test.sh
```

### 指标说明

| 指标 | 含义 |
|------|------|
| `gopherstore_cache_requests_total` | 总请求数 |
| `gopherstore_cache_hits_total` | 缓存命中数 |
| `gopherstore_cache_misses_total` | 缓存未命中数 |
| `gopherstore_cache_source_requests_total` | 回源（穿透到DB）次数 |
| `gopherstore_cache_penetrations_total` | DB 也查不到的穿透次数 |
| `gopherstore_cache_evictions_total` | 过期/容量淘汰次数 |
| `gopherstore_peer_requests_total` | 节点间 RPC 请求数 |
| `gopherstore_peer_errors_total` | 节点间 RPC 错误数 |
| `gopherstore_cache_load_seconds` | 回源耗时直方图 |

常用 PromQL：

```promql
# 命中率
sum(rate(gopherstore_cache_hits_total[1m])) / sum(rate(gopherstore_cache_requests_total[1m]))

# 穿透率
sum(rate(gopherstore_cache_penetrations_total[1m])) / sum(rate(gopherstore_cache_requests_total[1m]))

# 回源 P95 延迟
histogram_quantile(0.95, rate(gopherstore_cache_load_seconds_bucket[5m]))
```
