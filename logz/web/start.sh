#!/bin/bash

# 日志管理Web服务器启动脚本

echo "=========================================="
echo "    日志管理系统 Web 服务器"
echo "=========================================="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go"
    exit 1
fi

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 切换到项目根目录
cd "$PROJECT_DIR"

# 设置环境变量
export LOG_DIR="${LOG_DIR:-logs}"
export PORT="${PORT:-8080}"

# 创建日志目录
mkdir -p "$LOG_DIR"

echo "项目目录: $PROJECT_DIR"
echo "日志目录: $LOG_DIR"
echo "服务端口: $PORT"
echo ""

# 检查依赖
echo "检查依赖..."
go mod tidy

# 编译并运行
echo "启动Web服务器..."
echo "访问地址: http://localhost:$PORT"
echo "按 Ctrl+C 停止服务器"
echo ""

go run web/main.go 