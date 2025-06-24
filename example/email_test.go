package main

import (
	"fmt"
	"log"

	"github.com/HsiaoL1/trace"
	"github.com/HsiaoL1/trace/logz"
)

func main() {
	// 初始化日志
	logz.InitDevelopment()

	fmt.Println("=== 邮件配置测试 ===")

	// 方法1：从环境变量加载配置
	fmt.Println("1. 从环境变量加载SMTP配置...")
	trace.LoadSMTPConfigFromEnv()

	// 设置接收邮箱
	trace.SetEmail("test@example.com")

	// 测试邮件发送
	fmt.Println("2. 测试邮件发送...")
	err := trace.SendEmail("test@example.com", "测试邮件", "这是一封测试邮件")
	if err != nil {
		log.Printf("发送邮件失败: %v", err)
		fmt.Println("请检查环境变量配置：")
		fmt.Println("- SMTP_USER: 邮箱地址")
		fmt.Println("- SMTP_PASSWORD: 邮箱授权码")
		fmt.Println("- SMTP_HOST: SMTP服务器地址（可选，默认smtp.qq.com）")
		fmt.Println("- SMTP_PORT: SMTP端口（可选，默认587）")
	} else {
		fmt.Println("邮件发送成功！")
	}

	// 方法2：在代码中设置配置
	fmt.Println("\n3. 在代码中设置SMTP配置...")
	trace.SetSMTPConfig("smtp.qq.com", 587, "your-email@qq.com", "your-password")

	// 测试日志邮件通知
	fmt.Println("4. 测试日志邮件通知...")
	logz.ErrorWithEmail(true, "这是一条测试错误日志，会发送邮件通知")

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("请查看CONFIG.md文件了解详细配置说明")
}
