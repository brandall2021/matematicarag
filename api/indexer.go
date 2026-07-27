package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReindexRequest struct {
	DocumentID string `json:"document_id,omitempty"`
}

type ReindexResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}

func IndexerRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))
		r.Use(RoleMiddleware("ADMIN"))

		r.Post("/reindex", func(w http.ResponseWriter, r *http.Request) {
			var req ReindexRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.DocumentID != "" {
				go reindexDocument(db, req.DocumentID)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ReindexResponse{
					Status:  "started",
					Message: "Reindexación iniciada para el documento " + req.DocumentID,
				})
				return
			}

			go reindexAll(db)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ReindexResponse{
				Status:  "started",
				Message: "Reindexación completa iniciada",
			})
		})
	}
}

func reindexDocument(db *pgxpool.Pool, docID string) {
	ctx := context.Background()

	var filename, originalName, ext string
	err := db.QueryRow(ctx,
		`SELECT filename, original_name, type FROM documents WHERE id = $1`, docID,
	).Scan(&filename, &originalName, &ext)
	if err != nil {
		log.Printf("[REINDEX] document not found: %s", docID)
		return
	}

	qdrantDeleteByDocID(docID)
	db.Exec(ctx, `DELETE FROM document_chunks WHERE document_id = $1`, docID)

	filePath := filepath.Join("./uploads", filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("[REINDEX] file not found: %s", filePath)
		db.Exec(ctx, `UPDATE documents SET status = 'error' WHERE id = $1`, docID)
		return
	}

	processDocument(db, docID, filePath, ext, originalName)
	log.Printf("[REINDEX] document %s reindexed", docID)
}

func reindexAll(db *pgxpool.Pool) {
	ctx := context.Background()

	rows, err := db.Query(ctx, `SELECT id FROM documents WHERE status = 'indexed' OR status = 'error'`)
	if err != nil {
		log.Printf("[REINDEX] query error: %v", err)
		return
	}
	defer rows.Close()

	var docIDs []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			docIDs = append(docIDs, id)
		}
	}

	log.Printf("[REINDEX] reindexing %d documents", len(docIDs))
	for _, id := range docIDs {
		reindexDocument(db, id)
	}
	log.Printf("[REINDEX] complete reindex finished")
}
