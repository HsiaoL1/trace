package trace

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"
)

// TracedHTTPClient 带追踪功能的HTTP客户端
type TracedHTTPClient struct {
	client *http.Client
}

// NewTracedHTTPClient 创建新的带追踪功能的HTTP客户端
func NewTracedHTTPClient(timeout time.Duration) *TracedHTTPClient {
	return &TracedHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Do 执行HTTP请求，自动传递追踪上下文
func (c *TracedHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// 从context中获取追踪上下文
	traceCtx := GetTraceContextFromContext(ctx)

	// 如果context中没有追踪上下文，创建一个根span
	if !traceCtx.IsValid() {
		traceCtx = CreateRootSpan()
	}

	// 将追踪上下文设置到HTTP头部
	SetTraceContextToHttpHeader(ctx, traceCtx)

	// 执行HTTP请求
	return c.client.Do(req)
}

// Get 执行GET请求
func (c *TracedHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post 执行POST请求
func (c *TracedHTTPClient) Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// HTTPMiddleware HTTP中间件，用于自动处理追踪上下文
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从HTTP头部获取追踪上下文
		traceCtx := GetTraceContextFromHttpHeader(r)

		// 如果没有追踪上下文，创建一个根span
		if !traceCtx.IsValid() {
			traceCtx = CreateRootSpan()
		} else {
			// 如果有追踪上下文，创建子span
			traceCtx = CreateChildSpan(traceCtx)
		}

		// 将追踪上下文注入到请求的context中
		ctx := WithTraceContext(r.Context(), traceCtx)
		ctx = WithHttpRequest(ctx, r)

		// 将追踪信息添加到响应头部（可选）
		w.Header().Set(TraceIDHeader, traceCtx.TraceID)
		w.Header().Set(SpanIDHeader, traceCtx.SpanID)

		// 使用新的context处理请求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ExtractTraceContext 从HTTP请求中提取追踪上下文并注入到context
func ExtractTraceContext(r *http.Request) context.Context {
	traceCtx := GetTraceContextFromHttpHeader(r)

	if !traceCtx.IsValid() {
		traceCtx = CreateRootSpan()
	} else {
		traceCtx = CreateChildSpan(traceCtx)
	}

	ctx := WithTraceContext(r.Context(), traceCtx)
	ctx = WithHttpRequest(ctx, r)

	return ctx
}

// LogTraceContext 记录追踪上下文信息（用于调试）
func LogTraceContext(ctx context.Context, operation string) {
	traceCtx := GetTraceContextFromContext(ctx)
	if traceCtx.IsValid() {
		log.Printf("[TRACE] %s - %s", operation, traceCtx.String())
	}
}
