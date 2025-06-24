package trace

import (
	"crypto/tls"

	"gopkg.in/gomail.v2"
)

// 全局配置Email
var Email string

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
	// 获取配置
	emailHost := "smtp.qq.com"
	emailPort := 587
	emailUser := "1299720482@qq.com"
	emailPassword := "xkvdggdstkvebafi"
	// 创建邮件
	m := gomail.NewMessage()
	m.SetHeader("From", emailUser)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 创建邮件客户端
	d := gomail.NewDialer(emailHost, emailPort, emailUser, emailPassword)

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
