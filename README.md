# AI-Probe

一个基于 Go 并发的高性能 URL 健康检查器。支持并发探测、超时控制、优雅退出以及结构化 JSON 日志输出。

## ✨ 功能特性

- **并发检查**: 使用 Worker Pool 模式，支持自定义并发数，快速探测大量 URL。
- **超时控制**: 基于 `context.WithTimeout` 实现单次请求的精准超时，防止慢请求阻塞 Worker。
- **优雅退出**: 使用 `signal.NotifyContext` 监听系统信号（如 `Ctrl+C`），确保程序被中断时能安全释放资源。
- **结构化日志**: 集成 Go 标准库 `log/slog`，输出 JSON 格式日志，便于接入 ELK 或 Grafana 等监控系统。
- **配置管理**: 基于 `godotenv` 读取环境变量，实现配置与代码解耦，杜绝硬编码。

## 📁 项目结构

本项目遵循 Go 社区标准布局规范（golang-standards/project-layout）：

```text
.
├── cmd/
│   └── checker/
│       └── main.go          # 程序入口：初始化日志、加载配置、调用核心逻辑
├── internal/
│   ├── config/
│   │   ├── config.go        # 配置解析：将环境变量映射为 Config 结构体
│   │   └── config_test.go   # 配置模块的单元测试
│   └── checker/
│       ├── checker.go       # 核心逻辑：基于 Worker Pool 的并发 HTTP 探测
│       └── checker_test.go  # 核心逻辑的单元测试（基于 httptest 假服务器）
├── .env.example             # 环境变量模板
├── go.mod
└── README.md
```

## 🚀 快速开始

### 前置要求

- Go 1.21+

### 安装与运行

1. 克隆项目

   ```bash
   git clone https://github.com/username/ai-probe.git
   cd ai-probe
   ```

2. 配置环境变量

   ```bash
   cp .env.example .env
   ```

   编辑 `.env` 文件，填入你要检查的 URL 列表、超时时间（秒）以及并发数。参考模板：

   ```env
   TARGET_URLS=https://www.docker.com/,https://github.com/
   TIMEOUT_SECONDS=10
   CONCURRENCY=2
   ```

3. 运行程序

   ```bash
   go run ./cmd/checker/main.go
   ```

## 🧪 测试

本项目使用 `httptest` 进行核心逻辑的单元测试，不依赖真实的第三方网络环境，保证测试的稳定性。

```bash
go test -v -race ./...
```

## 🛠️ 设计实现

- **Worker Pool 模型**: 主协程负责投递任务（URL），多个 Worker 协程并发消费任务并发送 HTTP 请求，结果统一收敛到 `resultChan` 中进行汇总。
- **错误分类**: 准确区分 `timeout`（context 超时）、`error`（网络错误/被 WAF 拦截）以及业务层面的非 2xx 状态码，并在 `Result` 结构体中打上 `Success` 标签。
- **优雅退出**: 当收到 `SIGINT` 信号时，通过 `context` 级联取消所有正在进行的 HTTP 请求，并在退出前关闭所有通道。

## ⚠️ 已知限制与踩坑记录

- **WAF 拦截问题**: 在探测某些启用了强防护的站点（如 `chat.deepseek.com`）时，Go 标准库默认的 User-Agent 和 TLS 指纹极易被识别。实测会返回 `429 Too Many Requests` 或直接断开连接。
    - **后续优化**: 可以通过自定义 `http.Transport` 并伪装浏览器的请求头来绕过此类反爬策略。
- **网络依赖**: 单次探测的结果受限于本地网络环境，无法完全代表目标服务在全球范围的真实可用性。

## 📄 License

[MIT](https://choosealicense.com/licenses/mit/)