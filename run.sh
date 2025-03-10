#!/bin/bash
echo "正在启动 RAG 系统..."

# 启动后端服务
cd rag-backend && go run cmd/api/main.go cmd/api/route.go &
BACKEND_PID=$!

# 返回根目录
cd ..

# 启动前端服务
cd rag-frontend && npm start &
FRONTEND_PID=$!

echo "RAG 系统已启动！"
echo "后端服务运行在: http://localhost:8080"
echo "前端服务运行在: http://localhost:3000"
echo "按 Ctrl+C 停止所有服务"

# 等待用户按下Ctrl+C
trap "kill $BACKEND_PID $FRONTEND_PID; exit" INT
wait 