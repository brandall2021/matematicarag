package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecoveryPlan struct {
	ID                    string          `json:"id"`
	StudentID             string          `json:"student_id"`
	CourseID              string          `json:"course_id"`
	TriggerAssessmentID   *string         `json:"trigger_assessment_id,omitempty"`
	TriggerScore          float64         `json:"trigger_score"`
	Status                string          `json:"status"`
	Priority              int             `json:"priority"`
	ConceptsToReview      json.RawMessage `json:"concepts_to_review"`
	RecommendedActivities json.RawMessage `json:"recommended_activities"`
	TargetDate            *string         `json:"target_date,omitempty"`
	CompletedAt           *string         `json:"completed_at,omitempty"`
}

func RecoveryRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req struct {
				AssessmentID string  `json:"assessment_id"`
				Score        float64 `json:"score"`
				CourseID     string  `json:"course_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.CourseID == "" {
				req.CourseID = "matematica-1"
			}

			plan, err := CreateRecoveryPlan(db, studentID, req.AssessmentID, req.Score, req.CourseID)
			if err != nil {
				log.Printf("[RECOVERY] create error: %v", err)
				http.Error(w, `{"error":"failed to create recovery plan"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(plan)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			courseID := r.URL.Query().Get("course_id")

			plans, err := GetRecoveryPlans(db, studentID, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to get plans"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(plans)
		})

		r.Put("/{planID}/complete", func(w http.ResponseWriter, r *http.Request) {
			planID := chi.URLParam(r, "planID")
			if err := CompleteRecoveryPlan(db, planID); err != nil {
				http.Error(w, `{"error":"failed to complete plan"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Put("/{planID}/cancel", func(w http.ResponseWriter, r *http.Request) {
			planID := chi.URLParam(r, "planID")
			if err := CancelRecoveryPlan(db, planID); err != nil {
				http.Error(w, `{"error":"failed to cancel plan"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func CreateRecoveryPlan(db *pgxpool.Pool, studentID, assessmentID string, score float64, courseID string) (*RecoveryPlan, error) {
	ctx := context.Background()

	var weakConcepts []string
	db.QueryRow(ctx,
		`SELECT COALESCE(array_agg(concept_id ORDER BY mastery ASC), '{}')
		 FROM concept_mastery
		 WHERE student_id = $1 AND mastery < 0.5
		 LIMIT 5`, studentID).Scan(&weakConcepts)

	conceptsJSON, _ := json.Marshal(weakConcepts)
	activitiesJSON, _ := json.Marshal([]map[string]interface{}{
		{"type": "practice", "description": "Practicar ejercicios de los conceptos débiles"},
		{"type": "review", "description": "Revisar teoría y ejemplos resueltos"},
		{"type": "tutor", "description": "Sesión de tutoría enfocada en conceptos específicos"},
	})

	priority := 3
	if score < 0.3 {
		priority = 5
	} else if score < 0.5 {
		priority = 4
	} else if score < 0.6 {
		priority = 3
	}

	targetDate := time.Now().AddDate(0, 0, 14)

	var plan RecoveryPlan
	err := db.QueryRow(ctx,
		`INSERT INTO recovery_plans (student_id, course_id, trigger_assessment_id, trigger_score, priority, concepts_to_review, recommended_activities, target_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, student_id, course_id, trigger_score, status, priority, concepts_to_review, recommended_activities`,
		studentID, courseID, assessmentID, score, priority, conceptsJSON, activitiesJSON, targetDate,
	).Scan(&plan.ID, &plan.StudentID, &plan.CourseID, &plan.TriggerScore, &plan.Status, &plan.Priority, &plan.ConceptsToReview, &plan.RecommendedActivities)
	if err != nil {
		return nil, err
	}

	return &plan, nil
}

func GetRecoveryPlans(db *pgxpool.Pool, studentID, courseID string) ([]RecoveryPlan, error) {
	ctx := context.Background()
	query := `SELECT id, student_id, course_id, trigger_assessment_id, trigger_score, status, priority, concepts_to_review, recommended_activities, target_date, completed_at
	          FROM recovery_plans WHERE student_id = $1`
	args := []interface{}{studentID}
	argIdx := 2

	if courseID != "" {
		query += ` AND course_id = $` + string(rune('0'+argIdx))
		args = append(args, courseID)
		argIdx++
	}

	query += ` ORDER BY priority DESC, created_at DESC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []RecoveryPlan
	for rows.Next() {
		var p RecoveryPlan
		if err := rows.Scan(&p.ID, &p.StudentID, &p.CourseID, &p.TriggerAssessmentID, &p.TriggerScore, &p.Status, &p.Priority, &p.ConceptsToReview, &p.RecommendedActivities, &p.TargetDate, &p.CompletedAt); err != nil {
			continue
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func CompleteRecoveryPlan(db *pgxpool.Pool, planID string) error {
	ctx := context.Background()
	_, err := db.Exec(ctx,
		`UPDATE recovery_plans SET status = 'completed', completed_at = NOW(), updated_at = NOW() WHERE id = $1 AND status = 'active'`,
		planID)
	return err
}

func CancelRecoveryPlan(db *pgxpool.Pool, planID string) error {
	ctx := context.Background()
	_, err := db.Exec(ctx,
		`UPDATE recovery_plans SET status = 'cancelled', updated_at = NOW() WHERE id = $1 AND status = 'active'`,
		planID)
	return err
}
