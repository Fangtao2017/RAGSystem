package main

import (
	"fmt"
	"net/http"
	"rag-backend/internal/app"
	"rag-backend/internal/database"

	"github.com/gin-gonic/gin"
)

// 注册路由
func RegisterRoutes(router *gin.Engine) {
	// 文档上传
	router.POST("/upload", app.HandleUpload)

	// 获取文档列表
	router.GET("/documents", app.HandleListDocuments)

	// 删除文档
	router.DELETE("/document/:doc_id", app.HandleDeleteDocument)

	// 清空所有文档
	router.POST("/clear-all-documents", app.HandleClearAllDocuments)

	// 清理无效文档记录
	router.POST("/cleanup-invalid-documents", app.HandleCleanupInvalidDocuments)

	// 重新处理文档
	router.POST("/document/:doc_id/reprocess", app.HandleReprocessDocument)

	// 查询
	router.POST("/query", app.HandleQuery)

	// 获取文档状态
	router.GET("/status/:task_id", app.HandleStatus)

	// 清空向量数据库
	router.POST("/clear-vectors", HandleClearVectors)
}

// 处理清空向量数据库请求
func HandleClearVectors(c *gin.Context) {
	err := database.ClearQdrantCollection()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("清空向量数据库失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "向量数据库已清空"})
}
