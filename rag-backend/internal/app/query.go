package app

import (
	"net/http"

	"rag-backend/internal/database"

	"github.com/gin-gonic/gin"
)

// HandleQuery 处理RAG查询
func HandleQuery(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	// 检查查询是否为空
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "查询不能为空"})
		return
	}

	// 生成查询向量
	vector, err := database.GetEmbedding(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取向量失败: " + err.Error()})
		return
	}

	// 查询 Qdrant
	results, err := database.SearchQdrant(vector)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败: " + err.Error()})
		return
	}

	// 如果没有找到相关内容
	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"answer":  "抱歉，我没有找到与您问题相关的信息。",
			"sources": []string{},
		})
		return
	}

	// 调用 OpenAI 生成答案
	answer, err := database.GenerateResponse(results, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成答案失败: " + err.Error()})
		return
	}

	// 返回答案和来源
	c.JSON(http.StatusOK, gin.H{
		"answer":  answer,
		"sources": results,
	})
}
