package trace

import (
	"testing"
)

func TestGenerateTraceID(t *testing.T) {
	traceID1 := GenerateTraceID()
	traceID2 := GenerateTraceID()

	// 检查生成的trace ID是否有效
	if !traceID1.IsValid() {
		t.Error("Generated trace ID is not valid")
	}

	if !traceID2.IsValid() {
		t.Error("Generated trace ID is not valid")
	}

	// 检查两个trace ID是否不同
	if traceID1 == traceID2 {
		t.Error("Generated trace IDs should be different")
	}

	// 检查字符串表示
	str1 := traceID1.String()
	str2 := traceID2.String()

	if len(str1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Expected trace ID string length 32, got %d", len(str1))
	}

	if len(str2) != 32 {
		t.Errorf("Expected trace ID string length 32, got %d", len(str2))
	}

	t.Logf("Generated trace ID 1: %s", str1)
	t.Logf("Generated trace ID 2: %s", str2)
}

func TestGenerateSpanID(t *testing.T) {
	spanID1 := GenerateSpanID()
	spanID2 := GenerateSpanID()

	// 检查生成的span ID是否有效
	if !spanID1.IsValid() {
		t.Error("Generated span ID is not valid")
	}

	if !spanID2.IsValid() {
		t.Error("Generated span ID is not valid")
	}

	// 检查两个span ID是否不同
	if spanID1 == spanID2 {
		t.Error("Generated span IDs should be different")
	}

	// 检查字符串表示
	str1 := spanID1.String()
	str2 := spanID2.String()

	if len(str1) != 16 { // 8 bytes = 16 hex chars
		t.Errorf("Expected span ID string length 16, got %d", len(str1))
	}

	if len(str2) != 16 {
		t.Errorf("Expected span ID string length 16, got %d", len(str2))
	}

	t.Logf("Generated span ID 1: %s", str1)
	t.Logf("Generated span ID 2: %s", str2)
}

func TestTraceIDValidity(t *testing.T) {
	// 测试有效的trace ID
	validTraceID := GenerateTraceID()
	if !validTraceID.IsValid() {
		t.Error("Valid trace ID should return true for IsValid()")
	}

	// 测试无效的trace ID（全为0）
	var invalidTraceID TraceID
	if invalidTraceID.IsValid() {
		t.Error("Invalid trace ID (all zeros) should return false for IsValid()")
	}
}

func TestSpanIDValidity(t *testing.T) {
	// 测试有效的span ID
	validSpanID := GenerateSpanID()
	if !validSpanID.IsValid() {
		t.Error("Valid span ID should return true for IsValid()")
	}

	// 测试无效的span ID（全为0）
	var invalidSpanID SpanID
	if invalidSpanID.IsValid() {
		t.Error("Invalid span ID (all zeros) should return false for IsValid()")
	}
}