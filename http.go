package trace

import (
	"context"
	"fmt"
	"net/http"
)

// HTTP头部常量定义
const (
	TraceIDHeader      = "X-Trace-ID"
	SpanIDHeader       = "X-Span-ID"
	ParentSpanIDHeader = "X-Parent-Span-ID"
)

// TraceContext 表示追踪上下文
type TraceContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
}

// 定义trace context的key
type traceContextKey string

const (
	TraceContextKey traceContextKey = "trace_context"
	HttpRequestKey  traceContextKey = "http_request"
)

// SetTraceIDToHttpHeader 将trace ID设置到HTTP头部
func SetTraceIDToHttpHeader(ctx context.Context, traceID string) {
	if req, ok := ctx.Value(HttpRequestKey).(*http.Request); ok {
		req.Header.Set(TraceIDHeader, traceID)
	}
}

// SetSpanIDToHttpHeader 将span ID设置到HTTP头部
func SetSpanIDToHttpHeader(ctx context.Context, spanID string) {
	if req, ok := ctx.Value(HttpRequestKey).(*http.Request); ok {
		req.Header.Set(SpanIDHeader, spanID)
	}
}

// SetParentSpanIDToHttpHeader 将父span ID设置到HTTP头部
func SetParentSpanIDToHttpHeader(ctx context.Context, parentSpanID string) {
	if req, ok := ctx.Value(HttpRequestKey).(*http.Request); ok {
		req.Header.Set(ParentSpanIDHeader, parentSpanID)
	}
}

// SetTraceContextToHttpHeader 将完整的追踪上下文设置到HTTP头部
func SetTraceContextToHttpHeader(ctx context.Context, traceCtx TraceContext) {
	if req, ok := ctx.Value(HttpRequestKey).(*http.Request); ok {
		if traceCtx.TraceID != "" {
			req.Header.Set(TraceIDHeader, traceCtx.TraceID)
		}
		if traceCtx.SpanID != "" {
			req.Header.Set(SpanIDHeader, traceCtx.SpanID)
		}
		if traceCtx.ParentSpanID != "" {
			req.Header.Set(ParentSpanIDHeader, traceCtx.ParentSpanID)
		}
	}
}

// GetTraceContextFromHttpHeader 从HTTP头部获取追踪上下文
func GetTraceContextFromHttpHeader(req *http.Request) TraceContext {
	return TraceContext{
		TraceID:      req.Header.Get(TraceIDHeader),
		SpanID:       req.Header.Get(SpanIDHeader),
		ParentSpanID: req.Header.Get(ParentSpanIDHeader),
	}
}

// GetTraceContextFromContext 从context中获取追踪上下文
func GetTraceContextFromContext(ctx context.Context) TraceContext {
	if traceCtx, ok := ctx.Value(TraceContextKey).(TraceContext); ok {
		return traceCtx
	}
	return TraceContext{}
}

// WithTraceContext 将追踪上下文注入到context中
func WithTraceContext(ctx context.Context, traceCtx TraceContext) context.Context {
	return context.WithValue(ctx, TraceContextKey, traceCtx)
}

// WithHttpRequest 将HTTP请求注入到context中
func WithHttpRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, HttpRequestKey, req)
}

// CreateChildSpan 创建子span的上下文
// 当服务A调用服务B时，服务B会使用这个函数创建新的span
func CreateChildSpan(parentTraceCtx TraceContext) TraceContext {
	// 生成新的span ID作为当前span
	newSpanID := GenerateSpanID().String()

	return TraceContext{
		TraceID:      parentTraceCtx.TraceID, // 保持相同的trace ID
		SpanID:       newSpanID,              // 生成新的span ID
		ParentSpanID: parentTraceCtx.SpanID,  // 父span ID为上游的span ID
	}
}

// CreateRootSpan 创建根span的上下文
// 用于创建新的追踪链路
func CreateRootSpan() TraceContext {
	return TraceContext{
		TraceID:      GenerateTraceID().String(),
		SpanID:       GenerateSpanID().String(),
		ParentSpanID: "", // 根span没有父span
	}
}

// IsValidTraceContext 验证追踪上下文是否有效
func (tc TraceContext) IsValid() bool {
	return tc.TraceID != "" && tc.SpanID != ""
}

// String 返回追踪上下文的字符串表示
func (tc TraceContext) String() string {
	if tc.ParentSpanID != "" {
		return fmt.Sprintf("TraceID: %s, SpanID: %s, ParentSpanID: %s",
			tc.TraceID, tc.SpanID, tc.ParentSpanID)
	}
	return fmt.Sprintf("TraceID: %s, SpanID: %s", tc.TraceID, tc.SpanID)
}
