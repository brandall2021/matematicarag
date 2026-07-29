package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Question struct {
	ID               string          `json:"id"`
	Statement        string          `json:"statement"`
	Latex            string          `json:"latex,omitempty"`
	QuestionType     string          `json:"question_type"`
	Difficulty       int             `json:"difficulty"`
	ConceptID        string          `json:"concept_id,omitempty"`
	Competencies     []string        `json:"competencies"`
	ExpectedAnswer   string          `json:"expected_answer,omitempty"`
	AnswerOptions    json.RawMessage `json:"answer_options,omitempty"`
	Explanation      string          `json:"explanation,omitempty"`
	ExplanationLatex string          `json:"explanation_latex,omitempty"`
	Tags             []string        `json:"tags"`
	Source           string          `json:"source"`
	CreatedBy        string          `json:"created_by,omitempty"`
	ValidatedByMath  bool            `json:"validated_by_math"`
	Version          int             `json:"version"`
	IsActive         bool            `json:"is_active"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	CreatedAt        string          `json:"created_at,omitempty"`
}

func QuestionRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", CreateQuestionHandler(db, cfg))
		r.Get("/", ListQuestionsHandler(db))
		r.Get("/{questionID}", GetQuestionHandler(db))
		r.Put("/{questionID}", UpdateQuestionHandler(db, cfg))
		r.Delete("/{questionID}", DeleteQuestionHandler(db))
		r.Post("/validate/{questionID}", ValidateQuestionHandler(db, cfg))
	}
}

func CreateQuestionHandler(db *pgxpool.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var q Question
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(q.Statement) == "" {
			http.Error(w, `{"error":"statement is required"}`, http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(q.QuestionType) == "" {
			http.Error(w, `{"error":"question_type is required"}`, http.StatusBadRequest)
			return
		}
		if q.Difficulty < 1 || q.Difficulty > 5 {
			q.Difficulty = 1
		}

		if validationErr := ValidateQuestionByType(q); validationErr != "" {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, validationErr), http.StatusBadRequest)
			return
		}

		if q.Competencies == nil {
			q.Competencies = []string{}
		}
		if q.Tags == nil {
			q.Tags = []string{}
		}
		q.IsActive = true
		q.Version = 1

		mathClient := NewMathClient(cfg)
		if q.ExpectedAnswer != "" {
			result, err := mathClient.Verify(q.ExpectedAnswer, q.ExpectedAnswer, "")
			q.ValidatedByMath = err == nil && result != nil && result.Success
		}

		var id string
		err := db.QueryRow(r.Context(),
			`INSERT INTO questions (statement, latex, question_type, difficulty, concept_id,
			 competencies, expected_answer, answer_options, explanation, explanation_latex,
			 tags, source, created_by, validated_by_math, version, is_active, metadata)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			 RETURNING id`,
			q.Statement, q.Latex, q.QuestionType, q.Difficulty, q.ConceptID,
			q.Competencies, q.ExpectedAnswer, q.AnswerOptions, q.Explanation,
			q.ExplanationLatex, q.Tags, q.Source, q.CreatedBy, q.ValidatedByMath,
			q.Version, q.IsActive, q.Metadata,
		).Scan(&id)
		if err != nil {
			log.Printf("[QUESTION] insert failed: %v", err)
			http.Error(w, `{"error":"failed to create question"}`, http.StatusInternalServerError)
			return
		}

		q.ID = id
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(q)
	}
}

func ListQuestionsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		query := `SELECT id, statement, latex, question_type, difficulty, concept_id,
			competencies, expected_answer, answer_options, explanation, explanation_latex,
			tags, source, created_by, validated_by_math, version, is_active, metadata, created_at
			FROM questions WHERE is_active = true`

		var args []interface{}
		argIdx := 1

		if conceptID := r.URL.Query().Get("concept_id"); conceptID != "" {
			query += fmt.Sprintf(" AND concept_id = $%d", argIdx)
			args = append(args, conceptID)
			argIdx++
		}

		if qType := r.URL.Query().Get("type"); qType != "" {
			query += fmt.Sprintf(" AND question_type = $%d", argIdx)
			args = append(args, qType)
			argIdx++
		}

		if minDiff := r.URL.Query().Get("min_difficulty"); minDiff != "" {
			query += fmt.Sprintf(" AND difficulty >= $%d", argIdx)
			args = append(args, minDiff)
			argIdx++
		}

		if maxDiff := r.URL.Query().Get("max_difficulty"); maxDiff != "" {
			query += fmt.Sprintf(" AND difficulty <= $%d", argIdx)
			args = append(args, maxDiff)
			argIdx++
		}

		if source := r.URL.Query().Get("source"); source != "" {
			query += fmt.Sprintf(" AND source = $%d", argIdx)
			args = append(args, source)
			argIdx++
		}

		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			var parsed int
			if _, err := fmt.Sscanf(l, "%d", &parsed); err == nil && parsed > 0 && parsed <= 1000 {
				limit = parsed
			}
		}
		query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
		args = append(args, limit)

		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			log.Printf("[QUESTION] list query failed: %v", err)
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		questions := make([]Question, 0)
		for rows.Next() {
			var q Question
			var createdAt string
			if err := rows.Scan(
				&q.ID, &q.Statement, &q.Latex, &q.QuestionType, &q.Difficulty, &q.ConceptID,
				&q.Competencies, &q.ExpectedAnswer, &q.AnswerOptions, &q.Explanation,
				&q.ExplanationLatex, &q.Tags, &q.Source, &q.CreatedBy, &q.ValidatedByMath,
				&q.Version, &q.IsActive, &q.Metadata, &createdAt,
			); err != nil {
				log.Printf("[QUESTION] scan failed: %v", err)
				continue
			}
			q.CreatedAt = createdAt
			questions = append(questions, q)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(questions)
	}
}

func GetQuestionHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionID := chi.URLParam(r, "questionID")

		var q Question
		var createdAt string
		err := db.QueryRow(r.Context(),
			`SELECT id, statement, latex, question_type, difficulty, concept_id,
			competencies, expected_answer, answer_options, explanation, explanation_latex,
			tags, source, created_by, validated_by_math, version, is_active, metadata, created_at
			FROM questions WHERE id = $1`, questionID,
		).Scan(
			&q.ID, &q.Statement, &q.Latex, &q.QuestionType, &q.Difficulty, &q.ConceptID,
			&q.Competencies, &q.ExpectedAnswer, &q.AnswerOptions, &q.Explanation,
			&q.ExplanationLatex, &q.Tags, &q.Source, &q.CreatedBy, &q.ValidatedByMath,
			&q.Version, &q.IsActive, &q.Metadata, &createdAt,
		)
		if err != nil {
			http.Error(w, `{"error":"question not found"}`, http.StatusNotFound)
			return
		}
		q.CreatedAt = createdAt

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(q)
	}
}

func UpdateQuestionHandler(db *pgxpool.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionID := chi.URLParam(r, "questionID")

		var existing Question
		err := db.QueryRow(r.Context(),
			`SELECT id, statement, latex, question_type, difficulty, concept_id,
			competencies, expected_answer, answer_options, explanation, explanation_latex,
			tags, source, created_by, validated_by_math, version, is_active, metadata, created_at
			FROM questions WHERE id = $1`, questionID,
		).Scan(
			&existing.ID, &existing.Statement, &existing.Latex, &existing.QuestionType,
			&existing.Difficulty, &existing.ConceptID, &existing.Competencies,
			&existing.ExpectedAnswer, &existing.AnswerOptions, &existing.Explanation,
			&existing.ExplanationLatex, &existing.Tags, &existing.Source, &existing.CreatedBy,
			&existing.ValidatedByMath, &existing.Version, &existing.IsActive, &existing.Metadata,
			&existing.CreatedAt,
		)
		if err != nil {
			http.Error(w, `{"error":"question not found"}`, http.StatusNotFound)
			return
		}

		var updates Question
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if updates.Statement != "" {
			existing.Statement = updates.Statement
		}
		if updates.Latex != "" {
			existing.Latex = updates.Latex
		}
		if updates.QuestionType != "" {
			existing.QuestionType = updates.QuestionType
		}
		if updates.Difficulty != 0 {
			existing.Difficulty = updates.Difficulty
		}
		if updates.ConceptID != "" {
			existing.ConceptID = updates.ConceptID
		}
		if updates.Competencies != nil {
			existing.Competencies = updates.Competencies
		}
		if updates.ExpectedAnswer != "" {
			existing.ExpectedAnswer = updates.ExpectedAnswer
		}
		if updates.AnswerOptions != nil {
			existing.AnswerOptions = updates.AnswerOptions
		}
		if updates.Explanation != "" {
			existing.Explanation = updates.Explanation
		}
		if updates.ExplanationLatex != "" {
			existing.ExplanationLatex = updates.ExplanationLatex
		}
		if updates.Tags != nil {
			existing.Tags = updates.Tags
		}
		if updates.Source != "" {
			existing.Source = updates.Source
		}
		if updates.Metadata != nil {
			existing.Metadata = updates.Metadata
		}

		if validationErr := ValidateQuestionByType(existing); validationErr != "" {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, validationErr), http.StatusBadRequest)
			return
		}

		if existing.ExpectedAnswer != "" {
			mathClient := NewMathClient(cfg)
			result, err := mathClient.Verify(existing.ExpectedAnswer, existing.ExpectedAnswer, "")
			existing.ValidatedByMath = err == nil && result != nil && result.Success
		}

		existing.Version++

		_, err = db.Exec(r.Context(),
			`UPDATE questions SET
			 statement = $1, latex = $2, question_type = $3, difficulty = $4, concept_id = $5,
			 competencies = $6, expected_answer = $7, answer_options = $8, explanation = $9,
			 explanation_latex = $10, tags = $11, source = $12, validated_by_math = $13,
			 version = $14, metadata = $15
			 WHERE id = $16`,
			existing.Statement, existing.Latex, existing.QuestionType, existing.Difficulty,
			existing.ConceptID, existing.Competencies, existing.ExpectedAnswer,
			existing.AnswerOptions, existing.Explanation, existing.ExplanationLatex,
			existing.Tags, existing.Source, existing.ValidatedByMath, existing.Version,
			existing.Metadata, questionID,
		)
		if err != nil {
			log.Printf("[QUESTION] update failed: %v", err)
			http.Error(w, `{"error":"failed to update question"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existing)
	}
}

func DeleteQuestionHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionID := chi.URLParam(r, "questionID")

		tag, err := db.Exec(r.Context(),
			`UPDATE questions SET is_active = false WHERE id = $1`, questionID,
		)
		if err != nil {
			log.Printf("[QUESTION] soft delete failed: %v", err)
			http.Error(w, `{"error":"failed to delete question"}`, http.StatusInternalServerError)
			return
		}
		if tag.RowsAffected() == 0 {
			http.Error(w, `{"error":"question not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "question deleted"})
	}
}

func ValidateQuestionHandler(db *pgxpool.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionID := chi.URLParam(r, "questionID")

		var q Question
		var createdAt string
		err := db.QueryRow(r.Context(),
			`SELECT id, statement, latex, question_type, difficulty, concept_id,
			competencies, expected_answer, answer_options, explanation, explanation_latex,
			tags, source, created_by, validated_by_math, version, is_active, metadata, created_at
			FROM questions WHERE id = $1`, questionID,
		).Scan(
			&q.ID, &q.Statement, &q.Latex, &q.QuestionType, &q.Difficulty, &q.ConceptID,
			&q.Competencies, &q.ExpectedAnswer, &q.AnswerOptions, &q.Explanation,
			&q.ExplanationLatex, &q.Tags, &q.Source, &q.CreatedBy, &q.ValidatedByMath,
			&q.Version, &q.IsActive, &q.Metadata, &createdAt,
		)
		if err != nil {
			http.Error(w, `{"error":"question not found"}`, http.StatusNotFound)
			return
		}

		validated := false
		if q.ExpectedAnswer != "" {
			mathClient := NewMathClient(cfg)
			result, err := mathClient.Verify(q.ExpectedAnswer, q.ExpectedAnswer, "")
			validated = err == nil && result != nil && result.Success
			if !validated {
				log.Printf("[QUESTION] math validation failed for question %s: expected_answer=%s", questionID, q.ExpectedAnswer)
			}
		}

		_, err = db.Exec(r.Context(),
			`UPDATE questions SET validated_by_math = $1 WHERE id = $2`,
			validated, questionID,
		)
		if err != nil {
			log.Printf("[QUESTION] update validation status failed: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"question_id":       questionID,
			"validated_by_math": validated,
			"expected_answer":   q.ExpectedAnswer,
		})
	}
}

func ValidateQuestionByType(q Question) string {
	switch q.QuestionType {
	case "multiple_choice", "true_false":
		if len(q.AnswerOptions) == 0 {
			return fmt.Sprintf("%s questions require answer_options with at least 2 options", q.QuestionType)
		}
		var options []json.RawMessage
		if err := json.Unmarshal(q.AnswerOptions, &options); err != nil {
			return "answer_options must be a valid JSON array"
		}
		if len(options) < 2 {
			return fmt.Sprintf("%s questions require at least 2 answer options", q.QuestionType)
		}
	case "numeric", "algebraic_expression", "equation":
		if strings.TrimSpace(q.ExpectedAnswer) == "" {
			return fmt.Sprintf("%s questions require expected_answer", q.QuestionType)
		}
	case "step_by_step":
		if strings.TrimSpace(q.ExpectedAnswer) == "" {
			return "step_by_step questions require expected_answer"
		}
	case "exercise":
		// exercise questions are free-form and require no specific validation
	default:
		return fmt.Sprintf("unknown question_type: %s", q.QuestionType)
	}
	return ""
}
