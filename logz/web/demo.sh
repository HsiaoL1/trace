#!/bin/bash

# 日志管理系统 Web 界面演示脚本

echo "=========================================="
echo "    日志管理系统 Web 界面演示"
echo "=========================================="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go"
    exit 1
fi

# 设置环境变量
export LOG_DIR="logs"
export PORT="8080"

# 创建日志目录
mkdir -p "$LOG_DIR"

echo "日志目录: $LOG_DIR"
echo "服务端口: $PORT"
echo ""

# 启动日志生成器（后台运行）
echo "启动日志生成器..."
go run demo.go &
DEMO_PID=$!

# 等待一下让日志生成器启动
sleep 2

# 启动Web服务器（后台运行）
echo "启动Web服务器..."
go run main.go &
WEB_PID=$!

# 等待一下让Web服务器启动
sleep 3

echo ""
echo "=========================================="
echo "    演示已启动！"
echo "=========================================="
echo ""
echo "🌐 Web界面: http://localhost:$PORT"
echo "📁 日志目录: $LOG_DIR"
echo ""
echo "功能演示:"
echo "1. 打开浏览器访问 http://localhost:$PORT"
echo "2. 查看主页面统计信息和文件列表"
echo "3. 使用高级搜索功能"
echo "4. 点击文件查看详细日志"
echo "5. 访问错误日志页面"
echo ""
echo "按 Ctrl+C 停止演示..."

# 等待用户中断
trap 'cleanup' INT

wait

cleanup() {
    echo ""
    echo "正在停止演示..."
    
    # 停止日志生成器
    if kill -0 $DEMO_PID 2>/dev/null; then
        kill $DEMO_PID
        echo "✓ 日志生成器已停止"
    fi
    
    # 停止Web服务器
    if kill -0 $WEB_PID 2>/dev/null; then
        kill $WEB_PID
        echo "✓ Web服务器已停止"
    fi
    
    echo ""
    echo "演示已结束！"
    exit 0
} 