# ai-probe

一个基于 Go + Gin + GORM 的高并发服务健康探测组件。支持并发探测 URL 可用性、结构化日志、崩溃兜底、优雅退出、数据持久化与分页查询。

## ✨ 功能特性

- **并发探测**：基于 Worker Pool 模式，支持自定义并发数，快速探测大量 URL。
- **超时控制**：基于 `context.WithTimeout` 实现单次请求精准超时，并支持级联取消。
- **优雅退出**：使用 `signal.NotifyContext` 监听系统信号，配合 `srv.Shutdown` 实现平滑关闭。
- **结构化日志**：集成 `log/slog`，输出 JSON 格式访问日志，便于接入 ELK / Loki。
- **崩溃兜底**：自定义 Recovery 中间件，捕获 panic，返回 500 并记录堆栈，服务不崩。
- **数据持久化**：使用 GORM + MySQL 保存每次检查结果，支持分页查询历史记录。
- **连接池配置**：合理配置 `MaxOpenConns`、`MaxIdleConns`、`ConnMaxLifetime`，提升数据库性能。
- **参数安全**：分页参数做上下限校验，防止恶意大 `pageSize` 打垮数据库。

## 📁 项目结构

```text
.
├── cmd/
│   └── checker/
│       └── main.go               # 程序入口：组装配置、DB、路由、优雅退出
├── internal/
│   ├── checker/
│   │   ├── checker.go            # 并发探测核心逻辑（Worker Pool + Context）
│   │   └── checker_test.go       # 核心逻辑单元测试（httptest）
│   ├── config/
│   │   ├── config.go             # 环境变量解析为 Config 结构体
│   │   └── config_test.go        # 配置模块单元测试
│   ├── handler/
│   │   └── check.go              # HTTP 处理器：CheckHandler、GetResultsHandler
│   ├── middleware/
│   │   ├── logger.go             # SlogMiddleware：结构化访问日志
│   │   └── recovery.go           # RecoveryMiddleware：panic 兜底
│   ├── model/
│   │   └── check_result.go       # GORM 数据模型
│   └── repository/
│       ├── db.go                 # InitDB：连接 MySQL、AutoMigrate、连接池
│       └── check_repo.go         # CheckRepository：SaveResults、GetResults
├── .env.example                  # 环境变量模板
├── go.mod
└── README.md
```

## 🚀 快速开始

### 前置要求

- Go 1.21+
- MySQL 8.0（推荐使用 Docker 启动）
- Docker Desktop（用于运行 MySQL 容器）

### 1. 启动 MySQL（Docker）

```bash
docker run --name ai-probe-mysql \
  -e MYSQL_ROOT_PASSWORD=123456 \
  -e MYSQL_DATABASE=ai_probe \
  -p 3306:3306 \
  -d mysql:8.0
```

### 2. 克隆项目并配置环境变量

```bash
git clone https://github.com/your-username/ai-probe.git
cd ai-probe
cp .env.example .env
```

编辑 `.env`，填入你的配置（参考 `.env.example`）：

```env
TARGET_URLS=https://www.docker.com/,https://github.com/
TIMEOUT_SECONDS=10
CONCURRENCY=2
DB_DSN=root:123456@tcp(127.0.0.1:3306)/ai_probe?charset=utf8mb4&parseTime=True&loc=Local
```

### 3. 运行服务

```bash
go run ./cmd/checker/main.go
```

服务默认监听 `:8080`。

## 🧪 测试

```bash
go test -v -race ./...
```

## 📡 API 文档

### POST /api/v1/check

发起一次 URL 健康检查。

**请求体**：

```json
{
  "urls": ["https://github.com", "https://www.docker.com"]
}
```

**响应示例**：

```json
[
  {
    "URL": "https://github.com",
    "Status": "200 OK",
    "Duration": "890.5903ms",
    "Err": null,
    "Success": true
  }
]
```

### GET /api/v1/results

分页查询历史检查记录。

**查询参数**：

| 参数 | 默认值 | 说明 |
|---|---|---|
| `page` | 1 | 页码，最小 1 |
| `pageSize` | 20 | 每页条数，范围 1-100 |

**响应示例**：

```json
[
  {
    "ID": 1,
    "URL": "https://github.com",
    "Status": "200 OK",
    "Success": true,
    "Duration": 890590300,
    "ErrMsg": "",
    "CreatedAt": "2026-10-14T14:30:00+08:00"
  }
]
```

## 🛠️ 设计实现

- **Worker Pool**：主协程投递任务，多个 worker 并发消费，结果收敛到 `resultChan`。
- **Context 级联**：从 `c.Request.Context()` 到 `http.NewRequestWithContext` 到 GORM `WithContext`，实现全链路取消。
- **中间件洋葱模型**：`SlogMiddleware` 和 `RecoveryMiddleware` 按顺序包裹业务 Handler。
- **错误分层**：业务错误返回 400，数据库/服务端错误返回 500；对外模糊，对内详细。
- **连接池参数**：`SetMaxOpenConns(100)`、`SetMaxIdleConns(100)`、`SetConnMaxLifetime(time.Hour)`、`SetConnMaxIdleTime(10*time.Minute)`。

## ⚠️ 已知限制与踩坑记录

