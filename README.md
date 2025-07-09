# Trace - 分布式追踪和日志库

这是一个完整的分布式追踪和日志解决方案，集成了 Trace ID/Span ID 生成、HTTP 追踪上下文传递、结构化日志记录、日志聚合系统、Jaeger 集成和邮件通知功能。

## 🚀 功能特性

### 核心功能
- ✅ **Trace ID 和 Span ID 生成**：基于 UUID 的方案，符合 OpenTelemetry 规范
- ✅ **HTTP 追踪上下文传递**：自动在服务间传递追踪信息
- ✅ **结构化日志记录**：基于 logrus，支持多种格式和级别
- ✅ **日志聚合系统**：专门针对大规模日志处理进行优化，能够处理单服务单天 10G+ 的日志量
- ✅ **Jaeger 集成**：通过 OpenTelemetry 协议将追踪数据发送到 Jaeger
- ✅ **邮件通知功能**：重要错误自动邮件通知
- ✅ **分布式追踪支持**：完整的调用链路追踪
- ✅ **Web 界面**：提供直观的 Web 界面进行日志管理和查询

### 日志功能
- ✅ 支持多种日志级别（Debug, Info, Warn, Error, Fatal, Panic）
- ✅ 支持文本和 JSON 格式输出
- ✅ 支持文件输出和标准输出
- ✅ 支持结构化日志（带字段）
- ✅ 支持追踪上下文（Trace ID, Span ID）
- ✅ 支持调用者信息
- ✅ 支持便捷的初始化方法
- ✅ 支持邮件通知功能（Error, Fatal, Panic 级别）

### 大规模日志处理能力
- **🚀 分片轮转**：按大小分片，避免单个文件过大
- **🚀 索引机制**：使用 BoltDB 建立内存索引，加速查询
- **🚀 批量写入**：批量处理日志写入，提高性能
- **🚀 并发安全**：支持高并发写入和查询
- **🚀 自动压缩**：自动压缩历史日志文件
- **🚀 后台任务**：异步处理清理和压缩任务

## 📊 性能指标

| 功能 | 性能指标 | 说明 |
|------|----------|------|
| 写入速度 | 10,000+ 条/秒 | 批量写入 + 缓冲优化 |
| 查询速度 | 索引查询 10-100x 更快 | BoltDB 索引 + 文件偏移定位 |
| 并发能力 | 100+ 并发查询 | 读写锁分离 + 异步处理 |
| 存储效率 | 压缩节省 70%+ 空间 | 自动 gzip 压缩 |
| 文件管理 | 自动分片轮转 | 避免单个文件过大 |
| 内存使用 | 低内存占用 | 流式处理 + 批量操作 |

## 🚀 快速开始

### 1. 安装依赖

```bash
go get github.com/HsiaoL1/trace
```

### 2. 基本使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/HsiaoL1/trace"
    "github.com/HsiaoL1/trace/logz"
)

func main() {
    // 初始化日志
    logz.InitDevelopment()
    
    // 生成 Trace ID 和 Span ID
    traceID := trace.GenerateTraceID()
    spanID := trace.GenerateSpanID()
    
    fmt.Printf("Trace ID: %s\n", traceID.String())
    fmt.Printf("Span ID: %s\n", spanID.String())
    
    // 记录日志
    logz.Info("应用启动")
    logz.WithField("trace_id", traceID.String()).Info("追踪信息")
}
```

### 3. 大规模日志处理

```go
package main

import (
    "log"
    "github.com/HsiaoL1/trace/logz"
)

func main() {
    // 初始化带聚合功能的日志系统
    err := logz.InitWithAggregation(
        "./logs/app.log",           // 普通日志文件
        "./logs/aggregated",        // 聚合日志目录
        "user-service",             // 服务名
        500*1024*1024,             // 轮转大小 (500MB)
        50,                        // 最大备份数
    )
    if err != nil {
        log.Fatalf("初始化日志系统失败: %v", err)
    }
    defer logz.CloseAggregator()

    // 使用日志方法（会自动聚合）
    logz.Info("应用启动")
    logz.InfoWithTrace("trace-001", "span-001", "处理用户请求")
    logz.Error("发生错误")
}
```

### 4. Jaeger 集成

```go
package main

import (
    "context"
    "log"
    "github.com/HsiaoL1/trace"
)

func main() {
    // 设置环境变量
    os.Setenv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces")
    os.Setenv("JAEGER_SERVICE_NAME", "my-service")
    os.Setenv("JAEGER_ENABLED", "true")
    
    // 加载配置
    config := trace.LoadConfigFromEnv()
    
    // 初始化 Jaeger
    cleanup, err := trace.InitJaeger(&config.Jaeger)
    if err != nil {
        log.Fatalf("Failed to initialize Jaeger: %v", err)
    }
    defer cleanup()
    
    // 创建 span
    ctx := context.Background()
    ctx, span := trace.StartSpan(ctx, "my-operation")
    defer span.End()
    
    // 设置属性
    trace.SetAttribute(span, "user.id", "12345")
    
    // 记录事件
    trace.AddEvent(span, "processing started")
}
```

## 🌐 Web 界面

### 启动 Web 界面

```bash
# 进入 Web 目录
cd logz/web

