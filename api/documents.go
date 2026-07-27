package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledongthuc/pdf"
)

type Document struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"originalName"`
	Type         string    `json:"type"`
	Size         int64     `json:"size"`
	Status       string    `json:"status"`
	ChunkCount   int       `json:"chunkCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

type DocumentChunk struct {
	ID         string `json:"id"`
	DocID      string `json:"documentId"`
	ChunkIndex int    `json:"chunkIndex"`
	Content    string `json:"content"`
	Filename   string `json:"filename,omitempty"`
	Page       int    `json:"page,omitempty"`
	Section    string `json:"section,omitempty"`
}

type VectorSearchResult struct {
	ChunkID  string  `json:"chunkId"`
	DocID    string  `json:"documentId"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Filename string  `json:"filename"`
	Page     int     `json:"page"`
	Section  string  `json:"section"`
	URL      string  `json:"url"`
}

type PageContent struct {
	Page int
	Text string
}

type ChunkMeta struct {
	Text        string
	Page        int
	Section     string
	Topic       string
	ContentType string
	HasFormula  bool
	HasExample  bool
	HasExercise bool
	HasSolution bool
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
			r.ParseMultipartForm(50 << 20)

			file, handler, err := r.FormFile("file")
			if err != nil {
				http.Error(w, `{"error":"no file provided"}`, http.StatusBadRequest)
				return
			}
			defer file.Close()

			ext := strings.ToLower(filepath.Ext(handler.Filename))
			if ext != ".pdf" && ext != ".txt" && ext != ".md" && ext != ".docx" && ext != ".doc" {
				http.Error(w, `{"error":"tipo no soportado. Permitidos: pdf, docx, txt, md"}`, http.StatusBadRequest)
				return
			}

			filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
			savePath := filepath.Join(uploadDir, filename)
			dst, err := os.Create(savePath)
			if err != nil {
				http.Error(w, `{"error":"error al guardar archivo"}`, http.StatusInternalServerError)
				return
			}
			content, _ := io.ReadAll(file)
			dst.Write(content)
			dst.Close()

			var docID string
			err = db.QueryRow(r.Context(),
				`INSERT INTO documents (filename, original_name, type, size, status, uploaded_by)
				 VALUES ($1, $2, $3, $4, 'processing', $5) RETURNING id`,
				filename, handler.Filename, ext, handler.Size, userID,
			).Scan(&docID)
			if err != nil {
				http.Error(w, `{"error":"error al guardar registro"}`, http.StatusInternalServerError)
				return
			}

			go processDocument(db, docID, savePath, ext, handler.Filename)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Document{
				ID: docID, Filename: filename, OriginalName: handler.Filename,
				Type: ext, Size: handler.Size, Status: "processing",
			})
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			rows, err := db.Query(r.Context(),
				`SELECT id, filename, original_name, type, size, status, created_at
				 FROM documents WHERE uploaded_by = $1 ORDER BY created_at DESC`, userID,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to list documents"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
	docs := make([]Document, 0)
		for rows.Next() {
			var d Document
			if err := rows.Scan(&d.ID, &d.Filename, &d.OriginalName, &d.Type, &d.Size, &d.Status, &d.CreatedAt); err != nil {
				log.Printf("[DOCS] scan error: %v", err)
				continue
			}
			if d.Status == "indexed" {
				count, _ := qdrantCountByDocID(d.ID)
				d.ChunkCount = count
			}
			docs = append(docs, d)
		}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(docs)
		})

		r.Get("/{id}/chunks", func(w http.ResponseWriter, r *http.Request) {
			docID := chi.URLParam(r, "id")

			results, err := qdrantSearchByDocID(docID, 500)
			if err != nil {
				http.Error(w, `{"error":"failed to get chunks"}`, http.StatusInternalServerError)
				return
			}

			chunks := make([]DocumentChunk, len(results))
			for i, res := range results {
				chunks[i] = DocumentChunk{
					ID:         res.ID,
					DocID:      fmt.Sprintf("%v", res.Payload["document_id"]),
					ChunkIndex: int(res.Payload["chunk_index"].(float64)),
					Content:    res.Payload["content"].(string),
					Filename:   fmt.Sprintf("%v", res.Payload["filename"]),
				}
				if v, ok := res.Payload["page"].(float64); ok {
					chunks[i].Page = int(v)
				}
				if v, ok := res.Payload["section"].(string); ok {
					chunks[i].Section = v
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
			db.Exec(r.Context(), `DELETE FROM document_chunks WHERE document_id = $1`, docID)
			qdrantDeleteByDocID(docID)
			db.Exec(r.Context(), `DELETE FROM documents WHERE id = $1`, docID)
			os.Remove(filepath.Join(uploadDir, filename))
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func processDocument(db *pgxpool.Pool, docID, filePath, ext, originalName string) {
	ctx := context.Background()
	log.Printf("[DOCS] processing document %s (%s)", docID, originalName)

	pages, err := extractTextWithMetadata(filePath, ext)
	if err != nil {
		log.Printf("[DOCS] extract error for %s: %v", docID, err)
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	chunks := chunkTextWithMetadata(pages, 500, 50)
	if len(chunks) == 0 {
		log.Printf("[DOCS] no chunks for %s", docID)
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	// Insert chunks into document_chunks for text search
	for i, chunk := range chunks {
		_, err := db.Exec(ctx,
			`INSERT INTO document_chunks (document_id, chunk_index, content, page, section, topic, content_type, has_formula, has_example, has_exercise, has_solution)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			docID, i, chunk.Text, chunk.Page, chunk.Section, chunk.Topic, chunk.ContentType, chunk.HasFormula, chunk.HasExample, chunk.HasExercise, chunk.HasSolution,
		)
		if err != nil {
			log.Printf("[DOCS] chunk insert error for %s chunk %d: %v", docID, i, err)
		}
	}

	if err := ensureQdrantCollection(); err != nil {
		log.Printf("[DOCS] qdrant collection error for %s: %v", docID, err)
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}

	embeddings, err := generateEmbeddings(db, texts)
	if err != nil {
		log.Printf("[DOCS] embedding error for %s: %v", docID, err)
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	docURL := fmt.Sprintf("/documents?id=%s", docID)

	allOK := true
	for i, chunk := range chunks {
		chunkID := generateChunkID(docID, i)
		if err := qdrantUpsert(docID, i, chunkID, embeddings[i], chunk.Text, originalName, chunk.Page, chunk.Section, docURL); err != nil {
			log.Printf("[DOCS] qdrant upsert error for %s chunk %d: %v", docID, i, err)
			allOK = false
		}
	}

	if allOK {
		db.Exec(ctx, `UPDATE documents SET status = 'indexed' WHERE id = $1`, docID)
		log.Printf("[DOCS] document %s indexed successfully (%d chunks)", docID, len(chunks))
	} else {
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		log.Printf("[DOCS] document %s indexing failed (some upserts failed)", docID)
	}
}

