# Logz - 日志库

基于 logrus 的日志库，提供简单易用的日志功能，支持结构化日志、追踪上下文、多种输出格式和邮件通知功能。

## 功能特性

- ✅ 支持多种日志级别（Debug, Info, Warn, Error, Fatal, Panic）
- ✅ 支持文本和JSON格式输出
- ✅ 支持文件输出和标准输出
- ✅ 支持结构化日志（带字段）
- ✅ 支持追踪上下文（Trace ID, Span ID）
- ✅ 支持调用者信息
- ✅ 支持便捷的初始化方法
- ✅ 支持邮件通知功能（Error, Fatal, Panic级别）

## 快速开始

### 1. 基本使用

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

### 2. 设置日志级别

```go
// 设置日志级别
logz.SetLevel(logz.LevelDebug)  // 显示所有日志
logz.SetLevel(logz.LevelInfo)   // 只显示Info及以上级别
logz.SetLevel(logz.LevelError)  // 只显示Error及以上级别
```

### 3. 设置日志格式

```go
// 文本格式（默认）
logz.SetFormat(logz.FormatText)

// JSON格式
logz.SetFormat(logz.FormatJSON)
```

### 4. 设置输出位置

```go
// 输出到标准输出（默认）
logz.SetOutput(os.Stdout)

// 输出到文件
err := logz.SetFileOutput("/var/log/app.log")
if err != nil {
    log.Fatal(err)
}
```

## 高级功能

### 1. 结构化日志

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

### 2. 带追踪上下文的日志

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

### 3. 启用调用者信息

```go
// 启用调用者信息（显示文件名和行号）
logz.EnableCaller()

// 禁用调用者信息
logz.DisableCaller()
```

## 邮件通知功能

### 1. 配置邮箱

```go
import "github.com/HsiaoL1/trace"

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

## 完整示例

```go
package main

import (
    "errors"
    "time"
    
    "github.com/HsiaoL1/trace"
    "github.com/HsiaoL1/trace/logz"
    "github.com/sirupsen/logrus"
)

func main() {
    // 初始化日志配置
    logz.InitDevelopment()
    
    // 启用调用者信息
    logz.EnableCaller()
    
    // 设置接收邮箱
    trace.SetEmail("developer@example.com")
    
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
    traceID := "abc123def456"
    spanID := "span789"
    logz.InfoWithTrace(traceID, spanID, "处理用户请求")
    logz.ErrorWithTrace(traceID, spanID, "处理失败")
    
    // 带追踪上下文的邮件通知
    logz.ErrorWithTraceAndEmail(traceID, spanID, true, "服务调用失败")
    
    // 格式化日志
    logz.Infof("用户 %s 登录成功", "张三")
    logz.Errorf("处理请求失败: %v", err)
}
```

## 运行测试

```bash
go test -v
```

## 运行示例

```bash
cd example
go run main.go
```

## 最佳实践

1. **开发环境**：使用 `InitDevelopment()` 获得详细的调试信息
2. **生产环境**：使用 `InitProduction()` 输出JSON格式到文件
3. **结构化日志**：使用 `WithField()` 和 `WithFields()` 添加上下文信息
4. **追踪日志**：使用 `*WithTrace()` 方法记录分布式追踪信息
5. **错误处理**：使用 `WithError()` 记录错误详情
6. **日志级别**：根据环境设置合适的日志级别
7. **邮件通知**：为重要错误配置邮件通知，及时发现问题
8. **邮箱配置**：在生产环境中配置有效的邮箱地址
9. **异步处理**：邮件发送是异步的，不会影响程序性能
10. **错误处理**：邮件发送失败时会记录到日志中，避免循环调用 