# Trace ID 和 Span ID 生成方案

这个项目提供了生成分布式追踪中 Trace ID 和 Span ID 的多种方案实现，以及完整的HTTP追踪上下文传递功能。

## 当前实现方案

### 1. 基于UUID的方案（推荐）✅

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

## 最佳实践

1. **Trace ID**：使用16字节，确保全局唯一性
2. **Span ID**：使用8字节，在同一个Trace内唯一即可
3. **错误处理**：提供fallback机制（如时间戳）
4. **验证**：实现有效性检查方法
5. **测试**：确保生成的ID唯一性

## 使用示例

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

## 运行测试

```bash
go test -v
```

## 运行示例

```bash
go run example/main.go
``` 