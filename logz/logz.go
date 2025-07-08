package logz

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/HsiaoL1/trace"
	"github.com/sirupsen/logrus"
)

// Logger 日志器接口
type Logger interface {
	Debug(args ...any)
	Debugf(format string, args ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	Warn(args ...any)
	Warnf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Panic(args ...any)
	Panicf(format string, args ...any)
	WithField(key string, value any) *logrus.Entry
	WithFields(fields logrus.Fields) *logrus.Entry
	WithError(err error) *logrus.Entry
}

// DefaultLogger 默认日志器实现
type DefaultLogger struct {
	logrus *logrus.Logger
	mutex  sync.RWMutex
	config *LoggerConfig
}

// LoggerConfig 日志器配置
type LoggerConfig struct {
	Level          string
	Format         string
	Output         io.Writer
	FilePath       string
	EnableCaller   bool
	EmailConfig    *EmailConfig
	RotationConfig *RotationConfig
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Enabled   bool
	ToEmail   string
	OnLevels  []string // 哪些级别发送邮件
	Throttle  time.Duration // 邮件限流
	lastSent  time.Time
	mutex     sync.Mutex
}

// RotationConfig 轮转配置
type RotationConfig struct {
	MaxSize    int64
	MaxBackups int
	Enabled    bool
}

// 全局默认日志器
var defaultLogger *DefaultLogger
var loggerMutex sync.RWMutex

// 兼容性全局变量
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
	// 初始化默认日志器
	defaultConfig := &LoggerConfig{
		Level:        LevelInfo,
		Format:       FormatText,
		Output:       os.Stdout,
		EnableCaller: false,
	}
	defaultLogger = NewDefaultLogger(defaultConfig)
	
	// 兼容性设置
	Logrus = defaultLogger.logrus
}

// NewDefaultLogger 创建默认日志器
func NewDefaultLogger(config *LoggerConfig) *DefaultLogger {
	if config == nil {
		config = &LoggerConfig{
			Level:        LevelInfo,
			Format:       FormatText,
			Output:       os.Stdout,
			EnableCaller: false,
		}
	}
	
	logger := &DefaultLogger{
		logrus: logrus.New(),
		config: config,
	}
	
	logger.applyConfig()
	return logger
}

// applyConfig 应用配置
func (l *DefaultLogger) applyConfig() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.setLevel(l.config.Level)
	l.setFormat(l.config.Format)
	l.logrus.SetOutput(l.config.Output)
	l.logrus.SetReportCaller(l.config.EnableCaller)
	
	if l.config.FilePath != "" {
		l.setFileOutput(l.config.FilePath)
	}
}

// GetDefaultLogger 获取默认日志器
func GetDefaultLogger() *DefaultLogger {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	return defaultLogger
}

// SetDefaultLogger 设置默认日志器
func SetDefaultLogger(logger *DefaultLogger) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	defaultLogger = logger
	Logrus = logger.logrus // 更新兼容性变量
}

// setLevel 设置日志级别（内部方法）
func (l *DefaultLogger) setLevel(level string) {
	switch strings.ToLower(level) {
	case LevelDebug:
		l.logrus.SetLevel(logrus.DebugLevel)
	case LevelInfo:
		l.logrus.SetLevel(logrus.InfoLevel)
	case LevelWarn:
		l.logrus.SetLevel(logrus.WarnLevel)
	case LevelError:
		l.logrus.SetLevel(logrus.ErrorLevel)
	case LevelFatal:
		l.logrus.SetLevel(logrus.FatalLevel)
	case LevelPanic:
		l.logrus.SetLevel(logrus.PanicLevel)
	default:
		l.logrus.SetLevel(logrus.InfoLevel)
	}
	l.config.Level = level
}

// SetLevel 设置日志级别（全局函数，兼容性）
func SetLevel(level string) {
	defaultLogger.setLevel(level)
}