# 启动演示（包含日志生成器和 Web 服务器）
./demo.sh

# 或者分别启动
# 1. 启动日志生成器
go run demo.go &

# 2. 启动 Web 服务器
go run main.go
```

然后打开浏览器访问：http://localhost:8080

### Web 界面功能

#### 主页面功能
- **统计信息**：显示总文件数、总大小、最早/最新文件
- **高级搜索**：支持按 Trace ID、Span ID、级别、服务、消息内容、时间范围搜索
- **文件列表**：显示所有日志文件，支持查看和删除操作
- **实时刷新**：自动更新文件列表和统计信息

#### 日志查看页面
- **分页浏览**：支持大文件的分页查看
- **内容搜索**：在文件内容中搜索关键词
- **级别过滤**：按日志级别过滤显示
- **自动刷新**：实时监控日志文件变化
- **文件下载**：下载完整的日志文件
- **语法高亮**：根据日志级别显示不同颜色

#### 错误日志页面
- **错误统计**：显示今日、本周错误数量
- **错误列表**：专门展示 error 级别的日志
- **服务过滤**：按服务名过滤错误
- **时间范围**：支持多种时间范围过滤
- **错误详情**：查看完整的错误信息
- **导出功能**：导出错误日志为 CSV 格式

## 📋 Trace ID 和 Span ID 生成

### 基于 UUID 的方案（推荐）✅

**特点：**
- 使用 `crypto/rand` 生成16字节的 Trace ID 和8字节的 Span ID
- 符合 OpenTelemetry 规范
- 全局唯一性保证
- 高随机性，避免冲突

**优势：**
- 标准化的实现
- 与主流追踪系统兼容
- 性能优秀
- 安全性高

**适用场景：**
- 生产环境
- 需要与 OpenTelemetry 集成的系统
- 高并发场景

### 使用示例

```go
package main

import (
    "fmt"
    "github.com/HsiaoL1/trace"
)

func main() {
    // 生成 Trace ID
    traceID := trace.GenerateTraceID()
    fmt.Printf("Trace ID: %s\n", traceID.String())
    
    // 生成 Span ID
    spanID := trace.GenerateSpanID()
    fmt.Printf("Span ID: %s\n", spanID.String())
    
    // 验证有效性
    if traceID.IsValid() && spanID.IsValid() {
        fmt.Println("IDs are valid")
    }
}
```

## 🌐 HTTP 追踪功能

### 核心功能

1. **TraceContext 结构体**
```go
type TraceContext struct {
    TraceID      string // 追踪ID
    SpanID       string // 当前Span ID
    ParentSpanID string // 父Span ID
}
```

2. **HTTP 头部常量**
```go
const (
    TraceIDHeader      = "X-Trace-ID"
    SpanIDHeader       = "X-Span-ID"
    ParentSpanIDHeader = "X-Parent-Span-ID"
)
```

### 使用示例

#### 1. 服务间调用追踪

**上游服务（服务A）：**
```go
package main

import (
    "context"
    "net/http"
    "time"
    "github.com/HsiaoL1/trace"
)

func main() {
    // 创建带追踪功能的 HTTP 客户端
    client := trace.NewTracedHTTPClient(10 * time.Second)
    
    // 创建根 span（模拟外部请求）
    rootCtx := context.Background()
    rootTraceCtx := trace.CreateRootSpan()
    ctx := trace.WithTraceContext(rootCtx, rootTraceCtx)
    
    // 调用下游服务
    req, err := http.NewRequestWithContext(ctx, "GET", "http://service-b/api/data", nil)
    if err != nil {
        return
    }
    
    // 自动传递追踪上下文
    resp, err := client.Do(ctx, req)
    // 处理响应...
}
```

**下游服务（服务B）：**
```go
package main

import (
    "net/http"
    "github.com/HsiaoL1/trace"
)

func main() {
    mux := http.NewServeMux()
    
    // 使用追踪中间件
    mux.Handle("/api/data", trace.HTTPMiddleware(http.HandlerFunc(handleRequest)))
    
    http.ListenAndServe(":8080", mux)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 从 context 中获取追踪上下文
    traceCtx := trace.GetTraceContextFromContext(r.Context())
    
    // 现在可以访问：
    // traceCtx.TraceID      - 追踪ID（与上游相同）
    // traceCtx.SpanID       - 当前Span ID（新生成的）
    // traceCtx.ParentSpanID - 父Span ID（上游的Span ID）
    
    // 处理业务逻辑...
    
    // 返回响应
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status": "ok"}`))
}
```

#### 2. 手动设置 HTTP 头部

```go
// 设置单个头部
trace.SetTraceIDToHttpHeader(ctx, "abc123")
trace.SetSpanIDToHttpHeader(ctx, "def456")

