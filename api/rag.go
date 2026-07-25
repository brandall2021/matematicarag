package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RagQueryRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"topK,omitempty"`
}

type RagQueryResponse struct {
	Answer  string      `json:"answer"`
	Sources []RagSource `json:"sources"`
}

type RagSource struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Metadata string  `json:"metadata"`
}

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

			// Vector search
			results, err := VectorSearch(db, req.Query, req.TopK)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(RagQueryResponse{
					Answer:  "No se pudo realizar la busqueda vectorial: " + err.Error(),
					Sources: []RagSource{},
				})
				return
			}

			if len(results) == 0 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(RagQueryResponse{
					Answer:  "No se encontraron documentos relevantes. Subi material de estudio en Gestion Documental.",
					Sources: []RagSource{},
				})
				return
			}

			// Build context from search results
			var contextParts []string
			sources := make([]RagSource, len(results))
			for i, res := range results {
				contextParts = append(contextParts, res.Content)
				sources[i] = RagSource{
					ID:       res.ChunkID,
					Content:  truncateString(res.Content, 200),
					Score:    res.Score,
					Metadata: res.Filename,
				}
			}

			// Generate answer using AI with context
			systemPrompt := `Sos un tutor de matematicas de la UNT. Respondes en español usando la informacion de los documentos provistos como contexto.
Si la pregunta no esta en el contexto, lo indicas y respondes con tu conocimiento general.
Siempre citas de que documento sacas la informacion.`

			userPrompt := "Pregunta: " + req.Query + "\n\nContexto de documentos:\n" + strings.Join(contextParts, "\n---\n")

			apiKey := getAPIKey(db)
			if apiKey == "" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(RagQueryResponse{
					Answer:  strings.Join(contextParts, "\n\n"),
					Sources: sources,
				})
				return
			}

			messages := []OpenAIMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			}

			answer, callErr := callOpenAIWithHistory(db, messages, "")
			if callErr != nil {
				answer = strings.Join(contextParts, "\n\n")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(RagQueryResponse{
				Answer:  answer,
				Sources: sources,
			})
		})
	}
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
