package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/HsiaoL1/trace/logz"
	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化日志配置
	logz.InitDevelopment()

	// 启用调用者信息
	logz.EnableCaller()

	// 基本日志方法
	logz.Info("应用启动")
	logz.Debug("调试信息")
	logz.Warn("警告信息")
	logz.Error("错误信息")

	// 格式化日志
	logz.Infof("用户 %s 登录成功", "张三")
	logz.Errorf("处理请求失败: %v", errors.New("网络超时"))

	// 带字段的日志
	logz.WithField("user_id", "123").Info("用户操作")

	fields := logrus.Fields{
		"user_id": "123",
		"action":  "login",
		"ip":      "192.168.1.1",
		"time":    time.Now().Format("2006-01-02 15:04:05"),
	}
	logz.WithFields(fields).Info("用户登录")

	// 带错误的日志
	err := errors.New("数据库连接失败")
	logz.WithError(err).Error("系统错误")

	// 带追踪上下文的日志
	traceID := "abc123def456"
	spanID := "span789"

	logz.InfoWithTrace(traceID, spanID, "处理用户请求")
	logz.DebugWithTrace(traceID, spanID, "查询数据库")
	logz.ErrorWithTrace(traceID, spanID, "数据库查询失败")

	logz.InfofWithTrace(traceID, spanID, "用户 %s 的操作", "李四")
	logz.ErrorfWithTrace(traceID, spanID, "处理失败: %v", err)

	// 演示不同日志级别
	logz.SetLevel(logz.LevelDebug)
	logz.Debug("这条调试信息会显示")

	logz.SetLevel(logz.LevelInfo)
	logz.Debug("这条调试信息不会显示")
	logz.Info("这条信息会显示")

	// 演示JSON格式
	logz.SetFormat(logz.FormatJSON)
	logz.Info("JSON格式的日志")

	// 演示文本格式
	logz.SetFormat(logz.FormatText)
	logz.Info("文本格式的日志")

	// 模拟业务场景
	simulateBusinessScenario()

	// 演示日志聚合功能
	demonstrateAggregationFeatures()

	// 演示大规模日志处理
	demonstrateLargeScaleFeatures()
}

func simulateBusinessScenario() {
	logz.Info("=== 模拟业务场景 ===")

	// 用户注册
	userID := "user_123"
	logz.WithField("user_id", userID).Info("用户开始注册")

	// 验证邮箱
	logz.WithFields(logrus.Fields{
		"user_id": userID,
		"step":    "email_verification",
	}).Info("验证用户邮箱")

	// 创建用户
	logz.WithFields(logrus.Fields{
		"user_id": userID,
		"step":    "create_user",
	}).Info("创建用户账户")

	// 发送欢迎邮件
	logz.WithFields(logrus.Fields{
		"user_id": userID,
		"step":    "send_welcome_email",
	}).Info("发送欢迎邮件")

	// 模拟错误
	err := errors.New("邮件发送失败")
	logz.WithFields(logrus.Fields{
		"user_id": userID,
		"step":    "send_welcome_email",
		"error":   err.Error(),
	}).Error("邮件发送失败")

	logz.WithField("user_id", userID).Info("用户注册完成")
}

func demonstrateAggregationFeatures() {
	fmt.Println("\n=== 演示日志聚合功能 ===")

	// 初始化带聚合功能的日志系统
	err := logz.InitWithAggregation(
		"./logs/app.log",    // 普通日志文件
		"./logs/aggregated", // 聚合日志目录
		"user-service",      // 服务名
		100*1024*1024,       // 轮转大小 (100MB)
		10,                  // 最大备份数
	)
	if err != nil {
		log.Printf("初始化聚合日志系统失败: %v", err)
		return
	}
	defer logz.CloseAggregator()

	// 生成一些测试日志
	generateTestLogs()

	// 演示查询功能
	demonstrateQueryFeatures()

	// 演示统计功能
	demonstrateStatsFeatures()

	// 演示清理功能
	demonstrateCleanupFeatures()
}

