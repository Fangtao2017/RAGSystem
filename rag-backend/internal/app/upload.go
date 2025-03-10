package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"rag-backend/internal/database"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// HandleUpload 处理文件上传
func HandleUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}
	defer file.Close()

	fileName := header.Filename
	filePath := "./uploads/" + fileName

	// 确保uploads目录存在
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建上传目录"})
		return
	}

	// 创建并保存文件
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存文件"})
		return
	}
	defer out.Close()
	io.Copy(out, file)

	// 存入 MongoDB
	doc := bson.M{
		"name":       fileName,
		"content":    "", // 解析后再更新
		"status":     "processing",
		"uploadedAt": time.Now(),
	}
	res, err := database.MongoCollection.InsertOne(context.Background(), doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库存储失败"})
		return
	}

	// 获取插入的文档ID
	docID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无效的文档ID"})
		return
	}

	// 异步处理文档
	go processDocument(docID, filePath)

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功，正在处理文档",
		"docId":   docID.Hex(),
	})
}

// HandleStatus 查询任务进度
func HandleStatus(c *gin.Context) {
	ctx := context.Background()
	taskID := c.Param("task_id")

	// 转换taskID为ObjectID
	objectID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 从MongoDB查询文档状态
	var doc bson.M
	err = database.MongoCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	status, ok := doc["status"].(string)
	if !ok {
		status = "unknown"
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// HandleListDocuments 处理获取文档列表的请求
func HandleListDocuments(c *gin.Context) {
	ctx := context.Background()

	// 从MongoDB获取所有文档
	cursor, err := database.MongoCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取文档列表失败: %v", err)})
		return
	}
	defer cursor.Close(ctx)

	// 解码文档
	var documents []bson.M
	if err := cursor.All(ctx, &documents); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("解码文档失败: %v", err)})
		return
	}

	// 过滤掉无效的文档记录
	var validDocuments []bson.M
	for _, doc := range documents {
		// 检查文档是否有名称和状态
		if name, ok := doc["name"].(string); ok && name != "" {
			if status, ok := doc["status"].(string); ok && status != "" {
				validDocuments = append(validDocuments, doc)
			}
		}
	}

	fmt.Printf("找到 %d 个文档，其中 %d 个有效\n", len(documents), len(validDocuments))

	// 直接返回文档数组，不要包装在documents字段中
	c.JSON(http.StatusOK, validDocuments)
}

// HandleDeleteDocument 删除文档
func HandleDeleteDocument(c *gin.Context) {
	ctx := context.Background()
	docID := c.Param("doc_id")

	fmt.Printf("🗑️ 尝试删除文档: %s\n", docID)

	// 检查docID是否为空
	if docID == "" || docID == "undefined" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
		return
	}

	// 转换docID为ObjectID
	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		fmt.Printf("❌ 无效的文档ID格式: %s, 错误: %v\n", docID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID格式"})
		return
	}

	// 先获取文档信息，以便后续删除文件
	var doc bson.M
	err = database.MongoCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Printf("⚠️ 文档不存在: %s\n", docID)
			c.JSON(http.StatusNotFound, gin.H{"error": "文档不存在"})
		} else {
			fmt.Printf("❌ 查询文档失败: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文档失败"})
		}
		return
	}

	// 从MongoDB删除文档
	result, err := database.MongoCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		fmt.Printf("❌ 删除文档失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文档失败"})
		return
	}

	if result.DeletedCount == 0 {
		fmt.Printf("⚠️ 文档不存在或已被删除: %s\n", docID)
		c.JSON(http.StatusNotFound, gin.H{"error": "文档不存在或已被删除"})
		return
	}

	// 尝试删除文件系统中的文件
	if fileName, ok := doc["name"].(string); ok {
		filePath := "./uploads/" + fileName
		err := os.Remove(filePath)
		if err != nil {
			fmt.Printf("⚠️ 删除文件失败: %s, 错误: %v\n", filePath, err)
			// 不返回错误，因为文档已经从数据库中删除
		} else {
			fmt.Printf("✅ 文件删除成功: %s\n", filePath)
		}
	}

	// 尝试从Qdrant中删除向量
	// 注意：由于Qdrant API的限制，我们暂时不实现这部分功能
	// 在实际生产环境中，应该实现向量数据的清理
	fmt.Printf("⚠️ 注意：向量数据需要手动清理\n")

	fmt.Printf("✅ 文档删除成功: %s\n", docID)
	c.JSON(http.StatusOK, gin.H{"message": "文档删除成功"})
}

// HandleClearAllDocuments 处理清空所有文档的请求
func HandleClearAllDocuments(c *gin.Context) {
	ctx := context.Background()

	// 获取所有文档
	cursor, err := database.MongoCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取文档列表失败: %v", err)})
		return
	}
	defer cursor.Close(ctx)

	// 解码文档
	var documents []bson.M
	if err := cursor.All(ctx, &documents); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("解码文档失败: %v", err)})
		return
	}

	// 删除文件
	for _, doc := range documents {
		if filePath, ok := doc["filePath"].(string); ok && filePath != "" {
			// 尝试删除文件，但不中断流程
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("警告: 无法删除文件 %s: %v\n", filePath, err)
			} else {
				fmt.Printf("已删除文件: %s\n", filePath)
			}
		}
	}

	// 删除所有文档记录
	result, err := database.MongoCollection.DeleteMany(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除文档记录失败: %v", err)})
		return
	}

	fmt.Printf("已删除 %d 个文档记录\n", result.DeletedCount)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("已清空所有文档，共删除 %d 个记录", result.DeletedCount),
	})
}

// HandleReprocessDocument 处理重新处理文档的请求
func HandleReprocessDocument(c *gin.Context) {
	// 获取文档ID
	docID := c.Param("doc_id")
	if docID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文档ID"})
		return
	}

	// 将字符串ID转换为ObjectID
	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("无效的文档ID: %v", err)})
		return
	}

	ctx := context.Background()

	// 查找文档
	var doc bson.M
	err = database.MongoCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("找不到文档: %v", err)})
		return
	}

	// 获取文件路径
	filePath, ok := doc["filePath"].(string)
	if !ok || filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文档没有有效的文件路径"})
		return
	}

	// 更新文档状态为处理中
	_, err = database.MongoCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"status": "processing"}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新文档状态失败: %v", err)})
		return
	}

	// 异步处理文档
	go processDocument(objectID, filePath)

	c.JSON(http.StatusOK, gin.H{
		"message": "文档重新处理已开始",
		"docId":   docID,
	})
}

// HandleCleanupInvalidDocuments 处理清理无效文档记录的请求
func HandleCleanupInvalidDocuments(c *gin.Context) {
	ctx := context.Background()

	// 查找所有无效的文档记录
	filter := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$exists": false}},
			{"name": ""},
			{"name": bson.M{"$type": "null"}},
			{"status": bson.M{"$exists": false}},
			{"status": ""},
			{"status": bson.M{"$type": "null"}},
		},
	}

	// 删除无效的文档记录
	result, err := database.MongoCollection.DeleteMany(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除无效文档记录失败: %v", err)})
		return
	}

	fmt.Printf("已删除 %d 个无效文档记录\n", result.DeletedCount)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("已清理 %d 个无效文档记录", result.DeletedCount),
		"count":   result.DeletedCount,
	})
}
