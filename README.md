# GopherStore（v2）

基于 Go 实现的后端基础组件集合，包含轻量 RPC、分布式缓存和简易 ORM，配套本地运行与 Docker Compose 运行方式。

## v2 新增功能

- MySQL 方言与独立数据库容器
- 运行期与初始化分离（MySQL 走 `db/init.sql`，sqlite 使用初始化脚本）
- 可观测性：Prometheus 指标 + Grafana 仪表盘自动导入
- 多节点指标采集（node1/node2）
- peers 支持完整地址配置（`node1:8001,node2:8002`）
- 压测脚本（`scripts/load_test.sh`）

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

说明：
- 仅 `node1` 暴露对外端口（`9999`），`db` 与 `node2` 只在容器内通信。
- 节点列表通过环境变量 `PEERS` 配置（例如 `PEERS=8001,8002`）。
- `PEERS` 支持完整地址（例如 `PEERS=node1:8001,node2:8002`）。
- MySQL 初始化通过 `db/init.sql` 完成，服务启动不再自动建表/写入数据。
- 监控指标：`/metrics`（Prometheus 文本格式）。

## 监控

启动后可访问：
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`（默认用户/密码：`admin` / `admin`）
Grafana 已自动导入数据源与仪表盘（`GopherStore Cache`）。

Prometheus 抓取以下两个节点的指标：
- `node1:9100`
- `node2:9100`

### 监控面板使用说明

1. 打开 Grafana `http://localhost:3000`，使用 `admin/admin` 登录（首次登录会要求改密）。
2. 进入 Dashboards，打开 `GopherStore Cache`。
3. 运行压测脚本以产生可视化数据：
   ```bash
   bash ./scripts/load_test.sh
   ```
4. 在右上角时间范围选择 `Last 5 minutes` 或 `Last 15 minutes`，观察请求速率、命中率、RPC 失败数、p95 耗时等指标。

如需在 Prometheus 手工验证，可在 `http://localhost:9090` 的 Graph 页面执行：
```promql
sum(rate(geecache_requests_total[1m]))
```

## 数据初始化（本地 sqlite）

```bash
go run ./geecache/main/init_db.go
```

如需手动指定数据库：

```bash
DB_TYPE=sqlite3 DB_DSN=custom.db go run ./geecache/main/init_db.go
```