func generateTestLogs() {
	fmt.Println("生成测试日志...")

	// 模拟不同的TraceID和SpanID
	traceIDs := []string{"trace-001", "trace-002", "trace-003"}
	spanIDs := []string{"span-001", "span-002", "span-003"}

	for i := 0; i < 5; i++ {
		traceID := traceIDs[i%len(traceIDs)]
		spanID := spanIDs[i%len(spanIDs)]

		// 使用带追踪上下文的日志方法
		logz.InfoWithTrace(traceID, spanID, "用户登录成功", "user_id", i+1)
		logz.DebugWithTrace(traceID, spanID, "处理用户请求", "request_id", fmt.Sprintf("req-%d", i+1))

		if i%3 == 0 {
			logz.ErrorWithTrace(traceID, spanID, "数据库连接失败", "error_code", "DB_001")
		}

		// 使用普通日志方法（也会被聚合）
		logz.Info("系统运行正常", "uptime", fmt.Sprintf("%d分钟", i*5))
		logz.Warn("内存使用率较高", "memory_usage", "85%")

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("测试日志生成完成")
}

func demonstrateQueryFeatures() {
	fmt.Println("\n演示查询功能:")

	// 1. 根据TraceID查询
	fmt.Println("1. 根据TraceID查询:")
	result, err := logz.QueryLogsByTraceID("trace-001", "./logs/aggregated", 10, 0)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 条日志\n", result.Total)
		for i, entry := range result.Entries {
			fmt.Printf("  %d. [%s] %s (TraceID: %s, SpanID: %s)\n",
				i+1, entry.Level, entry.Message, entry.TraceID, entry.SpanID)
		}
	}

	// 2. 根据时间范围查询
	fmt.Println("\n2. 根据时间范围查询:")
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()
	result, err = logz.QueryLogsByTimeRange(startTime, endTime, "./logs/aggregated", 5, 0)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 条日志\n", result.Total)
		for i, entry := range result.Entries {
			fmt.Printf("  %d. [%s] %s (时间: %s)\n",
				i+1, entry.Level, entry.Message, entry.Timestamp)
		}
	}

	// 3. 根据日志级别查询
	fmt.Println("\n3. 根据日志级别查询:")
	result, err = logz.QueryLogsByLevel("error", "./logs/aggregated", 10, 0)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 条错误日志\n", result.Total)
		for i, entry := range result.Entries {
			fmt.Printf("  %d. %s (TraceID: %s)\n",
				i+1, entry.Message, entry.TraceID)
		}
	}
}

func demonstrateStatsFeatures() {
	fmt.Println("\n演示统计功能:")

	// 获取日志统计信息
	stats, err := logz.GetLogStatsDefault("./logs/aggregated")
	if err != nil {
		fmt.Printf("获取统计信息失败: %v\n", err)
		return
	}

	fmt.Printf("日志文件总数: %d\n", stats["total_files"])
	fmt.Printf("总大小: %d 字节\n", stats["total_size"])
	fmt.Printf("最旧文件: %s\n", stats["oldest_file"])
	fmt.Printf("最新文件: %s\n", stats["newest_file"])

	if oldestTime, ok := stats["oldest_time"].(time.Time); ok {
		fmt.Printf("最旧时间: %s\n", oldestTime.Format("2006-01-02 15:04:05"))
	}
	if newestTime, ok := stats["newest_time"].(time.Time); ok {
		fmt.Printf("最新时间: %s\n", newestTime.Format("2006-01-02 15:04:05"))
	}
}

