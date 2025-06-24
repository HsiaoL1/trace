# Trace - 分布式追踪和日志库

这是一个完整的分布式追踪和日志解决方案，提供Trace ID/Span ID生成、HTTP追踪上下文传递、结构化日志记录和邮件通知功能。

## 功能特性

### 核心功能
- ✅ **Trace ID 和 Span ID 生成**：基于UUID的方案，符合OpenTelemetry规范
- ✅ **HTTP追踪上下文传递**：自动在服务间传递追踪信息
- ✅ **结构化日志记录**：基于logrus，支持多种格式和级别
- ✅ **邮件通知功能**：重要错误自动邮件通知
- ✅ **分布式追踪支持**：完整的调用链路追踪

### 日志功能
- ✅ 支持多种日志级别（Debug, Info, Warn, Error, Fatal, Panic）
- ✅ 支持文本和JSON格式输出
- ✅ 支持文件输出和标准输出
- ✅ 支持结构化日志（带字段）
- ✅ 支持追踪上下文（Trace ID, Span ID）
- ✅ 支持调用者信息
- ✅ 支持便捷的初始化方法
- ✅ 支持邮件通知功能（Error, Fatal, Panic级别）

## 快速开始

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
    
    // 生成Trace ID和Span ID
    traceID := trace.GenerateTraceID()
    spanID := trace.GenerateSpanID()
    
    fmt.Printf("Trace ID: %s\n", traceID.String())
    fmt.Printf("Span ID: %s\n", spanID.String())
    
    // 记录日志
    logz.Info("应用启动")
    logz.WithField("trace_id", traceID.String()).Info("追踪信息")
}
```

## Trace ID 和 Span ID 生成

### 基于UUID的方案（推荐）✅

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
    // 生成Trace ID
    traceID := trace.GenerateTraceID()
    fmt.Printf("Trace ID: %s\n", traceID.String())
    
    // 生成Span ID
    spanID := trace.GenerateSpanID()
    fmt.Printf("Span ID: %s\n", spanID.String())
    
    // 验证有效性
    if traceID.IsValid() && spanID.IsValid() {
        fmt.Println("IDs are valid")
    }
}
```

## HTTP 追踪功能

### 核心功能

1. **TraceContext 结构体**
```go
type TraceContext struct {
    TraceID      string // 追踪ID
    SpanID       string // 当前Span ID
    ParentSpanID string // 父Span ID
}
```

2. **HTTP头部常量**
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
    // 创建带追踪功能的HTTP客户端
    client := trace.NewTracedHTTPClient(10 * time.Second)
    
    // 创建根span（模拟外部请求）
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
    // 从context中获取追踪上下文
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

#### 2. 手动设置HTTP头部

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

#### 3. 从HTTP头部获取追踪上下文

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 从HTTP头部获取追踪上下文
    traceCtx := trace.GetTraceContextFromHttpHeader(r)
    
    // 创建子span
    childTraceCtx := trace.CreateChildSpan(traceCtx)
    
    // 将追踪上下文注入到context中
    ctx := trace.WithTraceContext(r.Context(), childTraceCtx)
    
    // 使用新的context处理请求
    processRequest(ctx)
}
```

### 追踪链路示例

```
外部请求 → 服务A → 服务B → 服务C
   ↓         ↓       ↓       ↓
TraceID: abc123 (保持不变)
SpanID:  span1 → span2 → span3 → span4
Parent:   -    → span1 → span2 → span3
```

当服务C出错时，可以通过 `ParentSpanID` 知道是服务B调用的，通过 `TraceID` 可以追踪整个调用链路。

## 日志功能 (Logz)

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
logz.SetLevel(logz.LevelInfo)   // 只显示Info及以上级别
logz.SetLevel(logz.LevelError)  // 只显示Error及以上级别
```

### 设置日志格式

```go
// 文本格式（默认）
logz.SetFormat(logz.FormatText)

// JSON格式
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

## 邮件通知功能

### 1. 配置邮箱

**重要**：为了避免敏感信息泄露，请使用环境变量配置SMTP信息。

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

// 从环境变量加载SMTP配置
trace.LoadSMTPConfigFromEnv()

// 设置接收通知的邮箱地址
trace.SetEmail("developer@example.com")
```

#### 方法2：代码中设置

```go
import "github.com/HsiaoL1/trace"

// 设置SMTP配置
trace.SetSMTPConfig("smtp.qq.com", 587, "your-email@qq.com", "your-password")

// 设置接收通知的邮箱地址
trace.SetEmail("developer@example.com")
```

