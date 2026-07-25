package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledongthuc/pdf"
	pgvector "github.com/pgvector/pgvector-go"
)

type Document struct {
	ID           string `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"originalName"`
	Type         string `json:"type"`
	Size         int64  `json:"size"`
	Status       string `json:"status"`
	ChunkCount   int    `json:"chunkCount"`
	CreatedAt    string `json:"createdAt"`
}

type DocumentChunk struct {
	ID         string          `json:"id"`
	DocID      string          `json:"documentId"`
	ChunkIndex int             `json:"chunkIndex"`
	Content    string          `json:"content"`
	Embedding  []float32       `json:"-"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  string          `json:"createdAt"`
}

type VectorSearchResult struct {
	ChunkID   string  `json:"chunkId"`
	DocID     string  `json:"documentId"`
	Content   string  `json:"content"`
	Score     float64 `json:"score"`
	Filename  string  `json:"filename"`
}

var uploadDir = "./uploads"

func init() {
	os.MkdirAll(uploadDir, 0755)
}

func DocumentRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			r.ParseMultipartForm(50 << 20) // 50MB max

			file, handler, err := r.FormFile("file")
			if err != nil {
				http.Error(w, `{"error":"no file provided"}`, http.StatusBadRequest)
				return
			}
			defer file.Close()

			ext := strings.ToLower(filepath.Ext(handler.Filename))
			if ext != ".pdf" && ext != ".txt" && ext != ".md" && ext != ".docx" && ext != ".doc" {
				http.Error(w, `{"error":"unsupported file type. Allowed: pdf, docx, txt, md"}`, http.StatusBadRequest)
				return
			}

			// Save file
			filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
			savePath := filepath.Join(uploadDir, filename)
			dst, err := os.Create(savePath)
			if err != nil {
				http.Error(w, `{"error":"failed to save file"}`, http.StatusInternalServerError)
				return
			}
			content, _ := io.ReadAll(file)
			dst.Write(content)
			dst.Close()

			// Insert document record
			var docID string
			err = db.QueryRow(r.Context(),
				`INSERT INTO documents (filename, original_name, type, size, status, uploaded_by)
				 VALUES ($1, $2, $3, $4, 'processing', $5) RETURNING id`,
				filename, handler.Filename, ext, handler.Size, userID,
			).Scan(&docID)
			if err != nil {
				http.Error(w, `{"error":"failed to save document"}`, http.StatusInternalServerError)
				return
			}

			// Process in background: extract text, chunk, embed
			go processDocument(db, docID, savePath, ext)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Document{
				ID: docID, Filename: filename, OriginalName: handler.Filename,
				Type: ext, Size: handler.Size, Status: "processing",
			})
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			rows, err := db.Query(r.Context(),
				`SELECT d.id, d.filename, d.original_name, d.type, d.size, d.status,
				        COUNT(c.id)::int, d.created_at
				 FROM documents d LEFT JOIN document_chunks c ON c.document_id = d.id
				 WHERE d.uploaded_by = $1
				 GROUP BY d.id ORDER BY d.created_at DESC`, userID,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to list documents"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			docs := make([]Document, 0)
			for rows.Next() {
				var d Document
				if err := rows.Scan(&d.ID, &d.Filename, &d.OriginalName, &d.Type, &d.Size, &d.Status, &d.ChunkCount, &d.CreatedAt); err == nil {
					docs = append(docs, d)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(docs)
		})

		r.Get("/{id}/chunks", func(w http.ResponseWriter, r *http.Request) {
			docID := chi.URLParam(r, "id")
			rows, err := db.Query(r.Context(),
				`SELECT id, document_id, chunk_index, content, COALESCE(metadata, '{}'), created_at
				 FROM document_chunks WHERE document_id = $1 ORDER BY chunk_index`, docID,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to get chunks"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			chunks := make([]DocumentChunk, 0)
			for rows.Next() {
				var c DocumentChunk
				if err := rows.Scan(&c.ID, &c.DocID, &c.ChunkIndex, &c.Content, &c.Metadata, &c.CreatedAt); err == nil {
					chunks = append(chunks, c)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(chunks)
		})

		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			docID := chi.URLParam(r, "id")
			var filename string
			err := db.QueryRow(r.Context(), `SELECT filename FROM documents WHERE id = $1`, docID).Scan(&filename)
			if err != nil {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			db.Exec(r.Context(), `DELETE FROM documents WHERE id = $1`, docID)
			os.Remove(filepath.Join(uploadDir, filename))
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func processDocument(db *pgxpool.Pool, docID, filePath, ext string) {
	ctx := context.Background()
	text, err := extractText(filePath, ext)
	if err != nil {
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	chunks := chunkText(text, 500, 50)
	if len(chunks) == 0 {
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	embeddings, err := generateEmbeddings(db, chunks)
	if err != nil {
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	for i, chunk := range chunks {
		emb := pgvector.NewVector(embeddings[i])
		db.Exec(ctx,
			`INSERT INTO document_chunks (document_id, chunk_index, content, embedding) VALUES ($1, $2, $3, $4)`,
			docID, i, chunk, emb,
		)
	}

	db.Exec(ctx, `UPDATE documents SET status = 'indexed' WHERE id = $1`, docID)
}

func extractText(filePath, ext string) (string, error) {
	switch ext {
	case ".txt", ".md":
		data, err := os.ReadFile(filePath)
		return string(data), err
	case ".pdf":
		return extractPDFText(filePath)
	case ".docx":
		return extractDOCXText(filePath)
	}
	return "", fmt.Errorf("unsupported type: %s", ext)
}

func extractPDFText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func extractDOCXText(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			data, _ := io.ReadAll(rc)
			// Simple XML tag stripping
			text := string(data)
			text = strings.ReplaceAll(text, "</w:p>", "\n")
			text = strings.ReplaceAll(text, "</w:r>", "")
			text = strings.ReplaceAll(text, "</w:t>", "")
			// Strip all XML tags
			for {
				start := strings.Index(text, "<")
				if start == -1 {
					break
				}
				end := strings.Index(text[start:], ">")
				if end == -1 {
					break
				}
				text = text[:start] + text[start+end+1:]
			}
			return strings.TrimSpace(text), nil
		}
	}
	return "", fmt.Errorf("word/document.xml not found")
}

func chunkText(text string, chunkSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return nil
	}

	runes := []rune(text)
	var chunks []string

	for i := 0; i < len(runes); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		chunk = strings.TrimSpace(chunk)
		if len(chunk) > 20 {
			chunks = append(chunks, chunk)
		}
		if end >= len(runes) {
			break
		}
	}
	return chunks
}

func generateEmbeddings(db *pgxpool.Pool, texts []string) ([][]float32, error) {
	apiKey := getEmbeddingAPIKey(db)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key for embeddings")
	}

	reqBody := map[string]interface{}{
		"model": "text-embedding-3-small",
		"input": texts,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("invalid embedding response")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("embedding error: %s", result.Error.Message)
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}
	return embeddings, nil
}

func getOpenAIKey() string {
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		return v
	}
	return ""
}

func getEmbeddingAPIKey(db *pgxpool.Pool) string {
	keyName := getSetting(db, "AI_API_KEY_NAME")
	if keyName == "" {
		keyName = "OPENAI_API_KEY"
	}
	if v := getSetting(db, keyName); v != "" {
		return v
	}
	return getOpenAIKey()
}

func VectorSearch(db *pgxpool.Pool, query string, topK int) ([]VectorSearchResult, error) {
	apiKey := getEmbeddingAPIKey(db)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key for embeddings")
	}

	// Embed the query
	queryEmb, err := generateEmbeddings(db, []string{query})
	if err != nil {
		return nil, err
	}
	if len(queryEmb) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}

	emb := pgvector.NewVector(queryEmb[0])
	vec := fmt.Sprintf("[%s]", floatsToString(queryEmb[0]))

	rows, err := db.Query(context.Background(),
		`SELECT c.id, c.document_id, c.content,
		        1 - (c.embedding <=> $1::vector) AS score,
		        d.original_name
		 FROM document_chunks c
		 JOIN documents d ON d.id = c.document_id
		 ORDER BY c.embedding <=> $1::vector
		 LIMIT $2`, vec, topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	_ = emb // used for type checking
	results := make([]VectorSearchResult, 0)
	for rows.Next() {
		var r VectorSearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocID, &r.Content, &r.Score, &r.Filename); err == nil {
			results = append(results, r)
		}
	}
	return results, nil
}

func floatsToString(fs []float32) string {
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = fmt.Sprintf("%f", f)
	}
	return strings.Join(parts, ",")
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
