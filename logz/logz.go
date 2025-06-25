package logz

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HsiaoL1/trace"
	"github.com/sirupsen/logrus"
)

// 全局配置Logrus
var Logrus *logrus.Logger

// 日志级别常量
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
	LevelPanic = "panic"
)

// 日志格式常量
const (
	FormatText = "text"
	FormatJSON = "json"
)

// 初始化函数
func init() {
	Logrus = logrus.New()
	// 默认配置
	SetLevel(LevelInfo)
	SetFormat(FormatText)
	SetOutput(os.Stdout)
}

// SetLevel 设置日志级别
func SetLevel(level string) {
	switch strings.ToLower(level) {
	case LevelDebug:
		Logrus.SetLevel(logrus.DebugLevel)
	case LevelInfo:
		Logrus.SetLevel(logrus.InfoLevel)
	case LevelWarn:
		Logrus.SetLevel(logrus.WarnLevel)
	case LevelError:
		Logrus.SetLevel(logrus.ErrorLevel)
	case LevelFatal:
		Logrus.SetLevel(logrus.FatalLevel)
	case LevelPanic:
		Logrus.SetLevel(logrus.PanicLevel)
	default:
		Logrus.SetLevel(logrus.InfoLevel)
	}
}

// SetFormat 设置日志格式
func SetFormat(format string) {
	switch strings.ToLower(format) {
	case FormatJSON:
		Logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := filepath.Base(f.File)
				return "", fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	case FormatText:
		fallthrough
	default:
		Logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := filepath.Base(f.File)
				return "", fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	}
}

// SetOutput 设置日志输出位置
func SetOutput(output io.Writer) {
	Logrus.SetOutput(output)
}

// SetFileOutput 设置日志文件输出
func SetFileOutput(filepath string) error {
	// 确保目录存在
	lastSlash := strings.LastIndex(filepath, "/")
	if lastSlash > 0 {
		dir := filepath[:lastSlash]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %v", err)
		}
	}

	// 打开或创建日志文件
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 设置输出到文件
	Logrus.SetOutput(file)
	return nil
}

// SetFileOutputWithRotation 设置日志文件输出（带轮转）
func SetFileOutputWithRotation(filepath string, maxSize int64, maxBackups int) error {
	// 这里可以集成 logrotate 或其他轮转库
	// 暂时使用简单的文件输出
	return SetFileOutput(filepath)
}

// EnableCaller 启用调用者信息
func EnableCaller() {
	Logrus.SetReportCaller(true)
}

// DisableCaller 禁用调用者信息
func DisableCaller() {
	Logrus.SetReportCaller(false)
}

// sendEmailNotification 发送邮件通知
func sendEmailNotification(level, message string) {
	// 获取接收方邮箱
	toEmail := trace.GetEmail()
	if toEmail == "" {
		// 如果没有配置邮箱，直接返回
		return
	}

	// 获取调用者信息
	var callerInfo string
	if Logrus.ReportCaller {
		if pc, file, line, ok := runtime.Caller(3); ok {
			funcName := runtime.FuncForPC(pc).Name()
			callerInfo = fmt.Sprintf("调用位置: %s:%d (%s)", filepath.Base(file), line, funcName)
		}
	}

	// 构建邮件主题和内容
	subject := fmt.Sprintf("[%s] 系统日志告警 - %s", strings.ToUpper(level), time.Now().Format("2006-01-02 15:04:05"))

	body := fmt.Sprintf(`
		<h2>系统日志告警</h2>
		<p><strong>级别:</strong> %s</p>
		<p><strong>时间:</strong> %s</p>
		<p><strong>消息:</strong> %s</p>
		<p><strong>%s</strong></p>
		<hr>
		<p><em>此邮件由系统自动发送，请及时处理。</em></p>
	`, strings.ToUpper(level), time.Now().Format("2006-01-02 15:04:05"), message, callerInfo)

	// 异步发送邮件，避免阻塞日志记录
	go func() {
		if err := trace.SendEmail(toEmail, subject, body); err != nil {
			// 邮件发送失败时，记录到日志中（避免循环调用）
			Logrus.Errorf("发送邮件通知失败: %v", err)
		}
	}()
}

