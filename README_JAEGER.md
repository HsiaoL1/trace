# Jaeger 集成使用指南

这个trace库现在支持Jaeger集成，通过OpenTelemetry协议将追踪数据发送到Jaeger。

## 快速开始

### 1. 启动Jaeger

使用Docker启动Jaeger all-in-one：

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

### 2. 环境变量配置

```bash
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
export JAEGER_SERVICE_NAME="my-service"
export JAEGER_ENVIRONMENT="development"
export JAEGER_VERSION="1.0.0"
export JAEGER_ENABLED="true"
```

### 3. 基本使用

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
    
    // 初始化Jaeger
    cleanup, err := trace.InitJaeger(&config.Jaeger)
    if err != nil {
        log.Fatalf("Failed to initialize Jaeger: %v", err)
    }
    defer cleanup()
    
    // 创建span
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

## API 使用说明

### 初始化

```go
// 使用默认配置
config := trace.DefaultJaegerConfig()

// 或从环境变量加载
config := trace.LoadJaegerConfigFromEnv()

// 初始化Jaeger
cleanup, err := trace.InitJaeger(config)
if err != nil {
    log.Fatal(err)
}
defer cleanup()
```

### 创建Span

```go
// 创建根span
ctx, span := trace.StartSpan(context.Background(), "operation-name")
defer span.End()

// 创建子span
childCtx, childSpan := trace.StartSpan(ctx, "child-operation")
defer childSpan.End()
```

### 设置属性和事件

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

### HTTP中间件

```go
// 使用OpenTelemetry中间件
mux := http.NewServeMux()
mux.HandleFunc("/api/endpoint", handler)

// 包装中间件
wrappedHandler := trace.OpenTelemetryMiddleware(mux)

server := &http.Server{
    Addr:    ":8080",
    Handler: wrappedHandler,
}
```

### HTTP客户端

```go
// 创建带追踪的HTTP客户端
client := trace.NewTracedHTTPClient(10 * time.Second)

// 发送请求（自动传播追踪上下文）
resp, err := client.Get(ctx, "http://api.example.com/users")
```

## 配置选项

### 环境变量

| 变量名 | 默认值 | 描述 |
|--------|---------|------|
| `JAEGER_ENDPOINT` | `http://localhost:14268/api/traces` | Jaeger收集器端点 |
| `JAEGER_SERVICE_NAME` | `trace-service` | 服务名称 |
| `JAEGER_ENVIRONMENT` | `development` | 环境名称 |
| `JAEGER_VERSION` | `1.0.0` | 服务版本 |
| `JAEGER_ENABLED` | `true` | 是否启用Jaeger |
| `TRACE_LOG_LEVEL` | `info` | 日志级别 |
| `TRACE_SAMPLING_RATIO` | `1.0` | 采样比例 (0.0-1.0) |

### 程序配置

```go
config := &trace.JaegerConfig{
    Endpoint:    "http://localhost:14268/api/traces",
    ServiceName: "my-service",
    Environment: "production",
    Version:     "2.0.0",
    Enabled:     true,
}
```

## 示例

运行示例程序：

```bash
cd example
go run main.go
```

然后访问 Jaeger UI：http://localhost:16686

## 向后兼容

此库保持与之前版本的完全向后兼容性：

- 原有的自定义追踪功能继续工作
- 自定义HTTP头部 (`X-Trace-ID`, `X-Span-ID`, `X-Parent-Span-ID`) 仍然支持
- 现有的中间件和客户端代码无需修改

## 最佳实践

1. **服务命名**：使用有意义的服务名称，如 `user-service`、`order-api`
2. **属性设置**：添加业务相关的属性，如用户ID、请求ID等
3. **错误处理**：始终记录错误到span中
4. **采样配置**：生产环境建议降低采样比例以减少性能影响
5. **资源清理**：确保调用cleanup函数来正确关闭追踪器

## 故障排除

### 常见问题

1. **追踪数据未显示在Jaeger中**
   - 检查Jaeger是否正在运行
   - 验证端点配置是否正确
   - 确保防火墙允许连接

2. **性能影响**
   - 调整采样比例
   - 检查网络延迟到Jaeger收集器

3. **内存使用**
   - 确保调用cleanup函数
   - 监控span的生命周期

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