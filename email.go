package trace

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

// 全局配置Email
var Email string

// SMTP配置结构体
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// 全局SMTP配置
var smtpConfig SMTPConfig

// 设置SMTP配置
func SetSMTPConfig(host string, port int, user, password string) {
	smtpConfig = SMTPConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
}

// 从环境变量加载SMTP配置
func LoadSMTPConfigFromEnv() {
	host := getEnvOrDefault("SMTP_HOST", "smtp.qq.com")
	port := getEnvIntOrDefault("SMTP_PORT", 587)
	user := getEnvOrDefault("SMTP_USER", "")
	password := getEnvOrDefault("SMTP_PASSWORD", "")

	smtpConfig = SMTPConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
}

// 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 获取环境变量并转换为整数，如果不存在则返回默认值
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// 全局配置Email
func SetEmail(email string) {
	Email = email
}

// 全局配置Email
func GetEmail() string {
	return Email
}

// 发送邮件的方法
func SendEmail(to, subject, body string) error {
	// 检查SMTP配置是否完整
	if smtpConfig.User == "" || smtpConfig.Password == "" {
		return fmt.Errorf("SMTP配置不完整，请设置SMTP_USER和SMTP_PASSWORD环境变量")
	}

	// 创建邮件
	m := gomail.NewMessage()
	m.SetHeader("From", smtpConfig.User)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 创建邮件客户端
	d := gomail.NewDialer(smtpConfig.Host, smtpConfig.Port, smtpConfig.User, smtpConfig.Password)

	// 设置TLS配置
	d.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
