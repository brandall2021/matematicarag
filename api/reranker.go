package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HybridResult struct {
	ChunkID      string  `json:"chunk_id"`
	DocID        string  `json:"document_id"`
	Content      string  `json:"content"`
	Filename     string  `json:"filename"`
	Page         int     `json:"page"`
	Section      string  `json:"section"`
	URL          string  `json:"url"`
	VectorScore  float64 `json:"vector_score"`
	KeywordScore float64 `json:"keyword_score"`
	HybridScore  float64 `json:"hybrid_score"`
	RerankScore  float64 `json:"rerank_score"`
}

// HybridSearch combines vector search from Qdrant with keyword search from PostgreSQL
func HybridSearch(db *pgxpool.Pool, query string, topK int, filters map[string]interface{}, vectorWeight, keywordWeight float64) ([]HybridResult, error) {
	// 1. Vector search — retrieve more results for better fusion
	vectorTopK := topK * 4
	queryEmb, err := generateEmbeddings(db, []string{query})
	if err != nil {
		log.Printf("[HYBRID] embedding error: %v", err)
		// Fall back to keyword-only
		return hybridFromKeywordOnly(db, query, topK, filters)
	}
	if len(queryEmb) == 0 {
		return hybridFromKeywordOnly(db, query, topK, filters)
	}

	vectorResults, err := qdrantSearchWithFilters(queryEmb[0], vectorTopK, filters)
	if err != nil {
		log.Printf("[HYBRID] vector search error: %v", err)
		return hybridFromKeywordOnly(db, query, topK, filters)
	}

	// 2. Keyword search
	keywordTopK := topK * 4
	keywordResults, err := KeywordSearch(db, query, keywordTopK, filters)
	if err != nil {
		log.Printf("[HYBRID] keyword search error: %v", err)
		// Fall back to vector-only
		return hybridFromVectorOnly(vectorResults)
	}

	// 3. Normalize scores
	keywordResults = NormalizeTextSearchScores(keywordResults)

	// 4. Merge results by chunk_id, compute hybrid score
	merged := make(map[string]*HybridResult)

	for _, vr := range vectorResults {
		chunkID := vr.ID
		merged[chunkID] = &HybridResult{
			ChunkID:     chunkID,
			DocID:       fmt.Sprintf("%v", vr.Payload["document_id"]),
			Content:     fmt.Sprintf("%v", vr.Payload["content"]),
			Filename:    fmt.Sprintf("%v", vr.Payload["filename"]),
			VectorScore: vr.Score,
		}
		if v, ok := vr.Payload["page"].(float64); ok {
			merged[chunkID].Page = int(v)
		}
		if v, ok := vr.Payload["section"].(string); ok {
			merged[chunkID].Section = v
		}
		if v, ok := vr.Payload["url"].(string); ok {
			merged[chunkID].URL = v
		}
	}

	for _, kr := range keywordResults {
		if existing, ok := merged[kr.ChunkID]; ok {
			existing.KeywordScore = kr.Score
		} else {
			merged[kr.ChunkID] = &HybridResult{
				ChunkID:      kr.ChunkID,
				DocID:        kr.DocID,
				Content:      kr.Content,
				Filename:     kr.Filename,
				Page:         kr.Page,
				Section:      kr.Section,
				URL:          kr.URL,
				KeywordScore: kr.Score,
			}
		}
	}

	// 5. Compute hybrid scores
	results := make([]HybridResult, 0, len(merged))
	for _, r := range merged {
		r.HybridScore = r.VectorScore*vectorWeight + r.KeywordScore*keywordWeight
		results = append(results, *r)
	}

	// 6. Sort by hybrid_score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].HybridScore > results[j].HybridScore
	})

	// 7. Return top results (before reranking)
	if len(results) > topK*2 {
		results = results[:topK*2]
	}

	log.Printf("[HYBRID] query=%s, vector=%d, keyword=%d, merged=%d", query, len(vectorResults), len(keywordResults), len(results))
	return results, nil
}