// 设置完整的追踪上下文
traceCtx := trace.TraceContext{
    TraceID:      "abc123",
    SpanID:       "def456",
    ParentSpanID: "ghi789",
}
trace.SetTraceContextToHttpHeader(ctx, traceCtx)
```

#### 3. 从 HTTP 头部获取追踪上下文

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 从 HTTP 头部获取追踪上下文
    traceCtx := trace.GetTraceContextFromHttpHeader(r)
    
    // 创建子 span
    childTraceCtx := trace.CreateChildSpan(traceCtx)
    
    // 将追踪上下文注入到 context 中
    ctx := trace.WithTraceContext(r.Context(), childTraceCtx)
    
    // 使用新的 context 处理请求
    processRequest(ctx)
}
```

## 🔧 Jaeger 集成

### 快速开始

#### 1. 启动 Jaeger

使用 Docker 启动 Jaeger all-in-one：

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 5775:5775/udp \
  jaegertracing/all-in-one:latest
```

#### 2. 环境变量配置

```bash
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
export JAEGER_SERVICE_NAME="my-service"
export JAEGER_ENVIRONMENT="development"
export JAEGER_VERSION="1.0.0"
export JAEGER_ENABLED="true"
```

#### 3. 基本使用

```go
package main

import (
    "context"
    "log"
    "github.com/HsiaoL1/trace"
)

func main() {
    // 加载配置
    config := trace.LoadConfigFromEnv()
    
    // 初始化 Jaeger
    cleanup, err := trace.InitJaeger(&config.Jaeger)
    if err != nil {
        log.Fatalf("Failed to initialize Jaeger: %v", err)
    }
    defer cleanup()
    
    // 创建 span
    ctx := context.Background()
    ctx, span := trace.StartSpan(ctx, "my-operation")
    defer span.End()
    
    // 设置属性
    trace.SetAttribute(span, "user.id", "12345")
    
    // 记录事件
    trace.AddEvent(span, "processing started")
    
    // 你的业务逻辑...
}
```

### API 使用说明

#### 初始化

```go
// 使用默认配置
config := trace.DefaultJaegerConfig()

// 或从环境变量加载
config := trace.LoadJaegerConfigFromEnv()

// 初始化 Jaeger
cleanup, err := trace.InitJaeger(config)
if err != nil {
    log.Fatal(err)
}
defer cleanup()
```

#### 创建 Span

```go
// 创建根 span
ctx, span := trace.StartSpan(context.Background(), "operation-name")
defer span.End()

// 创建子 span
childCtx, childSpan := trace.StartSpan(ctx, "child-operation")
defer childSpan.End()
```

#### 设置属性和事件

```go
// 设置属性
trace.SetAttribute(span, "user.id", "12345")
trace.SetAttribute(span, "request.size", 1024)

// 添加事件
trace.AddEvent(span, "cache miss")
trace.AddEvent(span, "database query started")

// 记录错误
err := someOperation()
if err != nil {
    trace.RecordError(span, err)
}
```

#### HTTP 中间件

```go
// 使用 OpenTelemetry 中间件
mux := http.NewServeMux()
mux.HandleFunc("/api/endpoint", handler)

// 包装中间件
wrappedHandler := trace.OpenTelemetryMiddleware(mux)

server := &http.Server{
    Addr:    ":8080",
    Handler: wrappedHandler,
}
```

#### HTTP 客户端

```go
// 创建带追踪的 HTTP 客户端
client := trace.NewTracedHTTPClient(10 * time.Second)

// 发送请求（自动传播追踪上下文）
resp, err := client.Get(ctx, "http://api.example.com/users")
```

### 配置选项

#### 环境变量

| 变量名 | 默认值 | 描述 |
|--------|---------|------|
| `JAEGER_ENDPOINT` | `http://localhost:14268/api/traces` | Jaeger 收集器端点 |
| `JAEGER_SERVICE_NAME` | `trace-service` | 服务名称 |
| `JAEGER_ENVIRONMENT` | `development` | 环境名称 |
| `JAEGER_VERSION` | `1.0.0` | 服务版本 |
| `JAEGER_ENABLED` | `true` | 是否启用 Jaeger |
| `TRACE_LOG_LEVEL` | `info` | 日志级别 |
| `TRACE_SAMPLING_RATIO` | `1.0` | 采样比例 (0.0-1.0) |

#### 程序配置

```go
config := &trace.JaegerConfig{
    Endpoint:    "http://localhost:14268/api/traces",
    ServiceName: "my-service",
    Environment: "production",
    Version:     "2.0.0",
    Enabled:     true,
}
```

