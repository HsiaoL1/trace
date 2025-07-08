package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/HsiaoL1/trace"
)

func main() {
	// 加载配置
	config := trace.LoadConfigFromEnv()
	config.Validate()

	// 初始化Jaeger
	cleanup, err := trace.InitJaeger(&config.Jaeger)
	if err != nil {
		log.Fatalf("Failed to initialize Jaeger: %v", err)
	}
	defer cleanup()

	fmt.Println("Jaeger集成示例启动...")
	fmt.Printf("Jaeger endpoint: %s\n", config.Jaeger.Endpoint)
	fmt.Printf("Service name: %s\n", config.Jaeger.ServiceName)

	// 示例1: 基本span使用
	basicSpanExample()

	// 示例2: HTTP服务器
	go httpServerExample()

	// 示例3: HTTP客户端
	time.Sleep(1 * time.Second) // 等待服务器启动
	httpClientExample()

	// 保持程序运行一段时间，让traces被发送到Jaeger
	time.Sleep(5 * time.Second)
	fmt.Println("示例完成")
}

// basicSpanExample 基本span使用示例
func basicSpanExample() {
	ctx := context.Background()

	// 创建根span
	ctx, span := trace.StartSpan(ctx, "basic-operation")
	defer span.End()

	// 设置属性
	trace.SetAttribute(span, "user.id", "12345")
	trace.SetAttribute(span, "operation.type", "example")

	// 添加事件
	trace.AddEvent(span, "processing started")

	// 模拟一些工作
	time.Sleep(100 * time.Millisecond)

	// 创建子span
	_, childSpan := trace.StartSpan(ctx, "child-operation")
	trace.SetAttribute(childSpan, "step", "data-processing")
	
	// 模拟错误处理
	err := fmt.Errorf("模拟错误")
	trace.RecordError(childSpan, err)
	
	childSpan.End()

	trace.AddEvent(span, "processing completed")
	fmt.Println("✓ 基本span示例完成")
}

// httpServerExample HTTP服务器示例
func httpServerExample() {
	mux := http.NewServeMux()

	// 添加路由处理器
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		// 从请求中提取追踪上下文
		ctx := trace.ExtractOtelTraceContext(r)

		// 创建业务逻辑span
		ctx, span := trace.StartSpan(ctx, "get-users")
		defer span.End()

		trace.SetAttribute(span, "http.method", r.Method)
		trace.SetAttribute(span, "http.route", "/api/users")

		// 模拟数据库查询
		ctx, dbSpan := trace.StartSpan(ctx, "database-query")
		trace.SetAttribute(dbSpan, "db.operation", "SELECT")
		trace.SetAttribute(dbSpan, "db.table", "users")
		
		time.Sleep(50 * time.Millisecond) // 模拟数据库延迟
		dbSpan.End()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`))
	})

	// 使用OpenTelemetry中间件
	handler := trace.OpenTelemetryMiddleware(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	fmt.Println("✓ HTTP服务器启动在 :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP服务器错误: %v", err)
	}
}

// httpClientExample HTTP客户端示例
func httpClientExample() {
	ctx := context.Background()

	// 创建带追踪的HTTP客户端
	client := trace.NewTracedHTTPClient(10 * time.Second)

	// 创建请求span
	ctx, span := trace.StartSpan(ctx, "http-client-request")
	defer span.End()

	trace.SetAttribute(span, "client.name", "example-client")

	// 发送请求
	resp, err := client.Get(ctx, "http://localhost:8080/api/users")
	if err != nil {
		trace.RecordError(span, err)
		log.Printf("HTTP请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	trace.SetAttribute(span, "http.status_code", resp.StatusCode)
	fmt.Printf("✓ HTTP客户端请求完成，状态码: %d\n", resp.StatusCode)
}