func demonstrateCleanupFeatures() {
	fmt.Println("\n演示清理功能:")

	// 清理一周前的日志文件
	fmt.Println("清理一周前的日志文件...")
	err := logz.CleanupOldLogsDefault("./logs/aggregated")
	if err != nil {
		fmt.Printf("清理失败: %v\n", err)
	} else {
		fmt.Println("清理完成")
	}

	// 清理后再次获取统计信息
	fmt.Println("\n清理后的统计信息:")
	stats, err := logz.GetLogStatsDefault("./logs/aggregated")
	if err != nil {
		fmt.Printf("获取统计信息失败: %v\n", err)
		return
	}

	fmt.Printf("日志文件总数: %d\n", stats["total_files"])
	fmt.Printf("总大小: %d 字节\n", stats["total_size"])
}

func demonstrateLargeScaleFeatures() {
	fmt.Println("\n=== 大规模日志处理演示 ===")

	// 初始化针对大规模日志优化的聚合器
	err := logz.InitWithAggregation(
		"./logs/large-scale.log", // 普通日志文件
		"./logs/large-scale-agg", // 聚合日志目录
		"high-volume-service",    // 服务名
		500*1024*1024,            // 轮转大小 (500MB，适合大规模)
		50,                       // 最大备份数
	)
	if err != nil {
		log.Printf("初始化大规模日志系统失败: %v", err)
		return
	}
	defer logz.CloseAggregator()

	// 演示大规模日志生成
	demonstrateLargeScaleLogging()

	// 演示高性能查询
	demonstrateHighPerformanceQuery()

	// 演示索引功能
	demonstrateIndexFeatures()

	// 演示并发处理
	demonstrateConcurrentProcessing()
}

func demonstrateLargeScaleLogging() {
	fmt.Println("\n=== 大规模日志生成演示 ===")

	// 生成大量测试日志
	const numLogs = 1000 // 减少数量以便演示
	const numWorkers = 5

	var wg sync.WaitGroup
	startTime := time.Now()

	// 使用多个goroutine并发生成日志
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			generateLogsForWorker(workerID, numLogs/numWorkers)
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	fmt.Printf("生成了 %d 条日志，耗时: %v\n", numLogs, duration)
	fmt.Printf("平均速度: %.2f 条/秒\n", float64(numLogs)/duration.Seconds())
}

func generateLogsForWorker(workerID, numLogs int) {
	traceIDs := []string{
		"trace-user-001", "trace-user-002", "trace-user-003",
		"trace-order-001", "trace-order-002", "trace-order-003",
		"trace-payment-001", "trace-payment-002", "trace-payment-003",
	}

	spanIDs := []string{
		"span-auth", "span-db", "span-cache", "span-api", "span-external",
	}

	levels := []string{"info", "debug", "warn", "error"}

	for i := 0; i < numLogs; i++ {
		traceID := traceIDs[rand.Intn(len(traceIDs))]
		spanID := spanIDs[rand.Intn(len(spanIDs))]
		level := levels[rand.Intn(len(levels))]

		// 使用带追踪上下文的日志方法
		switch level {
		case "info":
			logz.InfoWithTrace(traceID, spanID, "处理用户请求",
				"worker_id", workerID,
				"request_id", fmt.Sprintf("req-%d-%d", workerID, i),
				"user_id", rand.Intn(10000))
		case "debug":
			logz.DebugWithTrace(traceID, spanID, "数据库查询",
				"worker_id", workerID,
				"query_time", rand.Intn(100))
		case "warn":
			logz.Warn("缓存未命中",
				"worker_id", workerID,
				"cache_key", fmt.Sprintf("key-%d", rand.Intn(1000)))
		case "error":
			logz.ErrorWithTrace(traceID, spanID, "外部服务调用失败",
				"worker_id", workerID,
				"error_code", fmt.Sprintf("ERR-%d", rand.Intn(100)))
		}

		// 偶尔生成一些普通日志
		if i%100 == 0 {
			logz.Info("系统状态检查",
				"worker_id", workerID,
				"memory_usage", fmt.Sprintf("%d%%", 60+rand.Intn(30)),
				"cpu_usage", fmt.Sprintf("%d%%", 20+rand.Intn(60)))
		}
	}
}

