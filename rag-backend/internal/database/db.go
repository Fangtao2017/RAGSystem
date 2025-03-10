package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	qdrant "github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	MongoClient      *mongo.Client
	MongoCollection  *mongo.Collection
	qdrantConn       *grpc.ClientConn
	qdrantClient     qdrant.PointsClient
	qdrantCollection string
	OpenAIClient     *openai.Client
)

func ConnectMongoDB() {
	// 从环境变量获取MongoDB连接信息
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017" // 默认值
	}

	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "admin" // 默认值
	}

	collectionName := os.Getenv("MONGODB_COLLECTION")
	if collectionName == "" {
		collectionName = "RAGDATA" // 默认值
	}

	// 连接MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		panic(err)
	}

	// 选择数据库和集合
	MongoCollection = client.Database(dbName).Collection(collectionName)
	fmt.Printf("✅ 连接到 MongoDB (%s.%s)\n", dbName, collectionName)
}

// 连接到Qdrant
func ConnectQdrant() {
	// 从环境变量获取Qdrant连接信息
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		// 尝试使用QDRANT_URL
		qdrantURL := os.Getenv("QDRANT_URL")
		if qdrantURL != "" {
			// 如果是HTTP URL，转换为gRPC地址
			qdrantURL = strings.Replace(qdrantURL, "http://", "", 1)
			qdrantURL = strings.Replace(qdrantURL, "https://", "", 1)
			if strings.HasSuffix(qdrantURL, ":6333") {
				// 如果是HTTP端口，改为gRPC端口
				qdrantURL = strings.Replace(qdrantURL, ":6333", ":6334", 1)
			}
			qdrantHost = qdrantURL
		} else {
			qdrantHost = "localhost:6334" // 默认值
		}
	}

	qdrantCollection = os.Getenv("QDRANT_COLLECTION")
	if qdrantCollection == "" {
		qdrantCollection = "documents"
	}

	var err error
	qdrantConn, err = grpc.Dial(
		qdrantHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("无法连接到Qdrant: %v", err)
	}

	// 初始化qdrantClient
	qdrantClient = qdrant.NewPointsClient(qdrantConn)

	// 检查集合是否存在，如果不存在则创建
	collectionsClient := qdrant.NewCollectionsClient(qdrantConn)
	ctx := context.Background()

	// 检查集合是否存在
	_, err = collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: qdrantCollection,
	})

	if err != nil {
		// 创建集合
		_, err = collectionsClient.Create(ctx, &qdrant.CreateCollection{
			CollectionName: qdrantCollection,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     1536, // OpenAI嵌入维度
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			log.Fatalf("无法创建Qdrant集合: %v", err)
		}
		fmt.Println("✅ 成功创建Qdrant集合:", qdrantCollection)
	} else {
		fmt.Println("✅ 成功连接到Qdrant集合:", qdrantCollection)
	}

	// 初始化OpenAI客户端
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		fmt.Println("⚠️ 未设置OPENAI_API_KEY环境变量")
	} else {
		OpenAIClient = openai.NewClient(openaiKey)
		fmt.Println("✅ 初始化OpenAI客户端成功")
	}
}

// 获取向量嵌入
func GetEmbedding(text string) ([]float32, error) {
	if OpenAIClient == nil {
		return nil, fmt.Errorf("OpenAI客户端未初始化")
	}

	// 从环境变量获取嵌入模型
	embeddingModel := os.Getenv("OPENAI_EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "text-embedding-ada-002" // 默认值
	}

	// 调用OpenAI API获取嵌入
	resp, err := OpenAIClient.CreateEmbeddings(
		context.Background(),
		openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.EmbeddingModel(embeddingModel),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API调用失败: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("OpenAI返回空嵌入")
	}

	// 将[]float64转换为[]float32
	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// 存储向量到Qdrant
func StoreVectorInQdrant(docID string, chunkIndex int, vector []float32, text string) error {
	ctx := context.Background()

	// 创建唯一的点ID
	// 使用时间戳和随机数生成唯一ID
	pointID := uint64(time.Now().UnixNano() + int64(chunkIndex))

	// 创建Qdrant点
	point := &qdrant.PointStruct{
		Id: &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Num{
				Num: pointID,
			},
		},
		Vectors: &qdrant.Vectors{
			VectorsOptions: &qdrant.Vectors_Vector{
				Vector: &qdrant.Vector{
					Data: vector,
				},
			},
		},
		Payload: map[string]*qdrant.Value{
			"text": {
				Kind: &qdrant.Value_StringValue{
					StringValue: text,
				},
			},
			"docId": {
				Kind: &qdrant.Value_StringValue{
					StringValue: docID,
				},
			},
			"chunkIndex": {
				Kind: &qdrant.Value_IntegerValue{
					IntegerValue: int64(chunkIndex),
				},
			},
		},
	}

	// 将点添加到Qdrant集合
	upsertPoints := &qdrant.UpsertPoints{
		CollectionName: qdrantCollection,
		Points:         []*qdrant.PointStruct{point},
	}

	_, err := qdrantClient.Upsert(ctx, upsertPoints)
	if err != nil {
		return fmt.Errorf("Qdrant插入失败: %v", err)
	}

	// 存储映射关系到MongoDB，以便后续查询
	mapping := bson.M{
		"pointId":    pointID,
		"docId":      docID,
		"chunkIndex": chunkIndex,
		"createdAt":  time.Now(),
	}

	_, err = MongoCollection.InsertOne(ctx, mapping)
	if err != nil {
		fmt.Printf("警告: 存储点ID映射失败: %v\n", err)
		// 继续执行，不中断流程
	}

	fmt.Printf("成功将文档块 %d 存储到Qdrant，点ID: %d\n", chunkIndex, pointID)
	return nil
}

