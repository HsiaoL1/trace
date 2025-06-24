package main

import (
	"context"
	"time"

	"github.com/HsiaoL1/trace"
	"github.com/HsiaoL1/trace/logz"
)

func main() {
	// 初始化日志
	logz.InitDevelopment()
	logz.EnableCaller()

	logz.Info("=== 分布式追踪与日志整合演示 ===")

	// 创建根span
	rootCtx := context.Background()
	rootTraceCtx := trace.CreateRootSpan()
	ctx := trace.WithTraceContext(rootCtx, rootTraceCtx)

	// 记录根请求日志
	logz.InfoWithTrace(rootTraceCtx.TraceID, rootTraceCtx.SpanID, "收到用户请求")

	// 模拟业务处理
	processUserRequest(ctx)

	logz.InfoWithTrace(rootTraceCtx.TraceID, rootTraceCtx.SpanID, "请求处理完成")
}

func processUserRequest(ctx context.Context) {
	traceCtx := trace.GetTraceContextFromContext(ctx)

	logz.InfoWithTrace(traceCtx.TraceID, traceCtx.SpanID, "开始处理用户请求")

	// 模拟数据库操作
	logz.DebugWithTrace(traceCtx.TraceID, traceCtx.SpanID, "查询用户信息")
	time.Sleep(50 * time.Millisecond)
	logz.InfoWithTrace(traceCtx.TraceID, traceCtx.SpanID, "用户信息查询成功")

	// 模拟外部服务调用
	logz.InfoWithTrace(traceCtx.TraceID, traceCtx.SpanID, "调用外部服务")
	time.Sleep(100 * time.Millisecond)
	logz.InfoWithTrace(traceCtx.TraceID, traceCtx.SpanID, "外部服务调用成功")

	// 模拟业务逻辑
	logz.DebugWithTrace(traceCtx.TraceID, traceCtx.SpanID, "执行业务逻辑")
	time.Sleep(30 * time.Millisecond)
	logz.InfoWithTrace(traceCtx.TraceID, traceCtx.SpanID, "业务逻辑执行完成")
}
