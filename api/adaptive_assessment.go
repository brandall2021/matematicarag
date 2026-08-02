package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdaptiveAssessment struct {
	StudentID         string   `json:"student_id"`
	AssessmentID      string   `json:"assessment_id"`
	CurrentLevel      int      `json:"current_level"`
	TargetLevel       int      `json:"target_level"`
	Performance       float64  `json:"performance"`
	ConceptsCovered   []string `json:"concepts_covered"`
	QuestionsAnswered int      `json:"questions_answered"`
	CorrectAnswers    int      `json:"correct_answers"`
}

func NewAdaptiveAssessment(db *pgxpool.Pool, studentID, assessmentID string) (*AdaptiveAssessment, error) {
	ctx := context.Background()

	var avgScore float64
	err := db.QueryRow(ctx,
		`SELECT COALESCE(AVG(percentage), 0)
		 FROM student_assessments
		 WHERE student_id = $1 AND status = 'graded'`,
		studentID,
	).Scan(&avgScore)
	if err != nil {
		log.Printf("[ADAPTIVE] failed to query avg score: %v", err)
		avgScore = 0
	}

	var level int
	switch {
	case avgScore >= 0.8:
		level = 4
	case avgScore >= 0.6:
		level = 3
	case avgScore >= 0.4:
		level = 2
	default:
		level = 1
	}

	return &AdaptiveAssessment{
		StudentID:         studentID,
		AssessmentID:      assessmentID,
		CurrentLevel:      level,
		TargetLevel:       5,
		Performance:       0,
		ConceptsCovered:   []string{},
		QuestionsAnswered: 0,
		CorrectAnswers:    0,
	}, nil
}

func (aa *AdaptiveAssessment) GetNextQuestion(db *pgxpool.Pool) (*Question, error) {
	ctx := context.Background()

	query := `SELECT id, statement, latex, question_type, difficulty, concept_id,
		 competencies, expected_answer, answer_options, explanation, explanation_latex,
		 tags, source, created_by, validated_by_math, version, is_active, metadata
		 FROM question_bank
		 WHERE difficulty = $1 AND is_active = true`
	args := []any{aa.CurrentLevel}

	if len(aa.ConceptsCovered) > 0 {
		placeholders := make([]string, 0, len(aa.ConceptsCovered))
		for i := range aa.ConceptsCovered {
			placeholders = append(placeholders, fmt.Sprintf("$%d", i+2))
			args = append(args, aa.ConceptsCovered[i])
		}
		query += " AND concept_id NOT IN (" + strings.Join(placeholders, ",") + ")"
	}
	query += " ORDER BY RANDOM() LIMIT 1"

	var q Question
	var createdAt *string
	err := db.QueryRow(ctx, query, args...).Scan(
		&q.ID, &q.Statement, &q.Latex, &q.QuestionType, &q.Difficulty, &q.ConceptID,
		&q.Competencies, &q.ExpectedAnswer, &q.AnswerOptions, &q.Explanation,
		&q.ExplanationLatex, &q.Tags, &q.Source, &q.CreatedBy, &q.ValidatedByMath,
		&q.Version, &q.IsActive, &q.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("no questions available at level %d: %w", aa.CurrentLevel, err)
	}
	if createdAt != nil {
		q.CreatedAt = *createdAt
	}

	return &q, nil
}

func (aa *AdaptiveAssessment) UpdatePerformance(answeredCorrectly bool) {
	aa.QuestionsAnswered++

	if answeredCorrectly {
		aa.CorrectAnswers++
	}

	if aa.QuestionsAnswered > 0 {
		aa.Performance = float64(aa.CorrectAnswers) / float64(aa.QuestionsAnswered)
	}

	if answeredCorrectly {
		aa.TargetLevel = aa.CurrentLevel + 1
		if aa.TargetLevel > 5 {
			aa.TargetLevel = 5
		}
	} else {
		aa.TargetLevel = aa.CurrentLevel - 1
		if aa.TargetLevel < 1 {
			aa.TargetLevel = 1
		}
	}

	aa.adjustLevel()
}