func extractTextWithMetadata(filePath, ext string) ([]PageContent, error) {
	switch ext {
	case ".pdf":
		return extractPDFTextWithPages(filePath)
	case ".txt", ".md":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return []PageContent{{Page: 0, Text: string(data)}}, nil
	case ".docx":
		text, err := extractDOCXText(filePath)
		if err != nil {
			return nil, err
		}
		return []PageContent{{Page: 0, Text: text}}, nil
	}
	return nil, fmt.Errorf("unsupported type: %s", ext)
}

func extractPDFTextWithPages(filePath string) ([]PageContent, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	numPages := r.NumPage()
	var pages []PageContent

	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		text, err := page.GetPlainText(nil)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		pages = append(pages, PageContent{Page: i, Text: text})
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no text content found in PDF")
	}
	return pages, nil
}

func extractDOCXText(filePath string) (string, error) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer archive.Close()

	for _, f := range archive.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			data, _ := io.ReadAll(rc)
			text := string(data)
			text = strings.ReplaceAll(text, "</w:p>", "\n")
			text = strings.ReplaceAll(text, "</w:r>", "")
			text = strings.ReplaceAll(text, "</w:t>", "")
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

var sectionRe = regexp.MustCompile(`^(?:(?:capitulo|cap\.?|chapter)\s+\d+|(?:\d{1,3}(?:\.\d{1,3}){0,2})\s+[A-ZÁÉÍÓÚÑ].{2,60}|#{1,3}\s+.+|(?:introduccion|conclusion|resumen|bibliografia|referencias|appendix|anexo).*)$`)

