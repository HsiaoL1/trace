# 日志管理系统 Web 界面

一个功能完整的日志管理系统，提供Web界面和RESTful API接口，支持日志聚合、查询、文件管理和第三方服务集成。

## 功能特性

### Web界面功能
- 📊 **日志文件管理**: 查看、搜索、删除日志文件
- 🔍 **高级搜索**: 支持多条件组合查询（TraceID、SpanID、级别、服务等）
- 📄 **文件内容查看**: 分页显示文件内容，支持内容搜索
- ⚠️ **错误日志页面**: 专门展示错误级别日志
- 📈 **统计信息**: 显示日志系统统计信息
- 🗂️ **文件压缩**: 自动压缩旧日志文件

### API接口功能
- 🔌 **RESTful API**: 完整的REST API接口
- 📝 **日志写入**: 支持第三方服务写入日志
- 🔍 **日志查询**: 多种查询方式（TraceID、SpanID、级别、服务等）
- 📊 **统计信息**: 获取系统统计信息
- 🗂️ **文件管理**: 文件列表、内容获取、删除等操作
- 💚 **健康检查**: 服务健康状态监控

## 快速开始

### 1. 启动服务器

```bash
# 进入web目录
cd web

# 启动服务器
./start.sh
```

服务器将在 `http://localhost:8080` 启动。

### 2. 访问Web界面

- **主页**: http://localhost:8080
- **错误日志**: http://localhost:8080/errors
- **API健康检查**: http://localhost:8080/api/v1/health

### 3. 测试API接口

```bash
# 运行API测试
python3 api_test.py
```

## 第三方服务集成

### API接口概览

| 功能 | 方法 | 端点 | 描述 |
|------|------|------|------|
| 健康检查 | GET | `/api/v1/health` | 检查服务状态 |
| 写入日志 | POST | `/api/v1/logs/write` | 写入日志条目 |
| 搜索日志 | POST | `/api/v1/logs/search` | 复杂条件搜索 |
| 按TraceID查询 | GET | `/api/v1/logs/trace/{id}` | 根据TraceID查询 |
| 按SpanID查询 | GET | `/api/v1/logs/span/{id}` | 根据SpanID查询 |
| 按级别查询 | GET | `/api/v1/logs/level/{level}` | 根据日志级别查询 |
| 按服务查询 | GET | `/api/v1/logs/service/{service}` | 根据服务名查询 |
| 获取错误日志 | GET | `/api/v1/logs/errors` | 获取所有错误日志 |
| 获取文件列表 | GET | `/api/v1/files` | 获取日志文件列表 |
| 获取文件内容 | GET | `/api/v1/files/content/{file}` | 获取文件内容 |
| 删除文件 | DELETE | `/api/v1/files/{file}` | 删除日志文件 |
| 获取统计信息 | GET | `/api/v1/stats` | 获取系统统计 |

### Python集成示例

```python
import requests
import json
from datetime import datetime

class LogManagerClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({'Content-Type': 'application/json'})
    
    def write_log(self, level, message, trace_id=None, span_id=None, service=None, **fields):
        """写入日志"""
        data = {
            "level": level,
            "message": message,
            "timestamp": datetime.now().isoformat()
        }
        
        if trace_id:
            data["trace_id"] = trace_id
        if span_id:
            data["span_id"] = span_id
        if service:
            data["service"] = service
        if fields:
            data["fields"] = fields
            
        response = self.session.post(f"{self.base_url}/api/v1/logs/write", json=data)
        return response.json()
    
    def search_logs(self, **kwargs):
        """搜索日志"""
        response = self.session.post(f"{self.base_url}/api/v1/logs/search", json=kwargs)
        return response.json()

# 使用示例
client = LogManagerClient()

# 写入日志
client.write_log(
    level="info",
    message="User login successful",
    trace_id="abc123",
    span_id="def456",
    service="auth-service",
    user_id="12345",
    ip="192.168.1.1"
)

# 搜索日志
result = client.search_logs(
    trace_id="abc123",
    level="error",
    limit=50
)
```

### Node.js集成示例

