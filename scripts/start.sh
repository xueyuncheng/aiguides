#!/bin/bash

# AIGuide 启动脚本
# 同时启动后端和前端服务

set -e

echo "🚀 启动 AIGuide 服务..."
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装。请先安装 Go 1.25.5 或更高版本。"
    exit 1
fi

# 检查 Node.js 是否安装
if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装。请先安装 Node.js 20+。"
    exit 1
fi

# 检查 pnpm 是否安装
if ! command -v pnpm &> /dev/null; then
    echo "❌ pnpm 未安装。请运行: npm install -g pnpm"
    exit 1
fi

# 检查配置文件
if [ ! -f "cmd/aiguide/aiguide.yaml" ]; then
    echo "❌ 配置文件 cmd/aiguide/aiguide.yaml 不存在。"
    exit 1
fi

# 检查是否已安装前端依赖
if [ ! -d "frontend/node_modules" ]; then
    echo "📦 安装前端依赖..."
    cd frontend
    pnpm install
    cd ..
    echo "✅ 前端依赖安装完成"
    echo ""
fi

# 启动后端服务
echo "🔧 启动后端服务 (端口 8080)..."
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml &
BACKEND_PID=$!
echo "✅ 后端服务已启动 (PID: $BACKEND_PID)"
echo ""

# 等待后端启动
sleep 3

# 启动前端服务
echo "🎨 启动前端服务 (端口 3000)..."
cd frontend
pnpm dev &
FRONTEND_PID=$!
echo "✅ 前端服务已启动 (PID: $FRONTEND_PID)"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ AIGuide 已成功启动！"
echo ""
echo "📍 前端地址: http://localhost:3000"
echo "📍 后端地址: http://localhost:8080"
echo ""
echo "按 Ctrl+C 停止所有服务"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 等待用户中断
wait
