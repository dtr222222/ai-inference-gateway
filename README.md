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

## 📄 License

[MIT](https://choosealicense.com/licenses/mit/)