# RAG系统

这是一个基于检索增强生成（Retrieval-Augmented Generation, RAG）的问答系统，使用Go语言开发后端，React开发前端。

## 功能特点

- 文档上传和处理
- 向量化存储和检索
- 基于OpenAI的智能问答
- 文档管理和重新处理

## 技术栈

### 后端

- Go语言
- Gin Web框架
- MongoDB（文档元数据存储）
- Qdrant（向量数据库）
- OpenAI API（向量嵌入和问答生成）

### 前端

- React
- Axios（HTTP请求）
- React Markdown（格式化显示）

## 安装和使用

### 环境要求

- Go 1.16+
- Node.js 14+
- MongoDB
- Qdrant

### 后端设置

1. 进入后端目录：
   ```
   cd rag-backend
   ```

2. 安装依赖：
   ```
   go mod tidy
   ```

3. 创建`.env`文件并配置：
   ```
   # 数据库配置
   MONGODB_URI=mongodb://localhost:27017
   MONGODB_DATABASE=admin
   MONGODB_COLLECTION=RAGDATA

   # Qdrant配置
   QDRANT_URL=http://localhost:6333
   QDRANT_HOST=localhost:6334
   QDRANT_COLLECTION=documents

   # OpenAI配置
   OPENAI_API_KEY=your_openai_api_key_here
   OPENAI_MODEL=gpt-4o
   OPENAI_EMBEDDING_MODEL=text-embedding-ada-002

   # 服务器配置
   SERVER_PORT=8080
   ```

4. 启动后端服务：
   ```
   go run cmd/api/main.go cmd/api/route.go
   ```

### 前端设置

1. 进入前端目录：
   ```
   cd rag-frontend
   ```

2. 安装依赖：
   ```
   npm install
   ```

3. 启动前端开发服务器：
   ```
   npm start
   ```

## 使用方法

1. 上传文档：支持PDF和TXT格式
2. 等待文档处理完成（状态变为"ready"）
3. 在查询框中输入问题
4. 系统会基于上传的文档内容生成回答

## 高级功能

- **重新处理文档**：如果需要更新文档的向量表示，可以使用"重新处理"功能
- **清空向量数据库**：清除所有向量数据，但保留文档记录
- **清空所有文档**：完全清除所有文档和相关数据
- **清理无效记录**：修复数据库中的无效记录

## 许可证

MIT 