func detectSection(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 || len(trimmed) > 100 {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if sectionRe.MatchString(lower) {
		return trimmed
	}
	return ""
}

func detectContentType(text string) string {
	lower := strings.ToLower(text)
	if matched, _ := regexp.MatchString(`(?i)(definicion|definición|define|se define)`, lower); matched {
		return "definition"
	}
	if matched, _ := regexp.MatchString(`(?i)(teorema|theorem|se demuestra|demostracion)`, lower); matched {
		return "theorem"
	}
	if matched, _ := regexp.MatchString(`(?i)(fórmula|formula|expresion|ecuacion|ecuación)`, lower); matched {
		return "formula"
	}
	if matched, _ := regexp.MatchString(`(?i)(ejemplo|ej\.|ej1|ej2|ejercicio resuelto)`, lower); matched {
		return "example"
	}
	if matched, _ := regexp.MatchString(`(?i)(ejercicio|resolver|calcular|determinar|hallar)`, lower); matched {
		return "exercise"
	}
	if matched, _ := regexp.MatchString(`(?i)(solucion|solución|resultado|respuesta)`, lower); matched {
		return "solution"
	}
	if matched, _ := regexp.MatchString(`(?i)(resumen|conclusion|conclusión)`, lower); matched {
		return "summary"
	}
	return "theory"
}

func detectContentFlags(text string) (hasFormula, hasExample, hasExercise, hasSolution bool) {
	lower := strings.ToLower(text)
	hasFormula, _ = regexp.MatchString(`(?i)(fórmula|formula|=|\\frac|\\int|\\sum|\$\$)`, lower)
	hasExample, _ = regexp.MatchString(`(?i)(ejemplo|ej\.|ej1|ej2)`, lower)
	hasExercise, _ = regexp.MatchString(`(?i)(ejercicio|resolver|calcular|determinar|hallar)`, lower)
	hasSolution, _ = regexp.MatchString(`(?i)(solucion|solución|resultado|respuesta)`, lower)
	return
}

func chunkTextWithMetadata(pages []PageContent, chunkSize, overlap int) []ChunkMeta {
	var allChunks []ChunkMeta
	currentSection := ""

	for _, page := range pages {
		text := strings.TrimSpace(page.Text)
		if len(text) == 0 {
			continue
		}

		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if s := detectSection(line); s != "" {
				currentSection = s
			}
		}

		runes := []rune(text)
		for i := 0; i < len(runes); i += chunkSize - overlap {
			end := i + chunkSize
			if end > len(runes) {
				end = len(runes)
			}
			chunk := strings.TrimSpace(string(runes[i:end]))
			if len(chunk) > 20 {
				hasFormula, hasExample, hasExercise, hasSolution := detectContentFlags(chunk)
				allChunks = append(allChunks, ChunkMeta{
					Text:        chunk,
					Page:        page.Page,
					Section:     currentSection,
					ContentType: detectContentType(chunk),
					HasFormula:  hasFormula,
					HasExample:  hasExample,
					HasExercise: hasExercise,
					HasSolution: hasSolution,
				})
			}
			if end >= len(runes) {
				break
			}
		}
	}
	return allChunks
}

func generateEmbeddings(db *pgxpool.Pool, texts []string) ([][]float32, error) {
	apiKey := getEmbeddingAPIKey(db)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key for embeddings")
	}
	log.Printf("[EMBED] using key: %s... (len=%d)", apiKey[:min(len(apiKey), 8)], len(apiKey))

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
	log.Printf("[EMBED] OpenAI response status: %d, body: %s", resp.StatusCode, string(respBody[:min(len(respBody), 500)]))
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

func getEmbeddingAPIKey(db *pgxpool.Pool) string {
	keyName := getSetting(db, "AI_API_KEY_NAME")
	if keyName == "" {
		keyName = "OPENAI_API_KEY"
	}
	if v := getSetting(db, keyName); v != "" {
		return v
	}
	return ""
}

func VectorSearch(db *pgxpool.Pool, query string, topK int) ([]VectorSearchResult, error) {
	apiKey := getEmbeddingAPIKey(db)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key for embeddings")
	}

	queryEmb, err := generateEmbeddings(db, []string{query})
	if err != nil {
		return nil, err
	}
	if len(queryEmb) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}

	results, err := qdrantSearch(queryEmb[0], topK)
	if err != nil {
		return nil, err
	}

	searchResults := make([]VectorSearchResult, len(results))
	for i, r := range results {
		sr := VectorSearchResult{
			ChunkID:  r.ID,
			DocID:    fmt.Sprintf("%v", r.Payload["document_id"]),
			Content:  r.Payload["content"].(string),
			Score:    r.Score,
			Filename: fmt.Sprintf("%v", r.Payload["filename"]),
		}
		if v, ok := r.Payload["page"].(float64); ok {
			sr.Page = int(v)
		}
		if v, ok := r.Payload["section"].(string); ok {
			sr.Section = v
		}
		if v, ok := r.Payload["url"].(string); ok {
			sr.URL = v
		}
		searchResults[i] = sr
	}
	return searchResults, nil
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
