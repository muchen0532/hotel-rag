package main

import (
	"fmt"
	"log"
	"net/http"

	"hotel-rag/internal/config"
	"hotel-rag/internal/db"
	"hotel-rag/internal/handler"
	"hotel-rag/internal/llm"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	vectorDB, err := db.LoadCSV(cfg.Data.CSVPath)
	if err != nil {
		log.Fatalf("加载数据失败: %v", err)
	}
	if err := vectorDB.LoadSummary(cfg.Data.SummaryPath); err != nil {
		log.Printf("警告: %v", err)
	}
	log.Printf("数据加载完成: %d 条记录，%d 家酒店", vectorDB.Count(), vectorDB.Hotels())

	llmClient, err := llm.NewClient(&cfg.LLM)
	if err != nil {
		log.Fatalf("初始化LLM失败: %v", err)
	}

	h := handler.New(vectorDB, llmClient, cfg.Data.TopK)

	// 初始化 Qdrant 向量检索
	if cfg.Qdrant.JinaAPIKey != "" {
		qdrantRetriever, err := db.NewQdrantRetriever(db.QdrantConfig{
			QdrantURL:  cfg.Qdrant.URL,
			Collection: cfg.Qdrant.Collection,
			JinaAPIKey: cfg.Qdrant.JinaAPIKey,
		})
		if err != nil {
			log.Printf("警告: 初始化Qdrant失败，使用关键词检索: %v", err)
		} else {
			h.WithRetriever(qdrantRetriever)
			log.Printf("向量检索已启用: %s / %s", cfg.Qdrant.URL, cfg.Qdrant.Collection)
		}
	} else {
		log.Printf("未配置 Qdrant，使用关键词检索")
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("服务启动: http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