func (aa *AdaptiveAssessment) adjustLevel() {
	if aa.QuestionsAnswered < 3 {
		return
	}

	rr := float64(aa.CorrectAnswers) / float64(aa.QuestionsAnswered)

	if rr >= 0.8 {
		aa.CurrentLevel = int(math.Min(float64(aa.CurrentLevel+1), 5))
	} else if rr < 0.4 {
		aa.CurrentLevel = int(math.Max(float64(aa.CurrentLevel-1), 1))
	}

	if aa.CurrentLevel < 1 {
		aa.CurrentLevel = 1
	}
	if aa.CurrentLevel > 5 {
		aa.CurrentLevel = 5
	}
}

func AdaptiveNextHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentID := r.Context().Value(UserIDKey).(string)
		assessmentID := chi.URLParam(r, "assessmentID")

		var req struct {
			AnsweredCorrectly bool   `json:"answered_correctly"`
			QuestionID        string `json:"question_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		aa, err := loadOrCreateAdaptiveAssessment(db, studentID, assessmentID)
		if err != nil {
			log.Printf("[ADAPTIVE] load error: %v", err)
			http.Error(w, `{"error":"failed to load adaptive state"}`, http.StatusInternalServerError)
			return
		}

		aa.UpdatePerformance(req.AnsweredCorrectly)

		if req.QuestionID != "" {
			var conceptID string
			err := db.QueryRow(r.Context(),
				`SELECT concept_id FROM question_bank WHERE id = $1`, req.QuestionID,
			).Scan(&conceptID)
			if err == nil && conceptID != "" {
				aa.ConceptsCovered = append(aa.ConceptsCovered, conceptID)
			}
		}

		nextQ, err := aa.GetNextQuestion(db)
		if err != nil {
			log.Printf("[ADAPTIVE] no next question: %v", err)
			http.Error(w, `{"error":"no more questions available at current level"}`, http.StatusNotFound)
			return
		}

		if err := saveAdaptiveAssessment(db, aa); err != nil {
			log.Printf("[ADAPTIVE] save error: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"next_question": nextQ,
			"adaptive_state": map[string]interface{}{
				"current_level":      aa.CurrentLevel,
				"performance":        aa.Performance,
				"questions_answered": aa.QuestionsAnswered,
				"correct_answers":    aa.CorrectAnswers,
				"concepts_covered":   aa.ConceptsCovered,
			},
		})
	}
}

func loadOrCreateAdaptiveAssessment(db *pgxpool.Pool, studentID, assessmentID string) (*AdaptiveAssessment, error) {
	ctx := context.Background()

	var metadata json.RawMessage
	err := db.QueryRow(ctx,
		`SELECT metadata FROM student_assessments
		 WHERE student_id = $1 AND assessment_id = $2 AND status = 'in_progress'
		 ORDER BY attempt_number DESC LIMIT 1`,
		studentID, assessmentID,
	).Scan(&metadata)

	if err == nil && metadata != nil {
		var state AdaptiveAssessment
		if err := json.Unmarshal(metadata, &state); err == nil {
			if state.StudentID == studentID && state.AssessmentID == assessmentID {
				return &state, nil
			}
		}
	}

	return NewAdaptiveAssessment(db, studentID, assessmentID)
}

func saveAdaptiveAssessment(db *pgxpool.Pool, aa *AdaptiveAssessment) error {
	ctx := context.Background()

	stateJSON, err := json.Marshal(aa)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx,
		`UPDATE student_assessments
		 SET metadata = $3
		 WHERE id = (
			SELECT id FROM student_assessments
			WHERE student_id = $1 AND assessment_id = $2 AND status = 'in_progress'
			ORDER BY attempt_number DESC LIMIT 1
		 )`,
		aa.StudentID, aa.AssessmentID, stateJSON,
	)
	return err
}