func hybridFromVectorOnly(vectorResults []QdrantSearchResult) ([]HybridResult, error) {
	results := make([]HybridResult, 0, len(vectorResults))
	for _, vr := range vectorResults {
		r := HybridResult{
			ChunkID:     vr.ID,
			DocID:       fmt.Sprintf("%v", vr.Payload["document_id"]),
			Content:     fmt.Sprintf("%v", vr.Payload["content"]),
			Filename:    fmt.Sprintf("%v", vr.Payload["filename"]),
			VectorScore: vr.Score,
			HybridScore: vr.Score,
		}
		if v, ok := vr.Payload["page"].(float64); ok {
			r.Page = int(v)
		}
		if v, ok := vr.Payload["section"].(string); ok {
			r.Section = v
		}
		if v, ok := vr.Payload["url"].(string); ok {
			r.URL = v
		}
		results = append(results, r)
	}
	return results, nil
}

func hybridFromKeywordOnly(db *pgxpool.Pool, query string, topK int, filters map[string]interface{}) ([]HybridResult, error) {
	keywordResults, err := KeywordSearch(db, query, topK, filters)
	if err != nil {
		return nil, err
	}
	keywordResults = NormalizeTextSearchScores(keywordResults)
	results := make([]HybridResult, 0, len(keywordResults))
	for _, kr := range keywordResults {
		results = append(results, HybridResult{
			ChunkID:      kr.ChunkID,
			DocID:        kr.DocID,
			Content:      kr.Content,
			Filename:     kr.Filename,
			Page:         kr.Page,
			Section:      kr.Section,
			URL:          kr.URL,
			KeywordScore: kr.Score,
			HybridScore:  kr.Score,
		})
	}
	return results, nil
}

// RerankResults uses LLM to evaluate relevance and return top K
func RerankResults(db *pgxpool.Pool, query string, results []HybridResult, topK int) ([]HybridResult, error) {
	if len(results) == 0 {
		return results, nil
	}
	if len(results) <= topK {
		return results, nil
	}

	// Build reranking prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pregunta: %s\n\n", query))
	sb.WriteString("Fragmentos:\n")
	for i, r := range results {
		content := r.Content
		if len(content) > 300 {
			content = content[:300] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, content))
	}
	sb.WriteString(fmt.Sprintf(`Responde SOLO con un JSON: {"scores": [%d numeros del 0 al 10]}`, len(results)))

	apiKey := getAPIKey(db)
	if apiKey == "" {
		// No API key — use hybrid score as rerank score
		for i := range results {
			results[i].RerankScore = results[i].HybridScore * 10
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].RerankScore > results[j].RerankScore
		})
		if len(results) > topK {
			results = results[:topK]
		}
		return results, nil
	}

	model := getModel(db, "")
	messages := []OpenAIMessage{
		{Role: "system", Content: "Eres un evaluador de relevancia. Asignas puntajes del 0 al 10 basado en cuán relevante es cada fragmento para responder la pregunta. Respondes SOLO con JSON."},
		{Role: "user", Content: sb.String()},
	}

	response, err := callOpenAIWithHistory(db, messages, model)
	if err != nil {
		log.Printf("[RERANK] LLM error: %v, using hybrid scores", err)
		for i := range results {
			results[i].RerankScore = results[i].HybridScore * 10
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].RerankScore > results[j].RerankScore
		})
		if len(results) > topK {
			results = results[:topK]
		}
		return results, nil
	}

	// Parse scores from response
	scores := parseRerankScores(response, len(results))

	// Apply rerank scores
	for i := range results {
		if i < len(scores) {
			results[i].RerankScore = scores[i]
		} else {
			results[i].RerankScore = 0
		}
	}

	// Sort by rerank_score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].RerankScore > results[j].RerankScore
	})

	// Return top K
	if len(results) > topK {
		results = results[:topK]
	}

	log.Printf("[RERANK] query=%s, reranked %d → %d results", query, len(results), topK)
	return results, nil
}

func parseRerankScores(response string, expected int) []float64 {
	scores := make([]float64, expected)
	// Try to find JSON in response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 {
		// Try to parse as plain numbers
		parts := strings.Fields(response)
		for i, p := range parts {
			if i >= expected {
				break
			}
			var s float64
			fmt.Sscanf(p, "%f", &s)
			scores[i] = s
		}
		return scores
	}

	var result struct {
		Scores []float64 `json:"scores"`
	}
	if err := json.Unmarshal([]byte(response[start:end+1]), &result); err != nil {
		return scores
	}
	for i, s := range result.Scores {
		if i >= expected {
			break
		}
		scores[i] = math.Max(0, math.Min(10, s))
	}
	return scores
}