```javascript
const axios = require('axios');

class LogManagerClient {
    constructor(baseUrl = 'http://localhost:8080') {
        this.baseUrl = baseUrl;
        this.client = axios.create({
            baseURL: baseUrl,
            headers: {
                'Content-Type': 'application/json'
            }
        });
    }

    async writeLog(level, message, options = {}) {
        const data = {
            level,
            message,
            timestamp: new Date().toISOString(),
            ...options
        };

        const response = await this.client.post('/api/v1/logs/write', data);
        return response.data;
    }

    async searchLogs(query) {
        const response = await this.client.post('/api/v1/logs/search', query);
        return response.data;
    }
}

// 使用示例
const client = new LogManagerClient();

client.writeLog('info', 'User login successful', {
    trace_id: 'abc123',
    span_id: 'def456',
    service: 'auth-service',
    fields: {
        user_id: '12345',
        ip: '192.168.1.1'
    }
});
```

### Go集成示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type LogManagerClient struct {
    baseURL string
    client  *http.Client
}

type LogWriteRequest struct {
    Level     string            `json:"level"`
    Message   string            `json:"message"`
    TraceID   string            `json:"trace_id,omitempty"`
    SpanID    string            `json:"span_id,omitempty"`
    Service   string            `json:"service,omitempty"`
    Fields    map[string]any    `json:"fields,omitempty"`
    Timestamp time.Time         `json:"timestamp,omitempty"`
}

func NewLogManagerClient(baseURL string) *LogManagerClient {
    return &LogManagerClient{
        baseURL: baseURL,
        client:  &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *LogManagerClient) WriteLog(req LogWriteRequest) error {
    if req.Timestamp.IsZero() {
        req.Timestamp = time.Now()
    }

    data, err := json.Marshal(req)
    if err != nil {
        return err
    }

    resp, err := c.client.Post(c.baseURL+"/api/v1/logs/write", "application/json", bytes.NewBuffer(data))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
    }

    return nil
}

// 使用示例
func main() {
    client := NewLogManagerClient("http://localhost:8080")

    err := client.WriteLog(LogWriteRequest{
        Level:   "info",
        Message: "User login successful",
        TraceID: "abc123",
        SpanID:  "def456",
        Service: "auth-service",
        Fields: map[string]any{
            "user_id": "12345",
            "ip":      "192.168.1.1",
        },
    })
    if err != nil {
        fmt.Printf("写入日志失败: %v\n", err)
    }
}
```

## API响应格式

所有API都使用标准JSON响应格式：

```json
{
  "success": true,
  "data": {},
  "error": "",
  "message": "",
  "code": 200
}
```

## 配置选项

### 环境变量

- `LOG_DIR`: 日志文件目录（默认: `logs`）
- `PORT`: 服务端口（默认: `8080`）

### 启动示例

```bash
# 自定义配置
LOG_DIR=/var/logs PORT=9090 ./start.sh
```

## 监控和运维

### 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

### 获取统计信息

```bash
curl http://localhost:8080/api/v1/stats
```

### 查看错误日志

```bash
curl http://localhost:8080/api/v1/logs/errors?limit=10
```

## 故障排除

### 常见问题

1. **模板文件找不到**
   - 确保在正确的目录下运行 `./start.sh`
   - 检查 `web/templates/` 目录是否存在

2. **API连接失败**
   - 检查服务器是否正在运行
   - 确认端口是否正确
   - 检查防火墙设置

3. **日志写入失败**
   - 检查日志目录权限
   - 确认聚合器是否正常初始化

### 日志文件位置

- 日志文件: `logs/` 目录
- 索引文件: `logs/index/` 目录
- 压缩文件: 自动添加 `.gz` 后缀

## 开发指南

### 添加新的API端点

1. 在 `api.go` 中添加新的处理函数
2. 在 `SetupAPIRoutes()` 中注册路由
3. 更新API文档

### 自定义响应格式

修改 `APIResponse` 结构体来定制响应格式。

### 添加认证

在生产环境中，建议添加API密钥或OAuth认证机制。

## 许可证

MIT License 