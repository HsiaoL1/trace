package trace

import (
	"context"
	"fmt"
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
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// 创建HTTP客户端span
	ctx, span := StartHTTPClientSpan(ctx, req.Method, req.URL.String())
	defer func() {
		if span != nil {
			span.End()
		}
	}()

	// 注入OpenTelemetry追踪上下文到请求头
	InjectTraceContext(ctx, req)

	// 从context中获取自定义追踪上下文（向后兼容）
	traceCtx := GetTraceContextFromContext(ctx)
	if !traceCtx.IsValid() {
		traceCtx = CreateRootSpan()
		ctx = WithTraceContext(ctx, traceCtx)
	}

	// 设置自定义追踪头部（向后兼容）
	ctx = WithHttpRequest(ctx, req)
	SetTraceContextToHttpHeader(ctx, traceCtx)

	// 执行HTTP请求
	resp, err := c.client.Do(req.WithContext(ctx))

	// 完成span
	FinishHTTPClientSpan(span, resp, err)

	return resp, err
}

// Get 执行GET请求
func (c *TracedHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	return c.Do(ctx, req)
}

// Post 执行POST请求
func (c *TracedHTTPClient) Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// HTTPMiddleware HTTP中间件，用于自动处理追踪上下文
// 注意：推荐使用 OpenTelemetryMiddleware 更标准的中间件
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if next == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

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
