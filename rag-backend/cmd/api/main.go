package main

import (
	"fmt"
	"log"
	"os"

	"rag-backend/internal/database"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到.env文件或加载失败，将使用默认值")
	}

	// 初始化 MongoDB 和 Qdrant 连接
	database.ConnectMongoDB()
	database.ConnectQdrant()

	// 创建 Gin 路由
	router := gin.Default()

	// 设置文件上传大小限制
	router.MaxMultipartMemory = 8 << 20 // 8 MiB

	// 配置CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	RegisterRoutes(router) // 使用本地的RegisterRoutes函数

	// 获取端口号
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080" // 默认端口
	}

	fmt.Printf("✅ 服务器启动成功，监听端口 %s\n", port)
	log.Fatal(router.Run(":" + port))
}
