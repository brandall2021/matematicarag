package api

import (
	"encoding/json"
	"net/http"

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
			response := RagQueryResponse{
				Answer:  "RAG pipeline placeholder - vector search integration pending",
				Sources: []RagSource{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})
	}
}
