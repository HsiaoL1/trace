package logz

import (
	"testing"
	"time"

	"github.com/HsiaoL1/trace"
)

func TestEmailNotification(t *testing.T) {
	// 设置接收邮箱
	trace.SetEmail("test@example.com")

	// 测试错误日志邮件通知
	t.Run("ErrorWithEmail", func(t *testing.T) {
		ErrorWithEmail(true, "测试错误日志邮件通知")
		// 等待异步邮件发送
		time.Sleep(100 * time.Millisecond)
	})

	// 测试格式化错误日志邮件通知
	t.Run("ErrorfWithEmail", func(t *testing.T) {
		ErrorfWithEmail(true, "测试格式化错误日志: %s", "邮件通知")
		// 等待异步邮件发送
		time.Sleep(100 * time.Millisecond)
	})

	// 测试带追踪上下文的错误日志邮件通知
	t.Run("ErrorWithTraceAndEmail", func(t *testing.T) {
		traceID := "test_trace_123"
		spanID := "test_span_456"
		ErrorWithTraceAndEmail(traceID, spanID, true, "测试带追踪上下文的错误日志邮件通知")
		// 等待异步邮件发送
		time.Sleep(100 * time.Millisecond)
	})

	// 测试格式化带追踪上下文的错误日志邮件通知
	t.Run("ErrorfWithTraceAndEmail", func(t *testing.T) {
		traceID := "test_trace_789"
		spanID := "test_span_012"
		ErrorfWithTraceAndEmail(traceID, spanID, true, "测试格式化带追踪上下文的错误日志: %s", "邮件通知")
		// 等待异步邮件发送
		time.Sleep(100 * time.Millisecond)
	})
}

func TestEmailNotificationWithoutEmail(t *testing.T) {
	// 清空邮箱配置
	trace.SetEmail("")

	// 测试没有配置邮箱时不会发送邮件
	t.Run("ErrorWithEmailNoConfig", func(t *testing.T) {
		ErrorWithEmail(true, "测试没有配置邮箱的错误日志")
		// 这里不会发送邮件，因为没有配置邮箱
	})
}

func TestEmailNotificationWithCaller(t *testing.T) {
	// 设置接收邮箱
	trace.SetEmail("test@example.com")

	// 启用调用者信息
	EnableCaller()

	t.Run("ErrorWithEmailAndCaller", func(t *testing.T) {
		ErrorWithEmail(true, "测试带调用者信息的错误日志邮件通知")
		// 等待异步邮件发送
		time.Sleep(100 * time.Millisecond)
	})

	// 禁用调用者信息
	DisableCaller()
}

func TestFatalEmailNotification(t *testing.T) {
	// 注意：这些测试会调用os.Exit(1)，所以需要谨慎使用
	// 在实际测试中，你可能想要跳过这些测试

	t.Skip("跳过Fatal测试，因为会调用os.Exit(1)")

	// 设置接收邮箱
	trace.SetEmail("test@example.com")

	// 测试致命错误日志邮件通知
	t.Run("FatalWithEmail", func(t *testing.T) {
		FatalWithEmail(true, "测试致命错误日志邮件通知")
	})

	// 测试格式化致命错误日志邮件通知
	t.Run("FatalfWithEmail", func(t *testing.T) {
		FatalfWithEmail(true, "测试格式化致命错误日志: %s", "邮件通知")
	})
}

func TestPanicEmailNotification(t *testing.T) {
	// 注意：这些测试会调用panic，所以需要谨慎使用
	// 在实际测试中，你可能想要跳过这些测试

	t.Skip("跳过Panic测试，因为会调用panic")

	// 设置接收邮箱
	trace.SetEmail("test@example.com")

	// 测试恐慌日志邮件通知
	t.Run("PanicWithEmail", func(t *testing.T) {
		PanicWithEmail(true, "测试恐慌日志邮件通知")
	})

	// 测试格式化恐慌日志邮件通知
	t.Run("PanicfWithEmail", func(t *testing.T) {
		PanicfWithEmail(true, "测试格式化恐慌日志: %s", "邮件通知")
	})
}
