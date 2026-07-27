package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RagQueryRequest struct {
	Query       string `json:"query"`
	CourseID    string `json:"course_id,omitempty"`
	UnitID      string `json:"unit_id,omitempty"`
	DocumentID  string `json:"document_id,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	TopK        int    `json:"top_k,omitempty"`
}

type RagQueryResponse struct {
	Answer     string         `json:"answer"`
	Citations  []RagCitation  `json:"citations"`
	Confidence string         `json:"confidence"`
	Retrieval  RetrievalInfo  `json:"retrieval"`
}

type RagCitation struct {
	ID         string  `json:"id"`
	DocumentID string  `json:"document_id"`
	Document   string  `json:"document"`
	Page       int     `json:"page"`
	Section    string  `json:"section"`
	ChunkID    string  `json:"chunk_id"`
	Score      float64 `json:"score"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
}

type RetrievalInfo struct {
	VectorResults   int `json:"vector_results"`
	KeywordResults  int `json:"keyword_results"`
	HybridResults   int `json:"hybrid_results"`
	RerankedResults int `json:"reranked_results"`
}

type RagSource struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Filename string  `json:"filename"`
	Page     int     `json:"page,omitempty"`
	Section  string  `json:"section,omitempty"`
	URL      string  `json:"url,omitempty"`
}

const defaultRAGPrompt = `Sos un tutor de matematicas de la UNT. Respondes usando la informacion de los documentos provistos como contexto.

REGLAS DE CITACION:
- SIEMPRE referencia las fuentes usando el formato: [SRC-XXX] donde XXX es el ID de la fuente
- Las fuentes estan numeradas como [SRC-001], [SRC-002], etc.
- Cada idea o parrafo relevante debe llevar su cita
- Si la informacion viene de un documento sin pagina conocida, igualmente cita con [SRC-XXX]
- Si la pregunta no esta en el contexto, indica que no fue encontrada
- Puedes citar multiples fuentes en una misma respuesta
- Las citas van al final de cada idea o parrafo relevante

REGLAS DE HONESTIDAD:
- No inventes informacion documental
- No inventes paginas
- No inventes nombres de documentos
- Si la informacion necesaria no esta en el contexto, indica que no fue encontrada en el material disponible`

func RagRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))
		r.Post("/query", func(w http.ResponseWriter, r *http.Request) {
			var req RagQueryRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Query == "" {
				http.Error(w, `{"error":"query is required"}`, http.StatusBadRequest)
				return
			}
			if req.TopK == 0 {
				req.TopK = 5
			}

			// Build filters
			filters := make(map[string]interface{})
			if req.DocumentID != "" {
				filters["document_id"] = req.DocumentID
			}
			if req.CourseID != "" {
				filters["course_id"] = req.CourseID
			}
			if req.UnitID != "" {
				filters["unit_id"] = req.UnitID
			}
			if req.ContentType != "" {
				filters["content_type"] = req.ContentType
			}

			// 1. Hybrid search
			vectorWeight := 0.60
			keywordWeight := 0.40
			hybridResults, err := HybridSearch(db, req.Query, req.TopK*4, filters, vectorWeight, keywordWeight)
			if err != nil {
				log.Printf("[RAG] hybrid search error: %v", err)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(RagQueryResponse{
					Answer:     "Error en la busqueda: " + err.Error(),
					Citations:  []RagCitation{},
					Confidence: "low",
				})
				return
			}

			retrievalInfo := RetrievalInfo{
				HybridResults: len(hybridResults),
			}

			if len(hybridResults) == 0 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(RagQueryResponse{
					Answer:     "No se encontraron documentos relevantes. Subi material de estudio en Gestion Documental.",
					Citations:  []RagCitation{},
					Confidence: "low",
					Retrieval:  retrievalInfo,
				})
				return
			}

			// 2. Rerank
			reranked, err := RerankResults(db, req.Query, hybridResults, req.TopK)
			if err != nil {
				log.Printf("[RAG] rerank error: %v", err)
				reranked = hybridResults
				if len(reranked) > req.TopK {
					reranked = reranked[:req.TopK]
				}
			}
			retrievalInfo.RerankedResults = len(reranked)

			// 3. Build citations
			citations := make([]RagCitation, len(reranked))
			var contextParts []string
			for i, r := range reranked {
				citationID := fmt.Sprintf("SRC-%03d", i+1)
				citations[i] = RagCitation{
					ID:         citationID,
					DocumentID: r.DocID,
					Document:   r.Filename,
					Page:       r.Page,
					Section:    r.Section,
					ChunkID:    r.ChunkID,
					Score:      math.Round(r.RerankScore*100) / 100,
					URL:        r.URL,
					Content:    r.Content,
				}
				contextParts = append(contextParts, fmt.Sprintf(
					"FUENTE [%s]\nDocumento: %s\nPágina: %d\nSección: %s\nContenido: %s",
					citationID, r.Filename, r.Page, r.Section, r.Content,
				))
			}

			// 4. Build context for LLM
			context := strings.Join(contextParts, "\n\n---\n\n")

			systemPrompt := getSetting(db, "RAG_SYSTEM_PROMPT")
			if systemPrompt == "" {
				systemPrompt = defaultRAGPrompt
			}

			userPrompt := fmt.Sprintf("Pregunta: %s\n\nContexto de documentos:\n%s\n\nResponde usando [SRC-XXX] para citar las fuentes.", req.Query, context)

			apiKey := getAPIKey(db)
			if apiKey == "" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(RagQueryResponse{
					Answer:     "API key no configurada. No se puede generar respuesta.",
					Citations:  citations,
					Confidence: "low",
					Retrieval:  retrievalInfo,
				})
				return
			}

			// 5. Call LLM
			messages := []OpenAIMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			}

			answer, callErr := callOpenAIWithHistory(db, messages, "")
			if callErr != nil {
				log.Printf("[RAG] LLM error: %v", callErr)
				// Return context without LLM summary
				answer = fmt.Sprintf("No se pudo generar una respuesta con IA. Fragmentos relevantes encontrados:\n\n%s", strings.Join(contextParts, "\n\n"))
			}

			// 6. Validate citations — remove invented ones
			answer = validateCitations(answer, citations)

			// 7. Calculate confidence
			confidence := calculateConfidence(reranked)

			// 8. Log retrieval info
			log.Printf("[RAG] query=%s, hybrid=%d, reranked=%d, citations=%d, confidence=%s",
				req.Query, retrievalInfo.HybridResults, retrievalInfo.RerankedResults, len(citations), confidence)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(RagQueryResponse{
				Answer:     answer,
				Citations:  citations,
				Confidence: confidence,
				Retrieval:  retrievalInfo,
			})
		})
	}
}

// validateCitations removes [SRC-XXX] references from the answer that don't exist in citations
func validateCitations(answer string, citations []RagCitation) string {
	validIDs := make(map[string]bool)
	for _, c := range citations {
		validIDs[c.ID] = true
	}

	re := regexp.MustCompile(`\[SRC-\d+\]`)
	return re.ReplaceAllStringFunc(answer, func(match string) string {
		if validIDs[match[1:len(match)-1]] {
			return match
		}
		return ""
	})
}

// calculateConfidence based on average rerank score
func calculateConfidence(results []HybridResult) string {
	if len(results) == 0 {
		return "low"
	}
	sum := 0.0
	for _, r := range results {
		sum += r.RerankScore
	}
	avg := sum / float64(len(results))
	if avg >= 7.0 {
		return "high"
	}
	if avg >= 4.0 {
		return "medium"
	}
	return "low"
}

func buildSourceLabel(filename string, page int, section string) string {
	parts := []string{filename}
	if page > 0 {
		parts = append(parts, fmt.Sprintf("pagina %d", page))
	}
	if section != "" {
		parts = append(parts, fmt.Sprintf("seccion \"%s\"", section))
	}
	return strings.Join(parts, ", ")
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