// setFormat 设置日志格式（内部方法）
func (l *DefaultLogger) setFormat(format string) {
	callerPrettyfier := func(f *runtime.Frame) (string, string) {
		filename := filepath.Base(f.File)
		return "", fmt.Sprintf("%s:%d", filename, f.Line)
	}
	
	switch strings.ToLower(format) {
	case FormatJSON:
		l.logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:  time.RFC3339,
			CallerPrettyfier: callerPrettyfier,
		})
	case FormatText:
		fallthrough
	default:
		l.logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:    true,
			TimestampFormat:  time.RFC3339,
			CallerPrettyfier: callerPrettyfier,
		})
	}
	l.config.Format = format
}

// SetFormat 设置日志格式（全局函数，兼容性）
func SetFormat(format string) {
	defaultLogger.setFormat(format)
}

// SetOutput 设置日志输出位置（全局函数，兼容性）
func SetOutput(output io.Writer) {
	defaultLogger.mutex.Lock()
	defer defaultLogger.mutex.Unlock()
	defaultLogger.logrus.SetOutput(output)
	defaultLogger.config.Output = output
}

// setFileOutput 设置日志文件输出（内部方法）
func (l *DefaultLogger) setFileOutput(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开或创建日志文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 设置输出到文件
	l.logrus.SetOutput(file)
	l.config.FilePath = filePath
	return nil
}

// SetFileOutput 设置日志文件输出（全局函数，兼容性）
func SetFileOutput(filePath string) error {
	return defaultLogger.setFileOutput(filePath)
}

// SetFileOutputWithRotation 设置日志文件输出（带轮转）
func SetFileOutputWithRotation(filePath string, maxSize int64, maxBackups int) error {
	if maxSize <= 0 {
		maxSize = 100 * 1024 * 1024 // 100MB
	}
	if maxBackups <= 0 {
		maxBackups = 3
	}
	
	defaultLogger.config.RotationConfig = &RotationConfig{
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		Enabled:    true,
	}
	
	// TODO: 这里可以集成 logrotate 或其他轮转库
	return SetFileOutput(filePath)
}

// EnableCaller 启用调用者信息（全局函数，兼容性）
func EnableCaller() {
	defaultLogger.mutex.Lock()
	defer defaultLogger.mutex.Unlock()
	defaultLogger.logrus.SetReportCaller(true)
	defaultLogger.config.EnableCaller = true
}

// DisableCaller 禁用调用者信息（全局函数，兼容性）
func DisableCaller() {
	defaultLogger.mutex.Lock()
	defer defaultLogger.mutex.Unlock()
	defaultLogger.logrus.SetReportCaller(false)
	defaultLogger.config.EnableCaller = false
}

// EmailNotifier 邮件通知器
type EmailNotifier struct {
	config    *EmailConfig
	throttle  map[string]time.Time
	mutex     sync.Mutex
}

// NewEmailNotifier 创建邮件通知器
func NewEmailNotifier(config *EmailConfig) *EmailNotifier {
	if config == nil {
		config = &EmailConfig{
			Enabled:  false,
			Throttle: 5 * time.Minute,
		}
	}
	
	return &EmailNotifier{
		config:   config,
		throttle: make(map[string]time.Time),
	}
}

