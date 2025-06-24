package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/HsiaoL1/trace"
)

// ServiceA 模拟服务A
type ServiceA struct {
	client *trace.TracedHTTPClient
}

// ServiceB 模拟服务B
type ServiceB struct{}

// ServiceC 模拟服务C
type ServiceC struct{}

// Response 通用响应结构
type Response struct {
	Message      string `json:"message"`
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
}

func main() {
	fmt.Println("=== Trace ID 和 Span ID 生成示例 ===")

	// 生成trace ID
	traceID := trace.GenerateTraceID()
	fmt.Printf("生成的 Trace ID: %s\n", traceID.String())
	fmt.Printf("Trace ID 是否有效: %t\n", traceID.IsValid())

	// 生成span ID
	spanID := trace.GenerateSpanID()
	fmt.Printf("生成的 Span ID: %s\n", spanID.String())
	fmt.Printf("Span ID 是否有效: %t\n", spanID.IsValid())

	// 模拟分布式追踪场景
	fmt.Println("\n=== 模拟分布式追踪场景 ===")

	// 根span
	rootTraceID := trace.GenerateTraceID()
	rootSpanID := trace.GenerateSpanID()

	fmt.Printf("根请求 - Trace ID: %s, Span ID: %s\n",
		rootTraceID.String(), rootSpanID.String())

	// 子span（使用相同的trace ID，但不同的span ID）
	childSpanID := trace.GenerateSpanID()
	fmt.Printf("子请求 - Trace ID: %s, Span ID: %s\n",
		rootTraceID.String(), childSpanID.String())

	// 另一个子span
	anotherChildSpanID := trace.GenerateSpanID()
	fmt.Printf("另一个子请求 - Trace ID: %s, Span ID: %s\n",
		rootTraceID.String(), anotherChildSpanID.String())

	// 验证ID的唯一性
	fmt.Println("\n=== 验证ID唯一性 ===")

	traceIDs := make(map[string]bool)
	spanIDs := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		tID := trace.GenerateTraceID()
		sID := trace.GenerateSpanID()

		if traceIDs[tID.String()] {
			log.Fatalf("发现重复的 Trace ID: %s", tID.String())
		}

		if spanIDs[sID.String()] {
			log.Fatalf("发现重复的 Span ID: %s", sID.String())
		}

		traceIDs[tID.String()] = true
		spanIDs[sID.String()] = true
	}

	fmt.Println("✓ 成功生成1000个唯一的 Trace ID 和 Span ID")

	// 运行服务间调用演示
	fmt.Println("\n=== 服务间调用演示 ===")
	runServiceDemo()
}

func runServiceDemo() {
	// 创建带追踪功能的HTTP客户端
	client := trace.NewTracedHTTPClient(10 * time.Second)

	// 创建服务实例
	serviceA := &ServiceA{client: client}
	serviceB := &ServiceB{}
	serviceC := &ServiceC{}

	// 启动服务B和C
	go startServiceB(serviceB)
	go startServiceC(serviceC)

	// 等待服务启动
	time.Sleep(100 * time.Millisecond)

	// 模拟服务A处理请求
	fmt.Println("=== 服务A处理请求 ===")

	// 创建根span（模拟外部请求）
	rootCtx := context.Background()
	rootTraceCtx := trace.CreateRootSpan()
	ctx := trace.WithTraceContext(rootCtx, rootTraceCtx)

	log.Printf("服务A收到请求: %s", rootTraceCtx.String())

	// 服务A调用服务B
	responseB, err := serviceA.CallServiceB(ctx)
	if err != nil {
		log.Printf("调用服务B失败: %v", err)
		return
	}

	log.Printf("服务B响应: %s", responseB.Message)

	// 服务A调用服务C
	responseC, err := serviceA.CallServiceC(ctx)
	if err != nil {
		log.Printf("调用服务C失败: %v", err)
		return
	}

	log.Printf("服务C响应: %s", responseC.Message)

	// 保持服务运行一段时间
	time.Sleep(2 * time.Second)
}

// CallServiceB 服务A调用服务B
func (s *ServiceA) CallServiceB(ctx context.Context) (*Response, error) {
	trace.LogTraceContext(ctx, "ServiceA.CallServiceB")

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8081/api/b", nil)
	if err != nil {
		return nil, err
	}

	// 执行请求
	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应
	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// CallServiceC 服务A调用服务C
func (s *ServiceA) CallServiceC(ctx context.Context) (*Response, error) {
	trace.LogTraceContext(ctx, "ServiceA.CallServiceC")

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8082/api/c", nil)
	if err != nil {
		return nil, err
	}

	// 执行请求
	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应
	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// startServiceB 启动服务B
func startServiceB(service *ServiceB) {
	mux := http.NewServeMux()

	// 使用追踪中间件
	mux.Handle("/api/b", trace.HTTPMiddleware(http.HandlerFunc(service.HandleRequest)))

	log.Println("服务B启动在端口 8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}

// startServiceC 启动服务C
func startServiceC(service *ServiceC) {
	mux := http.NewServeMux()

	// 使用追踪中间件
	mux.Handle("/api/c", trace.HTTPMiddleware(http.HandlerFunc(service.HandleRequest)))

	log.Println("服务C启动在端口 8082")
	if err := http.ListenAndServe(":8082", mux); err != nil {
		log.Fatal(err)
	}
}

// HandleRequest 服务B的处理函数
func (s *ServiceB) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// 从context中获取追踪上下文
	traceCtx := trace.GetTraceContextFromContext(r.Context())

	log.Printf("服务B收到请求: %s", traceCtx.String())

	// 模拟处理时间
	time.Sleep(100 * time.Millisecond)

	// 返回响应
	response := Response{
		Message:      "Hello from Service B",
		TraceID:      traceCtx.TraceID,
		SpanID:       traceCtx.SpanID,
		ParentSpanID: traceCtx.ParentSpanID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRequest 服务C的处理函数
func (s *ServiceC) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// 从context中获取追踪上下文
	traceCtx := trace.GetTraceContextFromContext(r.Context())

	log.Printf("服务C收到请求: %s", traceCtx.String())

	// 模拟处理时间
	time.Sleep(150 * time.Millisecond)

	// 返回响应
	response := Response{
		Message:      "Hello from Service C",
		TraceID:      traceCtx.TraceID,
		SpanID:       traceCtx.SpanID,
		ParentSpanID: traceCtx.ParentSpanID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
