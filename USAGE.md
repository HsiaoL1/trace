# 使用指南

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "github.com/HsiaoL1/trace"
)

func main() {
    // 生成Trace ID和Span ID
    traceID := trace.GenerateTraceID()
    spanID := trace.GenerateSpanID()
    
    fmt.Printf("Trace ID: %s\n", traceID.String())
    fmt.Printf("Span ID: %s\n", spanID.String())
}
```

### 2. HTTP服务端使用

```go
package main

import (
    "net/http"
    "github.com/HsiaoL1/trace"
)

func main() {
    mux := http.NewServeMux()
    
    // 使用追踪中间件
    mux.Handle("/api", trace.HTTPMiddleware(http.HandlerFunc(handleAPI)))
    
    http.ListenAndServe(":8080", mux)
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
    // 从context中获取追踪上下文
    traceCtx := trace.GetTraceContextFromContext(r.Context())
    
    // 记录追踪信息
    log.Printf("收到请求: %s", traceCtx.String())
    
    // 处理业务逻辑...
    w.Write([]byte("OK"))
}
```

### 3. HTTP客户端使用

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
    
    // 创建根span
    rootCtx := context.Background()
    rootTraceCtx := trace.CreateRootSpan()
    ctx := trace.WithTraceContext(rootCtx, rootTraceCtx)
    
    // 调用下游服务
    req, err := http.NewRequestWithContext(ctx, "GET", "http://api.example.com/data", nil)
    if err != nil {
        return
    }
    
    // 自动传递追踪上下文
    resp, err := client.Do(ctx, req)
    // 处理响应...
}
```

## 核心概念

### Trace ID
- 全局唯一的追踪标识符
- 在整个调用链路中保持不变
- 用于关联所有相关的操作

### Span ID
- 单个操作的标识符
- 每个服务调用都会生成新的Span ID
- 用于标识具体的操作

### Parent Span ID
- 父操作的Span ID
- 用于构建调用链路的层级关系
- 根span没有Parent Span ID

## 追踪链路示例

```
用户请求 → 网关服务 → 用户服务 → 订单服务 → 支付服务
   ↓         ↓         ↓         ↓         ↓
TraceID: abc123 (保持不变)
SpanID:  span1 → span2 → span3 → span4 → span5
Parent:   -    → span1 → span2 → span3 → span4
```

## 错误排查

当服务C出错时，可以通过以下信息定位问题：

1. **Trace ID**: `abc123` - 可以追踪整个调用链路
2. **Span ID**: `span5` - 当前出错的Span
3. **Parent Span ID**: `span4` - 知道是订单服务调用的

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