package main

import (
	"errors"
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