// sendEmailNotificationWithFormat 发送邮件通知（带格式化）
func sendEmailNotificationWithFormat(level, format string, args ...any) {
	// 获取接收方邮箱
	toEmail := trace.GetEmail()
	if toEmail == "" {
		// 如果没有配置邮箱，直接返回
		return
	}

	// 构建邮件内容
	fullMessage := fmt.Sprintf(format, args...)

	// 获取调用者信息
	var callerInfo string
	if Logrus.ReportCaller {
		if pc, file, line, ok := runtime.Caller(3); ok {
			funcName := runtime.FuncForPC(pc).Name()
			callerInfo = fmt.Sprintf("调用位置: %s:%d (%s)", filepath.Base(file), line, funcName)
		}
	}

	// 构建邮件主题和内容
	subject := fmt.Sprintf("[%s] 系统日志告警 - %s", strings.ToUpper(level), time.Now().Format("2006-01-02 15:04:05"))

	body := fmt.Sprintf(`
		<h2>系统日志告警</h2>
		<p><strong>级别:</strong> %s</p>
		<p><strong>时间:</strong> %s</p>
		<p><strong>消息:</strong> %s</p>
		<p><strong>%s</strong></p>
		<hr>
		<p><em>此邮件由系统自动发送，请及时处理。</em></p>
	`, strings.ToUpper(level), time.Now().Format("2006-01-02 15:04:05"), fullMessage, callerInfo)

	// 异步发送邮件，避免阻塞日志记录
	go func() {
		if err := trace.SendEmail(toEmail, subject, body); err != nil {
			// 邮件发送失败时，记录到日志中（避免循环调用）
			Logrus.Errorf("发送邮件通知失败: %v", err)
		}
	}()
}

// 日志方法