## 📝 日志功能 (Logz)

### 基本使用

```go
package main

import (
    "github.com/HsiaoL1/trace/logz"
)

func main() {
    // 初始化日志配置
    logz.InitDevelopment()
    
    // 基本日志方法
    logz.Info("应用启动")
    logz.Debug("调试信息")
    logz.Warn("警告信息")
    logz.Error("错误信息")
    
    // 格式化日志
    logz.Infof("用户 %s 登录成功", "张三")
    logz.Errorf("处理请求失败: %v", err)
}
```

### 设置日志级别

```go
// 设置日志级别
logz.SetLevel(logz.LevelDebug)  // 显示所有日志
logz.SetLevel(logz.LevelInfo)   // 只显示 Info 及以上级别
logz.SetLevel(logz.LevelError)  // 只显示 Error 及以上级别
```

### 设置日志格式

```go
// 文本格式（默认）
logz.SetFormat(logz.FormatText)

// JSON 格式
logz.SetFormat(logz.FormatJSON)
```

### 设置输出位置

```go
// 输出到标准输出（默认）
logz.SetOutput(os.Stdout)

// 输出到文件
err := logz.SetFileOutput("/var/log/app.log")
if err != nil {
    log.Fatal(err)
}
```

### 结构化日志

```go
// 添加单个字段
logz.WithField("user_id", "123").Info("用户登录")

// 添加多个字段
fields := logrus.Fields{
    "user_id": "123",
    "action":  "login",
    "ip":      "192.168.1.1",
    "time":    time.Now().Format("2006-01-02 15:04:05"),
}
logz.WithFields(fields).Info("用户操作")

// 添加错误字段
err := errors.New("数据库连接失败")
logz.WithError(err).Error("系统错误")
```

### 带追踪上下文的日志

```go
traceID := "abc123def456"
spanID := "span789"

// 带追踪上下文的日志方法
logz.InfoWithTrace(traceID, spanID, "处理用户请求")
logz.DebugWithTrace(traceID, spanID, "查询数据库")
logz.ErrorWithTrace(traceID, spanID, "数据库查询失败")

// 格式化版本
logz.InfofWithTrace(traceID, spanID, "用户 %s 的操作", "李四")
logz.ErrorfWithTrace(traceID, spanID, "处理失败: %v", err)
```

### 启用调用者信息

```go
// 启用调用者信息（显示文件名和行号）
logz.EnableCaller()

// 禁用调用者信息
logz.DisableCaller()
```

## 📊 大规模日志聚合系统

### 日志聚合功能

```go
// 手动创建聚合器
aggregator, err := logz.NewLogAggregator(
    "./logs/aggregated",  // 输出目录
    "my-service",         // 服务名
    500*1024*1024,       // 轮转大小 (500MB)
    50,                  // 最大备份数
)
if err != nil {
    log.Fatal(err)
}
defer aggregator.Close()

// 手动写入日志
entry := logz.LogEntry{
    Timestamp: time.Now().Format(time.RFC3339),
    Level:     "info",
    Message:   "用户登录成功",
    TraceID:   "trace-001",
    SpanID:    "span-001",
    Service:   "my-service",
}
aggregator.WriteLog(entry)
```

### 高性能查询功能

#### 1. 使用索引的快速查询

```go
// 使用索引的快速查询
result, err := logz.QueryLogsByTraceID("trace-001", "./logs/aggregated", 10, 0)
if err != nil {
    log.Printf("查询失败: %v", err)
} else {
    fmt.Printf("找到 %d 条日志\n", result.Total)
    for _, entry := range result.Entries {
        fmt.Printf("[%s] %s\n", entry.Level, entry.Message)
    }
}
```

#### 2. 按时间范围查询

```go
startTime := time.Now().Add(-1 * time.Hour)
endTime := time.Now()
result, err := logz.QueryLogsByTimeRange(startTime, endTime, "./logs/aggregated", 10, 0)
```

#### 3. 按日志级别查询

```go
result, err := logz.QueryLogsByLevel("error", "./logs/aggregated", 10, 0)
```

#### 4. 按服务名查询

```go
result, err := logz.QueryLogsByService("user-service", "./logs/aggregated", 10, 0)
```

#### 5. 按消息内容查询（支持正则表达式）

```go
result, err := logz.QueryLogsByMessage(".*登录.*", "./logs/aggregated", 10, 0)
```

#### 6. 强制使用索引或文件扫描

```go
// 强制使用索引查询
result, err := logz.QueryLogsWithIndex(logz.LogQuery{
    TraceID: "trace-001",
    Level:   "error",
    Limit:   100,
    Offset:  0,
}, "./logs/aggregated")

// 强制使用文件扫描查询
result, err := logz.QueryLogsWithoutIndex(logz.LogQuery{
    TraceID: "trace-001",
    Level:   "error",
    Limit:   100,
    Offset:  0,
}, "./logs/aggregated")
```

