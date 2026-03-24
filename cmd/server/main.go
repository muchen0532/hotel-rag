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

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("服务启动: http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