// Debug 调试日志
func Debug(args ...any) {
	Logrus.Debug(args...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...any) {
	Logrus.Debugf(format, args...)
}

// Info 信息日志
func Info(args ...any) {
	Logrus.Info(args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...any) {
	Logrus.Infof(format, args...)
}

// Warn 警告日志
func Warn(args ...any) {
	Logrus.Warn(args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...any) {
	Logrus.Warnf(format, args...)
}

// Error 错误日志
func Error(args ...any) {
	Logrus.Error(args...)
}

// ErrorWithEmail 错误日志（带邮件通知）
func ErrorWithEmail(sendEmail bool, args ...any) {
	Logrus.Error(args...)
	if sendEmail {
		message := fmt.Sprint(args...)
		sendEmailNotification("error", message)
	}
}

// Errorf 格式化错误日志
func Errorf(format string, args ...any) {
	Logrus.Errorf(format, args...)
}

// ErrorfWithEmail 格式化错误日志（带邮件通知）
func ErrorfWithEmail(sendEmail bool, format string, args ...any) {
	Logrus.Errorf(format, args...)
	if sendEmail {
		sendEmailNotificationWithFormat("error", format, args...)
	}
}

// Fatal 致命错误日志（会调用os.Exit(1)）
func Fatal(args ...any) {
	Logrus.Fatal(args...)
}

// FatalWithEmail 致命错误日志（带邮件通知，会调用os.Exit(1)）
func FatalWithEmail(sendEmail bool, args ...any) {
	Logrus.Fatal(args...)
	if sendEmail {
		message := fmt.Sprint(args...)
		sendEmailNotification("fatal", message)
	}
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...any) {
	Logrus.Fatalf(format, args...)
}

// FatalfWithEmail 格式化致命错误日志（带邮件通知）
func FatalfWithEmail(sendEmail bool, format string, args ...any) {
	Logrus.Fatalf(format, args...)
	if sendEmail {
		sendEmailNotificationWithFormat("fatal", format, args...)
	}
}

// Panic 恐慌日志（会调用panic）
func Panic(args ...any) {
	Logrus.Panic(args...)
}

// PanicWithEmail 恐慌日志（带邮件通知，会调用panic）
func PanicWithEmail(sendEmail bool, args ...any) {
	Logrus.Panic(args...)
	if sendEmail {
		message := fmt.Sprint(args...)
		sendEmailNotification("panic", message)
	}
}

// Panicf 格式化恐慌日志
func Panicf(format string, args ...any) {
	Logrus.Panicf(format, args...)
}

// PanicfWithEmail 格式化恐慌日志（带邮件通知）
func PanicfWithEmail(sendEmail bool, format string, args ...any) {
	Logrus.Panicf(format, args...)
	if sendEmail {
		sendEmailNotificationWithFormat("panic", format, args...)
	}
}

// WithField 添加字段
func WithField(key string, value any) *logrus.Entry {
	return Logrus.WithField(key, value)
}

// WithFields 添加多个字段
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Logrus.WithFields(fields)
}

// WithError 添加错误字段
func WithError(err error) *logrus.Entry {
	return Logrus.WithError(err)
}

// 带追踪上下文的日志方法

// DebugWithTrace 带追踪上下文的调试日志
func DebugWithTrace(traceID, spanID string, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Debug(args...)
}

// DebugfWithTrace 带追踪上下文的格式化调试日志
func DebugfWithTrace(traceID, spanID, format string, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Debugf(format, args...)
}

// InfoWithTrace 带追踪上下文的信息日志
func InfoWithTrace(traceID, spanID string, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Info(args...)
}

// InfofWithTrace 带追踪上下文的格式化信息日志
func InfofWithTrace(traceID, spanID, format string, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Infof(format, args...)
}

// ErrorWithTrace 带追踪上下文的错误日志
func ErrorWithTrace(traceID, spanID string, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Error(args...)
}

// ErrorWithTraceAndEmail 带追踪上下文的错误日志（带邮件通知）
func ErrorWithTraceAndEmail(traceID, spanID string, sendEmail bool, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Error(args...)
	if sendEmail {
		message := fmt.Sprint(args...)
		sendEmailNotification("error", message)
	}
}

// ErrorfWithTrace 带追踪上下文的格式化错误日志
func ErrorfWithTrace(traceID, spanID, format string, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Errorf(format, args...)
}

// ErrorfWithTraceAndEmail 带追踪上下文的格式化错误日志（带邮件通知）
func ErrorfWithTraceAndEmail(traceID, spanID string, sendEmail bool, format string, args ...any) {
	Logrus.WithFields(logrus.Fields{
		"trace_id": traceID,
		"span_id":  spanID,
	}).Errorf(format, args...)
	if sendEmail {
		sendEmailNotificationWithFormat("error", format, args...)
	}
}

// 便捷的初始化方法

// InitDefault 初始化默认配置
func InitDefault() {
	SetLevel(LevelInfo)
	SetFormat(FormatText)
	SetOutput(os.Stdout)
	EnableCaller()
}

// InitProduction 初始化生产环境配置
func InitProduction(logFile string) error {
	SetLevel(LevelInfo)
	SetFormat(FormatJSON)
	EnableCaller()

	if logFile != "" {
		return SetFileOutput(logFile)
	}
	return nil
}

// InitDevelopment 初始化开发环境配置
func InitDevelopment() {
	SetLevel(LevelDebug)
	SetFormat(FormatText)
	SetOutput(os.Stdout)
	EnableCaller()
}

// 日志聚合相关方法

// InitWithAggregation 初始化带聚合功能的日志系统
func InitWithAggregation(logFile, aggregateDir, serviceName string, rotationSize int64, maxBackups int) error {
	// 初始化基本配置
	SetLevel(LevelInfo)
	SetFormat(FormatJSON)
	EnableCaller()

	// 设置文件输出
	if logFile != "" {
		if err := SetFileOutput(logFile); err != nil {
			return err
		}
	}

	// 创建聚合器
	aggregator, err := NewLogAggregator(aggregateDir, serviceName, rotationSize, maxBackups)
	if err != nil {
		return err
	}

	// 设置全局聚合器
	SetGlobalAggregator(aggregator)

	// 添加聚合Hook
	hook := NewAggregatorHook(aggregator, serviceName)
	Logrus.AddHook(hook)

	return nil
}

// QueryLogsByTraceID 根据TraceID查询日志
func QueryLogsByTraceID(traceID, logDir string, limit, offset int) (*LogQueryResult, error) {
	query := LogQuery{
		TraceID:  traceID,
		Limit:    limit,
		Offset:   offset,
		UseIndex: true, // 启用索引
	}
	return QueryLogs(query, logDir)
}

// QueryLogsBySpanID 根据SpanID查询日志
func QueryLogsBySpanID(spanID, logDir string, limit, offset int) (*LogQueryResult, error) {
	query := LogQuery{
		SpanID:   spanID,
		Limit:    limit,
		Offset:   offset,
		UseIndex: true, // 启用索引
	}
	return QueryLogs(query, logDir)
}

// QueryLogsByTimeRange 根据时间范围查询日志
func QueryLogsByTimeRange(startTime, endTime time.Time, logDir string, limit, offset int) (*LogQueryResult, error) {
	query := LogQuery{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
		Offset:    offset,
		UseIndex:  false, // 时间范围查询不使用索引
	}
	return QueryLogs(query, logDir)
}

// QueryLogsByLevel 根据日志级别查询
func QueryLogsByLevel(level, logDir string, limit, offset int) (*LogQueryResult, error) {
	query := LogQuery{
		Level:    level,
		Limit:    limit,
		Offset:   offset,
		UseIndex: true, // 启用索引
	}
	return QueryLogs(query, logDir)
}

// QueryLogsByService 根据服务名查询日志
func QueryLogsByService(service, logDir string, limit, offset int) (*LogQueryResult, error) {
	query := LogQuery{
		Service:  service,
		Limit:    limit,
		Offset:   offset,
		UseIndex: true, // 启用索引
	}
	return QueryLogs(query, logDir)
}

// QueryLogsByMessage 根据消息内容查询日志（支持正则表达式）
func QueryLogsByMessage(message, logDir string, limit, offset int) (*LogQueryResult, error) {
	query := LogQuery{
		Message:  message,
		Limit:    limit,
		Offset:   offset,
		UseIndex: false, // 消息内容查询不使用索引
	}
	return QueryLogs(query, logDir)
}

// QueryLogsWithIndex 使用索引的复杂查询
func QueryLogsWithIndex(query LogQuery, logDir string) (*LogQueryResult, error) {
	query.UseIndex = true
	return QueryLogs(query, logDir)
}

// QueryLogsWithoutIndex 不使用索引的查询（强制文件扫描）
func QueryLogsWithoutIndex(query LogQuery, logDir string) (*LogQueryResult, error) {
	query.UseIndex = false
	return QueryLogs(query, logDir)
}

// CleanupOldLogsDefault 清理一周前的日志文件
func CleanupOldLogsDefault(logDir string) error {
	return CleanupOldLogs(logDir, 7)
}

// GetLogStatsDefault 获取日志统计信息
func GetLogStatsDefault(logDir string) (map[string]interface{}, error) {
	return GetLogStats(logDir)
}

// CloseAggregator 关闭全局聚合器
func CloseAggregator() error {
	aggregator := GetGlobalAggregator()
	if aggregator != nil {
		return aggregator.Close()
	}
	return nil
}
