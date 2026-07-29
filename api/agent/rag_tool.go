package agent

import (
	"context"
	"fmt"
)

func NewRagTool(hybridSearchFn func(ctx context.Context, query string, filters map[string]any, topK int, vectorWeight, keywordWeight float64) ([]map[string]any, error), rerankFn func(ctx context.Context, query string, results []map[string]any, topK int) ([]map[string]any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        "rag_search",
		Description: "Search academic material in Qdrant using hybrid search with reranking",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			query, _ := input["query"].(string)
			if query == "" {
				return nil, fmt.Errorf("query is required")
			}

			filters, _ := input["filters"].(map[string]any)
			if filters == nil {
				filters = make(map[string]any)
			}

			topK := 20
			if t, ok := input["top_k"].(float64); ok {
				topK = int(t)
			}

			results, err := hybridSearchFn(ctx, query, filters, topK, 0.60, 0.40)
			if err != nil {
				return nil, fmt.Errorf("hybrid search: %w", err)
			}

			reranked, err := rerankFn(ctx, query, results, 5)
			if err == nil {
				results = reranked
			}

			var resultItems []map[string]any
			var sources []map[string]any
			for _, r := range results {
				resultItems = append(resultItems, map[string]any{
					"chunk_id":    r["chunk_id"],
					"document_id": r["document_id"],
					"content":     r["content"],
					"score":       r["score"],
				})
				sources = append(sources, map[string]any{
					"document_id":    r["document_id"],
					"document_title": r["filename"],
					"page":           r["page"],
					"section":        r["section"],
					"score":          r["score"],
				})
			}

			return map[string]any{
				"results": resultItems,
				"sources": sources,
			}, nil
		},
	}
}
