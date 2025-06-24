package logz

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetLevel(t *testing.T) {
	// 测试设置不同级别
	testCases := []struct {
		input    string
		expected string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warning"},
		{LevelError, "error"},
		{LevelFatal, "fatal"},
		{LevelPanic, "panic"},
	}

	for _, tc := range testCases {
		SetLevel(tc.input)
		if Logrus.GetLevel().String() != tc.expected {
			t.Errorf("设置级别 %s 失败，当前级别: %s，期望: %s", tc.input, Logrus.GetLevel().String(), tc.expected)
		}
	}

	// 测试无效级别
	SetLevel("invalid")
	if Logrus.GetLevel().String() != "info" {
		t.Errorf("无效级别应该默认为 info，当前级别: %s", Logrus.GetLevel().String())
	}
}

func TestSetFormat(t *testing.T) {
	// 测试文本格式
	SetFormat(FormatText)
	if _, ok := Logrus.Formatter.(*logrus.TextFormatter); !ok {
		t.Error("设置文本格式失败")
	}

	// 测试JSON格式
	SetFormat(FormatJSON)
	if _, ok := Logrus.Formatter.(*logrus.JSONFormatter); !ok {
		t.Error("设置JSON格式失败")
	}
}

func TestSetFileOutput(t *testing.T) {
	// 测试文件输出
	testFile := "testdata/test.log"

	// 清理测试文件
	defer func() {
		os.RemoveAll("testdata")
		SetOutput(os.Stdout) // 恢复标准输出
	}()

	err := SetFileOutput(testFile)
	if err != nil {
		t.Errorf("设置文件输出失败: %v", err)
	}

	// 写入测试日志
	Info("测试日志消息")

	// 检查文件是否存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("日志文件未创建")
	}
}

func TestLogMethods(t *testing.T) {
	// 测试各种日志方法
	Debug("调试信息")
	Info("信息日志")
	Warn("警告信息")
	Error("错误信息")

	// 测试格式化日志
	Debugf("调试信息: %s", "test")
	Infof("信息日志: %s", "test")
	Warnf("警告信息: %s", "test")
	Errorf("错误信息: %s", "test")
}

func TestWithFields(t *testing.T) {
	// 测试带字段的日志
	entry := WithField("user_id", "123")
	entry.Info("用户登录")

	// 测试多个字段
	fields := logrus.Fields{
		"user_id": "123",
		"action":  "login",
		"ip":      "192.168.1.1",
	}
	WithFields(fields).Info("用户操作")
}

func TestWithTrace(t *testing.T) {
	// 测试带追踪上下文的日志
	traceID := "abc123"
	spanID := "def456"

	DebugWithTrace(traceID, spanID, "调试信息")
	InfoWithTrace(traceID, spanID, "信息日志")
	ErrorWithTrace(traceID, spanID, "错误信息")

	DebugfWithTrace(traceID, spanID, "调试信息: %s", "test")
	InfofWithTrace(traceID, spanID, "信息日志: %s", "test")
	ErrorfWithTrace(traceID, spanID, "错误信息: %s", "test")
}

func TestInitMethods(t *testing.T) {
	// 测试初始化方法
	InitDefault()
	if Logrus.GetLevel().String() != LevelInfo {
		t.Error("默认初始化级别设置错误")
	}

	InitDevelopment()
	if Logrus.GetLevel().String() != LevelDebug {
		t.Error("开发环境初始化级别设置错误")
	}

	// 测试生产环境初始化
	err := InitProduction("")
	if err != nil {
		t.Errorf("生产环境初始化失败: %v", err)
	}
	if Logrus.GetLevel().String() != LevelInfo {
		t.Error("生产环境初始化级别设置错误")
	}
}

func TestEnableDisableCaller(t *testing.T) {
	// 测试启用调用者信息
	EnableCaller()
	if !Logrus.ReportCaller {
		t.Error("启用调用者信息失败")
	}

	// 测试禁用调用者信息
	DisableCaller()
	if Logrus.ReportCaller {
		t.Error("禁用调用者信息失败")
	}
}
