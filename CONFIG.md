# 配置说明

## 环境变量配置

为了避免敏感信息泄露，本项目使用环境变量来管理SMTP配置。

### 必需的环境变量

| 变量名 | 说明 | 示例值 |
|--------|------|--------|
| `SMTP_USER` | SMTP用户名（邮箱地址） | `your-email@qq.com` |
| `SMTP_PASSWORD` | SMTP密码（邮箱授权码） | `your-email-password` |

### 可选的环境变量

| 变量名 | 说明 | 默认值 | 示例值 |
|--------|------|--------|--------|
| `SMTP_HOST` | SMTP服务器地址 | `smtp.qq.com` | `smtp.gmail.com` |
| `SMTP_PORT` | SMTP端口 | `587` | `465` |
| `NOTIFICATION_EMAIL` | 接收通知的邮箱地址 | - | `developer@example.com` |

## 配置方法

### 方法1：直接设置环境变量

```bash
# Linux/macOS
export SMTP_USER="your-email@qq.com"
export SMTP_PASSWORD="your-email-password"
export SMTP_HOST="smtp.qq.com"
export SMTP_PORT="587"
export NOTIFICATION_EMAIL="developer@example.com"

# Windows (CMD)
set SMTP_USER=your-email@qq.com
set SMTP_PASSWORD=your-email-password
set SMTP_HOST=smtp.qq.com
set SMTP_PORT=587
set NOTIFICATION_EMAIL=developer@example.com

# Windows (PowerShell)
$env:SMTP_USER="your-email@qq.com"
$env:SMTP_PASSWORD="your-email-password"
$env:SMTP_HOST="smtp.qq.com"
$env:SMTP_PORT="587"
$env:NOTIFICATION_EMAIL="developer@example.com"
```

### 方法2：使用.env文件（需要第三方库支持）

如果你使用 `godotenv` 库，可以创建 `.env` 文件：

```bash
# 复制示例文件
cp env.example .env

# 编辑.env文件，填入实际配置
```

然后在代码中加载：

```go
import "github.com/joho/godotenv"

func init() {
    godotenv.Load()
}
```

### 方法3：在代码中设置

```go
package main

import (
    "github.com/HsiaoL1/trace"
    "github.com/HsiaoL1/trace/logz"
)

func main() {
    // 设置SMTP配置
    trace.SetSMTPConfig("smtp.qq.com", 587, "your-email@qq.com", "your-password")
    
    // 设置接收邮箱
    trace.SetEmail("developer@example.com")
    
    // 初始化日志
    logz.InitDevelopment()
    
    // 测试邮件发送
    logz.ErrorWithEmail(true, "测试邮件通知")
}
```

## 常用邮箱配置

### QQ邮箱

```bash
SMTP_HOST=smtp.qq.com
SMTP_PORT=587
SMTP_USER=your-qq-email@qq.com
SMTP_PASSWORD=your-qq-authorization-code
```

**注意**：QQ邮箱需要使用授权码，不是登录密码。获取方法：
1. 登录QQ邮箱
2. 设置 → 账户 → POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务
3. 开启POP3/SMTP服务
4. 生成授权码

### Gmail

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-gmail@gmail.com
SMTP_PASSWORD=your-app-password
```

**注意**：Gmail需要使用应用专用密码，不是账户密码。获取方法：
1. 开启两步验证
2. 生成应用专用密码

### 163邮箱

```bash
SMTP_HOST=smtp.163.com
SMTP_PORT=587
SMTP_USER=your-163-email@163.com
SMTP_PASSWORD=your-163-authorization-code
```

## 安全建议

1. **永远不要将敏感信息提交到代码仓库**
2. **使用环境变量或配置文件管理敏感信息**
3. **在生产环境中使用密钥管理服务**
4. **定期更换邮箱授权码**
5. **使用专门的邮箱用于系统通知**

## 故障排除

### 错误：SMTP配置不完整

```
SMTP配置不完整，请设置SMTP_USER和SMTP_PASSWORD环境变量
```

**解决方案**：确保设置了 `SMTP_USER` 和 `SMTP_PASSWORD` 环境变量。

### 错误：认证失败

```
authentication failed
```

**解决方案**：
1. 检查邮箱地址是否正确
2. 检查授权码是否正确
3. 确认邮箱服务已开启SMTP功能

### 错误：连接超时

```
connection timeout
```

**解决方案**：
1. 检查网络连接
2. 检查SMTP服务器地址和端口是否正确
3. 检查防火墙设置

## 测试配置

可以使用以下代码测试邮件配置：

```go
package main

import (
    "fmt"
    "github.com/HsiaoL1/trace"
)

func main() {
    // 加载环境变量配置
    trace.LoadSMTPConfigFromEnv()
    
    // 设置接收邮箱
    trace.SetEmail("test@example.com")
    
    // 发送测试邮件
    err := trace.SendEmail("test@example.com", "测试邮件", "这是一封测试邮件")
    if err != nil {
        fmt.Printf("发送邮件失败: %v\n", err)
    } else {
        fmt.Println("邮件发送成功")
    }
}
``` 