// shouldSendEmail 检查是否应该发送邮件
func (n *EmailNotifier) shouldSendEmail(level string) bool {
	if !n.config.Enabled || n.config.ToEmail == "" {
		return false
	}
	
	// 检查级别是否在允许列表中
	if len(n.config.OnLevels) > 0 {
		found := false
		for _, allowedLevel := range n.config.OnLevels {
			if strings.EqualFold(allowedLevel, level) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// 检查限流
	n.mutex.Lock()
	defer n.mutex.Unlock()
	
	lastSent, exists := n.throttle[level]
	if exists && time.Since(lastSent) < n.config.Throttle {
		return false
	}
	
	n.throttle[level] = time.Now()
	return true
}

// sendEmailNotification 发送邮件通知
func (n *EmailNotifier) sendEmailNotification(_ context.Context, level, message string) {
	if !n.shouldSendEmail(level) {
		return
	}

	// 获取调用者信息
	var callerInfo string
	if pc, file, line, ok := runtime.Caller(4); ok {
		funcName := runtime.FuncForPC(pc).Name()
		callerInfo = fmt.Sprintf("调用位置: %s:%d (%s)", filepath.Base(file), line, funcName)
	}

	// 构建邮件内容
	now := time.Now()
	subject := fmt.Sprintf("[%s] 系统日志告警 - %s", strings.ToUpper(level), now.Format("2006-01-02 15:04:05"))

	body := fmt.Sprintf(`
		<h2>系统日志告警</h2>
		<p><strong>级别:</strong> %s</p>
		<p><strong>时间:</strong> %s</p>
		<p><strong>消息:</strong> %s</p>
		<p><strong>%s</strong></p>
		<hr>
		<p><em>此邮件由系统自动发送，请及时处理。</em></p>
	`, strings.ToUpper(level), now.Format("2006-01-02 15:04:05"), message, callerInfo)

	// 异步发送邮件，避免阻塞日志记录
	go func() {
		// 使用带超时的context
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		select {
		case <-ctx.Done():
			return
		default:
			if err := trace.SendEmail(n.config.ToEmail, subject, body); err != nil {
				// 避免循环调用，使用简单的输出
				fmt.Fprintf(os.Stderr, "[邮件通知失败] %v\n", err)
			}
		}
	}()
}

// 全局邮件通知器
var globalEmailNotifier *EmailNotifier
var emailMutex sync.RWMutex

// SetEmailConfig 设置邮件配置
func SetEmailConfig(config *EmailConfig) {
	emailMutex.Lock()
	defer emailMutex.Unlock()
	
	if config == nil {
		// 从环境变量加载配置
		config = &EmailConfig{
			Enabled:   os.Getenv("TRACE_EMAIL_ENABLED") == "true",
			ToEmail:   os.Getenv("TRACE_EMAIL_TO"),
			OnLevels:  []string{"error", "fatal", "panic"},
			Throttle:  5 * time.Minute,
		}
	}
	
	globalEmailNotifier = NewEmailNotifier(config)
	defaultLogger.config.EmailConfig = config
}

// getEmailNotifier 获取邮件通知器
func getEmailNotifier() *EmailNotifier {
	emailMutex.RLock()
	defer emailMutex.RUnlock()
	
	if globalEmailNotifier == nil {
		SetEmailConfig(nil) // 懒加载
	}
	return globalEmailNotifier
}

// sendEmailNotification 发送邮件通知（兼容性函数）
func sendEmailNotification(level, message string) {
	notifier := getEmailNotifier()
	if notifier != nil {
		notifier.sendEmailNotification(context.Background(), level, message)
	}
}

// sendEmailNotificationWithFormat 发送邮件通知（带格式化，兼容性函数）
func sendEmailNotificationWithFormat(level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	sendEmailNotification(level, message)
}

// 实现Logger接口
func (l *DefaultLogger) Debug(args ...any) {
	l.logrus.Debug(args...)
}

func (l *DefaultLogger) Debugf(format string, args ...any) {
	l.logrus.Debugf(format, args...)
}

func (l *DefaultLogger) Info(args ...any) {
	l.logrus.Info(args...)
}

func (l *DefaultLogger) Infof(format string, args ...any) {
	l.logrus.Infof(format, args...)
}

func (l *DefaultLogger) Warn(args ...any) {
	l.logrus.Warn(args...)
}

func (l *DefaultLogger) Warnf(format string, args ...any) {
	l.logrus.Warnf(format, args...)
}

func (l *DefaultLogger) Error(args ...any) {
	l.logrus.Error(args...)
}

func (l *DefaultLogger) Errorf(format string, args ...any) {
	l.logrus.Errorf(format, args...)
}

func (l *DefaultLogger) Fatal(args ...any) {
	l.logrus.Fatal(args...)
}

func (l *DefaultLogger) Fatalf(format string, args ...any) {
	l.logrus.Fatalf(format, args...)
}

func (l *DefaultLogger) Panic(args ...any) {
	l.logrus.Panic(args...)
}

func (l *DefaultLogger) Panicf(format string, args ...any) {
	l.logrus.Panicf(format, args...)
}

func (l *DefaultLogger) WithField(key string, value any) *logrus.Entry {
	return l.logrus.WithField(key, value)
}

func (l *DefaultLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.logrus.WithFields(fields)
}

func (l *DefaultLogger) WithError(err error) *logrus.Entry {
	return l.logrus.WithError(err)
}

// 全局日志方法（兼容性）

// Debug 调试日志
func Debug(args ...any) {
	defaultLogger.Debug(args...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...any) {
	defaultLogger.Debugf(format, args...)
}

// Info 信息日志
func Info(args ...any) {
	defaultLogger.Info(args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...any) {
	defaultLogger.Infof(format, args...)
}

// Warn 警告日志
func Warn(args ...any) {
	defaultLogger.Warn(args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...any) {
	defaultLogger.Warnf(format, args...)
}

// Error 错误日志
func Error(args ...any) {
	defaultLogger.Error(args...)
}

// ErrorWithEmail 错误日志（带邮件通知）
func ErrorWithEmail(sendEmail bool, args ...any) {
	defaultLogger.Error(args...)
	if sendEmail {
		message := fmt.Sprint(args...)
		sendEmailNotification("error", message)
	}
}

// Errorf 格式化错误日志
func Errorf(format string, args ...any) {
	defaultLogger.Errorf(format, args...)
}

// ErrorfWithEmail 格式化错误日志（带邮件通知）
func ErrorfWithEmail(sendEmail bool, format string, args ...any) {
	defaultLogger.Errorf(format, args...)
	if sendEmail {
		sendEmailNotificationWithFormat("error", format, args...)
	}
}

// Fatal 致命错误日志（会调用os.Exit(1)）
func Fatal(args ...any) {
	defaultLogger.Fatal(args...)
}

// FatalWithEmail 致命错误日志（带邮件通知，会调用os.Exit(1)）
func FatalWithEmail(sendEmail bool, args ...any) {
	if sendEmail {
		message := fmt.Sprint(args...)
		// 先发送邮件，再调用Fatal
		notifier := getEmailNotifier()
		if notifier != nil {
			// 同步发送，因为Fatal会立即退出
			if notifier.shouldSendEmail("fatal") {
				trace.SendEmail(notifier.config.ToEmail, 
					fmt.Sprintf("[FATAL] 系统致命错误 - %s", time.Now().Format("2006-01-02 15:04:05")),
					fmt.Sprintf("<h2>系统致命错误</h2><p>%s</p>", message))
			}
		}
	}
	Logrus.Fatal(args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...any) {
	defaultLogger.Fatalf(format, args...)
}

// FatalfWithEmail 格式化致命错误日志（带邮件通知）
func FatalfWithEmail(sendEmail bool, format string, args ...any) {
	if sendEmail {
		message := fmt.Sprintf(format, args...)
		// 先发送邮件，再调用Fatalf
		notifier := getEmailNotifier()
		if notifier != nil && notifier.shouldSendEmail("fatal") {
			trace.SendEmail(notifier.config.ToEmail, 
				fmt.Sprintf("[FATAL] 系统致命错误 - %s", time.Now().Format("2006-01-02 15:04:05")),
				fmt.Sprintf("<h2>系统致命错误</h2><p>%s</p>", message))
		}
	}
	Logrus.Fatalf(format, args...)
}

// Panic 恐慌日志（会调用panic）
func Panic(args ...any) {
	defaultLogger.Panic(args...)
}

// PanicWithEmail 恐慌日志（带邮件通知，会调用panic）
func PanicWithEmail(sendEmail bool, args ...any) {
	if sendEmail {
		message := fmt.Sprint(args...)
		// 先发送邮件，再panic
		notifier := getEmailNotifier()
		if notifier != nil && notifier.shouldSendEmail("panic") {
			trace.SendEmail(notifier.config.ToEmail, 
				fmt.Sprintf("[PANIC] 系统恐慌 - %s", time.Now().Format("2006-01-02 15:04:05")),
				fmt.Sprintf("<h2>系统恐慌</h2><p>%s</p>", message))
		}
	}
	Logrus.Panic(args...)
}

// Panicf 格式化恐慌日志
func Panicf(format string, args ...any) {
	defaultLogger.Panicf(format, args...)
}

// PanicfWithEmail 格式化恐慌日志（带邮件通知）
func PanicfWithEmail(sendEmail bool, format string, args ...any) {
	if sendEmail {
		message := fmt.Sprintf(format, args...)
		// 先发送邮件，再panic
		notifier := getEmailNotifier()
		if notifier != nil && notifier.shouldSendEmail("panic") {
			trace.SendEmail(notifier.config.ToEmail, 
				fmt.Sprintf("[PANIC] 系统恐慌 - %s", time.Now().Format("2006-01-02 15:04:05")),
				fmt.Sprintf("<h2>系统恐慌</h2><p>%s</p>", message))
		}
	}
	Logrus.Panicf(format, args...)
}

// WithField 添加字段
func WithField(key string, value any) *logrus.Entry {
	return defaultLogger.WithField(key, value)
}

// WithFields 添加多个字段
func WithFields(fields logrus.Fields) *logrus.Entry {
	return defaultLogger.WithFields(fields)
}

// WithError 添加错误字段
func WithError(err error) *logrus.Entry {
	return defaultLogger.WithError(err)
}

// 带追踪上下文的日志方法

// createTraceFields 创建追踪字段
func createTraceFields(traceID, spanID string) logrus.Fields {
	fields := logrus.Fields{}
	if traceID != "" {
		fields["trace_id"] = traceID
	}
	if spanID != "" {
		fields["span_id"] = spanID
	}
	return fields
}

// DebugWithTrace 带追踪上下文的调试日志
func DebugWithTrace(traceID, spanID string, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Debug(args...)
}

// DebugfWithTrace 带追踪上下文的格式化调试日志
func DebugfWithTrace(traceID, spanID, format string, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Debugf(format, args...)
}

// InfoWithTrace 带追踪上下文的信息日志
func InfoWithTrace(traceID, spanID string, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Info(args...)
}

// InfofWithTrace 带追踪上下文的格式化信息日志
func InfofWithTrace(traceID, spanID, format string, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Infof(format, args...)
}

// ErrorWithTrace 带追踪上下文的错误日志
func ErrorWithTrace(traceID, spanID string, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Error(args...)
}

// ErrorWithTraceAndEmail 带追踪上下文的错误日志（带邮件通知）
func ErrorWithTraceAndEmail(traceID, spanID string, sendEmail bool, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Error(args...)
	if sendEmail {
		message := fmt.Sprint(args...)
		sendEmailNotification("error", message)
	}
}

// ErrorfWithTrace 带追踪上下文的格式化错误日志
func ErrorfWithTrace(traceID, spanID, format string, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Errorf(format, args...)
}

// ErrorfWithTraceAndEmail 带追踪上下文的格式化错误日志（带邮件通知）
func ErrorfWithTraceAndEmail(traceID, spanID string, sendEmail bool, format string, args ...any) {
	defaultLogger.WithFields(createTraceFields(traceID, spanID)).Errorf(format, args...)
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
func GetLogStatsDefault(logDir string) (map[string]any, error) {
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

// Close 关闭默认日志器
func Close() error {
	// 关闭聚合器
	if err := CloseAggregator(); err != nil {
		return err
	}
	
	// 如果输出是文件，关闭文件句柄
	if closer, ok := defaultLogger.config.Output.(io.Closer); ok {
		return closer.Close()
	}
	
	return nil
}
