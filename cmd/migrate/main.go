package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/brandall2021/matematicarag/api"
	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}
	defer dbpool.Close()

	mode := "all"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "all":
		fmt.Println("=== Migrating all indexed documents ===")
		migrateAll(dbpool)

	case "status":
		fmt.Println("=== Migration Status ===")
		showStatus(dbpool)

	case "chunks-only":
		fmt.Println("=== Populating document_chunks only ===")
		populateChunksOnly(dbpool)

	default:
		fmt.Println("Usage: migrate [all|status|chunks-only]")
		fmt.Println("  all          - Full migration (chunks + re-upsert)")
		fmt.Println("  status       - Show migration status")
		fmt.Println("  chunks-only  - Only populate document_chunks")
		os.Exit(1)
	}

	os.Exit(0)
}

func migrateAll(dbpool *pgxpool.Pool) {
	ctx := context.Background()

	rows, err := dbpool.Query(ctx,
		`SELECT id, filename, original_name, type FROM documents WHERE status = 'indexed'`)
	if err != nil {
		log.Fatalf("Query error: %v", err)
	}
	defer rows.Close()

	var docs []struct {
		ID, Filename, OriginalName, Type string
	}

	for rows.Next() {
		var d struct {
			ID, Filename, OriginalName, Type string
		}
		if rows.Scan(&d.ID, &d.Filename, &d.OriginalName, &d.Type) == nil {
			docs = append(docs, d)
		}
	}

	fmt.Printf("Found %d documents to migrate\n\n", len(docs))

	for i, doc := range docs {
		fmt.Printf("[%d/%d] Migrating: %s (%s)\n", i+1, len(docs), doc.OriginalName, doc.ID)

		filePath := "./uploads/" + doc.Filename
		api.ProcessDocument(dbpool, doc.ID, filePath, doc.Type, doc.OriginalName)

		fmt.Printf("  ✓ Done\n")
	}

	fmt.Printf("\n=== Migration complete: %d documents ===\n", len(docs))
}

func showStatus(dbpool *pgxpool.Pool) {
	ctx := context.Background()

	var total int
	dbpool.QueryRow(ctx, `SELECT COUNT(*) FROM documents`).Scan(&total)

	var withChunks int
	dbpool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT document_id) FROM document_chunks
	`).Scan(&withChunks)

	var totalChunks int
	dbpool.QueryRow(ctx, `SELECT COUNT(*) FROM document_chunks`).Scan(&totalChunks)

	rows, _ := dbpool.Query(ctx, `
		SELECT content_type, COUNT(*) FROM document_chunks GROUP BY content_type ORDER BY COUNT(*) DESC
	`)
	defer rows.Close()

	fmt.Printf("Total documents:    %d\n", total)
	fmt.Printf("With chunks:        %d\n", withChunks)
	fmt.Printf("Total chunks:       %d\n", totalChunks)
	fmt.Printf("Pending migration:  %d\n", total-withChunks)

	fmt.Println("\nContent type breakdown:")
	for rows.Next() {
		var ct string
		var count int
		if rows.Scan(&ct, &count) == nil {
			fmt.Printf("  %-20s %d\n", ct, count)
		}
	}
}

func populateChunksOnly(dbpool *pgxpool.Pool) {
	ctx := context.Background()

	rows, err := dbpool.Query(ctx,
		`SELECT id, filename, original_name, type FROM documents WHERE status = 'indexed'`)
	if err != nil {
		log.Fatalf("Query error: %v", err)
	}
	defer rows.Close()

	var docs []struct {
		ID, Filename, OriginalName, Type string
	}

	for rows.Next() {
		var d struct {
			ID, Filename, OriginalName, Type string
		}
		if rows.Scan(&d.ID, &d.Filename, &d.OriginalName, &d.Type) == nil {
			docs = append(docs, d)
		}
	}

	fmt.Printf("Found %d documents\n\n", len(docs))

	for i, doc := range docs {
		fmt.Printf("[%d/%d] %s\n", i+1, len(docs), doc.OriginalName)

		filePath := "./uploads/" + doc.Filename
		api.PopulateChunksOnly(dbpool, doc.ID, filePath, doc.Type, doc.OriginalName)

		fmt.Printf("  ✓ Chunks populated\n")
	}
}