### 清理功能

```go
// 清理一周前的日志
err := logz.CleanupOldLogsDefault("./logs/aggregated")
if err != nil {
    log.Printf("清理失败: %v", err)
}

// 清理指定天数前的日志
err := logz.CleanupOldLogs("./logs/aggregated", 30) // 清理30天前的日志
```

### 统计功能

```go
stats, err := logz.GetLogStatsDefault("./logs/aggregated")
if err != nil {
    log.Printf("获取统计信息失败: %v", err)
} else {
    fmt.Printf("日志文件总数: %d\n", stats["total_files"])
    fmt.Printf("总大小: %d 字节 (%.2f MB)\n", stats["total_size"], float64(stats["total_size"].(int64))/1024/1024)
    fmt.Printf("最旧文件: %s\n", stats["oldest_file"])
    fmt.Printf("最新文件: %s\n", stats["newest_file"])
}
```

## 📧 邮件通知功能

### 1. 配置邮箱

**重要**：为了避免敏感信息泄露，请使用环境变量配置 SMTP 信息。

#### 方法1：环境变量配置（推荐）

```bash
# 设置环境变量
export SMTP_USER="your-email@qq.com"
export SMTP_PASSWORD="your-email-password"
export SMTP_HOST="smtp.qq.com"
export SMTP_PORT="587"
export NOTIFICATION_EMAIL="developer@example.com"
```

```go
import "github.com/HsiaoL1/trace"

// 从环境变量加载 SMTP 配置
trace.LoadSMTPConfigFromEnv()

// 设置接收通知的邮箱地址
trace.SetEmail("developer@example.com")
```

#### 方法2：代码中设置

```go
import "github.com/HsiaoL1/trace"

// 设置 SMTP 配置
trace.SetSMTPConfig("smtp.qq.com", 587, "your-email@qq.com", "your-password")

// 设置接收通知的邮箱地址
trace.SetEmail("developer@example.com")
```

### 2. 使用邮件通知

```go
// 错误日志（带邮件通知）
logz.ErrorWithEmail(true, "数据库连接失败")
logz.ErrorfWithEmail(true, "处理用户 %s 请求失败: %v", "张三", err)

// 带追踪上下文的错误日志（带邮件通知）
traceID := "abc123"
spanID := "span456"
logz.ErrorWithTraceAndEmail(traceID, spanID, true, "服务调用失败")
logz.ErrorfWithTraceAndEmail(traceID, spanID, true, "用户 %s 操作失败: %v", "李四", err)

// 致命错误日志（带邮件通知，会终止程序）
logz.FatalWithEmail(true, "系统配置错误，程序退出")
logz.FatalfWithEmail(true, "初始化失败: %v", err)

// 恐慌错误日志（带邮件通知，会崩溃程序）
logz.PanicWithEmail(true, "严重错误，程序崩溃")
logz.PanicfWithEmail(true, "内存不足: %v", err)
```

### 3. 邮件通知特性

- **异步发送**：邮件发送不会阻塞日志记录
- **调用者信息**：邮件内容包含错误发生的文件位置和函数名
- **结构化内容**：邮件包含错误级别、时间、消息和调用位置
- **HTML 格式**：邮件使用 HTML 格式，便于阅读
- **错误处理**：邮件发送失败时会记录到日志中

### 4. 邮件内容示例

邮件主题：`[ERROR] 系统日志告警 - 2025-06-24 16:57:57`

邮件内容：
```html
<h2>系统日志告警</h2>
<p><strong>级别:</strong> ERROR</p>
<p><strong>时间:</strong> 2025-06-24 16:57:57</p>
<p><strong>消息:</strong> 数据库连接失败</p>
<p><strong>调用位置: main.go:25 (main.processUserRequest)</strong></p>
<hr>
<p><em>此邮件由系统自动发送，请及时处理。</em></p>
```

## 🛠️ 便捷初始化方法

### 1. 开发环境配置

```go
logz.InitDevelopment()
// 等同于：
// - 级别：Debug
// - 格式：Text
// - 输出：标准输出
// - 调用者信息：启用
```

### 2. 生产环境配置

```go
// 输出到文件
err := logz.InitProduction("/var/log/app.log")
if err != nil {
    log.Fatal(err)
}

// 等同于：
// - 级别：Info
// - 格式：JSON
// - 输出：文件
// - 调用者信息：启用
```

### 3. 默认配置

```go
logz.InitDefault()
// 等同于：
// - 级别：Info
// - 格式：Text
// - 输出：标准输出
// - 调用者信息：启用
```

## 📋 日志级别

