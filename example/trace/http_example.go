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
	fmt.Println("=== HTTP 追踪示例 ===")

	// 创建带追踪功能的HTTP客户端
	client := trace.NewTracedHTTPClient(5 * time.Second)

	// 创建根span（模拟外部请求）
	rootCtx := context.Background()
	rootTraceCtx := trace.CreateRootSpan()
	ctx := trace.WithTraceContext(rootCtx, rootTraceCtx)

	fmt.Printf("根请求: %s\n", rootTraceCtx.String())

	// 模拟调用下游服务
	callDownstreamService(ctx, client)

	// 模拟调用另一个下游服务
	callAnotherDownstreamService(ctx, client)
}

func callDownstreamService(ctx context.Context, client *trace.TracedHTTPClient) {
	// 记录当前操作
	trace.LogTraceContext(ctx, "调用下游服务")

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/get", nil)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return
	}

	// 执行请求
	resp, err := client.Do(ctx, req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 从响应头部获取追踪信息
	traceID := resp.Header.Get(trace.TraceIDHeader)
	spanID := resp.Header.Get(trace.SpanIDHeader)

	fmt.Printf("下游服务响应 - TraceID: %s, SpanID: %s\n", traceID, spanID)
}

func callAnotherDownstreamService(ctx context.Context, client *trace.TracedHTTPClient) {
	// 记录当前操作
	trace.LogTraceContext(ctx, "调用另一个下游服务")

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", "https://httpbin.org/post", nil)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return
	}

	// 执行请求
	resp, err := client.Do(ctx, req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 从响应头部获取追踪信息
	traceID := resp.Header.Get(trace.TraceIDHeader)
	spanID := resp.Header.Get(trace.SpanIDHeader)

	fmt.Printf("另一个下游服务响应 - TraceID: %s, SpanID: %s\n", traceID, spanID)
}
