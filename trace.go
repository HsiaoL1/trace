package trace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
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
		timestamp := time.Now().UnixNano()
		copy(traceID[:8], fmt.Sprintf("%016x", timestamp))
		copy(traceID[8:], fmt.Sprintf("%016x", timestamp>>32))
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
		timestamp := time.Now().UnixNano()
		copy(spanID[:], fmt.Sprintf("%016x", timestamp)[:8])
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
