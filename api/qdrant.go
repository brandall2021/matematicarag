package api

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	qdrantURL    = "http://qdrant:6333"
	qdrantAPIKey = ""
	collection   = "matematica_chunks"
	qdrantClient = &http.Client{Timeout: 30 * time.Second}
)

func init() {
	if v := os.Getenv("QDRANT_URL"); v != "" {
		qdrantURL = v
	} else {
		host := os.Getenv("QDRANT_HOST")
		port := os.Getenv("QDRANT_PORT")
		if host != "" {
			if port == "" {
				port = "6333"
			}
			qdrantURL = fmt.Sprintf("http://%s:%s", host, port)
		}
	}
	if v := os.Getenv("QDRANT_API_KEY"); v != "" {
		qdrantAPIKey = v
	}
	// Try to ensure collection on startup (non-blocking)
	go func() {
		time.Sleep(2 * time.Second)
		if err := ensureQdrantCollection(); err != nil {
			fmt.Printf("[QDRANT] Collection init failed: %v\n", err)
		} else {
			fmt.Printf("[QDRANT] Collection '%s' ready at %s\n", collection, qdrantURL)
			if err := ensureQdrantIndices(); err != nil {
				fmt.Printf("[QDRANT] Index creation failed: %v\n", err)
			}
		}
	}()
}

type QdrantPoint struct {
	ID       string                 `json:"id"`
	Vector   []float32              `json:"vector"`
	Payload  map[string]interface{} `json:"payload"`
}

type QdrantUpsertRequest struct {
	Points []QdrantPoint `json:"points"`
}

type QdrantSearchRequest struct {
	Vector      []float32 `json:"vector"`
	TopK        int       `json:"top"`
	WithPayload bool      `json:"with_payload"`
}

type QdrantSearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Payload  map[string]interface{} `json:"payload"`
}

type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
}

type QdrantCollectionInfo struct {
	Result struct {
		Status string `json:"status"`
	} `json:"result"`
}

type ChunkPayload struct {
	DocumentID   string
	DocumentName string
	ChunkIndex   int
	Content      string
	Page         int
	Section      string
	Topic        string
	ContentType  string
	HasFormula   bool
	HasExample   bool
	HasExercise  bool
	HasSolution  bool
	CourseID     string
	UnitID       string
	URL          string
}

func generateChunkID(docID string, chunkIndex int) string {
	h := md5.Sum([]byte(fmt.Sprintf("%s_%d", docID, chunkIndex)))
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:])
}

func qdrantRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, qdrantURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if qdrantAPIKey != "" {
		req.Header.Set("api-key", qdrantAPIKey)
	}

	resp, err := qdrantClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant connection error: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func QdrantHealthCheck() ([]byte, error) {
	body, err := qdrantRequest("GET", "/collections/"+collection, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func ensureQdrantCollection() error {
	// Check if collection exists
	_, err := qdrantRequest("GET", "/collections/"+collection, nil)
	if err == nil {
		return nil // already exists
	}

	// Create collection
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     1536,
			"distance": "Cosine",
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("PUT", qdrantURL+"/collections/"+collection, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if qdrantAPIKey != "" {
		req.Header.Set("api-key", qdrantAPIKey)
	}

	resp, err := qdrantClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create qdrant collection: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant create collection error (%d): %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func ensureQdrantIndices() error {
	indices := []map[string]interface{}{
		{"field_name": "document_id", "field_schema": "keyword"},
		{"field_name": "course_id", "field_schema": "keyword"},
		{"field_name": "unit_id", "field_schema": "keyword"},
		{"field_name": "topic", "field_schema": "keyword"},
		{"field_name": "content_type", "field_schema": "keyword"},
		{"field_name": "page", "field_schema": "integer"},
	}
	for _, idx := range indices {
		_, err := qdrantRequest("PUT", "/collections/"+collection+"/index", idx)
		if err != nil {
			log.Printf("[QDRANT] index creation warning for %s: %v", idx["field_name"], err)
		}
	}
	return nil
}

func qdrantUpsert(docID, chunkID string, chunkIndex int, embedding []float32, meta ChunkPayload) error {
	payload := map[string]interface{}{
		"document_id":   meta.DocumentID,
		"document_name": meta.DocumentName,
		"chunk_index":   meta.ChunkIndex,
		"content":       meta.Content,
		"content_type":  meta.ContentType,
	}
	if meta.Page > 0 {
		payload["page"] = meta.Page
	}
	if meta.Section != "" {
		payload["section"] = meta.Section
	}
	if meta.Topic != "" {
		payload["topic"] = meta.Topic
	}
	if meta.CourseID != "" {
		payload["course_id"] = meta.CourseID
	}
	if meta.UnitID != "" {
		payload["unit_id"] = meta.UnitID
	}
	if meta.URL != "" {
		payload["url"] = meta.URL
	}
	if meta.HasFormula {
		payload["has_formula"] = true
	}
	if meta.HasExample {
		payload["has_example"] = true
	}
	if meta.HasExercise {
		payload["has_exercise"] = true
	}
	if meta.HasSolution {
		payload["has_solution"] = true
	}

	point := QdrantPoint{
		ID:      chunkID,
		Vector:  embedding,
		Payload: payload,
	}

	req := QdrantUpsertRequest{
		Points: []QdrantPoint{point},
	}

	_, err := qdrantRequest("PUT", "/collections/"+collection+"/points", req)
	return err
}

func qdrantSearch(queryEmbedding []float32, topK int) ([]QdrantSearchResult, error) {
	req := QdrantSearchRequest{
		Vector:      queryEmbedding,
		TopK:        topK,
		WithPayload: true,
	}

	body, err := qdrantRequest("POST", "/collections/"+collection+"/points/search", req)
	if err != nil {
		return nil, err
	}

	var resp QdrantSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func qdrantSearchWithFilters(queryEmbedding []float32, topK int, filters map[string]interface{}) ([]QdrantSearchResult, error) {
	searchReq := map[string]interface{}{
		"vector":       queryEmbedding,
		"top":          topK,
		"with_payload": true,
	}

	if len(filters) > 0 {
		var must []map[string]interface{}
		for key, value := range filters {
			if value != nil && value != "" {
				must = append(must, map[string]interface{}{
					"key": key,
					"match": map[string]interface{}{
						"value": value,
					},
				})
			}
		}
		if len(must) > 0 {
			searchReq["filter"] = map[string]interface{}{
				"must": must,
			}
		}
	}

	body, err := qdrantRequest("POST", "/collections/"+collection+"/points/search", searchReq)
	if err != nil {
		return nil, err
	}

	var resp QdrantSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func qdrantDeleteByDocID(docID string) error {
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "document_id",
					"match": map[string]interface{}{
						"value": docID,
					},
				},
			},
		},
	}
	_, err := qdrantRequest("POST", "/collections/"+collection+"/points/delete", body)
	return err
}

func qdrantCountByDocID(docID string) (int, error) {
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "document_id",
					"match": map[string]interface{}{
						"value": docID,
					},
				},
			},
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", qdrantURL+"/collections/"+collection+"/points/count", bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if qdrantAPIKey != "" {
		req.Header.Set("api-key", qdrantAPIKey)
	}

	resp, err := qdrantClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Result int `json:"result"`
	}
	json.Unmarshal(respBody, &result)
	return result.Result, nil
}

func qdrantSearchByDocID(docID string, limit int) ([]QdrantSearchResult, error) {
	// Use a dummy vector with filter to get all chunks for a document
	// Qdrant requires a vector for search, so we use a zero vector with high limit
	dummyVector := make([]float32, 1536)
	dummyVector[0] = 1.0 // non-zero to avoid issues

	req := map[string]interface{}{
		"vector":      dummyVector,
		"top":         limit,
		"with_payload": true,
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "document_id",
					"match": map[string]interface{}{
						"value": docID,
					},
				},
			},
		},
	}

	body, err := qdrantRequest("POST", "/collections/"+collection+"/points/search", req)
	if err != nil {
		return nil, err
	}

	var resp QdrantSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Result, nil
}