func demonstrateHighPerformanceQuery() {
	fmt.Println("\n=== 高性能查询演示 ===")

	// 测试索引查询性能
	testQueries := []struct {
		name string
		fn   func() (*logz.LogQueryResult, error)
	}{
		{
			name: "TraceID索引查询",
			fn: func() (*logz.LogQueryResult, error) {
				return logz.QueryLogsByTraceID("trace-user-001", "./logs/large-scale-agg", 100, 0)
			},
		},
		{
			name: "SpanID索引查询",
			fn: func() (*logz.LogQueryResult, error) {
				return logz.QueryLogsBySpanID("span-db", "./logs/large-scale-agg", 100, 0)
			},
		},
		{
			name: "级别索引查询",
			fn: func() (*logz.LogQueryResult, error) {
				return logz.QueryLogsByLevel("error", "./logs/large-scale-agg", 100, 0)
			},
		},
		{
			name: "时间范围查询",
			fn: func() (*logz.LogQueryResult, error) {
				startTime := time.Now().Add(-1 * time.Hour)
				endTime := time.Now()
				return logz.QueryLogsByTimeRange(startTime, endTime, "./logs/large-scale-agg", 100, 0)
			},
		},
	}

	for _, test := range testQueries {
		startTime := time.Now()
		result, err := test.fn()
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("❌ %s 失败: %v\n", test.name, err)
		} else {
			fmt.Printf("✅ %s: 找到 %d 条日志，耗时: %v\n", test.name, result.Total, duration)
		}
	}
}

func demonstrateIndexFeatures() {
	fmt.Println("\n=== 索引功能演示 ===")

	// 演示索引vs非索引查询的性能差异
	traceID := "trace-user-001"

	// 使用索引查询
	startTime := time.Now()
	resultWithIndex, err := logz.QueryLogsWithIndex(logz.LogQuery{
		TraceID: traceID,
		Limit:   100,
		Offset:  0,
	}, "./logs/large-scale-agg")
	indexDuration := time.Since(startTime)

	// 不使用索引查询
	startTime = time.Now()
	resultWithoutIndex, err2 := logz.QueryLogsWithoutIndex(logz.LogQuery{
		TraceID: traceID,
		Limit:   100,
		Offset:  0,
	}, "./logs/large-scale-agg")
	scanDuration := time.Since(startTime)

	if err != nil || err2 != nil {
		fmt.Printf("查询失败: %v, %v\n", err, err2)
		return
	}

	fmt.Printf("索引查询: 找到 %d 条日志，耗时: %v\n", resultWithIndex.Total, indexDuration)
	fmt.Printf("文件扫描: 找到 %d 条日志，耗时: %v\n", resultWithoutIndex.Total, scanDuration)

	if scanDuration > 0 && indexDuration > 0 {
		improvement := float64(scanDuration) / float64(indexDuration)
		fmt.Printf("性能提升: %.2fx\n", improvement)
	}
}

func demonstrateConcurrentProcessing() {
	fmt.Println("\n=== 并发处理演示 ===")

	const numQueries = 20 // 减少数量以便演示
	var wg sync.WaitGroup
	results := make(chan string, numQueries)

	// 并发执行多个查询
	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func(queryID int) {
			defer wg.Done()

			startTime := time.Now()
			result, err := logz.QueryLogsByTraceID(
				fmt.Sprintf("trace-user-%03d", queryID%10),
				"./logs/large-scale-agg",
				10,
				0,
			)
			duration := time.Since(startTime)

			if err != nil {
				results <- fmt.Sprintf("查询 %d 失败: %v", queryID, err)
			} else {
				results <- fmt.Sprintf("查询 %d: %d 条结果，耗时: %v", queryID, result.Total, duration)
			}
		}(i)
	}

	// 等待所有查询完成
	wg.Wait()
	close(results)

	// 收集结果
	successCount := 0
	for result := range results {
		fmt.Println(result)
		successCount++
	}

	fmt.Printf("并发查询完成: %d 个查询\n", successCount)
}
