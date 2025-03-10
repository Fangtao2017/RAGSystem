package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rag-backend/internal/database"

	"github.com/ledongthuc/pdf"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 从文件中提取文本内容
func extractTextFromFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".pdf":
		return extractTextFromPDF(filePath)
	case ".txt":
		return extractTextFromTXT(filePath)
	default:
		return "", fmt.Errorf("不支持的文件类型: %s", ext)
	}
}

// 从PDF文件中提取文本
func extractTextFromPDF(filePath string) (string, error) {
	fmt.Printf("开始提取PDF文本: %s\n", filePath)

	// 打开PDF文件
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开PDF文件失败: %v", err)
	}
	defer f.Close()

	// 获取页数
	totalPage := r.NumPage()
	fmt.Printf("PDF共有 %d 页\n", totalPage)

	var textBuilder strings.Builder

	// 逐页提取文本
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		fmt.Printf("处理第 %d/%d 页...\n", pageIndex, totalPage)

		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			fmt.Printf("警告: 第 %d 页文本提取失败: %v\n", pageIndex, err)
			continue
		}

		textBuilder.WriteString(text)
		fmt.Printf("第 %d 页提取了 %d 个字符\n", pageIndex, len(text))
	}

	result := textBuilder.String()
	fmt.Printf("PDF文本提取完成，共 %d 个字符\n", len(result))

	// 打印前200个字符作为预览
	preview := result
	if len(result) > 200 {
		preview = result[:200] + "..."
	}
	fmt.Printf("文本预览: %s\n", preview)

	return result, nil
}

// 从TXT文件中提取文本
func extractTextFromTXT(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// 将文本分割成块
func splitTextIntoChunks(text string, chunkSize int, overlap int) []string {
	fmt.Printf("开始分割文本，总长度: %d 字符，块大小: %d，重叠: %d\n", len(text), chunkSize, overlap)

	if len(text) == 0 {
		fmt.Println("警告: 输入文本为空")
		return []string{}
	}

	// 如果文本长度小于块大小，直接返回整个文本
	if len(text) <= chunkSize {
		fmt.Println("文本长度小于块大小，返回单个块")
		return []string{text}
	}

	var chunks []string

	// 按段落分割文本
	paragraphs := strings.Split(text, "\n")
	fmt.Printf("文本被分割为 %d 个段落\n", len(paragraphs))

	currentChunk := ""

	for _, paragraph := range paragraphs {
		// 跳过空段落
		if strings.TrimSpace(paragraph) == "" {
			continue
		}

		// 如果当前段落加上当前块的长度小于块大小，则将段落添加到当前块
		if len(currentChunk)+len(paragraph)+1 <= chunkSize {
			if currentChunk != "" {
				currentChunk += "\n"
			}
			currentChunk += paragraph
		} else {
			// 如果当前块不为空，将其添加到块列表
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)

				// 计算重叠部分
				words := strings.Fields(currentChunk)
				if len(words) > 0 {
					overlapWordCount := int(float64(len(words)) * float64(overlap) / float64(chunkSize))
					if overlapWordCount > 0 && overlapWordCount < len(words) {
						overlapText := strings.Join(words[len(words)-overlapWordCount:], " ")
						currentChunk = overlapText
					} else {
						currentChunk = ""
					}
				}
			}

			// 如果当前段落大于块大小，则将其分割成多个块
			if len(paragraph) > chunkSize {
				words := strings.Fields(paragraph)
				var subChunk string

				for _, word := range words {
					if len(subChunk)+len(word)+1 <= chunkSize {
						if subChunk != "" {
							subChunk += " "
						}
						subChunk += word
					} else {
						if subChunk != "" {
							chunks = append(chunks, subChunk)
						}
						subChunk = word
					}
				}

				if subChunk != "" {
					if currentChunk != "" {
						currentChunk += " " + subChunk
					} else {
						currentChunk = subChunk
					}
				}
			} else {
				// 否则，将当前段落作为新块的开始
				if currentChunk != "" {
					currentChunk += "\n" + paragraph
				} else {
					currentChunk = paragraph
				}
			}
		}
	}

	// 添加最后一个块
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	fmt.Printf("文本分割完成，共生成 %d 个块\n", len(chunks))

	// 打印每个块的大小和前50个字符
	for i, chunk := range chunks {
		preview := chunk
		if len(chunk) > 50 {
			preview = chunk[:50] + "..."
		}
		fmt.Printf("块 #%d: %d 字符, 预览: %s\n", i+1, len(chunk), preview)
	}

	return chunks
}

// 处理文档
func processDocument(docID primitive.ObjectID, filePath string) {
	// 提取文本
	content, err := extractTextFromFile(filePath)
	if err != nil {
		fmt.Printf("❌ 提取文本失败: %v\n", err)
		// 更新文档状态为失败
		ctx := context.Background()
		database.MongoCollection.UpdateOne(ctx,
			bson.M{"_id": docID},
			bson.M{"$set": bson.M{"status": "failed", "error": err.Error()}})
		return
	}

	// 分割文本为块
	chunks := splitTextIntoChunks(content, 1000, 200)
	fmt.Printf("文档被分割为 %d 个块\n", len(chunks))

	// 处理每个块
	for i, chunk := range chunks {
		fmt.Printf("处理块 %d/%d...\n", i+1, len(chunks))

		// 生成向量嵌入
		vector, err := database.GetEmbedding(chunk)
		if err != nil {
			fmt.Printf("❌ 块 %d 生成向量失败: %v\n", i+1, err)
			continue
		}

		// 存储向量到Qdrant
		err = database.StoreVectorInQdrant(docID.Hex(), i, vector, chunk)
		if err != nil {
			fmt.Printf("❌ 块 %d Qdrant存储失败: %v\n", i+1, err)
			continue
		}
	}

	// 更新文档状态为已处理
	ctx := context.Background()
	database.MongoCollection.UpdateOne(ctx,
		bson.M{"_id": docID},
		bson.M{"$set": bson.M{
			"status":      "ready",
			"chunks":      len(chunks),
			"processedAt": time.Now(),
		}})

	fmt.Printf("✅ 文档处理完成，ID: %s\n", docID.Hex())
}