| 级别 | 常量 | 说明 | 邮件通知 |
|------|------|------|----------|
| Debug | `LevelDebug` | 调试信息，开发时使用 | ❌ |
| Info | `LevelInfo` | 一般信息，记录程序运行状态 | ❌ |
| Warn | `LevelWarn` | 警告信息，可能的问题 | ❌ |
| Error | `LevelError` | 错误信息，程序错误 | ✅ |
| Fatal | `LevelFatal` | 致命错误，程序退出 | ✅ |
| Panic | `LevelPanic` | 恐慌错误，程序崩溃 | ✅ |

## 📄 日志格式

### 文本格式示例

```
INFO[2025-06-24T16:57:57+08:00]logz.go:146 应用启动
DEBU[2025-06-24T16:57:57+08:00]logz.go:136 调试信息
WARN[2025-06-24T16:57:57+08:00]logz.go:156 警告信息
ERRO[2025-06-24T16:57:57+08:00]logz.go:166 错误信息
```

### JSON 格式示例

```json
{
  "level": "info",
  "msg": "用户登录",
  "time": "2025-06-24T16:57:57+08:00",
  "user_id": "123",
  "action": "login",
  "ip": "192.168.1.1"
}
```

### 聚合日志格式

聚合日志以 JSON 格式存储，每条日志占一行：

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "msg": "用户登录成功",
  "trace_id": "trace-001",
  "span_id": "span-001",
  "service": "user-service",
  "caller": "main.go:25",
  "file_id": "user-service_2024-01-15_001",
  "offset": 1024,
  "fields": {
    "user_id": "123",
    "ip": "192.168.1.1"
  }
}
```

## 🔧 其他可选方案

### 2. 基于时间戳的方案

```go
func GenerateTraceIDWithTimestamp() TraceID {
    var traceID TraceID
    timestamp := time.Now().UnixNano()
    machineID := getMachineID() // 获取机器ID
    
    // 组合时间戳和机器ID
    binary.BigEndian.PutUint64(traceID[:8], uint64(timestamp))
    binary.BigEndian.PutUint64(traceID[8:], machineID)
    
    return traceID
}
```

**优势：**
- 包含时间信息，便于调试
- 可以包含机器标识
- 相对有序

**劣势：**
- 时钟回拨可能导致冲突
- 需要额外的机器ID管理

### 3. 基于序列号的方案

```go
type IDGenerator struct {
    sequence uint64
    mutex    sync.Mutex
}

func (g *IDGenerator) GenerateTraceID() TraceID {
    g.mutex.Lock()
    defer g.mutex.Unlock()
    
    g.sequence++
    var traceID TraceID
    binary.BigEndian.PutUint64(traceID[:8], uint64(time.Now().UnixNano()))
    binary.BigEndian.PutUint64(traceID[8:], g.sequence)
    
    return traceID
}
```

**优势：**
- 保证单调递增
- 性能高
- 便于排序

**劣势：**
- 需要维护状态
- 重启后序列号重置

### 4. 雪花算法（Snowflake）方案

```go
type SnowflakeGenerator struct {
    machineID uint64
    sequence  uint64
    lastTime  int64
    mutex     sync.Mutex
}

func (s *SnowflakeGenerator) GenerateTraceID() TraceID {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    now := time.Now().UnixNano() / 1e6 // 毫秒
    
    if now == s.lastTime {
        s.sequence++
    } else {
        s.sequence = 0
        s.lastTime = now
    }
    
    var traceID TraceID
    id := (now << 22) | (s.machineID << 12) | s.sequence
    binary.BigEndian.PutUint64(traceID[:8], uint64(id))
    
    return traceID
}
```

**优势：**
- 包含时间戳、机器ID和序列号
- 全局唯一且有序
- 性能优秀

**劣势：**
- 实现复杂度较高
- 需要机器ID管理

## 🎯 使用建议

### 生产环境推荐
1. **首选方案**：基于 UUID 的方案（当前实现）
2. **备选方案**：雪花算法（如果需要有序性）

### 开发环境推荐
1. **调试友好**：基于时间戳的方案
2. **简单测试**：基于序列号的方案

## 📊 性能对比

| 方案 | 性能 | 唯一性 | 有序性 | 复杂度 |
|------|------|--------|--------|--------|
| UUID | 高 | 极高 | 无 | 低 |
| 时间戳 | 高 | 高 | 部分 | 低 |
| 序列号 | 极高 | 高 | 是 | 中 |
| 雪花算法 | 高 | 极高 | 是 | 高 |

## 🔧 完整示例

```go
package main

import (
    "context"
    "errors"
    "net/http"
    "time"
    
    "github.com/HsiaoL1/trace"
    "github.com/HsiaoL1/trace/logz"
    "github.com/sirupsen/logrus"
)

