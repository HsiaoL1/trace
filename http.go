package trace

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
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

// SetTraceContextToHttpHeader 将完整的追踪上下文设置到HTTP头部
// 同时设置自定义头部和OpenTelemetry标准头部
func SetTraceContextToHttpHeader(ctx context.Context, traceCtx TraceContext) {
	if req, ok := ctx.Value(HttpRequestKey).(*http.Request); ok {
		// 设置自定义头部（向后兼容）
		if traceCtx.TraceID != "" {
			req.Header.Set(TraceIDHeader, traceCtx.TraceID)
		}
		if traceCtx.SpanID != "" {
			req.Header.Set(SpanIDHeader, traceCtx.SpanID)
		}
		if traceCtx.ParentSpanID != "" {
			req.Header.Set(ParentSpanIDHeader, traceCtx.ParentSpanID)
		}
		// 同时注入OpenTelemetry标准追踪上下文
		InjectTraceContext(ctx, req)
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
	if !parentTraceCtx.IsValid() {
		return CreateRootSpan()
	}
	
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

// IsValid 验证追踪上下文是否有效
func (tc TraceContext) IsValid() bool {
	return tc.TraceID != "" && tc.SpanID != "" && 
		   len(strings.TrimSpace(tc.TraceID)) > 0 && 
		   len(strings.TrimSpace(tc.SpanID)) > 0
}

// String 返回追踪上下文的字符串表示
func (tc TraceContext) String() string {
	if tc.ParentSpanID != "" {
		return fmt.Sprintf("TraceID: %s, SpanID: %s, ParentSpanID: %s",
			tc.TraceID, tc.SpanID, tc.ParentSpanID)
	}
	return fmt.Sprintf("TraceID: %s, SpanID: %s", tc.TraceID, tc.SpanID)
}

// OpenTelemetryMiddleware OpenTelemetry HTTP中间件
func OpenTelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从请求头部提取追踪上下文
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// 创建span
		tracer := otel.Tracer("github.com/HsiaoL1/trace/http")
		spanName := generateSpanName(r)
		ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		// 设置HTTP相关属性
		setHTTPServerSpanAttributes(span, r)

		// 创建响应writer包装器来捕获状态码
		wrappedWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		// 将上下文传递给下一个处理器
		next.ServeHTTP(wrappedWriter, r.WithContext(ctx))

		// 设置响应属性
		setHTTPResponseSpanAttributes(span, wrappedWriter.statusCode)
	})
}

// responseWriter 包装器用于捕获HTTP响应状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

// generateSpanName 生成span名称
func generateSpanName(r *http.Request) string {
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	return fmt.Sprintf("%s %s", r.Method, path)
}

// setHTTPServerSpanAttributes 设置HTTP服务器span属性
func setHTTPServerSpanAttributes(span trace.Span, r *http.Request) {
	span.SetAttributes(
		semconv.HTTPMethod(r.Method),
		semconv.HTTPURL(r.URL.String()),
		semconv.HTTPRoute(r.URL.Path),
	)

	if r.URL.Scheme != "" {
		span.SetAttributes(semconv.HTTPScheme(r.URL.Scheme))
	}

	if r.ContentLength > 0 {
		span.SetAttributes(semconv.HTTPRequestContentLength(int(r.ContentLength)))
	}

	if userAgent := r.UserAgent(); userAgent != "" {
		span.SetAttributes(attribute.String("http.user_agent", userAgent))
	}

	if remoteAddr := r.RemoteAddr; remoteAddr != "" {
		// 提取IP地址（移除端口）
		if colonIndex := strings.LastIndex(remoteAddr, ":"); colonIndex != -1 {
			remoteAddr = remoteAddr[:colonIndex]
		}
		span.SetAttributes(attribute.String("net.peer.ip", remoteAddr))
	}

	if host := r.Host; host != "" {
		span.SetAttributes(attribute.String("http.host", host))
	}
}

// setHTTPResponseSpanAttributes 设置HTTP响应span属性
func setHTTPResponseSpanAttributes(span trace.Span, statusCode int) {
	span.SetAttributes(semconv.HTTPStatusCode(statusCode))

	// 根据状态码设置span状态
	if statusCode >= 400 {
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}
}

// InjectTraceContext 注入追踪上下文到HTTP请求
func InjectTraceContext(ctx context.Context, req *http.Request) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// ExtractOtelTraceContext 从HTTP请求提取OpenTelemetry追踪上下文
func ExtractOtelTraceContext(req *http.Request) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))
}

// StartHTTPClientSpan 为HTTP客户端请求创建span
func StartHTTPClientSpan(ctx context.Context, method, url string) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/HsiaoL1/trace/http-client")
	spanName := fmt.Sprintf("%s %s", method, url)
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
	
	// 设置HTTP客户端属性
	span.SetAttributes(
		semconv.HTTPMethod(method),
		semconv.HTTPURL(url),
	)
	
	return ctx, span
}

// FinishHTTPClientSpan 完成HTTP客户端span
func FinishHTTPClientSpan(span trace.Span, resp *http.Response, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if resp != nil {
		span.SetAttributes(
			semconv.HTTPStatusCode(resp.StatusCode),
			attribute.String("http.response.content_length", strconv.FormatInt(resp.ContentLength, 10)),
		)
		
		if resp.StatusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		}
	}
	span.End()
}