- **WAF 拦截**：部分站点（如 `chat.deepseek.com`）会识别 Go 默认 User-Agent 和 TLS 指纹，返回 429 或直接断开。后续可通过自定义 `http.Transport` 伪装浏览器请求头绕过。
- **分页边界**：客户端传入超大 `pageSize` 时必须强制限制（已实现上限 100）。
- **时区与 parseTime**：DSN 必须包含 `parseTime=True&loc=Local`，否则 `time.Time` 类型读写会出错。
- **GORM 慢 SQL 日志**：默认输出到标准输出，与 JSON 日志混在一起，后续可配置 GORM 使用 `slog` 或调高阈值。

## 🐳 Docker 化部署

本项目已完成容器化，使用 Docker Compose 编排 Go 服务与 MySQL 数据库。**一条命令即可启动全套服务。**

### 架构

```text
┌────────────────────────────────────────────────┐
│ Docker Compose Network (my-checker_default)    │
│                                                │
│  ┌──────────────────┐    ┌──────────────────┐  │
│  │ ai_probe         │───→│ mysql            │  │
│  │ (Go 服务)        │    │ (MySQL 8.0)      │  │
│  │ 容器内监听 :8080 │    │ 容器内监听 :3306 │  │
│  └────────┬─────────┘    └────────┬─────────┘  │
│           │                       │            │
│           │                       │            │
│  ┌────────▼───────────────────────▼─────────┐  │
│  │ Volume: mysql_data                       │  │
│  │ (MySQL 数据持久化到宿主机)                  │  │
│  └──────────────────────────────────────────┘  │
└────────────────────────────────────────────────┘
           │
      宿主机端口 8000
           │
        localhost:8000
```

### 一键启动

**前置要求**：Docker Desktop 已启动。

```bash
# 构建镜像并启动所有服务
docker compose up --build
```

**启动流程**：
1. 创建 `mysql` 容器，执行初始化（建库 `ai_probe`、建用户）。
2. `healthcheck` 每 10 秒探测一次 MySQL 是否就绪。
3. MySQL 状态变为 `healthy` 后，才启动 `ai_probe` 容器。
4. Go 服务连上 `mysql:3306`，开始监听 8080。
5. 宿主机可通过 `localhost:8000` 访问。

**首次启动约需 30-60 秒**（等 MySQL 初始化）。

### 访问接口

宿主机端口为 **8000**（映射到容器内的 8080）：

```bash
# 发起检查
curl -X POST http://localhost:8000/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://github.com"]}'

# 查询历史
curl http://localhost:8000/api/v1/results
```

### 常用命令

```bash
# 查看容器状态
docker compose ps

# 查看日志
docker compose logs -f ai_probe
docker compose logs -f mysql

# 停止服务（保留数据）
docker compose down

# 停止服务并删除数据卷（谨慎使用）
docker compose down -v
```

### 关键设计说明

| 设计点 | 实现方式 | 目的 |
|---|---|---|
| **多阶段构建** | builder（编译）+ alpine（运行） | 镜像从 800MB 压到 22MB |
| **静态编译** | `CGO_ENABLED=0` | 二进制不依赖 glibc，能在 alpine 运行 |
| **服务发现** | DSN 用 `mysql:3306` 而非 `127.0.0.1` | Compose 网络内通过服务名互访 |
| **健康检查** | `mysqladmin ping -h 127.0.0.1` | 强制走 TCP，避免 socket 假就绪 |
| **启动顺序** | `depends_on` + `condition: service_healthy` | 防止 Go 服务在 MySQL 就绪前启动 |
| **数据持久化** | 命名卷 `mysql_data` | 容器删除后数据不丢 |
| **环境变量注入** | compose 的 `environment` 字段 | 与本地 `.env` 开发解耦 |

### 配置说明

Docker 环境下，配置**不再来自 `.env` 文件**，而是由 `docker-compose.yml` 的 `environment` 字段注入：

```yaml
environment:
  - TIMEOUT_SECONDS=10
  - CONCURRENCY=2
  - DB_DSN=root:123456@tcp(mysql:3306)/ai_probe?charset=utf8mb4&parseTime=True&loc=Local
```

- `.env` 文件已被 `.dockerignore` 排除，**不会打进镜像**。
- 代码里的 `godotenv.Load()` 在找不到 `.env` 时会**降级为 warning**，继续使用系统环境变量。

### 常见问题

**Q：为什么端口是 8000，不是 8080？**
A：`ports: "8000:8080"` 表示宿主机用 8000，容器内服务监听 8080。避免和本机其他服务撞端口。

**Q：数据存在哪里？**
A：命名卷 `mysql_data`。可以用 `docker volume inspect my-checker_mysql_data` 查看具体位置。

**Q：怎么连数据库调试？**
A：`mysql` 服务默认**不暴露端口给宿主机**（更安全）。如需本地连接，在 `docker-compose.yml` 的 `mysql` 服务下加：
```yaml
ports:
  - "3307:3306"
```
然后用 Navicat/DBeaver 连 `localhost:3307`。

**Q：`connection refused` 怎么排查？**
A：通常是 MySQL 还没就绪。检查：
1. `docker compose ps` —— mysql 是不是 `healthy`？
2. `docker compose logs mysql` —— 有没有 `ready for connections`？
3. `healthcheck` 的 `test` 是否用了 `-h 127.0.0.1`（而不是 `localhost`）？

## 📄 License

[MIT](https://choosealicense.com/licenses/mit/)