func main() {
    // 初始化日志配置
    logz.InitDevelopment()
    logz.EnableCaller()
    
    // 初始化日志聚合
    err := logz.InitWithAggregation(
        "./logs/app.log",
        "./logs/aggregated",
        "demo-service",
        100*1024*1024, // 100MB
        20,
    )
    if err != nil {
        log.Fatal(err)
    }
    defer logz.CloseAggregator()
    
    // 设置接收邮箱
    trace.SetEmail("developer@example.com")
    
    // 初始化 Jaeger
    config := trace.LoadConfigFromEnv()
    cleanup, err := trace.InitJaeger(&config.Jaeger)
    if err != nil {
        log.Printf("Failed to initialize Jaeger: %v", err)
    } else {
        defer cleanup()
    }
    
    // 生成追踪ID
    traceID := trace.GenerateTraceID()
    spanID := trace.GenerateSpanID()
    
    // 基本日志
    logz.Info("应用启动")
    
    // 结构化日志
    logz.WithField("user_id", "123").Info("用户登录")
    
    // 多字段日志
    fields := logrus.Fields{
        "user_id": "123",
        "action":  "login",
        "ip":      "192.168.1.1",
        "time":    time.Now().Format("2006-01-02 15:04:05"),
    }
    logz.WithFields(fields).Info("用户操作")
    
    // 错误日志
    err = errors.New("数据库连接失败")
    logz.WithError(err).Error("系统错误")
    
    // 带邮件通知的错误日志
    logz.ErrorWithEmail(true, "严重错误，需要立即处理")
    logz.ErrorfWithEmail(true, "处理失败: %v", err)
    
    // 带追踪上下文的日志
    logz.InfoWithTrace(traceID.String(), spanID.String(), "处理用户请求")
    logz.ErrorWithTrace(traceID.String(), spanID.String(), "处理失败")
    
    // 带追踪上下文的邮件通知
    logz.ErrorWithTraceAndEmail(traceID.String(), spanID.String(), true, "服务调用失败")
    
    // 格式化日志
    logz.Infof("用户 %s 登录成功", "张三")
    logz.Errorf("处理请求失败: %v", err)
    
    // HTTP 追踪示例
    client := trace.NewTracedHTTPClient(10 * time.Second)
    rootCtx := context.Background()
    rootTraceCtx := trace.CreateRootSpan()
    ctx := trace.WithTraceContext(rootCtx, rootTraceCtx)
    
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://api.example.com/data", nil)
    client.Do(ctx, req)
    
    // Jaeger 追踪示例
    ctx, span := trace.StartSpan(ctx, "main-operation")
    defer span.End()
    
    trace.SetAttribute(span, "user.id", "123")
    trace.AddEvent(span, "processing started")
    
    // 日志查询示例
    result, err := logz.QueryLogsByTraceID(traceID.String(), "./logs/aggregated", 10, 0)
    if err == nil {
        logz.Infof("找到 %d 条相关日志", result.Total)
    }
}
```

## 🎯 最佳实践

1. **在服务入口使用中间件**
```go
mux.Handle("/api", trace.HTTPMiddleware(http.HandlerFunc(handler)))
```

2. **使用带追踪功能的 HTTP 客户端**
```go
client := trace.NewTracedHTTPClient(timeout)
```

3. **记录关键操作的追踪信息**
```go
trace.LogTraceContext(ctx, "数据库查询")
```

4. **在日志中包含追踪信息**
```go
log.Printf("[%s] 处理请求", traceCtx.TraceID)
```

5. **开发环境**：使用 `InitDevelopment()` 获得详细的调试信息
6. **生产环境**：使用 `InitProduction()` 输出 JSON 格式到文件
7. **结构化日志**：使用 `WithField()` 和 `WithFields()` 添加上下文信息
8. **追踪日志**：使用 `*WithTrace()` 方法记录分布式追踪信息
9. **错误处理**：使用 `WithError()` 记录错误详情
10. **日志级别**：根据环境设置合适的日志级别
11. **邮件通知**：为重要错误配置邮件通知，及时发现问题
12. **邮箱配置**：在生产环境中配置有效的邮箱地址
13. **异步处理**：邮件发送是异步的，不会影响程序性能
14. **错误处理**：邮件发送失败时会记录到日志中，避免循环调用
15. **日志聚合**：使用日志聚合功能统一管理大规模日志
16. **索引查询**：利用索引功能提高日志查询性能
17. **Web 界面**：使用 Web 界面方便地查看和管理日志

## 🔧 常见问题

### Q: 如何手动设置追踪上下文？
A: 使用 `trace.WithTraceContext()` 函数：
```go
ctx := trace.WithTraceContext(context.Background(), traceCtx)
```

### Q: 如何从 HTTP 头部获取追踪信息？
A: 使用 `trace.GetTraceContextFromHttpHeader()` 函数：
```go
traceCtx := trace.GetTraceContextFromHttpHeader(req)
```

### Q: 如何创建子 span？
A: 使用 `trace.CreateChildSpan()` 函数：
```go
childTraceCtx := trace.CreateChildSpan(parentTraceCtx)
```

### Q: 如何验证追踪上下文是否有效？
A: 使用 `IsValid()` 方法：
```go
if traceCtx.IsValid() {
    // 处理有效的追踪上下文
}
```

### Q: 如何优化大规模日志查询性能？
A: 使用索引查询：
```go
result, err := logz.QueryLogsByTraceID("trace-001", "./logs/aggregated", 100, 0)
```

### Q: 如何启动 Web 界面？
A: 进入 `logz/web` 目录，运行 `./demo.sh` 或手动启动：
```bash
cd logz/web
go run demo.go &
go run main.go
```

### Q: 如何配置 Jaeger 集成？
A: 设置环境变量：
```bash
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
export JAEGER_SERVICE_NAME="my-service"
export JAEGER_ENABLED="true"
```

## 🚀 运行测试

```bash
go test -v
```

## 🚀 运行示例

```bash
go run example/main.go
```

## 📁 项目结构

```
trace/
├── README.md           # 项目文档
├── go.mod             # Go 模块文件
├── go.sum             # 依赖校验文件
├── trace.go           # Trace ID/Span ID 生成
├── trace_test.go      # Trace 测试
├── http.go            # HTTP 追踪功能
├── http_client.go     # HTTP 客户端
├── email.go           # 邮件功能
├── jaeger.go          # Jaeger 集成
├── logz/              # 日志库
│   ├── README.md      # 日志库文档
│   ├── logz.go        # 日志核心功能
│   ├── logz_test.go   # 日志测试
│   ├── file.go        # 日志聚合功能
│   ├── file_test.go   # 聚合测试
│   ├── web/           # Web 界面
│   │   ├── main.go    # Web 服务器
│   │   ├── api.go     # API 接口
│   │   ├── demo.go    # 演示程序
│   │   ├── demo.sh    # 启动脚本
│   │   ├── static/    # 静态文件
│   │   └── templates/ # HTML 模板
│   └── example/       # 使用示例
│       └── main.go    # 主示例
└── example/           # 项目示例
    ├── main.go        # 主示例
    ├── demo/          # 演示代码
    ├── span/          # Span 示例
    └── trace/         # Trace 示例
