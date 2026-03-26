package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"hotel-rag/internal/db"
	"hotel-rag/internal/llm"
)

// Retriever 检索接口
type Retriever interface {
	Search(ctx context.Context, query string, topK int) ([]db.SearchResult, error)
}

type Handler struct {
	db        *db.VectorDB
	retriever Retriever
	llm       llm.Client
	topK      int
}

func New(vectorDB *db.VectorDB, llmClient llm.Client, topK int) *Handler {
	return &Handler{db: vectorDB, llm: llmClient, topK: topK}
}

// WithRetriever 设置向量检索器
func (h *Handler) WithRetriever(r Retriever) {
	h.retriever = r
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/query", cors(h.query))
	mux.HandleFunc("/stats", cors(h.stats))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
}

func (h *Handler) query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Question == "" {
		http.Error(w, "请提供question字段", http.StatusBadRequest)
		return
	}

	// 检索：优先用向量检索，失败或未配置则使用关键词检索
	var results []db.SearchResult
	retrieverUsed := "keyword"

	if h.retriever != nil {
		var err error
		results, err = h.retriever.Search(r.Context(), req.Question, h.topK)
		if err != nil {
			fmt.Printf("向量检索失败，使用关键词检索: %v\n", err)
		} else {
			retrieverUsed = "vector"
		}
	}

	if retrieverUsed == "keyword" {
		results = h.db.Search(req.Question, h.topK)
	}

	context := h.db.BuildContext(results)
	summary := h.db.Summary()
	prompt := fmt.Sprintf("全局统计摘要：\n%s\n\n检索到的相关数据：\n%s\n\n用户问题：%s",
		summary, context, req.Question)

	// fmt.Printf("[prompt] %s\n", prompt)

	fmt.Printf("[retriever=%s] %s\n", retrieverUsed, req.Question)

	answer, err := h.llm.Ask(r.Context(), prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{
		"answer":  answer,
		"context": context,
	})
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"total_records": h.db.Count(),
		"hotels":        h.db.Hotels(),
		"brands":        h.db.Brands(),
		"status":        "ok",
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}