// 在Qdrant中搜索相似内容
func SearchQdrant(vector []float32) ([]string, error) {
	if qdrantClient == nil {
		return nil, fmt.Errorf("Qdrant客户端未初始化")
	}

	ctx := context.Background()

	// 设置相似度阈值 - 降低到0.2以获取更多结果
	scoreThreshold := float32(0.2)

	fmt.Printf("执行向量搜索，相似度阈值: %.2f\n", scoreThreshold)

	// 搜索请求
	searchReq := &qdrant.SearchPoints{
		CollectionName: qdrantCollection,
		Vector:         vector,
		Limit:          30, // 增加到30个结果
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Include{
				Include: &qdrant.PayloadIncludeSelector{
					Fields: []string{"text", "docId", "chunkIndex"},
				},
			},
		},
		ScoreThreshold: &scoreThreshold,
	}

	// 执行搜索
	searchResp, err := qdrantClient.Search(ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("Qdrant搜索失败: %v", err)
	}

	// 提取结果
	var results []string
	fmt.Printf("搜索结果数量: %d\n", len(searchResp.GetResult()))

	for i, point := range searchResp.GetResult() {
		score := point.GetScore()
		docId := "unknown"
		chunkIndex := -1

		if docIdVal, ok := point.GetPayload()["docId"]; ok {
			docId = docIdVal.GetStringValue()
		}

		if chunkIndexVal, ok := point.GetPayload()["chunkIndex"]; ok {
			chunkIndex = int(chunkIndexVal.GetIntegerValue())
		}

		fmt.Printf("结果 #%d: 相似度=%.4f, 文档ID=%s, 块索引=%d\n",
			i+1, score, docId, chunkIndex)

		if text, ok := point.GetPayload()["text"]; ok {
			textValue := text.GetStringValue()
			preview := textValue
			if len(textValue) > 100 {
				preview = textValue[:100] + "..."
			}
			fmt.Printf("  预览: %s\n", preview)
			results = append(results, textValue)
		}
	}

	if len(results) == 0 {
		fmt.Println("警告: 没有找到相关内容")
	} else {
		fmt.Printf("成功找到 %d 个相关内容\n", len(results))
	}

	return results, nil
}

