package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	qdrantURL    = "http://sistemas-qdrant-gjlncs:6333"
	qdrantAPIKey = "0ylktnefcidr4f6dvkmwfoxc4nrgtywh"
	collection   = "matematica_chunks"
)

func init() {
	if v := os.Getenv("QDRANT_URL"); v != "" {
		qdrantURL = v
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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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

func qdrantUpsert(docID string, chunkIndex int, chunkID string, embedding []float32, content string, filename string) error {
	point := QdrantPoint{
		ID:     chunkID,
		Vector: embedding,
		Payload: map[string]interface{}{
			"document_id":  docID,
			"chunk_index":  chunkIndex,
			"content":      content,
			"filename":     filename,
		},
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

	resp, err := http.DefaultClient.Do(req)
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
