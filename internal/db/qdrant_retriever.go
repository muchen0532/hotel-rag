package db

// qdrant_retriever.go
// 实现 handler.Retriever 接口，用 Jina Embedding + Qdrant 做语义检索

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type QdrantConfig struct {
	QdrantURL      string
	Collection     string
	JinaAPIKey     string
	EmbeddingModel string
	TopK           int
}

type QdrantRetriever struct {
	cfg    QdrantConfig
	client *http.Client
}

func NewQdrantRetriever(cfg QdrantConfig) (*QdrantRetriever, error) {
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "jina-embeddings-v3"
	}
	if cfg.TopK == 0 {
		cfg.TopK = 5
	}
	return &QdrantRetriever{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (r *QdrantRetriever) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = r.cfg.TopK
	}

	vec, err := r.embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	return r.searchQdrant(ctx, vec, topK)
}

type jinaEmbedReq struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	Task  string   `json:"task"`
}

type jinaEmbedResp struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Detail string `json:"detail"`
}

func (r *QdrantRetriever) embed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(jinaEmbedReq{
		Model: r.cfg.EmbeddingModel,
		Input: []string{text},
		Task:  "retrieval.query",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.jina.ai/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.JinaAPIKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result jinaEmbedResp
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse embed response: %w", err)
	}
	if result.Detail != "" {
		return nil, fmt.Errorf("jina error: %s", result.Detail)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return result.Data[0].Embedding, nil
}

type qdrantSearchReq struct {
	Vector      []float64 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
}

type qdrantSearchResp struct {
	Result []struct {
		Score   float64 `json:"score"`
		Payload struct {
			HotelID   string  `json:"hotel_id"`
			HotelName string  `json:"hotel_name"`
			Date      string  `json:"date"`
			Occupancy float64 `json:"occupancy"`
			BrandTier string  `json:"brand_tier"`
			District  string  `json:"district"`
		} `json:"payload"`
	} `json:"result"`
}

func (r *QdrantRetriever) searchQdrant(ctx context.Context, vec []float64, topK int) ([]SearchResult, error) {
	reqBody, _ := json.Marshal(qdrantSearchReq{
		Vector:      vec,
		Limit:       topK,
		WithPayload: true,
	})

	url := fmt.Sprintf("%s/collections/%s/points/search", r.cfg.QdrantURL, r.cfg.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant status %d: %s", resp.StatusCode, string(raw))
	}

	var result qdrantSearchResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse qdrant response: %w", err)
	}

	records := make([]SearchResult, 0, len(result.Result))
	for _, hit := range result.Result {
		records = append(records, SearchResult{
			Record: HotelRecord{
				HotelID:   hit.Payload.HotelID,
				HotelName: hit.Payload.HotelName,
				Date:      hit.Payload.Date,
				Occupancy: hit.Payload.Occupancy,
				Brand:     hit.Payload.BrandTier,
				District:  hit.Payload.District,
			},
			Score: hit.Score,
		})
	}
	return records, nil
}
