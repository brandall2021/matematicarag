package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brandall2021/matematicarag/api"
	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TestQuery struct {
	Query    string
	Expected []string // Expected document filenames or keywords in results
}

func main() {
	cfg := config.Load()

	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}
	defer dbpool.Close()

	// Test queries covering different scenarios
	tests := []TestQuery{
		{
			Query:    "¿Que es un espacio vectorial?",
			Expected: []string{"algebra", "lineal"},
		},
		{
			Query:    "Teorema de pitagoras",
			Expected: []string{"geometria", "pitagoras"},
		},
		{
			Query:    "Derivada de una funcion",
			Expected: []string{"calculo", "derivada"},
		},
		{
			Query:    "Definicion de numero primo",
			Expected: []string{"numero", "primo"},
		},
		{
			Query:    "Limite de una funcion",
			Expected: []string{"limite", "calculo"},
		},
	}

	fmt.Println("=== RAG Evaluation Tests ===")
	fmt.Printf("Running %d test queries...\n\n", len(tests))

	var totalHits, totalTests int
	var totalDuration time.Duration

	for i, test := range tests {
		start := time.Now()

		// Run hybrid search
		hybridResults, err := api.HybridSearch(dbpool, test.Query, 20, nil, 0.60, 0.40)
		if err != nil {
			log.Printf("[FAIL] Query %d: hybrid search error: %v", i+1, err)
			continue
		}

		// Rerank
		reranked, err := api.RerankResults(dbpool, test.Query, hybridResults, 5)
		if err != nil {
			log.Printf("[WARN] Query %d: rerank failed, using hybrid results: %v", i+1, err)
			reranked = hybridResults
			if len(reranked) > 5 {
				reranked = reranked[:5]
			}
		}

		duration := time.Since(start)
		totalDuration += duration

		// Check results
		hits := 0
		for _, result := range reranked {
			contentLower := strings.ToLower(result.Content + " " + result.Filename)
			for _, expected := range test.Expected {
				if strings.Contains(contentLower, strings.ToLower(expected)) {
					hits++
					break
				}
			}
		}

		totalHits += hits
		totalTests += len(test.Expected)

		status := "PASS"
		if hits == 0 {
			status = "PARTIAL"
		}

		fmt.Printf("[%s] Query %d: %q\n", status, i+1, test.Query)
		fmt.Printf("  Results: %d | Hits: %d/%d | Duration: %v\n",
			len(reranked), hits, len(test.Expected), duration.Round(time.Millisecond))

		for j, r := range reranked {
			score := fmt.Sprintf("%.2f", r.RerankScore)
			fmt.Printf("  %d. [Score:%s] %s (p.%d) - %s\n",
				j+1, score, r.Filename, r.Page, truncate(r.Content, 80))
		}
		fmt.Println()
	}

	// Summary
	precision := float64(totalHits) / float64(totalTests) * 100
	avgDuration := totalDuration / time.Duration(len(tests))

	fmt.Println("=== Summary ===")
	fmt.Printf("Total Queries:   %d\n", len(tests))
	fmt.Printf("Total Hits:      %d/%d (%.1f%%)\n", totalHits, totalTests, precision)
	fmt.Printf("Avg Duration:    %v\n", avgDuration.Round(time.Millisecond))
	fmt.Println("================")

	os.Exit(0)
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
