package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/HsiaoL1/trace/logz"
)

func main() {
	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("创建日志目录失败:", err)
	}

	// 初始化带聚合功能的日志系统
	err := logz.InitWithAggregation(
		filepath.Join(logDir, "demo.log"), // 日志文件
		logDir,                            // 聚合目录
		"demo-service",                    // 服务名
		10*1024*1024,                      // 轮转大小 (10MB)
		5,                                 // 最大备份数
	)
	if err != nil {
		log.Fatal("初始化日志系统失败:", err)
	}
	defer logz.CloseAggregator()

	fmt.Println("开始生成测试日志...")
	fmt.Println("日志目录:", logDir)
	fmt.Println("Web界面: http://localhost:8080")
	fmt.Println("按 Ctrl+C 停止...")

	// 生成不同类型的日志
	go generateInfoLogs()
	go generateErrorLogs()
	go generateWarnLogs()

	// 保持程序运行
	select {}
}

func generateInfoLogs() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		traceID := fmt.Sprintf("trace_%d", time.Now().Unix())
		spanID := fmt.Sprintf("span_%d", time.Now().UnixNano()%1000)

		logz.InfoWithTrace(traceID, spanID, "这是一条信息日志")
		logz.InfofWithTrace(traceID, spanID, "用户 %s 登录了系统", "user123")
		logz.InfofWithTrace(traceID, spanID, "处理请求 %s，耗时 %dms", "GET /api/users", 150)
	}
}

func generateErrorLogs() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	errorCount := 0
	for range ticker.C {
		errorCount++
		traceID := fmt.Sprintf("error_trace_%d", errorCount)
		spanID := fmt.Sprintf("error_span_%d", errorCount)

		switch errorCount % 4 {
		case 0:
			logz.ErrorWithTraceAndEmail(traceID, spanID, true, "数据库连接失败")
		case 1:
			logz.ErrorfWithTraceAndEmail(traceID, spanID, true, "API调用失败: %s", "timeout")
		case 2:
			logz.ErrorWithTraceAndEmail(traceID, spanID, false, "文件读取错误")
		case 3:
			logz.ErrorfWithTraceAndEmail(traceID, spanID, false, "用户 %s 权限不足", "guest")
		}
	}
}

func generateWarnLogs() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	warnCount := 0
	for range ticker.C {
		warnCount++
		traceID := fmt.Sprintf("warn_trace_%d", warnCount)
		spanID := fmt.Sprintf("warn_span_%d", warnCount)

		switch warnCount % 3 {
		case 0:
			logz.InfoWithTrace(traceID, spanID, "系统资源使用率较高")
		case 1:
			logz.InfofWithTrace(traceID, spanID, "缓存命中率下降: %d%%", 75)
		case 2:
			logz.InfoWithTrace(traceID, spanID, "检测到异常访问模式")
		}
	}
}
