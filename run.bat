@echo off
echo 正在启动 RAG 系统...

:: 启动后端服务
start cmd /k "cd rag-backend && go run cmd/api/main.go cmd/api/route.go"

:: 启动前端服务
start cmd /k "cd rag-frontend && npm start"

echo RAG 系统已启动！
echo 后端服务运行在: http://localhost:8080
echo 前端服务运行在: http://localhost:3000 