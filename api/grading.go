package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Rubric struct {
	ID           string          `json:"id"`
	AssessmentID string          `json:"assessment_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	RubricType   string          `json:"rubric_type"`
	MaxScore     float64         `json:"max_score"`
	Criteria     json.RawMessage `json:"criteria"`
}

type RubricCriterion struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	MaxPoints   float64       `json:"max_points"`
	Levels      []RubricLevel `json:"levels"`
}

type RubricLevel struct {
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Points      float64 `json:"points"`
}

func GradeRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/answer/{answerID}", func(w http.ResponseWriter, r *http.Request) {
			answerID := chi.URLParam(r, "answerID")
			var req struct {
				Score    float64 `json:"score"`
				Feedback string  `json:"feedback"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}

			err := ManualGradeAnswer(db, answerID, req.Score, req.Feedback)
			if err != nil {
				http.Error(w, `{"error":"failed to grade"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Post("/rubric/{assessmentID}", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			var rubric Rubric
			if err := json.NewDecoder(r.Body).Decode(&rubric); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			rubric.AssessmentID = assessmentID

			err := CreateRubric(db, &rubric)
			if err != nil {
				http.Error(w, `{"error":"failed to create rubric"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(rubric)
		})

		r.Get("/rubric/{assessmentID}", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			rubrics, err := GetRubrics(db, assessmentID)
			if err != nil {
				http.Error(w, `{"error":"failed to get rubrics"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rubrics)
		})

		r.Post("/evaluate/{answerID}", func(w http.ResponseWriter, r *http.Request) {
			answerID := chi.URLParam(r, "answerID")
			var req struct {
				RubricID string `json:"rubric_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}

			result, err := EvaluateWithRubric(db, answerID, req.RubricID)
			if err != nil {
				http.Error(w, `{"error":"evaluation failed"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/batch-grade/{assessmentID}", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			err := BatchAutoGrade(db, cfg, assessmentID)
			if err != nil {
				http.Error(w, `{"error":"batch grading failed"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func ManualGradeAnswer(db *pgxpool.Pool, answerID string, score float64, feedback string) error {
	ctx := context.Background()
	_, err := db.Exec(ctx,
		`UPDATE student_answers
		 SET score = $2, points_earned = $2 * points_possible, feedback = $3, grading_method = 'manual', graded_at = NOW()
		 WHERE id = $1`,
		answerID, score, feedback)
	return err
}

func CreateRubric(db *pgxpool.Pool, rubric *Rubric) error {
	ctx := context.Background()
	return db.QueryRow(ctx,
		`INSERT INTO rubrics (assessment_id, name, description, rubric_type, max_score, criteria)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		rubric.AssessmentID, rubric.Name, rubric.Description, rubric.RubricType, rubric.MaxScore, rubric.Criteria,
	).Scan(&rubric.ID)
}

func GetRubrics(db *pgxpool.Pool, assessmentID string) ([]Rubric, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT id, assessment_id, name, description, rubric_type, max_score, criteria
		 FROM rubrics WHERE assessment_id = $1`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rubrics []Rubric
	for rows.Next() {
		var r Rubric
		if err := rows.Scan(&r.ID, &r.AssessmentID, &r.Name, &r.Description, &r.RubricType, &r.MaxScore, &r.Criteria); err != nil {
			continue
		}
		rubrics = append(rubrics, r)
	}
	return rubrics, nil
}

func EvaluateWithRubric(db *pgxpool.Pool, answerID, rubricID string) (map[string]interface{}, error) {
	ctx := context.Background()

	var answer StudentAnswer
	err := db.QueryRow(ctx,
		`SELECT id, answer, score, points_earned, points_possible, feedback
		 FROM student_answers WHERE id = $1`, answerID,
	).Scan(&answer.ID, &answer.Answer, &answer.Score, &answer.PointsEarned, &answer.PointsPossible, &answer.Feedback)
	if err != nil {
		return nil, err
	}

	var rubric Rubric
	err = db.QueryRow(ctx,
		`SELECT id, name, criteria, max_score FROM rubrics WHERE id = $1`, rubricID,
	).Scan(&rubric.ID, &rubric.Name, &rubric.Criteria, &rubric.MaxScore)
	if err != nil {
		return nil, err
	}

	var criteria []RubricCriterion
	json.Unmarshal(rubric.Criteria, &criteria)

	totalScore := 0.0
	maxPossible := 0.0
	evaluation := []map[string]interface{}{}

	for _, c := range criteria {
		maxPossible += c.MaxPoints
		bestLevel := RubricLevel{Points: 0}
		for _, l := range c.Levels {
			if l.Points > bestLevel.Points {
				bestLevel = l
			}
		}
		totalScore += bestLevel.Points
		evaluation = append(evaluation, map[string]interface{}{
			"criterion":   c.Name,
			"level":       bestLevel.Label,
			"points":      bestLevel.Points,
			"max_points":  c.MaxPoints,
			"description": bestLevel.Description,
		})
	}

	normalizedScore := 0.0
	if maxPossible > 0 {
		normalizedScore = totalScore / maxPossible
	}

	rubricScores, _ := json.Marshal(map[string]interface{}{
		"rubric_id":   rubricID,
		"rubric_name": rubric.Name,
		"total":       totalScore,
		"max":         maxPossible,
		"normalized":  normalizedScore,
		"evaluation":  evaluation,
	})

	_, err = db.Exec(ctx,
		`UPDATE student_answers
		 SET rubric_scores = $2, score = $3, points_earned = $3 * points_possible, grading_method = 'hybrid', graded_at = NOW()
		 WHERE id = $1`,
		answerID, rubricScores, normalizedScore)
	if err != nil {
		log.Printf("[GRADING] failed to update answer: %v", err)
	}

	return map[string]interface{}{
		"answer_id":        answerID,
		"rubric_name":      rubric.Name,
		"total_score":      totalScore,
		"max_score":        maxPossible,
		"normalized_score": normalizedScore,
		"evaluation":       evaluation,
	}, nil
}

func BatchAutoGrade(db *pgxpool.Pool, cfg *config.Config, assessmentID string) error {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT sa.id, sa.answer, aq.exercise_id, aq.points
		 FROM student_answers sa
		 JOIN assessment_questions aq ON sa.question_id = aq.id
		 JOIN student_assessments sas ON sa.student_assessment_id = sas.id
		 WHERE sas.assessment_id = $1 AND sa.grading_method = 'auto' AND sa.score = 0`,
		assessmentID)
	if err != nil {
		return err
	}
	defer rows.Close()

	mathClient := NewMathClient(cfg)
	graded := 0

	for rows.Next() {
		var answerID, answer string
		var exerciseID *string
		var points int
		if err := rows.Scan(&answerID, &answer, &exerciseID, &points); err != nil {
			continue
		}

		if exerciseID == nil {
			continue
		}

		var expectedAnswer string
		db.QueryRow(ctx, `SELECT expected_answer FROM exercises WHERE id = $1`, *exerciseID).Scan(&expectedAnswer)

		verifyResult, err := mathClient.Verify(answer, expectedAnswer, "")
		if err != nil {
			continue
		}

		score := 0.0
		feedback := "Respuesta incorrecta"
		if verifyResult != nil && verifyResult.Success {
			score = 1.0
			feedback = "Correcto"
		} else if verifyResult != nil && verifyResult.Method != "" {
			score = 0.3
			feedback = "Parcialmente correcto"
		}

		pointsEarned := score * float64(points)
		db.Exec(ctx,
			`UPDATE student_answers
			 SET score = $2, points_earned = $3, feedback = $4, math_verified = true, graded_at = NOW()
			 WHERE id = $1`,
			answerID, score, pointsEarned, feedback)
		graded++
	}

	log.Printf("[GRADING] batch graded %d answers for assessment %s", graded, assessmentID)
	return nil
}
