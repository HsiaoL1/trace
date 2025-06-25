package logz

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLogAggregator(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	aggregateDir := filepath.Join(tempDir, "aggregated")

	// 创建聚合器
	aggregator, err := NewLogAggregator(aggregateDir, "test-service", 1024*1024, 5)
	if err != nil {
		t.Fatalf("创建聚合器失败: %v", err)
	}

	// 测试写入日志
	testEntries := []LogEntry{
		{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "info",
			Message:   "测试日志1",
			TraceID:   "trace-001",
			SpanID:    "span-001",
			Service:   "test-service",
		},
		{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "error",
			Message:   "测试错误",
			TraceID:   "trace-002",
			SpanID:    "span-002",
			Service:   "test-service",
		},
	}

	for _, entry := range testEntries {
		if err := aggregator.WriteLog(entry); err != nil {
			t.Errorf("写入日志失败: %v", err)
		}
	}

	// 强制刷新批量缓冲区
	aggregator.flushBatch()

	// 验证文件是否创建
	files, err := filepath.Glob(filepath.Join(aggregateDir, "*.log"))
	if err != nil {
		t.Fatalf("获取文件列表失败: %v", err)
	}

	if len(files) == 0 {
		t.Error("没有创建日志文件")
	}

	// 关闭聚合器
	aggregator.Close()
}

