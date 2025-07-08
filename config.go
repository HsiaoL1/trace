package trace

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 追踪配置
type Config struct {
	// Jaeger配置
	Jaeger JaegerConfig `json:"jaeger" yaml:"jaeger"`

	// 日志配置
	LogLevel string `json:"log_level" yaml:"log_level"`
	LogFile  string `json:"log_file" yaml:"log_file"`

	// 采样配置
	SamplingRatio float64 `json:"sampling_ratio" yaml:"sampling_ratio"`

	// 邮件配置
	SMTP SMTPConfig `json:"smtp" yaml:"smtp"`

	// 其他配置
	Debug   bool `json:"debug" yaml:"debug"`
	Metrics bool `json:"metrics" yaml:"metrics"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Jaeger:        *DefaultJaegerConfig(),
		LogLevel:      "info",
		LogFile:       "",
		SamplingRatio: 1.0, // 100%采样
		SMTP:          DefaultSMTPConfig(),
		Debug:         false,
		Metrics:       true,
	}
}

// LoadConfigFromEnv 从环境变量加载配置
func LoadConfigFromEnv() *Config {
	config := DefaultConfig()

	// 加载Jaeger配置
	config.Jaeger = *LoadJaegerConfigFromEnv()

	// 加载日志配置
	if logLevel := os.Getenv("TRACE_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = strings.ToLower(logLevel)
	}

	if logFile := os.Getenv("TRACE_LOG_FILE"); logFile != "" {
		config.LogFile = logFile
	}

	// 加载采样配置
	if samplingRatio := os.Getenv("TRACE_SAMPLING_RATIO"); samplingRatio != "" {
		if ratio, err := strconv.ParseFloat(samplingRatio, 64); err == nil {
			if ratio >= 0.0 && ratio <= 1.0 {
				config.SamplingRatio = ratio
			}
		}
	}

	// 加载SMTP配置
	config.SMTP = LoadSMTPConfigFromEnv()

	// 加载其他配置
	if debug := os.Getenv("TRACE_DEBUG"); debug != "" {
		if parsed, err := strconv.ParseBool(debug); err == nil {
			config.Debug = parsed
		}
	}

	if metrics := os.Getenv("TRACE_METRICS"); metrics != "" {
		if parsed, err := strconv.ParseBool(metrics); err == nil {
			config.Metrics = parsed
		}
	}

	return config
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 验证采样比例
	if c.SamplingRatio < 0.0 || c.SamplingRatio > 1.0 {
		return fmt.Errorf("sampling ratio must be between 0.0 and 1.0, got %f", c.SamplingRatio)
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug":   true,
		"info":    true,
		"warn":    true,
		"warning": true,
		"error":   true,
		"fatal":   true,
		"panic":   true,
	}

	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return fmt.Errorf("invalid log level: %s, valid levels are: debug, info, warn, warning, error, fatal, panic", c.LogLevel)
	}

	// 验证Jaeger配置
	if err := validateJaegerConfig(&c.Jaeger); err != nil {
		return fmt.Errorf("invalid Jaeger config: %w", err)
	}

	return nil
}

// Fix 修复配置中的错误值
func (c *Config) Fix() {
	if c == nil {
		return
	}

	// 修复采样比例
	if c.SamplingRatio < 0.0 || c.SamplingRatio > 1.0 {
		c.SamplingRatio = 1.0
	}

	// 修复日志级别
	validLogLevels := map[string]bool{
		"debug":   true,
		"info":    true,
		"warn":    true,
		"warning": true,
		"error":   true,
		"fatal":   true,
		"panic":   true,
	}

	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		c.LogLevel = "info"
	}

	// 修复Jaeger配置
	if c.Jaeger.ServiceName == "" {
		c.Jaeger.ServiceName = "trace-service"
	}
	if c.Jaeger.Endpoint == "" {
		c.Jaeger.Endpoint = "http://localhost:4318/v1/traces"
	}
	if c.Jaeger.Environment == "" {
		c.Jaeger.Environment = "development"
	}
	if c.Jaeger.Version == "" {
		c.Jaeger.Version = "1.0.0"
	}
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	if c == nil {
		return "<nil config>"
	}
	return fmt.Sprintf("Config{Jaeger: %+v, LogLevel: %s, LogFile: %s, SamplingRatio: %f, Debug: %t, Metrics: %t}",
		c.Jaeger, c.LogLevel, c.LogFile, c.SamplingRatio, c.Debug, c.Metrics)
}
