package db

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type HotelRecord struct {
	HotelID   string
	HotelName string
	Date      string
	Occupancy float64
	Brand     string
	District  string
}

type SearchResult struct {
	Record HotelRecord
	Score  float64
}

type VectorDB struct {
	records []HotelRecord
	summary string // Python预计算的统计摘要，原始json字符串
}

// LoadCSV 加载CSV到内存
func LoadCSV(path string) (*VectorDB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开CSV失败: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReader(f))
	headers, err := r.Read()
	if err != nil {
		return nil, err
	}

	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	col := func(row []string, names ...string) string {
		for _, name := range names {
			if i, ok := idx[name]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
		}
		return ""
	}

	var records []HotelRecord
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		occ, _ := strconv.ParseFloat(col(row, "occupancy", "occupancy_rate", "入住率"), 64)
		records = append(records, HotelRecord{
			HotelID:   col(row, "hotel_id", "hotel", "酒店id"),
			HotelName: col(row, "hotel_name", "name", "酒店名", "酒店名称"),
			Date:      col(row, "date", "日期", "ds"),
			Occupancy: occ,
			Brand:     col(row, "brand_tier", "brand", "品牌"),
			District:  col(row, "district", "区域"),
		})
	}

	return &VectorDB{records: records}, nil
}

// LoadSummary 读取Python生成的summary.json
func (db *VectorDB) LoadSummary(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取summary.json失败: %w", err)
	}
	db.summary = string(data)
	return nil
}

// Summary 返回Python预计算的统计摘要
func (db *VectorDB) Summary() string {
	if db.summary == "" {
		return "统计摘要未加载，请先运行 python scripts/gen_data.py 生成 summary.json"
	}
	return db.summary
}

func (db *VectorDB) Count() int { return len(db.records) }

func (db *VectorDB) Hotels() int {
	m := make(map[string]struct{})
	for _, r := range db.records {
		m[r.HotelID] = struct{}{}
	}
	return len(m)
}

func (db *VectorDB) Brands() int {
	m := make(map[string]struct{})
	for _, r := range db.records {
		m[r.Brand] = struct{}{}
	}
	return len(m)
}

// 中文关键词映射
var keywordMap = map[string]string{
	"cbd":  "cbd",
	"市中心":  "cbd",
	"郊区":   "suburban",
	"交通枢纽": "transport_hub",
	"经济型":  "economy",
	"中档":   "midscale",
	"高端":   "premium",
	"区域":   "",
	"酒店":   "",
	"入住率":  "",
}

func normalizeKeywords(query string) []string {
	query = strings.ToLower(query)
	for zh, en := range keywordMap {
		if strings.Contains(query, zh) {
			if en != "" {
				query = strings.ReplaceAll(query, zh, " "+en+" ")
			} else {
				query = strings.ReplaceAll(query, zh, " ")
			}
		}
	}
	result := make([]string, 0)
	for _, w := range strings.Fields(query) {
		if w != "" {
			result = append(result, w)
		}
	}
	return result
}

// Search 关键词检索，返回topK条最相关记录
func (db *VectorDB) Search(query string, topK int) []SearchResult {
	keywords := normalizeKeywords(query)

	results := make([]SearchResult, 0, topK)
	for _, rec := range db.records {
		text := strings.ToLower(fmt.Sprintf("%s %s %s %s %s %.2f",
			rec.HotelID, rec.HotelName, rec.Date, rec.Brand, rec.District, rec.Occupancy))
		coreFields := strings.ToLower(rec.District + " " + rec.Brand + " " + rec.HotelID + " " + rec.HotelName)

		score := 0.0
		for _, kw := range keywords {
			if strings.Contains(coreFields, kw) {
				score += 3.0
			} else if strings.Contains(text, kw) {
				score += 1.0
			}
		}
		if score > 0 {
			results = append(results, SearchResult{rec, score})
		}
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

// BuildContext 把检索结果格式化成文字
func (db *VectorDB) BuildContext(results []SearchResult) string {
	if len(results) == 0 {
		return "未找到相关记录。"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("检索到 %d 条相关记录：\n", len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s | %s | 入住率:%.1f%% | %s | %s\n",
			i+1, r.Record.HotelID, r.Record.HotelName, r.Record.Date,
			r.Record.Occupancy*100, r.Record.Brand, r.Record.District))
	}
	return sb.String()
}
