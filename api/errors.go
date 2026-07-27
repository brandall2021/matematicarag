package api

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrorTaxonomy = map[string][]string{
	"conceptual":       {"misconception", "wrong_concept", "incomplete_understanding"},
	"algebraic":        {"distributive_property", "factoring_error", "expansion_error"},
	"arithmetic":       {"addition", "subtraction", "multiplication", "division", "power"},
	"sign":             {"sign_change", "double_negative", "sign_in_distribution"},
	"formula":          {"wrong_formula", "formula_misapplication", "missing_term"},
	"method_selection": {"wrong_method", "unnecessary_complexity"},
	"notation":         {"notation_error", "undefined_variable"},
	"domain":           {"domain_violation", "division_by_zero"},
	"logical":          {"logical_gap", "invalid_inference", "circular_reasoning"},
	"incomplete":       {"missing_solution", "missing_case", "incomplete_answer"},
}

type ErrorPattern struct {
	ConceptID    string `json:"concept_id"`
	ErrorType    string `json:"error_type"`
	ErrorSubtype string `json:"error_subtype"`
	Count        int    `json:"count"`
	Severity     string `json:"severity"`
}

func RecordError(db *pgxpool.Pool, studentID, conceptID, errorType, errorSubtype string) {
	ctx := context.Background()
	severity := calculateSeverity(db, studentID, conceptID, errorType)

	_, err := db.Exec(ctx,
		`INSERT INTO student_errors (student_id, concept_id, error_type, error_subtype, count, severity, last_occurred_at)
		 VALUES ($1, $2, $3, $4, 1, $5, NOW())
		 ON CONFLICT (student_id, concept_id, error_type, error_subtype) DO UPDATE SET
		   count = student_errors.count + 1,
		   severity = $5,
		   last_occurred_at = NOW()`,
		studentID, conceptID, errorType, errorSubtype, severity)
	if err != nil {
		log.Printf("[ERRORS] record error: %v", err)
	}
}

func calculateSeverity(db *pgxpool.Pool, studentID, conceptID, errorType string) string {
	ctx := context.Background()
	var count int
	db.QueryRow(ctx,
		`SELECT COALESCE(SUM(count), 0) FROM student_errors
		 WHERE student_id = $1 AND concept_id = $2 AND error_type = $3`,
		studentID, conceptID, errorType).Scan(&count)

	switch {
	case count >= 8:
		return "critical"
	case count >= 5:
		return "high"
	case count >= 3:
		return "medium"
	default:
		return "low"
	}
}

func DetectPatterns(db *pgxpool.Pool, studentID string) ([]ErrorPattern, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT concept_id, error_type, error_subtype, count, severity
		 FROM student_errors
		 WHERE student_id = $1
		 ORDER BY count DESC
		 LIMIT 10`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []ErrorPattern
	for rows.Next() {
		var p ErrorPattern
		if err := rows.Scan(&p.ConceptID, &p.ErrorType, &p.ErrorSubtype, &p.Count, &p.Severity); err != nil {
			continue
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}