```

## 📝 向后兼容

此库保持与之前版本的完全向后兼容性：

- 原有的自定义追踪功能继续工作
- 自定义 HTTP 头部 (`X-Trace-ID`, `X-Span-ID`, `X-Parent-Span-ID`) 仍然支持
- 现有的中间件和客户端代码无需修改
- 日志聚合功能可以独立使用
- Jaeger 集成是可选的，不影响现有功能

## 📖 运行演示

### 1. 基本演示
```bash
go run example/main.go
```

### 2. Jaeger 演示
```bash
cd example
go run main.go
```

然后访问 Jaeger UI：http://localhost:16686

### 3. Web 界面演示
```bash
cd logz/web
./demo.sh
```

然后访问 Web 界面：http://localhost:8080

## 📊 最佳实践

1. **服务命名**：使用有意义的服务名称，如 `user-service`、`order-api`
2. **属性设置**：添加业务相关的属性，如用户ID、请求ID等
3. **错误处理**：始终记录错误到 span 中
4. **采样配置**：生产环境建议降低采样比例以减少性能影响
5. **资源清理**：确保调用 cleanup 函数来正确关闭追踪器
6. **日志级别**：根据环境设置合适的日志级别
7. **日志聚合**：使用日志聚合功能统一管理大规模日志
8. **索引查询**：利用索引功能提高日志查询性能
9. **邮件通知**：为重要错误配置邮件通知
10. **Web 界面**：使用 Web 界面方便地查看和管理日志

## 🔧 故障排除

### 常见问题

1. **追踪数据未显示在 Jaeger 中**
   - 检查 Jaeger 是否正在运行
   - 验证端点配置是否正确
   - 确保防火墙允许连接

2. **性能影响**
   - 调整采样比例
   - 检查网络延迟到 Jaeger 收集器

3. **内存使用**
   - 确保调用 cleanup 函数
   - 监控 span 的生命周期

4. **日志查询慢**
   - 使用索引查询
   - 合理设置查询限制
   - 定期清理旧文件

5. **Web 界面无法访问**
   - 确认服务器已启动
   - 检查防火墙设置
   - 查看服务器日志

### 调试

启用调试日志：

```bash
export TRACE_LOG_LEVEL=debug
```

检查追踪数据：

```go
// 记录追踪上下文
trace.LogTraceContext(ctx, "operation-name")
```

检查日志聚合：

```go
// 查看聚合统计
stats, err := logz.GetLogStatsDefault("./logs/aggregated")
```

这个统一的 README 文档整合了所有功能模块，为用户提供了完整的使用指南。