// 生成回答
func GenerateResponse(results []string, query string) (string, error) {
	if OpenAIClient == nil {
		return "", fmt.Errorf("OpenAI客户端未初始化")
	}

	// 合并检索结果作为上下文，限制总长度并去除重复内容
	contextText := ""
	totalLength := 0
	maxContextLength := 6000
	uniqueTexts := make(map[string]bool)

	// 即使没有找到相关内容，也尝试使用模型自身的知识回答
	hasRelevantContext := len(results) > 0

	for _, text := range results {
		// 如果文本太短或已经包含在上下文中，跳过
		if len(text) < 50 || uniqueTexts[text[:50]] {
			continue
		}

		// 标记这个文本已处理
		uniqueTexts[text[:50]] = true

		// 检查是否超出长度限制
		if totalLength+len(text) > maxContextLength {
			break
		}

		if contextText != "" {
			contextText += "\n\n---\n\n"
		}
		contextText += text
		totalLength += len(text)
	}

	fmt.Printf("使用 %d 个字符的上下文生成回答\n", totalLength)

	// 构建提示模板，强调更好的格式和段落分隔
	var prompt string
	if hasRelevantContext {
		prompt = fmt.Sprintf(`I want you to answer the question "%s" by combining information from the provided context and your own knowledge.

Here is the context information from my knowledge base:
%s

Guidelines for your answer:
1. Prioritize information from the provided context when it's relevant to the question
2. Supplement with your own knowledge when the context is incomplete
3. Format your answer with CLEAR and DISTINCT paragraphs:
   - Use proper paragraph breaks between different topics or concepts
   - Add an empty line between paragraphs for better readability
   - Start each main section with a heading (e.g., "## Section Title")
4. When comparing different methods, approaches, or concepts:
   - Discuss each method in its own separate paragraph with proper spacing
   - Use clear headings to distinguish between them (e.g., "## Method 1" and "## Method 2")
   - Add empty lines between different methods/approaches
5. For lists and bullet points:
   - Each point should be on its own line
   - Use proper indentation and spacing
   - Group related points together
6. Explain technical concepts clearly
7. Maintain a professional, educational tone

Your answer should be well-structured with proper spacing and formatting to enhance readability.
IMPORTANT: Make sure to include empty lines between paragraphs and sections.`,
			query, contextText)
	} else {
		// 如果没有相关上下文，完全依赖模型自身的知识
		prompt = fmt.Sprintf(`Please answer the question "%s" based on your knowledge.

Guidelines for your answer:
1. Format your answer with CLEAR and DISTINCT paragraphs:
   - Use proper paragraph breaks between different topics or concepts
   - Add an empty line between paragraphs for better readability
   - Start each main section with a heading (e.g., "## Section Title")
2. When comparing different methods, approaches, or concepts:
   - Discuss each method in its own separate paragraph with proper spacing
   - Use clear headings to distinguish between them (e.g., "## Method 1" and "## Method 2")
   - Add empty lines between different methods/approaches
3. For lists and bullet points:
   - Each point should be on its own line
   - Use proper indentation and spacing
4. Explain technical concepts clearly
5. Maintain a professional, educational tone

IMPORTANT: Make sure to include empty lines between paragraphs and sections.
If you don't have information about this topic, please respond with "I'm sorry, I don't have enough information to answer this question."`,
			query)
	}

	// 尝试使用GPT-4模型（如果环境变量中设置了）
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		// 检查是否有GPT-4模型环境变量
		gpt4Model := os.Getenv("OPENAI_GPT4_MODEL")
		if gpt4Model != "" {
			model = gpt4Model
			fmt.Println("使用GPT-4模型生成回答")
		} else {
			model = "gpt-3.5-turbo" // 默认使用gpt-3.5-turbo
			fmt.Println("使用GPT-3.5模型生成回答")
		}
	}

	resp, err := OpenAIClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a technical expert that provides well-structured answers with excellent formatting. Pay special attention to paragraph breaks, section headings, and proper spacing between different concepts or methods being discussed. Your responses should be easy to read with clear visual separation between different topics. ALWAYS include empty lines between paragraphs and sections.",
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
			Temperature: 0.2,
			MaxTokens:   1500, // 增加token限制以允许更完整的回答
		},
	)

	if err != nil {
		return "", fmt.Errorf("OpenAI API调用失败: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI未返回任何回答")
	}

	// 确保回答中的段落之间有空行
	answer := resp.Choices[0].Message.Content

	// 替换单个换行为双换行，确保段落之间有空行
	answer = strings.ReplaceAll(answer, "\n\n", "DOUBLE_NEWLINE_PLACEHOLDER")
	answer = strings.ReplaceAll(answer, "\n", "\n\n")
	answer = strings.ReplaceAll(answer, "DOUBLE_NEWLINE_PLACEHOLDER", "\n\n")

	return answer, nil
}

// 检查文本是否已包含在上下文中
func containsText(context, text string) bool {
	// 如果文本长度超过100个字符，只检查前100个字符
	checkLength := 100
	if len(text) < checkLength {
		checkLength = len(text)
	}

	// 检查前checkLength个字符是否已包含在上下文中
	return strings.Contains(context, text[:checkLength])
}

// 清空Qdrant集合
func ClearQdrantCollection() error {
	if qdrantConn == nil {
		return fmt.Errorf("Qdrant连接未初始化")
	}

	ctx := context.Background()
	collectionsClient := qdrant.NewCollectionsClient(qdrantConn)

	// 删除集合
	_, err := collectionsClient.Delete(ctx, &qdrant.DeleteCollection{
		CollectionName: qdrantCollection,
	})

	if err != nil {
		return fmt.Errorf("删除Qdrant集合失败: %v", err)
	}

	fmt.Printf("已删除Qdrant集合: %s\n", qdrantCollection)

	// 重新创建集合
	_, err = collectionsClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: qdrantCollection,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     1536, // OpenAI嵌入维度
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("重新创建Qdrant集合失败: %v", err)
	}

	fmt.Printf("已重新创建Qdrant集合: %s\n", qdrantCollection)
	return nil
}
