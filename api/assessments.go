package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Assessment struct {
	ID               string          `json:"id"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	CourseID         string          `json:"course_id"`
	AssessmentType   string          `json:"assessment_type"`
	Mode             string          `json:"mode"`
	TimeLimitMinutes int             `json:"time_limit_minutes"`
	MaxAttempts      int             `json:"max_attempts"`
	ShuffleQuestions bool            `json:"shuffle_questions"`
	ShowResults      bool            `json:"show_results"`
	ShowSolutions    bool            `json:"show_solutions"`
	PassingScore     float64         `json:"passing_score"`
	TotalPoints      int             `json:"total_points"`
	CreatedBy        string          `json:"created_by"`
	Status           string          `json:"status"`
	PublishedAt      *string         `json:"published_at,omitempty"`
	ExpiresAt        *string         `json:"expires_at,omitempty"`
	Metadata         json.RawMessage `json:"metadata"`
}

type AssessmentQuestion struct {
	ID                string          `json:"id"`
	AssessmentID      string          `json:"assessment_id"`
	ExerciseID        *string         `json:"exercise_id,omitempty"`
	QuestionOrder     int             `json:"question_order"`
	Points            int             `json:"points"`
	QuestionType      string          `json:"question_type"`
	StatementOverride string          `json:"statement_override"`
	Metadata          json.RawMessage `json:"metadata"`
	Exercise          *Exercise       `json:"exercise,omitempty"`
}

type StudentAssessment struct {
	ID               string  `json:"id"`
	StudentID        string  `json:"student_id"`
	AssessmentID     string  `json:"assessment_id"`
	Status           string  `json:"status"`
	StartedAt        string  `json:"started_at"`
	SubmittedAt      *string `json:"submitted_at,omitempty"`
	TimeSpentSeconds int     `json:"time_spent_seconds"`
	AttemptNumber    int     `json:"attempt_number"`
	TotalScore       float64 `json:"total_score"`
	MaxScore         float64 `json:"max_score"`
	Percentage       float64 `json:"percentage"`
	Passed           bool    `json:"passed"`
	Feedback         string  `json:"feedback"`
}

type StudentAnswer struct {
	ID                  string          `json:"id"`
	StudentAssessmentID string          `json:"student_assessment_id"`
	QuestionID          string          `json:"question_id"`
	Answer              string          `json:"answer"`
	Procedure           json.RawMessage `json:"procedure"`
	IsCorrect           bool            `json:"is_correct"`
	Score               float64         `json:"score"`
	PointsEarned        float64         `json:"points_earned"`
	PointsPossible      int             `json:"points_possible"`
	MathVerified        bool            `json:"math_verified"`
	RubricScores        json.RawMessage `json:"rubric_scores"`
	Feedback            string          `json:"feedback"`
	TimeSpentSeconds    int             `json:"time_spent_seconds"`
}

func AssessmentRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			teacherID := r.Context().Value(UserIDKey).(string)
			var req struct {
				Title            string   `json:"title"`
				Description      string   `json:"description"`
				CourseID         string   `json:"course_id"`
				AssessmentType   string   `json:"assessment_type"`
				Mode             string   `json:"mode"`
				TimeLimitMinutes int      `json:"time_limit_minutes"`
				MaxAttempts      int      `json:"max_attempts"`
				PassingScore     float64  `json:"passing_score"`
				TotalPoints      int      `json:"total_points"`
				QuestionIDs      []string `json:"question_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.CourseID == "" {
				req.CourseID = "matematica-1"
			}
			if req.AssessmentType == "" {
				req.AssessmentType = "formative"
			}
			if req.TotalPoints == 0 {
				req.TotalPoints = 100
			}
			if req.PassingScore == 0 {
				req.PassingScore = 0.6
			}

			assessment, err := CreateAssessment(db, teacherID, &req)
			if err != nil {
				log.Printf("[ASSESSMENT] create error: %v", err)
				http.Error(w, `{"error":"failed to create assessment"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(assessment)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			assessmentType := r.URL.Query().Get("type")
			status := r.URL.Query().Get("status")

			assessments, err := ListAssessments(db, courseID, assessmentType, status)
			if err != nil {
				http.Error(w, `{"error":"failed to list assessments"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assessments)
		})

		r.Get("/{assessmentID}", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			assessment, err := GetAssessment(db, assessmentID)
			if err != nil {
				http.Error(w, `{"error":"assessment not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assessment)
		})

		r.Put("/{assessmentID}", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			var updates map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if err := UpdateAssessment(db, assessmentID, updates); err != nil {
				http.Error(w, `{"error":"failed to update"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Delete("/{assessmentID}", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			if err := DeleteAssessment(db, assessmentID); err != nil {
				http.Error(w, `{"error":"failed to delete"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Post("/{assessmentID}/publish", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			if err := PublishAssessment(db, assessmentID); err != nil {
				http.Error(w, `{"error":"failed to publish"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Post("/{assessmentID}/start", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			assessmentID := chi.URLParam(r, "assessmentID")

			sa, questions, err := StartAssessment(db, studentID, assessmentID)
			if err != nil {
				log.Printf("[ASSESSMENT] start error: %v", err)
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"student_assessment": sa,
				"questions":          questions,
			})
		})

		r.Post("/{assessmentID}/submit", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			assessmentID := chi.URLParam(r, "assessmentID")
			var req struct {
				Answers []struct {
					QuestionID string   `json:"question_id"`
					Answer     string   `json:"answer"`
					Procedure  []string `json:"procedure,omitempty"`
				} `json:"answers"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}

			result, err := SubmitAssessment(db, cfg, studentID, assessmentID, req.Answers)
			if err != nil {
				log.Printf("[ASSESSMENT] submit error: %v", err)
				http.Error(w, `{"error":"failed to submit"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Get("/{assessmentID}/results", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			assessmentID := chi.URLParam(r, "assessmentID")

			result, err := GetAssessmentResult(db, studentID, assessmentID)
			if err != nil {
				http.Error(w, `{"error":"no results found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Get("/{assessmentID}/student-results", func(w http.ResponseWriter, r *http.Request) {
			assessmentID := chi.URLParam(r, "assessmentID")
			results, err := GetAssessmentStudentResults(db, assessmentID)
			if err != nil {
				http.Error(w, `{"error":"failed to get results"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		})
	}
}

func CreateAssessment(db *pgxpool.Pool, teacherID string, req *struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	CourseID         string   `json:"course_id"`
	AssessmentType   string   `json:"assessment_type"`
	Mode             string   `json:"mode"`
	TimeLimitMinutes int      `json:"time_limit_minutes"`
	MaxAttempts      int      `json:"max_attempts"`
	PassingScore     float64  `json:"passing_score"`
	TotalPoints      int      `json:"total_points"`
	QuestionIDs      []string `json:"question_ids"`
}) (*Assessment, error) {
	ctx := context.Background()
	var a Assessment
	err := db.QueryRow(ctx,
		`INSERT INTO assessments (title, description, course_id, assessment_type, mode, time_limit_minutes, max_attempts, passing_score, total_points, created_by, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'draft')
		 RETURNING id, title, description, course_id, assessment_type, mode, time_limit_minutes, max_attempts, shuffle_questions, show_results, show_solutions, passing_score, total_points, created_by, status, metadata`,
		req.Title, req.Description, req.CourseID, req.AssessmentType, req.Mode,
		req.TimeLimitMinutes, req.MaxAttempts, req.PassingScore, req.TotalPoints, teacherID,
	).Scan(&a.ID, &a.Title, &a.Description, &a.CourseID, &a.AssessmentType, &a.Mode,
		&a.TimeLimitMinutes, &a.MaxAttempts, &a.ShuffleQuestions, &a.ShowResults, &a.ShowSolutions,
		&a.PassingScore, &a.TotalPoints, &a.CreatedBy, &a.Status, &a.Metadata)
	if err != nil {
		return nil, err
	}

	for i, exID := range req.QuestionIDs {
		_, err := db.Exec(ctx,
			`INSERT INTO assessment_questions (assessment_id, exercise_id, question_order, points)
			 VALUES ($1, $2, $3, $4)`,
			a.ID, exID, i+1, req.TotalPoints/len(req.QuestionIDs))
		if err != nil {
			log.Printf("[ASSESSMENT] failed to add question: %v", err)
		}
	}

	return &a, nil
}

func GetAssessment(db *pgxpool.Pool, assessmentID string) (map[string]interface{}, error) {
	ctx := context.Background()
	var a Assessment
	err := db.QueryRow(ctx,
		`SELECT id, title, description, course_id, assessment_type, mode, time_limit_minutes, max_attempts, shuffle_questions, show_results, show_solutions, passing_score, total_points, created_by, status, metadata
		 FROM assessments WHERE id = $1`, assessmentID,
	).Scan(&a.ID, &a.Title, &a.Description, &a.CourseID, &a.AssessmentType, &a.Mode,
		&a.TimeLimitMinutes, &a.MaxAttempts, &a.ShuffleQuestions, &a.ShowResults, &a.ShowSolutions,
		&a.PassingScore, &a.TotalPoints, &a.CreatedBy, &a.Status, &a.Metadata)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx,
		`SELECT aq.id, aq.assessment_id, aq.exercise_id, aq.question_order, aq.points, aq.question_type, aq.statement_override, aq.metadata,
		        e.statement, e.latex, e.expected_answer, e.difficulty, e.concept_id
		 FROM assessment_questions aq
		 LEFT JOIN exercises e ON aq.exercise_id = e.id
		 WHERE aq.assessment_id = $1
		 ORDER BY aq.question_order`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []map[string]interface{}
	for rows.Next() {
		var q struct {
			ID                string
			AssessmentID      string
			ExerciseID        *string
			QuestionOrder     int
			Points            int
			QuestionType      string
			StatementOverride string
			Metadata          json.RawMessage
			Statement         *string
			Latex             *string
			ExpectedAnswer    *string
			Difficulty        *int
			ConceptID         *string
		}
		if err := rows.Scan(&q.ID, &q.AssessmentID, &q.ExerciseID, &q.QuestionOrder, &q.Points, &q.QuestionType, &q.StatementOverride, &q.Metadata,
			&q.Statement, &q.Latex, &q.ExpectedAnswer, &q.Difficulty, &q.ConceptID); err != nil {
			continue
		}
		question := map[string]interface{}{
			"id":                 q.ID,
			"question_order":     q.QuestionOrder,
			"points":             q.Points,
			"question_type":      q.QuestionType,
			"statement_override": q.StatementOverride,
		}
		if q.Statement != nil {
			question["statement"] = *q.Statement
		}
		if q.Latex != nil {
			question["latex"] = *q.Latex
		}
		if q.Difficulty != nil {
			question["difficulty"] = *q.Difficulty
		}
		if q.ConceptID != nil {
			question["concept_id"] = *q.ConceptID
		}
		questions = append(questions, question)
	}

	return map[string]interface{}{
		"assessment": a,
		"questions":  questions,
	}, nil
}

func ListAssessments(db *pgxpool.Pool, courseID, assessmentType, status string) ([]Assessment, error) {
	ctx := context.Background()
	query := `SELECT id, title, description, course_id, assessment_type, mode, time_limit_minutes, max_attempts, shuffle_questions, show_results, show_solutions, passing_score, total_points, created_by, status, metadata
	           FROM assessments WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if courseID != "" {
		query += ` AND course_id = $` + fmt.Sprintf("%d", argIdx)
		args = append(args, courseID)
		argIdx++
	}
	if assessmentType != "" {
		query += ` AND assessment_type = $` + fmt.Sprintf("%d", argIdx)
		args = append(args, assessmentType)
		argIdx++
	}
	if status != "" {
		query += ` AND status = $` + fmt.Sprintf("%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query += ` ORDER BY created_at DESC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assessments []Assessment
	for rows.Next() {
		var a Assessment
		if err := rows.Scan(&a.ID, &a.Title, &a.Description, &a.CourseID, &a.AssessmentType, &a.Mode,
			&a.TimeLimitMinutes, &a.MaxAttempts, &a.ShuffleQuestions, &a.ShowResults, &a.ShowSolutions,
			&a.PassingScore, &a.TotalPoints, &a.CreatedBy, &a.Status, &a.Metadata); err != nil {
			continue
		}
		assessments = append(assessments, a)
	}
	return assessments, nil
}

func UpdateAssessment(db *pgxpool.Pool, assessmentID string, updates map[string]interface{}) error {
	ctx := context.Background()
	_, err := db.Exec(ctx,
		`UPDATE assessments SET updated_at = NOW() WHERE id = $1`, assessmentID)
	return err
}

func DeleteAssessment(db *pgxpool.Pool, assessmentID string) error {
	ctx := context.Background()
	_, err := db.Exec(ctx, `DELETE FROM assessments WHERE id = $1`, assessmentID)
	return err
}

func PublishAssessment(db *pgxpool.Pool, assessmentID string) error {
	ctx := context.Background()
	_, err := db.Exec(ctx,
		`UPDATE assessments SET status = 'published', published_at = NOW(), updated_at = NOW() WHERE id = $1 AND status = 'draft'`,
		assessmentID)
	return err
}

func StartAssessment(db *pgxpool.Pool, studentID, assessmentID string) (*StudentAssessment, []AssessmentQuestion, error) {
	ctx := context.Background()

	var assessment Assessment
	err := db.QueryRow(ctx,
		`SELECT id, assessment_type, time_limit_minutes, max_attempts, status, passing_score, total_points
		 FROM assessments WHERE id = $1`, assessmentID,
	).Scan(&assessment.ID, &assessment.AssessmentType, &assessment.TimeLimitMinutes,
		&assessment.MaxAttempts, &assessment.Status, &assessment.PassingScore, &assessment.TotalPoints)
	if err != nil {
		return nil, nil, err
	}

	if assessment.Status != "published" {
		return nil, nil, fmt.Errorf("assessment is not published")
	}

	var attemptCount int
	db.QueryRow(ctx,
		`SELECT COUNT(*) FROM student_assessments WHERE student_id = $1 AND assessment_id = $2`,
		studentID, assessmentID).Scan(&attemptCount)

	if attemptCount >= assessment.MaxAttempts {
		return nil, nil, fmt.Errorf("maximum attempts reached")
	}

	var sa StudentAssessment
	err = db.QueryRow(ctx,
		`INSERT INTO student_assessments (student_id, assessment_id, attempt_number, max_score)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, student_id, assessment_id, status, started_at, attempt_number, total_score, max_score`,
		studentID, assessmentID, attemptCount+1, assessment.TotalPoints,
	).Scan(&sa.ID, &sa.StudentID, &sa.AssessmentID, &sa.Status, &sa.StartedAt,
		&sa.AttemptNumber, &sa.TotalScore, &sa.MaxScore)
	if err != nil {
		return nil, nil, err
	}

	rows, err := db.Query(ctx,
		`SELECT aq.id, aq.assessment_id, aq.exercise_id, aq.question_order, aq.points, aq.question_type, aq.statement_override, aq.metadata
		 FROM assessment_questions aq
		 WHERE aq.assessment_id = $1
		 ORDER BY aq.question_order`, assessmentID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var questions []AssessmentQuestion
	for rows.Next() {
		var q AssessmentQuestion
		if err := rows.Scan(&q.ID, &q.AssessmentID, &q.ExerciseID, &q.QuestionOrder, &q.Points, &q.QuestionType, &q.StatementOverride, &q.Metadata); err != nil {
			continue
		}
		questions = append(questions, q)
	}

	return &sa, questions, nil
}

func SubmitAssessment(db *pgxpool.Pool, cfg *config.Config, studentID, assessmentID string, answers []struct {
	QuestionID string   `json:"question_id"`
	Answer     string   `json:"answer"`
	Procedure  []string `json:"procedure,omitempty"`
}) (*map[string]interface{}, error) {
	ctx := context.Background()

	var sa StudentAssessment
	err := db.QueryRow(ctx,
		`SELECT id, assessment_id, status, attempt_number
		 FROM student_assessments
		 WHERE student_id = $1 AND assessment_id = $2 AND status = 'in_progress'
		 ORDER BY attempt_number DESC LIMIT 1`,
		studentID, assessmentID,
	).Scan(&sa.ID, &sa.AssessmentID, &sa.Status, &sa.AttemptNumber)
	if err != nil {
		return nil, fmt.Errorf("no active assessment found")
	}

	mathClient := NewMathClient(cfg)
	totalPoints := 0.0
	maxPoints := 0.0

	for _, ans := range answers {
		var question AssessmentQuestion
		err := db.QueryRow(ctx,
			`SELECT id, points, exercise_id FROM assessment_questions WHERE id = $1`, ans.QuestionID,
		).Scan(&question.ID, &question.Points, &question.ExerciseID)
		if err != nil {
			continue
		}

		maxPoints += float64(question.Points)
		score := 0.0
		isCorrect := false
		mathVerified := false
		feedback := ""

		if question.ExerciseID != nil {
			var expectedAnswer string
			db.QueryRow(ctx, `SELECT expected_answer FROM exercises WHERE id = $1`, *question.ExerciseID).Scan(&expectedAnswer)

			verifyResult, err := mathClient.Verify(ans.Answer, expectedAnswer, "")
			if err == nil && verifyResult != nil {
				isCorrect = verifyResult.Success
				mathVerified = true
				if isCorrect {
					score = 1.0
					feedback = "Correcto"
				} else {
					score = 0.3
					feedback = "Respuesta incorrecta"
				}
			} else {
				isCorrect = strings.EqualFold(strings.TrimSpace(ans.Answer), strings.TrimSpace(expectedAnswer))
				if isCorrect {
					score = 1.0
					feedback = "Correcto"
				} else {
					feedback = "Respuesta incorrecta"
				}
			}
		}

		pointsEarned := score * float64(question.Points)
		totalPoints += pointsEarned

		procedureJSON, _ := json.Marshal(ans.Procedure)
		_, err = db.Exec(ctx,
			`INSERT INTO student_answers (student_assessment_id, question_id, answer, procedure, is_correct, score, points_earned, points_possible, math_verified, feedback, grading_method)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			sa.ID, ans.QuestionID, ans.Answer, procedureJSON, isCorrect, score, pointsEarned, question.Points, mathVerified, feedback, "auto")
		if err != nil {
			log.Printf("[ASSESSMENT] failed to save answer: %v", err)
		}
	}

	percentage := 0.0
	if maxPoints > 0 {
		percentage = totalPoints / maxPoints
	}

	var passingScore float64
	db.QueryRow(ctx, `SELECT passing_score FROM assessments WHERE id = $1`, assessmentID).Scan(&passingScore)
	passed := percentage >= passingScore

	now := time.Now()
	_, err = db.Exec(ctx,
		`UPDATE student_assessments
		 SET status = 'graded', submitted_at = $3, total_score = $4, max_score = $5, percentage = $6, passed = $7, graded_at = $3, updated_at = NOW()
		 WHERE id = $1 AND student_id = $2`,
		sa.ID, studentID, now, totalPoints, maxPoints, percentage, passed)
	if err != nil {
		return nil, err
	}

	return &map[string]interface{}{
		"student_assessment_id": sa.ID,
		"total_score":           totalPoints,
		"max_score":             maxPoints,
		"percentage":            percentage,
		"passed":                passed,
		"status":                "graded",
	}, nil
}

func GetAssessmentResult(db *pgxpool.Pool, studentID, assessmentID string) (map[string]interface{}, error) {
	ctx := context.Background()
	var sa StudentAssessment
	err := db.QueryRow(ctx,
		`SELECT id, student_id, assessment_id, status, started_at, submitted_at, time_spent_seconds, attempt_number, total_score, max_score, percentage, passed, feedback
		 FROM student_assessments
		 WHERE student_id = $1 AND assessment_id = $2
		 ORDER BY attempt_number DESC LIMIT 1`,
		studentID, assessmentID,
	).Scan(&sa.ID, &sa.StudentID, &sa.AssessmentID, &sa.Status, &sa.StartedAt, &sa.SubmittedAt,
		&sa.TimeSpentSeconds, &sa.AttemptNumber, &sa.TotalScore, &sa.MaxScore, &sa.Percentage, &sa.Passed, &sa.Feedback)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx,
		`SELECT sa.id, sa.question_id, sa.answer, sa.is_correct, sa.score, sa.points_earned, sa.points_possible, sa.feedback, aq.question_order
		 FROM student_answers sa
		 JOIN assessment_questions aq ON sa.question_id = aq.id
		 WHERE sa.student_assessment_id = $1
		 ORDER BY aq.question_order`, sa.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var answers []map[string]interface{}
	for rows.Next() {
		var a struct {
			ID             string
			QuestionID     string
			Answer         string
			IsCorrect      bool
			Score          float64
			PointsEarned   float64
			PointsPossible int
			Feedback       string
			QuestionOrder  int
		}
		if err := rows.Scan(&a.ID, &a.QuestionID, &a.Answer, &a.IsCorrect, &a.Score, &a.PointsEarned, &a.PointsPossible, &a.Feedback, &a.QuestionOrder); err != nil {
			continue
		}
		answers = append(answers, map[string]interface{}{
			"id":              a.ID,
			"question_id":     a.QuestionID,
			"answer":          a.Answer,
			"is_correct":      a.IsCorrect,
			"score":           a.Score,
			"points_earned":   a.PointsEarned,
			"points_possible": a.PointsPossible,
			"feedback":        a.Feedback,
			"question_order":  a.QuestionOrder,
		})
	}

	return map[string]interface{}{
		"student_assessment": sa,
		"answers":            answers,
	}, nil
}

func GetAssessmentStudentResults(db *pgxpool.Pool, assessmentID string) ([]map[string]interface{}, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT sa.id, sa.student_id, u.name, sa.status, sa.total_score, sa.max_score, sa.percentage, sa.passed, sa.attempt_number, sa.submitted_at
		 FROM student_assessments sa
		 JOIN users u ON sa.student_id = u.id
		 WHERE sa.assessment_id = $1
		 ORDER BY sa.percentage DESC`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var r struct {
			ID            string
			StudentID     string
			StudentName   string
			Status        string
			TotalScore    float64
			MaxScore      float64
			Percentage    float64
			Passed        bool
			AttemptNumber int
			SubmittedAt   *string
		}
		if err := rows.Scan(&r.ID, &r.StudentID, &r.StudentName, &r.Status, &r.TotalScore, &r.MaxScore, &r.Percentage, &r.Passed, &r.AttemptNumber, &r.SubmittedAt); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":             r.ID,
			"student_id":     r.StudentID,
			"student_name":   r.StudentName,
			"status":         r.Status,
			"total_score":    r.TotalScore,
			"max_score":      r.MaxScore,
			"percentage":     r.Percentage,
			"passed":         r.Passed,
			"attempt_number": r.AttemptNumber,
			"submitted_at":   r.SubmittedAt,
		})
	}
	return results, nil
}
