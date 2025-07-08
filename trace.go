package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TraceID 表示分布式追踪的trace ID
type TraceID [16]byte

// SpanID 表示分布式追踪的span ID
type SpanID [8]byte

// GenerateTraceID 生成一个新的trace ID
// 使用UUID v4格式，符合OpenTelemetry规范
func GenerateTraceID() TraceID {
	var traceID TraceID
	_, err := rand.Read(traceID[:])
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为fallback
		log.Printf("Warning: crypto/rand failed, using timestamp fallback: %v", err)
		timestamp := time.Now().UnixNano()
		// 正确的字节数据填充
		for i := 0; i < 8; i++ {
			traceID[i] = byte(timestamp >> (8 * i))
		}
		for i := 8; i < 16; i++ {
			traceID[i] = byte(timestamp >> (8 * (i - 8)))
		}
	}
	return traceID
}

// GenerateSpanID 生成一个新的span ID
// 使用8字节随机数，符合OpenTelemetry规范
func GenerateSpanID() SpanID {
	var spanID SpanID
	_, err := rand.Read(spanID[:])
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为fallback
		log.Printf("Warning: crypto/rand failed, using timestamp fallback: %v", err)
		timestamp := time.Now().UnixNano()
		// 正确的字节数据填充
		for i := 0; i < 8; i++ {
			spanID[i] = byte(timestamp >> (8 * i))
		}
	}
	return spanID
}

// String 返回trace ID的十六进制字符串表示
func (t TraceID) String() string {
	return hex.EncodeToString(t[:])
}

// String 返回span ID的十六进制字符串表示
func (s SpanID) String() string {
	return hex.EncodeToString(s[:])
}

// IsValid 检查trace ID是否有效（不全为0）
func (t TraceID) IsValid() bool {
	for _, b := range t {
		if b != 0 {
			return true
		}
	}
	return false
}

// IsValid 检查span ID是否有效（不全为0）
func (s SpanID) IsValid() bool {
	for _, b := range s {
		if b != 0 {
			return true
		}
	}
	return false
}

// StartSpan 开始一个新的span
func StartSpan(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/HsiaoL1/trace")
	ctx, span := tracer.Start(ctx, operationName, opts...)
	// 添加基本的span属性
	span.SetAttributes(
		attribute.String("component", "github.com/HsiaoL1/trace"),
		attribute.String("span.kind", "internal"),
	)
	return ctx, span
}

// RecordError 记录错误到span
func RecordError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// AddEvent 添加事件到span
func AddEvent(span trace.Span, name string, attributes ...attribute.KeyValue) {
	if span == nil || name == "" {
		return
	}
	span.AddEvent(name, trace.WithAttributes(attributes...))
}

// SetAttribute 设置span属性
func SetAttribute(span trace.Span, key string, value any) {
	if span == nil {
		return
	}
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int32:
		span.SetAttributes(attribute.Int(key, int(v)))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float32:
		span.SetAttributes(attribute.Float64(key, float64(v)))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	default:
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// ConvertToOtelTraceID 将自定义TraceID转换为OpenTelemetry TraceID
func ConvertToOtelTraceID(traceID TraceID) trace.TraceID {
	return trace.TraceID(traceID)
}

// ConvertToOtelSpanID 将自定义SpanID转换为OpenTelemetry SpanID
func ConvertToOtelSpanID(spanID SpanID) trace.SpanID {
	return trace.SpanID(spanID)
}
