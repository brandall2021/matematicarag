package adaptive

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ErrorAnalyzer struct {
	db *pgxpool.Pool
}

func NewErrorAnalyzer(db *pgxpool.Pool) *ErrorAnalyzer {
	return &ErrorAnalyzer{db: db}
}

var errorKeywords = map[string][]string{
	"conceptual":       {"misconception", "wrong concept", "incomplete understanding"},
	"algebraic":        {"distributive", "factoring", "expansion"},
	"arithmetic":       {"addition", "subtraction", "multiplication", "division", "power"},
	"sign":             {"sign change", "double negative", "sign in distribution"},
	"formula":          {"wrong formula", "formula misapplication", "missing term"},
	"method_selection": {"wrong method", "unnecessary complexity"},
	"notation":         {"notation", "undefined variable"},
	"domain":           {"domain violation", "division by zero"},
	"logical":          {"logical gap", "invalid inference", "circular reasoning"},
	"incomplete":       {"missing solution", "missing case", "incomplete answer"},
}

func (ea *ErrorAnalyzer) ClassifyError(errorDetail string) string {
	lower := strings.ToLower(errorDetail)
	for etype, keywords := range errorKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return etype
			}
		}
	}
	return "unknown"
}

func (ea *ErrorAnalyzer) IsRecurrent(studentID, conceptID, errorType string) bool {
	ctx := context.Background()
	var count int
	ea.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(count), 0) FROM student_errors
		 WHERE student_id = $1 AND concept_id = $2 AND error_type = $3`,
		studentID, conceptID, errorType).Scan(&count)
	return count >= 3
}

func (ea *ErrorAnalyzer) GetSeverity(count int) string {
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

func (ea *ErrorAnalyzer) RecordError(studentID, courseID, conceptID, errorType, errorDetail string) {
	ctx := context.Background()
	severity := "low"

	var currentCount int
	ea.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(count), 0) FROM student_errors
		 WHERE student_id = $1 AND concept_id = $2 AND error_type = $3`,
		studentID, conceptID, errorType).Scan(&currentCount)
	severity = ea.GetSeverity(currentCount + 1)

	subtype := ""
	if parts := strings.SplitN(errorDetail, ":", 2); len(parts) > 1 {
		subtype = strings.TrimSpace(parts[0])
	}

	_, err := ea.db.Exec(ctx,
		`INSERT INTO student_errors (student_id, concept_id, error_type, error_subtype, count, severity, last_occurred_at)
		 VALUES ($1, $2, $3, $4, 1, $5, NOW())
		 ON CONFLICT (student_id, concept_id, error_type, error_subtype) DO UPDATE SET
		   count = student_errors.count + 1,
		   severity = $5,
		   last_occurred_at = NOW()`,
		studentID, conceptID, errorType, subtype, severity)
	if err != nil {
		log.Printf("[ADAPTIVE_ERROR] record error: %v", err)
	}
}
