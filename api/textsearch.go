package api

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TextSearchResult struct {
	ChunkID  string  `json:"chunkId"`
	DocID    string  `json:"documentId"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Filename string  `json:"filename"`
	Page     int     `json:"page"`
	Section  string  `json:"section"`
	URL      string  `json:"url"`
}

func KeywordSearch(db *pgxpool.Pool, query string, topK int, filters map[string]interface{}) ([]TextSearchResult, error) {
	querySQL := `
		SELECT dc.id, dc.document_id, dc.content, ts_rank(dc.content_tsv, plainto_tsquery('spanish', $1)) as rank,
		       d.original_name, dc.page, dc.section,
		       COALESCE('/documents?id=' || dc.document_id, '')
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.document_id
		WHERE dc.content_tsv @@ plainto_tsquery('spanish', $1)
	`

	args := []interface{}{query}
	argIdx := 2

	if v, ok := filters["document_id"]; ok && v != nil && v != "" {
		querySQL += fmt.Sprintf(" AND dc.document_id = $%d", argIdx)
		args = append(args, v)
		argIdx++
	}
	if v, ok := filters["course_id"]; ok && v != nil && v != "" {
		querySQL += fmt.Sprintf(" AND dc.course_id = $%d", argIdx)
		args = append(args, v)
		argIdx++
	}
	if v, ok := filters["unit_id"]; ok && v != nil && v != "" {
		querySQL += fmt.Sprintf(" AND dc.unit_id = $%d", argIdx)
		args = append(args, v)
		argIdx++
	}
	if v, ok := filters["content_type"]; ok && v != nil && v != "" {
		querySQL += fmt.Sprintf(" AND dc.content_type = $%d", argIdx)
		args = append(args, v)
		argIdx++
	}

	querySQL += " ORDER BY rank DESC"
	querySQL += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, topK)

	log.Printf("[TEXTSEARCH] query: %s, args: %v", query, args)

	rows, err := db.Query(context.Background(), querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("text search error: %v", err)
	}
	defer rows.Close()

	var results []TextSearchResult
	for rows.Next() {
		var r TextSearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocID, &r.Content, &r.Score, &r.Filename, &r.Page, &r.Section, &r.URL); err != nil {
			log.Printf("[TEXTSEARCH] scan error: %v", err)
			continue
		}
		results = append(results, r)
	}

	return results, nil
}

func NormalizeTextSearchScores(results []TextSearchResult) []TextSearchResult {
	if len(results) == 0 {
		return results
	}
	maxScore := 0.0
	for _, r := range results {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}
	if maxScore == 0 {
		return results
	}
	for i := range results {
		results[i].Score = results[i].Score / maxScore
	}
	return results
}