**详细配置说明请查看 [CONFIG.md](CONFIG.md) 文件**

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
- **HTML格式**：邮件使用HTML格式，便于阅读
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

## 便捷初始化方法

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

## 日志级别

| 级别 | 常量 | 说明 | 邮件通知 |
|------|------|------|----------|
| Debug | `LevelDebug` | 调试信息，开发时使用 | ❌ |
| Info | `LevelInfo` | 一般信息，记录程序运行状态 | ❌ |
| Warn | `LevelWarn` | 警告信息，可能的问题 | ❌ |
| Error | `LevelError` | 错误信息，程序错误 | ✅ |
| Fatal | `LevelFatal` | 致命错误，程序退出 | ✅ |
| Panic | `LevelPanic` | 恐慌错误，程序崩溃 | ✅ |

## 日志格式

### 文本格式示例

```
INFO[2025-06-24T16:57:57+08:00]logz.go:146 应用启动
DEBU[2025-06-24T16:57:57+08:00]logz.go:136 调试信息
WARN[2025-06-24T16:57:57+08:00]logz.go:156 警告信息
ERRO[2025-06-24T16:57:57+08:00]logz.go:166 错误信息
```

### JSON格式示例

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

## 其他可选方案

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

## 使用建议

### 生产环境推荐
1. **首选方案**：基于UUID的方案（当前实现）
2. **备选方案**：雪花算法（如果需要有序性）

### 开发环境推荐
1. **调试友好**：基于时间戳的方案
2. **简单测试**：基于序列号的方案

## 性能对比

| 方案 | 性能 | 唯一性 | 有序性 | 复杂度 |
|------|------|--------|--------|--------|
| UUID | 高 | 极高 | 无 | 低 |
| 时间戳 | 高 | 高 | 部分 | 低 |
| 序列号 | 极高 | 高 | 是 | 中 |
| 雪花算法 | 高 | 极高 | 是 | 高 |

## 完整示例

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
    
    // 设置接收邮箱
    trace.SetEmail("developer@example.com")
    
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
    err := errors.New("数据库连接失败")
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
    
    // HTTP追踪示例
    client := trace.NewTracedHTTPClient(10 * time.Second)
    rootCtx := context.Background()
    rootTraceCtx := trace.CreateRootSpan()
    ctx := trace.WithTraceContext(rootCtx, rootTraceCtx)
    
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://api.example.com/data", nil)
    client.Do(ctx, req)
}
```

## 最佳实践

1. **在服务入口使用中间件**
```go
mux.Handle("/api", trace.HTTPMiddleware(http.HandlerFunc(handler)))
```

2. **使用带追踪功能的HTTP客户端**
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
6. **生产环境**：使用 `InitProduction()` 输出JSON格式到文件
7. **结构化日志**：使用 `WithField()` 和 `WithFields()` 添加上下文信息
8. **追踪日志**：使用 `*WithTrace()` 方法记录分布式追踪信息
9. **错误处理**：使用 `WithError()` 记录错误详情
10. **日志级别**：根据环境设置合适的日志级别
11. **邮件通知**：为重要错误配置邮件通知，及时发现问题
12. **邮箱配置**：在生产环境中配置有效的邮箱地址
13. **异步处理**：邮件发送是异步的，不会影响程序性能
14. **错误处理**：邮件发送失败时会记录到日志中，避免循环调用

## 常见问题

### Q: 如何手动设置追踪上下文？
A: 使用 `trace.WithTraceContext()` 函数：
```go
ctx := trace.WithTraceContext(context.Background(), traceCtx)
```

### Q: 如何从HTTP头部获取追踪信息？
A: 使用 `trace.GetTraceContextFromHttpHeader()` 函数：
```go
traceCtx := trace.GetTraceContextFromHttpHeader(req)
```

### Q: 如何创建子span？
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

## 运行测试

```bash
go test -v
```

## 运行示例

```bash
go run example/main.go
```

## 项目结构

```
trace/
├── README.md           # 项目文档
├── go.mod             # Go模块文件
├── go.sum             # 依赖校验文件
├── trace.go           # Trace ID/Span ID生成
├── trace_test.go      # Trace测试
├── http.go            # HTTP追踪功能
├── http_client.go     # HTTP客户端
├── email.go           # 邮件功能
├── logz/              # 日志库
│   ├── logz.go        # 日志核心功能
│   ├── logz_test.go   # 日志测试
│   ├── email_test.go  # 邮件通知测试
│   └── example/       # 使用示例
└── example/           # 项目示例
    ├── main.go        # 主示例
    ├── demo/          # 演示代码
    ├── span/          # Span示例
    └── trace/         # Trace示例
``` 