package api

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ProcessDocument(db *pgxpool.Pool, docID, filePath, docType, originalName string) {
	processDocument(db, docID, filePath, docType, originalName)
}

func PopulateChunksOnly(db *pgxpool.Pool, docID, filePath, docType, originalName string) {
	ctx := context.Background()

	var existing int
	db.QueryRow(ctx, `SELECT COUNT(*) FROM document_chunks WHERE document_id = $1`, docID).Scan(&existing)
	if existing > 0 {
		log.Printf("[MIGRATE] chunks already exist for %s, skipping", docID)
		return
	}

	pages, err := extractTextWithMetadata(filePath, docType)
	if err != nil {
		log.Printf("[MIGRATE] extract error for %s: %v", filePath, err)
		return
	}

	chunkSize := 500
	overlap := 50
	if v := getSetting(db, "CHUNK_SIZE"); v != "" {
		fmt.Sscanf(v, "%d", &chunkSize)
	}
	if v := getSetting(db, "CHUNK_OVERLAP"); v != "" {
		fmt.Sscanf(v, "%d", &overlap)
	}

	chunks := chunkTextWithMetadata(pages, chunkSize, overlap)

	for i, chunk := range chunks {
		db.Exec(ctx,
			`INSERT INTO document_chunks (document_id, chunk_index, content, page, section, topic, content_type, has_formula, has_example, has_exercise, has_solution)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			docID, i, chunk.Text, chunk.Page, chunk.Section, chunk.Topic, chunk.ContentType, chunk.HasFormula, chunk.HasExample, chunk.HasExercise, chunk.HasSolution,
		)
	}

	log.Printf("[MIGRATE] inserted %d chunks for %s", len(chunks), docID)
}
