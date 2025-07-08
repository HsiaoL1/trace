package trace

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

// SMTPConfig SMTP配置结构体
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	TLSEnabled bool
	InsecureSkipVerify bool
}

// EmailSender 邮件发送器接口
type EmailSender interface {
	SendEmail(to, subject, body string) error
	SetSMTPConfig(config SMTPConfig)
	GetSMTPConfig() SMTPConfig
}

// DefaultEmailSender 默认邮件发送器实现
type DefaultEmailSender struct {
	config SMTPConfig
}

// NewEmailSender 创建新的邮件发送器
func NewEmailSender() EmailSender {
	return &DefaultEmailSender{
		config: DefaultSMTPConfig(),
	}
}

// DefaultSMTPConfig 默认SMTP配置
func DefaultSMTPConfig() SMTPConfig {
	return SMTPConfig{
		Host:     "smtp.qq.com",
		Port:     587,
		User:     "",
		Password: "",
		TLSEnabled: true,
		InsecureSkipVerify: false,
	}
}

// SetSMTPConfig 设置SMTP配置
func (e *DefaultEmailSender) SetSMTPConfig(config SMTPConfig) {
	e.config = config
}

// GetSMTPConfig 获取SMTP配置
func (e *DefaultEmailSender) GetSMTPConfig() SMTPConfig {
	return e.config
}

// LoadSMTPConfigFromEnv 从环境变量加载SMTP配置
func LoadSMTPConfigFromEnv() SMTPConfig {
	config := DefaultSMTPConfig()
	
	if host := os.Getenv("SMTP_HOST"); host != "" {
		config.Host = host
	}
	
	if port := getEnvIntOrDefault("SMTP_PORT", 587); port > 0 {
		config.Port = port
	}
	
	if user := os.Getenv("SMTP_USER"); user != "" {
		config.User = user
	}
	
	if password := os.Getenv("SMTP_PASSWORD"); password != "" {
		config.Password = password
	}
	
	if tlsEnabled := getEnvBoolOrDefault("SMTP_TLS_ENABLED", true); tlsEnabled {
		config.TLSEnabled = tlsEnabled
	}
	
	if insecureSkipVerify := getEnvBoolOrDefault("SMTP_INSECURE_SKIP_VERIFY", false); insecureSkipVerify {
		config.InsecureSkipVerify = insecureSkipVerify
	}
	
	return config
}

// getEnvIntOrDefault 获取环境变量并转换为整数，如果不存在则返回默认值
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBoolOrDefault 获取环境变量并转换为布尔值，如果不存在则返回默认值
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}


// SendEmail 发送邮件的方法
func (e *DefaultEmailSender) SendEmail(to, subject, body string) error {
	// 验证输入参数
	if err := e.validateEmailParams(to, subject, body); err != nil {
		return fmt.Errorf("invalid email parameters: %w", err)
	}

	// 检查SMTP配置是否完整
	if err := e.validateSMTPConfig(); err != nil {
		return fmt.Errorf("invalid SMTP config: %w", err)
	}

	// 创建邮件
	m := gomail.NewMessage()
	m.SetHeader("From", e.config.User)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 创建邮件客户端
	d := gomail.NewDialer(e.config.Host, e.config.Port, e.config.User, e.config.Password)

	// 设置TLS配置
	if e.config.TLSEnabled {
		d.TLSConfig = &tls.Config{
			ServerName:         e.config.Host,
			InsecureSkipVerify: e.config.InsecureSkipVerify,
		}
	}

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// validateEmailParams 验证邮件参数
func (e *DefaultEmailSender) validateEmailParams(to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("recipient email cannot be empty")
	}
	if subject == "" {
		return fmt.Errorf("email subject cannot be empty")
	}
	if body == "" {
		return fmt.Errorf("email body cannot be empty")
	}
	return nil
}

// validateSMTPConfig 验证SMTP配置
func (e *DefaultEmailSender) validateSMTPConfig() error {
	if e.config.Host == "" {
		return fmt.Errorf("SMTP host cannot be empty")
	}
	if e.config.Port <= 0 || e.config.Port > 65535 {
		return fmt.Errorf("SMTP port must be between 1 and 65535")
	}
	if e.config.User == "" {
		return fmt.Errorf("SMTP user cannot be empty, please set SMTP_USER environment variable")
	}
	if e.config.Password == "" {
		return fmt.Errorf("SMTP password cannot be empty, please set SMTP_PASSWORD environment variable")
	}
	return nil
}

// SendEmailWithConfig 使用指定配置发送邮件（全局函数，向后兼容）
func SendEmailWithConfig(config SMTPConfig, to, subject, body string) error {
	sender := &DefaultEmailSender{config: config}
	return sender.SendEmail(to, subject, body)
}

// SendEmail 使用默认配置发送邮件（全局函数，向后兼容）
func SendEmail(to, subject, body string) error {
	config := LoadSMTPConfigFromEnv()
	return SendEmailWithConfig(config, to, subject, body)
}