func TestQueryLogs(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	aggregateDir := filepath.Join(tempDir, "aggregated")

	// 创建聚合器
	aggregator, err := NewLogAggregator(aggregateDir, "test-service", 1024*1024, 5)
	if err != nil {
		t.Fatalf("创建聚合器失败: %v", err)
	}

	// 写入测试数据
	now := time.Now()
	testEntries := []LogEntry{
		{
			Timestamp: now.Format(time.RFC3339),
			Level:     "info",
			Message:   "用户登录成功",
			TraceID:   "trace-001",
			SpanID:    "span-001",
			Service:   "test-service",
		},
		{
			Timestamp: now.Format(time.RFC3339),
			Level:     "error",
			Message:   "数据库连接失败",
			TraceID:   "trace-002",
			SpanID:    "span-002",
			Service:   "test-service",
		},
		{
			Timestamp: now.Format(time.RFC3339),
			Level:     "info",
			Message:   "系统启动",
			TraceID:   "trace-003",
			SpanID:    "span-003",
			Service:   "other-service",
		},
	}

	for _, entry := range testEntries {
		if err := aggregator.WriteLog(entry); err != nil {
			t.Errorf("写入日志失败: %v", err)
		}
	}

	// 强制刷新批量缓冲区
	aggregator.flushBatch()

	// 关闭聚合器
	aggregator.Close()

	// 验证文件是否创建
	files, err := filepath.Glob(filepath.Join(aggregateDir, "*.log"))
	if err != nil {
		t.Fatalf("获取文件列表失败: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("没有创建日志文件")
	}

	// 测试按TraceID查询
	t.Run("QueryByTraceID", func(t *testing.T) {
		result, err := QueryLogsByTraceID("trace-001", aggregateDir, 10, 0)
		if err != nil {
			t.Errorf("查询失败: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("期望找到1条日志，实际找到%d条", result.Total)
		}
		if len(result.Entries) > 0 && result.Entries[0].Message != "用户登录成功" {
			t.Errorf("期望消息为'用户登录成功'，实际为'%s'", result.Entries[0].Message)
		}
	})

	// 测试按级别查询
	t.Run("QueryByLevel", func(t *testing.T) {
		result, err := QueryLogsByLevel("error", aggregateDir, 10, 0)
		if err != nil {
			t.Errorf("查询失败: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("期望找到1条错误日志，实际找到%d条", result.Total)
		}
		if len(result.Entries) > 0 && result.Entries[0].Message != "数据库连接失败" {
			t.Errorf("期望消息为'数据库连接失败'，实际为'%s'", result.Entries[0].Message)
		}
	})

	// 测试按服务名查询
	t.Run("QueryByService", func(t *testing.T) {
		result, err := QueryLogsByService("test-service", aggregateDir, 10, 0)
		if err != nil {
			t.Errorf("查询失败: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("期望找到2条test-service日志，实际找到%d条", result.Total)
		}
	})

	// 测试按时间范围查询
	t.Run("QueryByTimeRange", func(t *testing.T) {
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)
		result, err := QueryLogsByTimeRange(startTime, endTime, aggregateDir, 10, 0)
		if err != nil {
			t.Errorf("查询失败: %v", err)
		}
		if result.Total != 3 {
			t.Errorf("期望找到3条日志，实际找到%d条", result.Total)
		}
	})

	// 测试按消息内容查询
	t.Run("QueryByMessage", func(t *testing.T) {
		result, err := QueryLogsByMessage(".*登录.*", aggregateDir, 10, 0)
		if err != nil {
			t.Errorf("查询失败: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("期望找到1条包含'登录'的日志，实际找到%d条", result.Total)
		}
	})
}

func TestCleanupOldLogs(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	aggregateDir := filepath.Join(tempDir, "aggregated")

	// 创建聚合器
	aggregator, err := NewLogAggregator(aggregateDir, "test-service", 1024*1024, 5)
	if err != nil {
		t.Fatalf("创建聚合器失败: %v", err)
	}
	defer aggregator.Close()

	// 写入一些测试数据
	now := time.Now()
	entry := LogEntry{
		Timestamp: now.Format(time.RFC3339),
		Level:     "info",
		Message:   "测试日志",
		TraceID:   "trace-001",
		SpanID:    "span-001",
		Service:   "test-service",
	}

	if err := aggregator.WriteLog(entry); err != nil {
		t.Errorf("写入日志失败: %v", err)
	}

	// 验证文件存在
	files, err := filepath.Glob(filepath.Join(aggregateDir, "*.log"))
	if err != nil {
		t.Fatalf("获取文件列表失败: %v", err)
	}

	if len(files) == 0 {
		t.Error("没有创建日志文件")
	}

	// 测试清理功能（由于文件是刚创建的，应该不会被清理）
	err = CleanupOldLogs(aggregateDir, 1) // 清理1天前的文件
	if err != nil {
		t.Errorf("清理失败: %v", err)
	}

	// 验证文件仍然存在
	files, err = filepath.Glob(filepath.Join(aggregateDir, "*.log"))
	if err != nil {
		t.Fatalf("获取文件列表失败: %v", err)
	}

	if len(files) == 0 {
		t.Error("文件被意外清理了")
	}
}

func TestGetLogStats(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	aggregateDir := filepath.Join(tempDir, "aggregated")

	// 创建聚合器
	aggregator, err := NewLogAggregator(aggregateDir, "test-service", 1024*1024, 5)
	if err != nil {
		t.Fatalf("创建聚合器失败: %v", err)
	}
	defer aggregator.Close()

	// 写入一些测试数据
	now := time.Now()
	entry := LogEntry{
		Timestamp: now.Format(time.RFC3339),
		Level:     "info",
		Message:   "测试日志",
		TraceID:   "trace-001",
		SpanID:    "span-001",
		Service:   "test-service",
	}

	if err := aggregator.WriteLog(entry); err != nil {
		t.Errorf("写入日志失败: %v", err)
	}

	// 获取统计信息
	stats, err := GetLogStats(aggregateDir)
	if err != nil {
		t.Errorf("获取统计信息失败: %v", err)
	}

	// 验证统计信息
	if stats["total_files"].(int) != 1 {
		t.Errorf("期望文件数为1，实际为%d", stats["total_files"])
	}

	if stats["total_size"].(int64) <= 0 {
		t.Errorf("期望文件大小大于0，实际为%d", stats["total_size"])
	}

	if stats["oldest_file"] == "" {
		t.Error("最旧文件名不能为空")
	}

	if stats["newest_file"] == "" {
		t.Error("最新文件名不能为空")
	}
}

func TestAggregatorHook(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	aggregateDir := filepath.Join(tempDir, "aggregated")

	// 创建聚合器
	aggregator, err := NewLogAggregator(aggregateDir, "test-service", 1024*1024, 5)
	if err != nil {
		t.Fatalf("创建聚合器失败: %v", err)
	}
	defer aggregator.Close()

	// 创建Hook
	hook := NewAggregatorHook(aggregator, "test-service")

	// 创建logrus条目
	entry := &logrus.Entry{
		Time:    time.Now(),
		Level:   logrus.InfoLevel,
		Message: "Hook测试日志",
		Data: logrus.Fields{
			"trace_id": "trace-hook-001",
			"span_id":  "span-hook-001",
			"user_id":  "123",
		},
	}

	// 测试Hook
	if err := hook.Fire(entry); err != nil {
		t.Errorf("Hook执行失败: %v", err)
	}

	// 验证日志是否被写入
	files, err := filepath.Glob(filepath.Join(aggregateDir, "*.log"))
	if err != nil {
		t.Fatalf("获取文件列表失败: %v", err)
	}

	if len(files) == 0 {
		t.Error("Hook没有写入日志文件")
	}
}
