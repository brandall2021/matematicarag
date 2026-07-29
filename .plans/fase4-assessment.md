# Fase 4 — Sistema de Evaluación y Calificación Inteligente

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a comprehensive assessment and evaluation system that supports multiple assessment types (diagnostic, formative, summative, recovery), automated grading with mathematical equivalence validation, rubric-based evaluation, student performance analytics, competency tracking, personalized recovery plans, and early warning alerts for at-risk students.

**Architecture:** Extend the existing Go+Chi backend with new API route groups (`/api/assessments`, `/api/grading`, `/api/analytics/v2`, `/api/recovery`). Add PostgreSQL tables for assessments, questions, rubrics, grades, student analytics, recovery plans, and alerts. The grading engine combines Math Engine validation, rule-based scoring, and rubric evaluation. Frontend gets assessment-taking interface, teacher grading dashboard, analytics visualizations, and student recovery view.

**Tech Stack:** Go 1.25 + Chi v5, Python 3.11 + Flask + SymPy, Angular 20 + Material + KaTeX, PostgreSQL 16, Qdrant v1.12, OpenAI API

## Global Constraints

- Go backend uses `pgxpool.Pool` for all DB access (no ORM)
- Migrations are inline strings in `internal/database/database.go`
- All new API routes require `AuthMiddleware`; teacher routes additionally require `RoleMiddleware("TEACHER", "ADMIN")`
- Student data isolation: students see only their own data via `UserIDKey` from JWT
- Grading never relies on LLM alone — combine Math Engine + rules + rubrics + LLM when needed
- All generated questions must be validated by Math Engine before storage
- Assessments support partial scoring and mathematical equivalence (no string comparison for answers)
- Practice mode allows hints and multiple attempts; assessment mode is controlled/timed/no hints
- All configurable parameters use env vars with sensible defaults in `config.go`
- Frontend uses Angular signals, standalone components, Material Design
- No comments in code unless explicitly requested

---

## File Structure

### New Files (Backend)

| File | Responsibility |
|------|---------------|
| `api/assessments.go` | Assessment CRUD, question selection, submission |
| `api/grading.go` | Grading engine, rubric evaluation, partial scoring |
| `api/analytics_v2.go` | Student/course analytics, competency tracking |
| `api/recovery.go` | Recovery plans, personalized recommendations |
| `api/alerts.go` | Academic alerts, early warning system |

### New Files (Frontend)

| File | Responsibility |
|------|---------------|
| `frontend/src/app/modules/assessment/assessment.component.ts` | Assessment taking interface |
| `frontend/src/app/modules/analytics/analytics.component.ts` | Analytics dashboard |
| `frontend/src/app/core/services/assessment.service.ts` | HTTP client for assessment API |

### Modified Files

| File | Change |
|------|--------|
| `internal/database/database.go` | Add assessment system migration statements |
| `cmd/server/main.go` | Register new route groups |
| `internal/config/config.go` | Add assessment config params |
| `math-service/app.py` | Add `/math/grade-step` and `/math/validate-assessment` endpoints |
| `frontend/src/app/app.routes.ts` | Add assessment and analytics routes |
| `frontend/src/app/shared/layout.component.ts` | Add nav items for new pages |

---

## Task 1: Database Schema — Assessment System Tables

**Files:**
- Modify: `internal/database/database.go`

**Interfaces:**
- Produces: 7 new PostgreSQL tables + indexes for assessment system

This is the foundation. Every subsequent task depends on these tables existing.

- [ ] **Step 1: Add assessment tables to migrations**

Add the following migration strings to the `migrations` slice in `database.go`, after the existing learning system migrations:

```go
// Assessment System
`CREATE TABLE IF NOT EXISTS assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
    assessment_type VARCHAR(30) NOT NULL DEFAULT 'formative'
        CHECK (assessment_type IN ('diagnostic','formative','summative','recovery','practice')),
    mode VARCHAR(20) NOT NULL DEFAULT 'fixed'
        CHECK (mode IN ('fixed','generated','adaptive')),
    time_limit_minutes INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 1,
    shuffle_questions BOOLEAN DEFAULT true,
    show_results BOOLEAN DEFAULT true,
    show_solutions BOOLEAN DEFAULT false,
    passing_score REAL DEFAULT 0.6 CHECK (passing_score BETWEEN 0.0 AND 1.0),
    total_points INTEGER DEFAULT 100,
    created_by UUID REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','published','archived')),
    published_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_assessments_course ON assessments(course_id)`,
`CREATE INDEX IF NOT EXISTS idx_assessments_type ON assessments(assessment_type)`,
`CREATE INDEX IF NOT EXISTS idx_assessments_status ON assessments(status)`,

`CREATE TABLE IF NOT EXISTS assessment_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    exercise_id UUID REFERENCES exercises(id),
    question_order INTEGER NOT NULL DEFAULT 0,
    points INTEGER NOT NULL DEFAULT 10,
    question_type VARCHAR(20) NOT NULL DEFAULT 'exercise'
        CHECK (question_type IN ('exercise','generated','text','multiple_choice')),
    statement_override TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_assessment_questions_assessment ON assessment_questions(assessment_id)`,

`CREATE TABLE IF NOT EXISTS rubrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    rubric_type VARCHAR(20) NOT NULL DEFAULT 'analytic'
        CHECK (rubric_type IN ('analytic','holistic')),
    max_score REAL NOT NULL DEFAULT 1.0,
    criteria JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_rubrics_assessment ON rubrics(assessment_id)`,

`CREATE TABLE IF NOT EXISTS student_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress'
        CHECK (status IN ('in_progress','submitted','graded','returned')),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    submitted_at TIMESTAMP WITH TIME ZONE,
    graded_at TIMESTAMP WITH TIME ZONE,
    time_spent_seconds INTEGER DEFAULT 0,
    attempt_number INTEGER DEFAULT 1,
    total_score REAL DEFAULT 0.0,
    max_score REAL DEFAULT 0.0,
    percentage REAL DEFAULT 0.0,
    passed BOOLEAN DEFAULT false,
    graded_by UUID REFERENCES users(id),
    feedback TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(student_id, assessment_id, attempt_number)
)`,
`CREATE INDEX IF NOT EXISTS idx_student_assessments_student ON student_assessments(student_id)`,
`CREATE INDEX IF NOT EXISTS idx_student_assessments_assessment ON student_assessments(assessment_id)`,

`CREATE TABLE IF NOT EXISTS student_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_assessment_id UUID NOT NULL REFERENCES student_assessments(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES assessment_questions(id) ON DELETE CASCADE,
    answer TEXT DEFAULT '',
    procedure JSONB DEFAULT '[]',
    is_correct BOOLEAN DEFAULT false,
    score REAL DEFAULT 0.0 CHECK (score BETWEEN 0.0 AND 1.0),
    points_earned REAL DEFAULT 0.0,
    points_possible INTEGER DEFAULT 10,
    math_verified BOOLEAN DEFAULT false,
    rubric_scores JSONB DEFAULT '{}',
    feedback TEXT DEFAULT '',
    time_spent_seconds INTEGER DEFAULT 0,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    graded_at TIMESTAMP WITH TIME ZONE,
    grading_method VARCHAR(20) DEFAULT 'auto'
        CHECK (grading_method IN ('auto','manual','hybrid'))
)`,
`CREATE INDEX IF NOT EXISTS idx_student_answers_assessment ON student_answers(student_assessment_id)`,
`CREATE INDEX IF NOT EXISTS idx_student_answers_question ON student_answers(question_id)`,

`CREATE TABLE IF NOT EXISTS student_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
    total_assessments INTEGER DEFAULT 0,
    passed_assessments INTEGER DEFAULT 0,
    average_score REAL DEFAULT 0.0,
    average_time_seconds INTEGER DEFAULT 0,
    competency_level VARCHAR(20) DEFAULT 'beginner'
        CHECK (competency_level IN ('beginner','developing','proficient','advanced','exceptional')),
    competency_score REAL DEFAULT 0.0,
    weakest_concepts JSONB DEFAULT '[]',
    strongest_concepts JSONB DEFAULT '[]',
    improvement_trend REAL DEFAULT 0.0,
    study_streak_days INTEGER DEFAULT 0,
    last_assessment_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(student_id, course_id)
)`,
`CREATE INDEX IF NOT EXISTS idx_student_analytics_student ON student_analytics(student_id)`,

`CREATE TABLE IF NOT EXISTS recovery_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
    trigger_assessment_id UUID REFERENCES assessments(id),
    trigger_score REAL DEFAULT 0.0,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','completed','expired','cancelled')),
    priority INTEGER DEFAULT 1 CHECK (priority BETWEEN 1 AND 5),
    concepts_to_review JSONB DEFAULT '[]',
    recommended_activities JSONB DEFAULT '[]',
    target_date TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_recovery_plans_student ON recovery_plans(student_id)`,

`CREATE TABLE IF NOT EXISTS academic_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning'
        CHECK (severity IN ('info','warning','critical')),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    concept_id VARCHAR(100) DEFAULT '',
    assessment_id UUID REFERENCES assessments(id),
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_by UUID REFERENCES users(id),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_academic_alerts_student ON academic_alerts(student_id)`,
`CREATE INDEX IF NOT EXISTS idx_academic_alerts_type ON academic_alerts(alert_type)`,
`CREATE INDEX IF NOT EXISTS idx_academic_alerts_severity ON academic_alerts(severity)`,
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add internal/database/database.go
git commit -m "feat(fase4): add assessment system schema — assessments, questions, rubrics, grades, analytics, recovery, alerts"
```

---

## Task 2: Assessment CRUD & Question Selection

**Files:**
- Create: `api/assessments.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `AssessmentRoutes(db, cfg)` → route group
- Produces: `CreateAssessment()`, `GetAssessment()`, `ListAssessments()`, `UpdateAssessment()`, `DeleteAssessment()`
- Produces: `StartAssessment()`, `SubmitAssessment()`, `GetStudentAssessment()`

- [ ] **Step 1: Create `api/assessments.go`**

```go
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

type Assessment struct {
	ID                string          `json:"id"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	CourseID          string          `json:"course_id"`
	AssessmentType    string          `json:"assessment_type"`
	Mode              string          `json:"mode"`
	TimeLimitMinutes  int             `json:"time_limit_minutes"`
	MaxAttempts       int             `json:"max_attempts"`
	ShuffleQuestions  bool            `json:"shuffle_questions"`
	ShowResults       bool            `json:"show_results"`
	ShowSolutions     bool            `json:"show_solutions"`
	PassingScore      float64         `json:"passing_score"`
	TotalPoints       int             `json:"total_points"`
	CreatedBy         string          `json:"created_by"`
	Status            string          `json:"status"`
	PublishedAt       *string         `json:"published_at,omitempty"`
	ExpiresAt         *string         `json:"expires_at,omitempty"`
	Metadata          json.RawMessage `json:"metadata"`
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
	ID              string  `json:"id"`
	StudentID       string  `json:"student_id"`
	AssessmentID    string  `json:"assessment_id"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"started_at"`
	SubmittedAt     *string `json:"submitted_at,omitempty"`
	TimeSpentSeconds int    `json:"time_spent_seconds"`
	AttemptNumber   int     `json:"attempt_number"`
	TotalScore      float64 `json:"total_score"`
	MaxScore        float64 `json:"max_score"`
	Percentage      float64 `json:"percentage"`
	Passed          bool    `json:"passed"`
	Feedback        string  `json:"feedback"`
}

type StudentAnswer struct {
	ID                   string          `json:"id"`
	StudentAssessmentID  string          `json:"student_assessment_id"`
	QuestionID           string          `json:"question_id"`
	Answer               string          `json:"answer"`
	Procedure            json.RawMessage `json:"procedure"`
	IsCorrect            bool            `json:"is_correct"`
	Score                float64         `json:"score"`
	PointsEarned         float64         `json:"points_earned"`
	PointsPossible       int             `json:"points_possible"`
	MathVerified         bool            `json:"math_verified"`
	RubricScores         json.RawMessage `json:"rubric_scores"`
	Feedback             string          `json:"feedback"`
	TimeSpentSeconds     int             `json:"time_spent_seconds"`
}

func AssessmentRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			teacherID := r.Context().Value(UserIDKey).(string)
			var req struct {
				Title            string  `json:"title"`
				Description      string  `json:"description"`
				CourseID         string  `json:"course_id"`
				AssessmentType   string  `json:"assessment_type"`
				Mode             string  `json:"mode"`
				TimeLimitMinutes int     `json:"time_limit_minutes"`
				MaxAttempts      int     `json:"max_attempts"`
				PassingScore     float64 `json:"passing_score"`
				TotalPoints      int     `json:"total_points"`
				QuestionIDs     []string `json:"question_ids"`
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
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	CourseID         string  `json:"course_id"`
	AssessmentType   string  `json:"assessment_type"`
	Mode             string  `json:"mode"`
	TimeLimitMinutes int     `json:"time_limit_minutes"`
	MaxAttempts      int     `json:"max_attempts"`
	PassingScore     float64 `json:"passing_score"`
	TotalPoints      int     `json:"total_points"`
	QuestionIDs     []string `json:"question_ids"`
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
		query += ` AND course_id = $` + string(rune('0'+argIdx))
		args = append(args, courseID)
		argIdx++
	}
	if assessmentType != "" {
		query += ` AND assessment_type = $` + string(rune('0'+argIdx))
		args = append(args, assessmentType)
		argIdx++
	}
	if status != "" {
		query += ` AND status = $` + string(rune('0'+argIdx))
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
```

- [ ] **Step 2: Add import for `fmt` and `strings` to assessments.go**

Ensure the import section includes `"fmt"` and `"strings"`.

- [ ] **Step 3: Register AssessmentRoutes in main.go**

Add inside the auth group in `cmd/server/main.go`:

```go
r.Route("/assessments", api.AssessmentRoutes(db, cfg))
```

- [ ] **Step 4: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 5: Commit**

```bash
git add api/assessments.go cmd/server/main.go
git commit -m "feat(fase4): assessment CRUD, question selection, submission & results"
```

---

## Task 3: Grading Engine & Rubric Evaluation

**Files:**
- Create: `api/grading.go`
- Modify: `math-service/app.py`

**Interfaces:**
- Produces: `GradeRoutes(db, cfg)` → route group
- Produces: `GradeAnswer()`, `EvaluateRubric()`, `CalculatePartialScore()`
- Consumes: `MathClient` for equivalence checking

- [ ] **Step 1: Create `api/grading.go`**

```go
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
	Name        string            `json:"name"`
	Description string            `json:"description"`
	MaxPoints   float64           `json:"max_points"`
	Levels      []RubricLevel     `json:"levels"`
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
				Score   float64 `json:"score"`
				Feedback string `json:"feedback"`
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
			"criterion":     c.Name,
			"level":         bestLevel.Label,
			"points":        bestLevel.Points,
			"max_points":    c.MaxPoints,
			"description":   bestLevel.Description,
		})
	}

	normalizedScore := 0.0
	if maxPossible > 0 {
		normalizedScore = totalScore / maxPossible
	}

	rubricScores, _ := json.Marshal(map[string]interface{}{
		"rubric_id":  rubricID,
		"rubric_name": rubric.Name,
		"total":      totalScore,
		"max":        maxPossible,
		"normalized": normalizedScore,
		"evaluation": evaluation,
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
		"answer_id":       answerID,
		"rubric_name":     rubric.Name,
		"total_score":     totalScore,
		"max_score":       maxPossible,
		"normalized_score": normalizedScore,
		"evaluation":      evaluation,
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
```

- [ ] **Step 2: Register GradeRoutes in main.go**

Add inside the auth group:

```go
r.Route("/grading", api.GradeRoutes(db, cfg))
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add api/grading.go cmd/server/main.go
git commit -m "feat(fase4): grading engine — manual, rubric evaluation, batch auto-grade"
```

---

## Task 4: Student Analytics & Competency Tracking

**Files:**
- Create: `api/analytics_v2.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `AnalyticsRoutes(db, cfg)` → route group
- Produces: `UpdateStudentAnalytics()`, `GetStudentAnalytics()`, `GetCourseAnalytics()`
- Produces: `GetCompetencyReport()`, `GetPerformanceTrend()`

- [ ] **Step 1: Create `api/analytics_v2.go`**

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentAnalyticsData struct {
	ID                string          `json:"id"`
	StudentID         string          `json:"student_id"`
	CourseID          string          `json:"course_id"`
	TotalAssessments  int             `json:"total_assessments"`
	PassedAssessments int             `json:"passed_assessments"`
	AverageScore      float64         `json:"average_score"`
	AverageTimeSeconds int            `json:"average_time_seconds"`
	CompetencyLevel   string          `json:"competency_level"`
	CompetencyScore   float64         `json:"competency_score"`
	WeakestConcepts   json.RawMessage `json:"weakest_concepts"`
	StrongestConcepts json.RawMessage `json:"strongest_concepts"`
	ImprovementTrend  float64         `json:"improvement_trend"`
	StudyStreakDays   int             `json:"study_streak_days"`
}

type CourseAnalyticsData struct {
	CourseID          string  `json:"course_id"`
	TotalStudents     int     `json:"total_students"`
	TotalAssessments  int     `json:"total_assessments"`
	AverageScore      float64 `json:"average_score"`
	PassRate          float64 `json:"pass_rate"`
	AverageCompetency float64 `json:"average_competency"`
}

func AnalyticsRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/student/{studentID}", func(w http.ResponseWriter, r *http.Request) {
			studentID := chi.URLParam(r, "studentID")
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			analytics, err := GetStudentAnalytics(db, studentID, courseID)
			if err != nil {
				http.Error(w, `{"error":"no analytics found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(analytics)
		})

		r.Get("/course/{courseID}", func(w http.ResponseWriter, r *http.Request) {
			courseID := chi.URLParam(r, "courseID")

			analytics, err := GetCourseAnalytics(db, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to get analytics"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(analytics)
		})

		r.Get("/student/{studentID}/competency", func(w http.ResponseWriter, r *http.Request) {
			studentID := chi.URLParam(r, "studentID")
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			report, err := GetCompetencyReport(db, studentID, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to get competency report"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(report)
		})

		r.Get("/student/{studentID}/trend", func(w http.ResponseWriter, r *http.Request) {
			studentID := chi.URLParam(r, "studentID")
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			trend, err := GetPerformanceTrend(db, studentID, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to get trend"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(trend)
		})

		r.Post("/student/{studentID}/update", func(w http.ResponseWriter, r *http.Request) {
			studentID := chi.URLParam(r, "studentID")
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			err := UpdateStudentAnalytics(db, studentID, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to update analytics"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func UpdateStudentAnalytics(db *pgxpool.Pool, studentID, courseID string) error {
	ctx := context.Background()

	var totalAssessments, passedAssessments int
	var averageScore float64
	db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN passed THEN 1 ELSE 0 END), 0), COALESCE(AVG(percentage), 0)
		 FROM student_assessments sa
		 JOIN assessments a ON sa.assessment_id = a.id
		 WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.status = 'graded'`,
		studentID, courseID).Scan(&totalAssessments, &passedAssessments, &averageScore)

	var averageTime int
	db.QueryRow(ctx,
		`SELECT COALESCE(AVG(time_spent_seconds), 0)
		 FROM student_assessments sa
		 JOIN assessments a ON sa.assessment_id = a.id
		 WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.status = 'graded'`,
		studentID, courseID).Scan(&averageTime)

	competencyLevel := "beginner"
	competencyScore := averageScore
	switch {
	case competencyScore >= 0.9:
		competencyLevel = "exceptional"
	case competencyScore >= 0.8:
		competencyLevel = "advanced"
	case competencyScore >= 0.7:
		competencyLevel = "proficient"
	case competencyScore >= 0.5:
		competencyLevel = "developing"
	}

	var weakestConcepts, strongestConcepts json.RawMessage
	db.QueryRow(ctx,
		`SELECT COALESCE(json_agg(concept_id ORDER BY mastery ASC), '[]')
		 FROM concept_mastery
		 WHERE student_id = $1 AND mastery < 0.5
		 LIMIT 5`, studentID).Scan(&weakestConcepts)

	db.QueryRow(ctx,
		`SELECT COALESCE(json_agg(concept_id ORDER BY mastery DESC), '[]')
		 FROM concept_mastery
		 WHERE student_id = $1 AND mastery >= 0.7
		 LIMIT 5`, studentID).Scan(&strongestConcepts)

	var improvementTrend float64
	db.QueryRow(ctx,
		`SELECT COALESCE(
		   (SELECT AVG(percentage) FROM student_assessments sa
		    JOIN assessments a ON sa.assessment_id = a.id
		    WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.status = 'graded'
		    AND sa.submitted_at > NOW() - INTERVAL '30 days') -
		   (SELECT AVG(percentage) FROM student_assessments sa
		    JOIN assessments a ON sa.assessment_id = a.id
		    WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.status = 'graded'
		    AND sa.submitted_at BETWEEN NOW() - INTERVAL '60 days' AND NOW() - INTERVAL '30 days'),
		   0)`, studentID, courseID).Scan(&improvementTrend)

	_, err := db.Exec(ctx,
		`INSERT INTO student_analytics (student_id, course_id, total_assessments, passed_assessments, average_score, average_time_seconds, competency_level, competency_score, weakest_concepts, strongest_concepts, improvement_trend, last_assessment_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		 ON CONFLICT (student_id, course_id) DO UPDATE SET
		   total_assessments = $3, passed_assessments = $4, average_score = $5,
		   average_time_seconds = $6, competency_level = $7, competency_score = $8,
		   weakest_concepts = $9, strongest_concepts = $10, improvement_trend = $11,
		   last_assessment_at = NOW(), updated_at = NOW()`,
		studentID, courseID, totalAssessments, passedAssessments, averageScore,
		averageTime, competencyLevel, competencyScore, weakestConcepts, strongestConcepts, improvementTrend)
	return err
}

func GetStudentAnalytics(db *pgxpool.Pool, studentID, courseID string) (*StudentAnalyticsData, error) {
	ctx := context.Background()
	var a StudentAnalyticsData
	err := db.QueryRow(ctx,
		`SELECT id, student_id, course_id, total_assessments, passed_assessments, average_score, average_time_seconds, competency_level, competency_score, weakest_concepts, strongest_concepts, improvement_trend, study_streak_days
		 FROM student_analytics
		 WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID,
	).Scan(&a.ID, &a.StudentID, &a.CourseID, &a.TotalAssessments, &a.PassedAssessments,
		&a.AverageScore, &a.AverageTimeSeconds, &a.CompetencyLevel, &a.CompetencyScore,
		&a.WeakestConcepts, &a.StrongestConcepts, &a.ImprovementTrend, &a.StudyStreakDays)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func GetCourseAnalytics(db *pgxpool.Pool, courseID string) (*CourseAnalyticsData, error) {
	ctx := context.Background()
	var a CourseAnalyticsData
	a.CourseID = courseID

	db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT student_id) FROM student_analytics WHERE course_id = $1`,
		courseID).Scan(&a.TotalStudents)

	db.QueryRow(ctx,
		`SELECT COUNT(*) FROM assessments WHERE course_id = $1 AND status = 'published'`,
		courseID).Scan(&a.TotalAssessments)

	db.QueryRow(ctx,
		`SELECT COALESCE(AVG(percentage), 0) FROM student_assessments sa
		 JOIN assessments a ON sa.assessment_id = a.id
		 WHERE a.course_id = $1 AND sa.status = 'graded'`,
		courseID).Scan(&a.AverageScore)

	db.QueryRow(ctx,
		`SELECT COALESCE(AVG(CASE WHEN passed THEN 1.0 ELSE 0.0 END), 0) FROM student_assessments sa
		 JOIN assessments a ON sa.assessment_id = a.id
		 WHERE a.course_id = $1 AND sa.status = 'graded'`,
		courseID).Scan(&a.PassRate)

	db.QueryRow(ctx,
		`SELECT COALESCE(AVG(competency_score), 0) FROM student_analytics WHERE course_id = $1`,
		courseID).Scan(&a.AverageCompetency)

	return &a, nil
}

func GetCompetencyReport(db *pgxpool.Pool, studentID, courseID string) (map[string]interface{}, error) {
	ctx := context.Background()

	var analytics StudentAnalyticsData
	err := db.QueryRow(ctx,
		`SELECT competency_level, competency_score, weakest_concepts, strongest_concepts, improvement_trend
		 FROM student_analytics
		 WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID,
	).Scan(&analytics.CompetencyLevel, &analytics.CompetencyScore, &analytics.WeakestConcepts, &analytics.StrongestConcepts, &analytics.ImprovementTrend)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx,
		`SELECT cm.concept_id, c.name, cm.mastery, cm.status, cm.attempts, cm.correct
		 FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE cm.student_id = $1 AND c.course_id = $2
		 ORDER BY cm.mastery ASC`, studentID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concepts []map[string]interface{}
	for rows.Next() {
		var conceptID, name, status string
		var mastery float64
		var attempts, correct int
		if err := rows.Scan(&conceptID, &name, &mastery, &status, &attempts, &correct); err != nil {
			continue
		}
		concepts = append(concepts, map[string]interface{}{
			"concept_id": conceptID,
			"name":       name,
			"mastery":    mastery,
			"status":     status,
			"attempts":   attempts,
			"correct":    correct,
		})
	}

	return map[string]interface{}{
		"competency_level":  analytics.CompetencyLevel,
		"competency_score":  analytics.CompetencyScore,
		"weakest_concepts":  analytics.WeakestConcepts,
		"strongest_concepts": analytics.StrongestConcepts,
		"improvement_trend": analytics.ImprovementTrend,
		"concept_details":   concepts,
	}, nil
}

func GetPerformanceTrend(db *pgxpool.Pool, studentID, courseID string) (map[string]interface{}, error) {
	ctx := context.Background()

	rows, err := db.Query(ctx,
		`SELECT DATE_TRUNC('week', submitted_at) as week, AVG(percentage) as avg_score, COUNT(*) as assessment_count
		 FROM student_assessments sa
		 JOIN assessments a ON sa.assessment_id = a.id
		 WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.status = 'graded'
		 GROUP BY DATE_TRUNC('week', submitted_at)
		 ORDER BY week DESC
		 LIMIT 12`, studentID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weeks []map[string]interface{}
	for rows.Next() {
		var week string
		var avgScore float64
		var count int
		if err := rows.Scan(&week, &avgScore, &count); err != nil {
			continue
		}
		weeks = append(weeks, map[string]interface{}{
			"week":            week,
			"average_score":   avgScore,
			"assessment_count": count,
		})
	}

	return map[string]interface{}{
		"student_id": studentID,
		"course_id":  courseID,
		"trend":      weeks,
	}, nil
}
```

- [ ] **Step 2: Register AnalyticsRoutes in main.go**

Add inside the auth group:

```go
r.Route("/analytics/v2", api.AnalyticsRoutes(db, cfg))
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add api/analytics_v2.go cmd/server/main.go
git commit -m "feat(fase4): student analytics & competency tracking"
```

---

## Task 5: Recovery Plans & Academic Alerts

**Files:**
- Create: `api/recovery.go`
- Create: `api/alerts.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `RecoveryRoutes(db, cfg)` → route group
- Produces: `AlertRoutes(db, cfg)` → route group
- Produces: `CreateRecoveryPlan()`, `GetRecoveryPlans()`, `CompleteRecoveryPlan()`
- Produces: `CreateAlert()`, `GetAlerts()`, `AcknowledgeAlert()`, `CheckForAlerts()`

- [ ] **Step 1: Create `api/recovery.go`**

```go
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
	ID                   string          `json:"id"`
	StudentID            string          `json:"student_id"`
	CourseID             string          `json:"course_id"`
	TriggerAssessmentID  *string         `json:"trigger_assessment_id,omitempty"`
	TriggerScore         float64         `json:"trigger_score"`
	Status               string          `json:"status"`
	Priority             int             `json:"priority"`
	ConceptsToReview     json.RawMessage `json:"concepts_to_review"`
	RecommendedActivities json.RawMessage `json:"recommended_activities"`
	TargetDate           *string         `json:"target_date,omitempty"`
	CompletedAt          *string         `json:"completed_at,omitempty"`
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
```

- [ ] **Step 2: Create `api/alerts.go`**

```go
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

type AcademicAlert struct {
	ID              string          `json:"id"`
	StudentID       string          `json:"student_id"`
	AlertType       string          `json:"alert_type"`
	Severity        string          `json:"severity"`
	Title           string          `json:"title"`
	Message         string          `json:"message"`
	ConceptID       string          `json:"concept_id"`
	AssessmentID    *string         `json:"assessment_id,omitempty"`
	Acknowledged    bool            `json:"acknowledged"`
	AcknowledgedBy  *string         `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *string         `json:"acknowledged_at,omitempty"`
	Metadata        json.RawMessage `json:"metadata"`
}

func AlertRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			severity := r.URL.Query().Get("severity")

			alerts, err := GetAlerts(db, studentID, severity)
			if err != nil {
				http.Error(w, `{"error":"failed to get alerts"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(alerts)
		})

		r.Put("/{alertID}/acknowledge", func(w http.ResponseWriter, r *http.Request) {
			alertID := chi.URLParam(r, "alertID")
			userID := r.Context().Value(UserIDKey).(string)

			if err := AcknowledgeAlert(db, alertID, userID); err != nil {
				http.Error(w, `{"error":"failed to acknowledge"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Post("/check", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			alerts, err := CheckForAlerts(db, studentID, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to check alerts"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"alerts_created": len(alerts),
				"alerts":         alerts,
			})
		})

		r.Get("/all", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			severity := r.URL.Query().Get("severity")

			alerts, err := GetAllAlerts(db, courseID, severity)
			if err != nil {
				http.Error(w, `{"error":"failed to get alerts"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(alerts)
		})
	}
}

func CreateAlert(db *pgxpool.Pool, studentID, alertType, severity, title, message, conceptID string, metadata json.RawMessage) (*AcademicAlert, error) {
	ctx := context.Background()
	var alert AcademicAlert
	err := db.QueryRow(ctx,
		`INSERT INTO academic_alerts (student_id, alert_type, severity, title, message, concept_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, student_id, alert_type, severity, title, message, concept_id, acknowledged, metadata`,
		studentID, alertType, severity, title, message, conceptID, metadata,
	).Scan(&alert.ID, &alert.StudentID, &alert.AlertType, &alert.Severity, &alert.Title, &alert.Message, &alert.ConceptID, &alert.Acknowledged, &alert.Metadata)
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func GetAlerts(db *pgxpool.Pool, studentID, severity string) ([]AcademicAlert, error) {
	ctx := context.Background()
	query := `SELECT id, student_id, alert_type, severity, title, message, concept_id, assessment_id, acknowledged, acknowledged_by, acknowledged_at, metadata
	          FROM academic_alerts WHERE student_id = $1`
	args := []interface{}{studentID}
	argIdx := 2

	if severity != "" {
		query += ` AND severity = $` + string(rune('0'+argIdx))
		args = append(args, severity)
		argIdx++
	}

	query += ` ORDER BY created_at DESC LIMIT 50`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []AcademicAlert
	for rows.Next() {
		var a AcademicAlert
		if err := rows.Scan(&a.ID, &a.StudentID, &a.AlertType, &a.Severity, &a.Title, &a.Message, &a.ConceptID, &a.AssessmentID, &a.Acknowledged, &a.AcknowledgedBy, &a.AcknowledgedAt, &a.Metadata); err != nil {
			continue
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

func AcknowledgeAlert(db *pgxpool.Pool, alertID, userID string) error {
	ctx := context.Background()
	_, err := db.Exec(ctx,
		`UPDATE academic_alerts SET acknowledged = true, acknowledged_by = $2, acknowledged_at = NOW() WHERE id = $1 AND acknowledged = false`,
		alertID, userID)
	return err
}

func CheckForAlerts(db *pgxpool.Pool, studentID, courseID string) ([]AcademicAlert, error) {
	ctx := context.Background()
	var alerts []AcademicAlert

	var failedCount int
	db.QueryRow(ctx,
		`SELECT COUNT(*) FROM student_assessments sa
		 JOIN assessments a ON sa.assessment_id = a.id
		 WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.passed = false AND sa.status = 'graded'`,
		studentID, courseID).Scan(&failedCount)

	if failedCount >= 3 {
		alert, err := CreateAlert(db, studentID, "multiple_failures", "critical",
			"Múltiples evaluaciones reprobadas",
			"Has reprobado "+string(rune('0'+failedCount))+" evaluaciones. Se recomienda un plan de recuperación.",
			"", nil)
		if err == nil {
			alerts = append(alerts, *alert)
		}
	}

	var lowMasteryConcepts int
	db.QueryRow(ctx,
		`SELECT COUNT(*) FROM concept_mastery
		 WHERE student_id = $1 AND mastery < 0.3 AND attempts >= 3`,
		studentID).Scan(&lowMasteryConcepts)

	if lowMasteryConcepts > 0 {
		alert, err := CreateAlert(db, studentID, "low_mastery", "warning",
			"Conceptos con bajo dominio",
			"Tienes "+string(rune('0'+lowMasteryConcepts))+" conceptos con dominio bajo. Considera practicar más.",
			"", nil)
		if err == nil {
			alerts = append(alerts, *alert)
		}
	}

	return alerts, nil
}

func GetAllAlerts(db *pgxpool.Pool, courseID, severity string) ([]map[string]interface{}, error) {
	ctx := context.Background()
	query := `SELECT aa.id, aa.student_id, u.name, aa.alert_type, aa.severity, aa.title, aa.message, aa.acknowledged, aa.created_at
	          FROM academic_alerts aa
	          JOIN users u ON aa.student_id = u.id
	          WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if courseID != "" {
		query += ` AND aa.student_id IN (SELECT student_id FROM student_analytics WHERE course_id = $` + string(rune('0'+argIdx)) + `)`
		args = append(args, courseID)
		argIdx++
	}
	if severity != "" {
		query += ` AND aa.severity = $` + string(rune('0'+argIdx))
		args = append(args, severity)
		argIdx++
	}

	query += ` ORDER BY aa.created_at DESC LIMIT 100`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, studentID, studentName, alertType, severity, title, message string
		var acknowledged bool
		var createdAt string
		if err := rows.Scan(&id, &studentID, &studentName, &alertType, &severity, &title, &message, &acknowledged, &createdAt); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":           id,
			"student_id":   studentID,
			"student_name": studentName,
			"alert_type":   alertType,
			"severity":     severity,
			"title":        title,
			"message":      message,
			"acknowledged": acknowledged,
			"created_at":   createdAt,
		})
	}
	return results, nil
}
```

- [ ] **Step 3: Register routes in main.go**

Add inside the auth group:

```go
r.Route("/recovery", api.RecoveryRoutes(db, cfg))
r.Route("/alerts", api.AlertRoutes(db, cfg))
```

- [ ] **Step 4: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 5: Commit**

```bash
git add api/recovery.go api/alerts.go cmd/server/main.go
git commit -m "feat(fase4): recovery plans & academic alerts"
```

---

## Task 6: Assessment Service (Frontend)

**Files:**
- Create: `frontend/src/app/core/services/assessment.service.ts`
- Modify: `frontend/src/app/app.routes.ts`
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Produces: `AssessmentService` with HTTP methods for all assessment endpoints
- Produces: Angular routes for assessment and analytics pages

- [ ] **Step 1: Create `frontend/src/app/core/services/assessment.service.ts`**

```typescript
import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface Assessment {
  id: string;
  title: string;
  description: string;
  course_id: string;
  assessment_type: 'diagnostic' | 'formative' | 'summative' | 'recovery' | 'practice';
  mode: 'fixed' | 'generated' | 'adaptive';
  time_limit_minutes: number;
  max_attempts: number;
  shuffle_questions: boolean;
  show_results: boolean;
  show_solutions: boolean;
  passing_score: number;
  total_points: number;
  created_by: string;
  status: 'draft' | 'published' | 'archived';
  published_at?: string;
  expires_at?: string;
  metadata?: any;
}

export interface AssessmentQuestion {
  id: string;
  assessment_id: string;
  exercise_id?: string;
  question_order: number;
  points: number;
  question_type: 'exercise' | 'generated' | 'text' | 'multiple_choice';
  statement_override: string;
  statement?: string;
  latex?: string;
  difficulty?: number;
  concept_id?: string;
}

export interface StudentAssessment {
  id: string;
  student_id: string;
  assessment_id: string;
  status: 'in_progress' | 'submitted' | 'graded' | 'returned';
  started_at: string;
  submitted_at?: string;
  time_spent_seconds: number;
  attempt_number: number;
  total_score: number;
  max_score: number;
  percentage: number;
  passed: boolean;
  feedback: string;
}

export interface StudentAnswer {
  id: string;
  student_assessment_id: string;
  question_id: string;
  answer: string;
  procedure?: string[];
  is_correct: boolean;
  score: number;
  points_earned: number;
  points_possible: number;
  math_verified: boolean;
  rubric_scores?: any;
  feedback: string;
  time_spent_seconds: number;
  question_order?: number;
}

export interface Rubric {
  id: string;
  assessment_id: string;
  name: string;
  description: string;
  rubric_type: 'analytic' | 'holistic';
  max_score: number;
  criteria: any[];
}

export interface StudentAnalytics {
  id: string;
  student_id: string;
  course_id: string;
  total_assessments: number;
  passed_assessments: number;
  average_score: number;
  average_time_seconds: number;
  competency_level: 'beginner' | 'developing' | 'proficient' | 'advanced' | 'exceptional';
  competency_score: number;
  weakest_concepts: string[];
  strongest_concepts: string[];
  improvement_trend: number;
  study_streak_days: number;
}

export interface RecoveryPlan {
  id: string;
  student_id: string;
  course_id: string;
  trigger_assessment_id?: string;
  trigger_score: number;
  status: 'active' | 'completed' | 'expired' | 'cancelled';
  priority: number;
  concepts_to_review: string[];
  recommended_activities: any[];
  target_date?: string;
  completed_at?: string;
}

export interface AcademicAlert {
  id: string;
  student_id: string;
  alert_type: string;
  severity: 'info' | 'warning' | 'critical';
  title: string;
  message: string;
  concept_id: string;
  assessment_id?: string;
  acknowledged: boolean;
  acknowledged_by?: string;
  acknowledged_at?: string;
  metadata?: any;
}

@Injectable({ providedIn: 'root' })
export class AssessmentService {
  private baseUrl = environment.apiUrl + '/api';

  constructor(private http: HttpClient) {}

  createAssessment(assessment: Partial<Assessment> & { question_ids?: string[] }): Observable<Assessment> {
    return this.http.post<Assessment>(`${this.baseUrl}/assessments/`, assessment);
  }

  listAssessments(courseId?: string, type?: string, status?: string): Observable<Assessment[]> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    if (type) params = params.set('type', type);
    if (status) params = params.set('status', status);
    return this.http.get<Assessment[]>(`${this.baseUrl}/assessments/`, { params });
  }

  getAssessment(assessmentId: string): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/assessments/${assessmentId}`);
  }

  updateAssessment(assessmentId: string, updates: Partial<Assessment>): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/assessments/${assessmentId}`, updates);
  }

  deleteAssessment(assessmentId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/assessments/${assessmentId}`);
  }

  publishAssessment(assessmentId: string): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/assessments/${assessmentId}/publish`, {});
  }

  startAssessment(assessmentId: string): Observable<{ student_assessment: StudentAssessment; questions: AssessmentQuestion[] }> {
    return this.http.post<{ student_assessment: StudentAssessment; questions: AssessmentQuestion[] }>(
      `${this.baseUrl}/assessments/${assessmentId}/start`, {}
    );
  }

  submitAssessment(assessmentId: string, answers: { question_id: string; answer: string; procedure?: string[] }[]): Observable<any> {
    return this.http.post<any>(`${this.baseUrl}/assessments/${assessmentId}/submit`, { answers });
  }

  getAssessmentResult(assessmentId: string): Observable<{ student_assessment: StudentAssessment; answers: StudentAnswer[] }> {
    return this.http.get<{ student_assessment: StudentAssessment; answers: StudentAnswer[] }>(
      `${this.baseUrl}/assessments/${assessmentId}/results`
    );
  }

  getAssessmentStudentResults(assessmentId: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/assessments/${assessmentId}/student-results`);
  }

  manualGradeAnswer(answerId: string, score: number, feedback: string): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/grading/answer/${answerId}`, { score, feedback });
  }

  createRubric(assessmentId: string, rubric: Partial<Rubric>): Observable<Rubric> {
    return this.http.post<Rubric>(`${this.baseUrl}/grading/rubric/${assessmentId}`, rubric);
  }

  getRubrics(assessmentId: string): Observable<Rubric[]> {
    return this.http.get<Rubric[]>(`${this.baseUrl}/grading/rubric/${assessmentId}`);
  }

  evaluateWithRubric(answerId: string, rubricId: string): Observable<any> {
    return this.http.post<any>(`${this.baseUrl}/grading/evaluate/${answerId}`, { rubric_id: rubricId });
  }

  batchAutoGrade(assessmentId: string): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/grading/batch-grade/${assessmentId}`, {});
  }

  getStudentAnalytics(studentId: string, courseId?: string): Observable<StudentAnalytics> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<StudentAnalytics>(`${this.baseUrl}/analytics/v2/student/${studentId}`, { params });
  }

  getCourseAnalytics(courseId: string): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/analytics/v2/course/${courseId}`);
  }

  getCompetencyReport(studentId: string, courseId?: string): Observable<any> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<any>(`${this.baseUrl}/analytics/v2/student/${studentId}/competency`, { params });
  }

  getPerformanceTrend(studentId: string, courseId?: string): Observable<any> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<any>(`${this.baseUrl}/analytics/v2/student/${studentId}/trend`, { params });
  }

  updateStudentAnalytics(studentId: string, courseId?: string): Observable<void> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.post<void>(`${this.baseUrl}/analytics/v2/student/${studentId}/update`, {}, { params });
  }

  createRecoveryPlan(assessmentId: string, score: number, courseId?: string): Observable<RecoveryPlan> {
    return this.http.post<RecoveryPlan>(`${this.baseUrl}/recovery/`, {
      assessment_id: assessmentId,
      score,
      course_id: courseId || 'matematica-1'
    });
  }

  getRecoveryPlans(courseId?: string): Observable<RecoveryPlan[]> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<RecoveryPlan[]>(`${this.baseUrl}/recovery/`, { params });
  }

  completeRecoveryPlan(planId: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/recovery/${planId}/complete`, {});
  }

  cancelRecoveryPlan(planId: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/recovery/${planId}/cancel`, {});
  }

  getAlerts(severity?: string): Observable<AcademicAlert[]> {
    let params = new HttpParams();
    if (severity) params = params.set('severity', severity);
    return this.http.get<AcademicAlert[]>(`${this.baseUrl}/alerts/`, { params });
  }

  acknowledgeAlert(alertId: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/alerts/${alertId}/acknowledge`, {});
  }

  checkForAlerts(courseId?: string): Observable<{ alerts_created: number; alerts: AcademicAlert[] }> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.post<{ alerts_created: number; alerts: AcademicAlert[] }>(
      `${this.baseUrl}/alerts/check`, {}, { params }
    );
  }

  getAllAlerts(courseId?: string, severity?: string): Observable<any[]> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    if (severity) params = params.set('severity', severity);
    return this.http.get<any[]>(`${this.baseUrl}/alerts/all`, { params });
  }
}
```

- [ ] **Step 2: Verify Angular compilation**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/core/services/assessment.service.ts
git commit -m "feat(fase4): assessment service — HTTP client for all assessment endpoints"
```

---

## Task 7: Assessment Component (Frontend)

**Files:**
- Create: `frontend/src/app/modules/assessment/assessment.component.ts`
- Modify: `frontend/src/app/app.routes.ts`
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Produces: Assessment taking interface with timer, question navigation, answer submission
- Produces: Assessment results view with score breakdown

- [ ] **Step 1: Create `frontend/src/app/modules/assessment/assessment.component.ts`**

```typescript
import { Component, OnInit, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatRadioModule } from '@angular/material/radio';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatTabsModule } from '@angular/material/tabs';
import { MatChipsModule } from '@angular/material/chips';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { AssessmentService, Assessment, AssessmentQuestion, StudentAssessment, StudentAnswer } from '../../core/services/assessment.service';

@Component({
  selector: 'app-assessment',
  standalone: true,
  imports: [
    CommonModule, FormsModule, MatCardModule, MatButtonModule, MatIconModule,
    MatProgressBarModule, MatRadioModule, MatInputModule, MatFormFieldModule,
    MatTabsModule, MatChipsModule, MatSnackBarModule
  ],
  template: `
    @if (loading()) {
      <div class="loading-container">
        <mat-progress-bar mode="indeterminate"></mat-progress-bar>
        <p>Cargando evaluación...</p>
      </div>
    } @else if (error()) {
      <div class="error-container">
        <mat-icon color="warn">error</mat-icon>
        <p>{{ error() }}</p>
        <button mat-raised-button color="primary" (click)="goBack()">Volver</button>
      </div>
    } @else if (mode() === 'list') {
      <div class="assessment-list">
        <h2>Evaluaciones Disponibles</h2>
        @for (assessment of assessments(); track assessment.id) {
          <mat-card class="assessment-card" (click)="selectAssessment(assessment)">
            <mat-card-header>
              <mat-card-title>{{ assessment.title }}</mat-card-title>
              <mat-card-subtitle>
                {{ assessment.assessment_type | titlecase }} · {{ assessment.total_points }} puntos
              </mat-card-subtitle>
            </mat-card-header>
            <mat-card-content>
              <p>{{ assessment.description }}</p>
              <div class="assessment-meta">
                @if (assessment.time_limit_minutes > 0) {
                  <mat-chip>{{ assessment.time_limit_minutes }} min</mat-chip>
                }
                <mat-chip>{{ assessment.max_attempts }} intento(s)</mat-chip>
                <mat-chip>Aprobar: {{ (assessment.passing_score * 100) | number:'1.0-0' }}%</mat-chip>
              </div>
            </mat-card-content>
          </mat-card>
        }
      </div>
    } @else if (mode() === 'taking') {
      <div class="assessment-taking">
        <div class="assessment-header">
          <h2>{{ currentAssessment()?.title }}</h2>
          @if (timeRemaining() !== null) {
            <div class="timer" [class.warning]="timeRemaining()! < 300">
              <mat-icon>timer</mat-icon>
              {{ formatTime(timeRemaining()!) }}
            </div>
          }
          <div class="progress">
            Pregunta {{ currentIndex() + 1 }} de {{ questions().length }}
          </div>
        </div>

        <mat-progress-bar [value]="((currentIndex() + 1) / questions().length) * 100"></mat-progress-bar>

        @if (currentQuestion()) {
          <mat-card class="question-card">
            <mat-card-header>
              <mat-card-subtitle>
                Pregunta {{ currentIndex() + 1 }} · {{ currentQuestion()!.points }} puntos
              </mat-card-subtitle>
            </mat-card-header>
            <mat-card-content>
              @if (currentQuestion()!.latex) {
                <div class="question-latex" [innerHTML]="currentQuestion()!.latex"></div>
              } @else {
                <p class="question-statement">{{ currentQuestion()!.statement }}</p>
              }

              <mat-form-field appearance="outline" class="answer-field">
                <mat-label>Tu respuesta</mat-label>
                <input matInput [(ngModel)]="currentAnswer" placeholder="Escribe tu respuesta...">
              </mat-form-field>
            </mat-card-content>
            <mat-card-actions>
              <button mat-button (click)="previousQuestion()" [disabled]="currentIndex() === 0">
                <mat-icon>chevron_left</mat-icon> Anterior
              </button>
              <span class="spacer"></span>
              @if (currentIndex() < questions().length - 1) {
                <button mat-raised-button color="primary" (click)="nextQuestion()">
                  Siguiente <mat-icon>chevron_right</mat-icon>
                </button>
              } @else {
                <button mat-raised-button color="accent" (click)="submitAssessment()">
                  <mat-icon>send</mat-icon> Enviar Evaluación
                </button>
              }
            </mat-card-actions>
          </mat-card>
        }

        <div class="question-nav">
          @for (q of questions(); track q.id; let i = $index) {
            <button mat-mini-fab
              [color]="i === currentIndex() ? 'primary' : (answers()[q.id] ? 'accent' : '')"
              (click)="goToQuestion(i)">
              {{ i + 1 }}
            </button>
          }
        </div>
      </div>
    } @else if (mode() === 'results') {
      <div class="assessment-results">
        <mat-card class="results-header">
          <mat-card-header>
            <mat-card-title>Resultados de la Evaluación</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <div class="score-display">
              <div class="score-circle" [class.passed]="result()!.passed">
                {{ (result()!.percentage * 100) | number:'1.0-0' }}%
              </div>
              <div class="score-details">
                <p>{{ result()!.total_score }} / {{ result()!.max_score }} puntos</p>
                <p [class]="result()!.passed ? 'passed' : 'failed'">
                  {{ result()!.passed ? 'Aprobado' : 'No aprobado' }}
                </p>
              </div>
            </div>
          </mat-card-content>
        </mat-card>

        <div class="answers-breakdown">
          <h3>Detalle de Respuestas</h3>
          @for (answer of resultAnswers(); track answer.id; let i = $index) {
            <mat-card class="answer-card" [class.correct]="answer.is_correct" [class.incorrect]="!answer.is_correct">
              <mat-card-header>
                <mat-card-subtitle>
                  Pregunta {{ i + 1 }} · {{ answer.points_earned }}/{{ answer.points_possible }} puntos
                </mat-card-subtitle>
              </mat-card-header>
              <mat-card-content>
                <p><strong>Tu respuesta:</strong> {{ answer.answer }}</p>
                <p><strong>Resultado:</strong>
                  <span [class]="answer.is_correct ? 'correct-text' : 'incorrect-text'">
                    {{ answer.is_correct ? 'Correcto' : 'Incorrecto' }}
                  </span>
                </p>
                <p *ngIf="answer.feedback">{{ answer.feedback }}</p>
              </mat-card-content>
            </mat-card>
          }
        </div>

        <button mat-raised-button color="primary" (click)="goBack()">Volver a Evaluaciones</button>
      </div>
    }
  `,
  styles: [`
    .loading-container, .error-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 400px;
      gap: 16px;
    }
    .assessment-list {
      max-width: 800px;
      margin: 0 auto;
      padding: 24px;
    }
    .assessment-card {
      margin-bottom: 16px;
      cursor: pointer;
      transition: transform 0.2s;
    }
    .assessment-card:hover {
      transform: translateY(-2px);
    }
    .assessment-meta {
      display: flex;
      gap: 8px;
      margin-top: 12px;
    }
    .assessment-taking {
      max-width: 800px;
      margin: 0 auto;
      padding: 24px;
    }
    .assessment-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
    }
    .timer {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 1.2em;
      font-weight: bold;
    }
    .timer.warning {
      color: #f44336;
    }
    .question-card {
      margin: 24px 0;
    }
    .question-latex, .question-statement {
      font-size: 1.1em;
      margin-bottom: 24px;
    }
    .answer-field {
      width: 100%;
    }
    mat-card-actions {
      display: flex;
      align-items: center;
    }
    .spacer {
      flex: 1;
    }
    .question-nav {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-top: 24px;
      justify-content: center;
    }
    .results-header {
      text-align: center;
      margin-bottom: 24px;
    }
    .score-display {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 32px;
      padding: 32px;
    }
    .score-circle {
      font-size: 3em;
      font-weight: bold;
      width: 150px;
      height: 150px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      border: 8px solid #f44336;
    }
    .score-circle.passed {
      border-color: #4caf50;
    }
    .passed { color: #4caf50; }
    .failed { color: #f44336; }
    .correct-text { color: #4caf50; }
    .incorrect-text { color: #f44336; }
    .answers-breakdown {
      margin-bottom: 24px;
    }
    .answer-card {
      margin-bottom: 12px;
      border-left: 4px solid;
    }
    .answer-card.correct {
      border-left-color: #4caf50;
    }
    .answer-card.incorrect {
      border-left-color: #f44336;
    }
  `]
})
export class AssessmentComponent implements OnInit {
  loading = signal(true);
  error = signal<string | null>(null);
  mode = signal<'list' | 'taking' | 'results'>('list');
  assessments = signal<Assessment[]>([]);
  currentAssessment = signal<Assessment | null>(null);
  questions = signal<AssessmentQuestion[]>([]);
  currentIndex = signal(0);
  currentAnswer = '';
  answers = signal<Record<string, string>>({});
  studentAssessment = signal<StudentAssessment | null>(null);
  timeRemaining = signal<number | null>(null);
  result = signal<StudentAssessment | null>(null);
  resultAnswers = signal<StudentAnswer[]>([]);

  private timerInterval: any;

  currentQuestion = computed(() => {
    const qs = this.questions();
    const idx = this.currentIndex();
    return qs.length > idx ? qs[idx] : null;
  });

  constructor(
    private assessmentService: AssessmentService,
    private route: ActivatedRoute,
    private router: Router,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit() {
    this.loadAssessments();
  }

  loadAssessments() {
    this.loading.set(true);
    this.assessmentService.listAssessments(undefined, undefined, 'published').subscribe({
      next: (assessments) => {
        this.assessments.set(assessments || []);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set('Error al cargar evaluaciones');
        this.loading.set(false);
      }
    });
  }

  selectAssessment(assessment: Assessment) {
    this.currentAssessment.set(assessment);
    this.loading.set(true);
    this.assessmentService.startAssessment(assessment.id).subscribe({
      next: (resp) => {
        this.studentAssessment.set(resp.student_assessment);
        this.questions.set(resp.questions || []);
        this.currentIndex.set(0);
        this.mode.set('taking');
        this.loading.set(false);
        if (assessment.time_limit_minutes > 0) {
          this.startTimer(assessment.time_limit_minutes * 60);
        }
      },
      error: (err) => {
        this.snackBar.open(err.error?.error || 'Error al iniciar evaluación', 'Cerrar', { duration: 3000 });
        this.loading.set(false);
      }
    });
  }

  startTimer(seconds: number) {
    this.timeRemaining.set(seconds);
    this.timerInterval = setInterval(() => {
      const current = this.timeRemaining();
      if (current !== null && current > 0) {
        this.timeRemaining.set(current - 1);
      } else {
        clearInterval(this.timerInterval);
        this.submitAssessment();
      }
    }, 1000);
  }

  formatTime(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  nextQuestion() {
    this.saveCurrentAnswer();
    if (this.currentIndex() < this.questions().length - 1) {
      this.currentIndex.set(this.currentIndex() + 1);
      this.loadCurrentAnswer();
    }
  }

  previousQuestion() {
    this.saveCurrentAnswer();
    if (this.currentIndex() > 0) {
      this.currentIndex.set(this.currentIndex() - 1);
      this.loadCurrentAnswer();
    }
  }

  goToQuestion(index: number) {
    this.saveCurrentAnswer();
    this.currentIndex.set(index);
    this.loadCurrentAnswer();
  }

  saveCurrentAnswer() {
    const q = this.currentQuestion();
    if (q) {
      const current = this.answers();
      this.answers.set({ ...current, [q.id]: this.currentAnswer });
    }
  }

  loadCurrentAnswer() {
    const q = this.currentQuestion();
    if (q) {
      this.currentAnswer = this.answers()[q.id] || '';
    }
  }

  submitAssessment() {
    this.saveCurrentAnswer();
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }

    const assessment = this.currentAssessment();
    if (!assessment) return;

    const answersArray = Object.entries(this.answers()).map(([question_id, answer]) => ({
      question_id,
      answer,
      procedure: []
    }));

    this.loading.set(true);
    this.assessmentService.submitAssessment(assessment.id, answersArray).subscribe({
      next: (resp) => {
        this.result.set({
          ...this.studentAssessment()!,
          total_score: resp.total_score,
          max_score: resp.max_score,
          percentage: resp.percentage,
          passed: resp.passed,
          status: 'graded'
        } as StudentAssessment);

        this.assessmentService.getAssessmentResult(assessment.id).subscribe({
          next: (resultResp) => {
            this.resultAnswers.set(resultResp.answers || []);
            this.mode.set('results');
            this.loading.set(false);
          },
          error: () => {
            this.mode.set('results');
            this.loading.set(false);
          }
        });
      },
      error: (err) => {
        this.snackBar.open('Error al enviar evaluación', 'Cerrar', { duration: 3000 });
        this.loading.set(false);
      }
    });
  }

  goBack() {
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }
    this.mode.set('list');
    this.currentAssessment.set(null);
    this.loadAssessments();
  }

  ngOnDestroy() {
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }
  }
}
```

- [ ] **Step 2: Add route to app.routes.ts**

Add the assessment route:

```typescript
{
  path: 'assessment',
  loadComponent: () => import('./modules/assessment/assessment.component').then(m => m.AssessmentComponent),
  canActivate: [authGuard]
},
```

- [ ] **Step 3: Add nav item to layout.component.ts**

Add to the navigation items:

```typescript
{ icon: 'quiz', label: 'Evaluaciones', route: '/assessment' },
```

- [ ] **Step 4: Verify Angular compilation**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build`
Expected: BUILD SUCCESSFUL

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/assessment/assessment.component.ts frontend/src/app/app.routes.ts frontend/src/app/shared/layout.component.ts
git commit -m "feat(fase4): assessment component — taking interface, timer, results view"
```

---

## Task 8: Analytics Component (Frontend)

**Files:**
- Create: `frontend/src/app/modules/analytics/analytics.component.ts`
- Modify: `frontend/src/app/app.routes.ts`
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Produces: Analytics dashboard with competency tracking, performance trends, alerts

- [ ] **Step 1: Create `frontend/src/app/modules/analytics/analytics.component.ts`**

```typescript
import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatTabsModule } from '@angular/material/tabs';
import { MatChipsModule } from '@angular/material/chips';
import { MatListModule } from '@angular/material/list';
import { AssessmentService, StudentAnalytics, RecoveryPlan, AcademicAlert } from '../../core/services/assessment.service';
import { LearningService, StudentProfile, ConceptMastery } from '../../core/services/learning.service';

@Component({
  selector: 'app-analytics',
  standalone: true,
  imports: [
    CommonModule, MatCardModule, MatButtonModule, MatIconModule,
    MatProgressBarModule, MatTabsModule, MatChipsModule, MatListModule
  ],
  template: `
    <div class="analytics-container">
      <h2>Panel de Analíticas</h2>

      @if (loading()) {
        <mat-progress-bar mode="indeterminate"></mat-progress-bar>
      } @else {
        <mat-tab-group>
          <mat-tab label="Resumen">
            <div class="analytics-grid">
              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>assessment</mat-icon>
                  <mat-card-title>Evaluaciones</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value">{{ analytics()?.total_assessments || 0 }}</div>
                  <div class="stat-label">Total realizadas</div>
                  <div class="stat-detail">
                    {{ analytics()?.passed_assessments || 0 }} aprobadas
                    ({{ ((analytics()?.passed_assessments || 0) / (analytics()?.total_assessments || 1) * 100) | number:'1.0-0' }}%)
                  </div>
                </mat-card-content>
              </mat-card>

              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>grade</mat-icon>
                  <mat-card-title>Promedio</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value">{{ ((analytics()?.average_score || 0) * 100) | number:'1.0-0' }}%</div>
                  <mat-progress-bar [value]="(analytics()?.average_score || 0) * 100"></mat-progress-bar>
                </mat-card-content>
              </mat-card>

              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>emoji_events</mat-icon>
                  <mat-card-title>Competencia</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value competency-badge" [attr.data-level]="analytics()?.competency_level">
                    {{ analytics()?.competency_level | titlecase }}
                  </div>
                  <mat-progress-bar [value]="(analytics()?.competency_score || 0) * 100"></mat-progress-bar>
                </mat-card-content>
              </mat-card>

              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>trending_up</mat-icon>
                  <mat-card-title>Tendencia</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value" [class.positive]="(analytics()?.improvement_trend || 0) > 0" [class.negative]="(analytics()?.improvement_trend || 0) < 0">
                    {{ (analytics()?.improvement_trend || 0) > 0 ? '+' : '' }}{{ ((analytics()?.improvement_trend || 0) * 100) | number:'1.0-0' }}%
                  </div>
                  <div class="stat-label">Últimos 30 días</div>
                </mat-card-content>
              </mat-card>
            </div>
          </mat-tab>

          <mat-tab label="Conceptos">
            <div class="concepts-section">
              <mat-card>
                <mat-card-header>
                  <mat-card-title>Conceptos Débiles</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  @if (analytics()?.weakest_concepts?.length) {
                    <mat-list>
                      @for (concept of analytics()?.weakest_concepts; track concept) {
                        <mat-list-item>
                          <mat-icon matListItemIcon>warning</mat-icon>
                          <span matListItemTitle>{{ concept }}</span>
                        </mat-list-item>
                      }
                    </mat-list>
                  } @else {
                    <p>No hay conceptos débiles identificados</p>
                  }
                </mat-card-content>
              </mat-card>

              <mat-card>
                <mat-card-header>
                  <mat-card-title>Conceptos Fuertes</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  @if (analytics()?.strongest_concepts?.length) {
                    <mat-list>
                      @for (concept of analytics()?.strongest_concepts; track concept) {
                        <mat-list-item>
                          <mat-icon matListItemIcon>check_circle</mat-icon>
                          <span matListItemTitle>{{ concept }}</span>
                        </mat-list-item>
                      }
                    </mat-list>
                  } @else {
                    <p>Aún no hay conceptos dominados</p>
                  }
                </mat-card-content>
              </mat-card>
            </div>
          </mat-tab>

          <mat-tab label="Planes de Recuperación">
            <div class="recovery-section">
              @if (recoveryPlans().length) {
                @for (plan of recoveryPlans(); track plan.id) {
                  <mat-card class="recovery-card">
                    <mat-card-header>
                      <mat-card-title>Plan de Recuperación</mat-card-title>
                      <mat-card-subtitle>
                        <mat-chip [color]="plan.priority >= 4 ? 'warn' : 'accent'">
                          Prioridad: {{ plan.priority }}/5
                        </mat-chip>
                        <mat-chip>{{ plan.status | titlecase }}</mat-chip>
                      </mat-card-subtitle>
                    </mat-card-header>
                    <mat-card-content>
                      <p><strong>Conceptos a repasar:</strong></p>
                      <div class="concept-chips">
                        @for (concept of plan.concepts_to_review; track concept) {
                          <mat-chip>{{ concept }}</mat-chip>
                        }
                      </div>
                      @if (plan.target_date) {
                        <p><strong>Fecha objetivo:</strong> {{ plan.target_date | date }}</p>
                      }
                    </mat-card-content>
                    <mat-card-actions>
                      @if (plan.status === 'active') {
                        <button mat-raised-button color="primary" (click)="completePlan(plan.id)">
                          Marcar como Completado
                        </button>
                        <button mat-button (click)="cancelPlan(plan.id)">Cancelar</button>
                      }
                    </mat-card-actions>
                  </mat-card>
                }
              } @else {
                <mat-card>
                  <mat-card-content>
                    <p>No tienes planes de recuperación activos</p>
                  </mat-card-content>
                </mat-card>
              }
            </div>
          </mat-tab>

          <mat-tab label="Alertas">
            <div class="alerts-section">
              @if (alerts().length) {
                @for (alert of alerts(); track alert.id) {
                  <mat-card class="alert-card" [class.critical]="alert.severity === 'critical'" [class.warning]="alert.severity === 'warning'">
                    <mat-card-header>
                      <mat-icon mat-card-avatar [color]="alert.severity === 'critical' ? 'warn' : 'accent'">
                        {{ alert.severity === 'critical' ? 'error' : 'warning' }}
                      </mat-icon>
                      <mat-card-title>{{ alert.title }}</mat-card-title>
                      <mat-card-subtitle>{{ alert.severity | titlecase }} · {{ alert.created_at | date }}</mat-card-subtitle>
                    </mat-card-header>
                    <mat-card-content>
                      <p>{{ alert.message }}</p>
                    </mat-card-content>
                    <mat-card-actions>
                      @if (!alert.acknowledged) {
                        <button mat-button (click)="acknowledgeAlert(alert.id)">Marcar como Leído</button>
                      }
                    </mat-card-actions>
                  </mat-card>
                }
              } @else {
                <mat-card>
                  <mat-card-content>
                    <p>No hay alertas activas</p>
                  </mat-card-content>
                </mat-card>
              }
            </div>
          </mat-tab>
        </mat-tab-group>
      }
    </div>
  `,
  styles: [`
    .analytics-container {
      max-width: 1200px;
      margin: 0 auto;
      padding: 24px;
    }
    .analytics-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 24px;
      margin-top: 24px;
    }
    .stat-card {
      text-align: center;
    }
    .stat-value {
      font-size: 2.5em;
      font-weight: bold;
      margin: 16px 0 8px;
    }
    .stat-label {
      color: #666;
    }
    .stat-detail {
      margin-top: 8px;
      color: #4caf50;
    }
    .competency-badge {
      text-transform: capitalize;
    }
    .positive { color: #4caf50; }
    .negative { color: #f44336; }
    .concepts-section, .recovery-section, .alerts-section {
      display: flex;
      flex-direction: column;
      gap: 16px;
      margin-top: 24px;
    }
    .concept-chips {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin: 8px 0;
    }
    .recovery-card {
      border-left: 4px solid #ff9800;
    }
    .alert-card {
      margin-bottom: 12px;
    }
    .alert-card.critical {
      border-left: 4px solid #f44336;
    }
    .alert-card.warning {
      border-left: 4px solid #ff9800;
    }
  `]
})
export class AnalyticsComponent implements OnInit {
  loading = signal(true);
  analytics = signal<StudentAnalytics | null>(null);
  recoveryPlans = signal<RecoveryPlan[]>([]);
  alerts = signal<AcademicAlert[]>([]);

  constructor(
    private assessmentService: AssessmentService,
    private learningService: LearningService
  ) {}

  ngOnInit() {
    this.loadData();
  }

  loadData() {
    this.loading.set(true);
    this.assessmentService.getStudentAnalytics('current').subscribe({
      next: (data) => {
        this.analytics.set(data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
      }
    });

    this.assessmentService.getRecoveryPlans().subscribe({
      next: (plans) => this.recoveryPlans.set(plans || []),
      error: () => {}
    });

    this.assessmentService.getAlerts().subscribe({
      next: (alerts) => this.alerts.set(alerts || []),
      error: () => {}
    });
  }

  completePlan(planId: string) {
    this.assessmentService.completeRecoveryPlan(planId).subscribe({
      next: () => {
        this.recoveryPlans.update(plans =>
          plans.map(p => p.id === planId ? { ...p, status: 'completed' as const } : p)
        );
      }
    });
  }

  cancelPlan(planId: string) {
    this.assessmentService.cancelRecoveryPlan(planId).subscribe({
      next: () => {
        this.recoveryPlans.update(plans =>
          plans.map(p => p.id === planId ? { ...p, status: 'cancelled' as const } : p)
        );
      }
    });
  }

  acknowledgeAlert(alertId: string) {
    this.assessmentService.acknowledgeAlert(alertId).subscribe({
      next: () => {
        this.alerts.update(alerts =>
          alerts.map(a => a.id === alertId ? { ...a, acknowledged: true } : a)
        );
      }
    });
  }
}
```

- [ ] **Step 2: Add route to app.routes.ts**

Add the analytics route:

```typescript
{
  path: 'analytics',
  loadComponent: () => import('./modules/analytics/analytics.component').then(m => m.AnalyticsComponent),
  canActivate: [authGuard]
},
```

- [ ] **Step 3: Add nav item to layout.component.ts**

Add to the navigation items:

```typescript
{ icon: 'analytics', label: 'Analíticas', route: '/analytics' },
```

- [ ] **Step 4: Verify Angular compilation**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build`
Expected: BUILD SUCCESSFUL

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/analytics/analytics.component.ts frontend/src/app/app.routes.ts frontend/src/app/shared/layout.component.ts
git commit -m "feat(fase4): analytics dashboard — competency tracking, recovery plans, alerts"
```

---

## Task 9: Config Updates

**Files:**
- Modify: `internal/config/config.go`

**Interfaces:**
- Adds assessment configuration parameters

- [ ] **Step 1: Add assessment config fields**

Add to the Config struct:

```go
// Assessment Engine
AssessmentDefaultTimeLimit   int     `json:"assessment_default_time_limit"`
AssessmentMaxAttempts        int     `json:"assessment_max_attempts"`
AssessmentPassingScore       float64 `json:"assessment_passing_score"`
AssessmentAutoGradeEnabled   bool    `json:"assessment_auto_grade_enabled"`
AssessmentRecoveryThreshold  float64 `json:"assessment_recovery_threshold"`
AssessmentAlertThreshold     float64 `json:"assessment_alert_threshold"`
```

Add to the Load function:

```go
// Assessment Engine
AssessmentDefaultTimeLimit:   getEnvInt("ASSESSMENT_DEFAULT_TIME_LIMIT", 60),
AssessmentMaxAttempts:        getEnvInt("ASSESSMENT_MAX_ATTEMPTS", 3),
AssessmentPassingScore:       getEnvFloat("ASSESSMENT_PASSING_SCORE", 0.6),
AssessmentAutoGradeEnabled:   getEnvBool("ASSESSMENT_AUTO_GRADE_ENABLED", true),
AssessmentRecoveryThreshold:  getEnvFloat("ASSESSMENT_RECOVERY_THRESHOLD", 0.6),
AssessmentAlertThreshold:     getEnvFloat("ASSESSMENT_ALERT_THRESHOLD", 0.4),
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(fase4): assessment configuration parameters"
```

---

## Task 10: Final Verification & Documentation

**Files:**
- Modify: `README.md`

**Interfaces:**
- Documents Phase 4 features and API endpoints

- [ ] **Step 1: Add Phase 4 section to README.md**

Add after the Phase 3 section:

```markdown
## Fase 4 — Sistema de Evaluación y Calificación Inteligente

### Características
- **Evaluaciones Múltiples:** Diagnósticas, formativas, sumativas, de recuperación y práctica
- **Modos de Evaluación:** Fija (profesor selecciona), generada (IA crea con RAG), adaptativa (se ajusta al rendimiento)
- **Calificación Inteligente:** Validación matemática + reglas + rúbricas + LLM cuando es necesario
- **Puntuación Parcial:** Soporte para evaluación paso a paso con detección de errores
- **Analíticas de Estudiantes:** Nivel de competencia, tendencias de rendimiento, conceptos débiles/fuertes
- **Planes de Recuperación:** Recomendaciones personalizadas basadas en rendimiento
- **Alertas Académicas:** Sistema de temprana detección de estudiantes en riesgo

### API Endpoints

#### Evaluaciones (`/api/assessments`)
- `POST /` — Crear evaluación
- `GET /` — Listar evaluaciones (filtros: course_id, type, status)
- `GET /{id}` — Obtener evaluación con preguntas
- `PUT /{id}` — Actualizar evaluación
- `DELETE /{id}` — Eliminar evaluación
- `POST /{id}/publish` — Publicar evaluación
- `POST /{id}/start` — Iniciar evaluación (estudiante)
- `POST /{id}/submit` — Enviar respuestas
- `GET /{id}/results` — Ver resultados (estudiante)
- `GET /{id}/student-results` — Ver resultados de todos los estudiantes

#### Calificación (`/api/grading`)
- `POST /answer/{id}` — Calificación manual
- `POST /rubric/{assessment_id}` — Crear rúbrica
- `GET /rubric/{assessment_id}` — Obtener rúbricas
- `POST /evaluate/{answer_id}` — Evaluar con rúbrica
- `POST /batch-grade/{assessment_id}` — Calificación automática en lote

#### Analíticas (`/api/analytics/v2`)
- `GET /student/{id}` — Analíticas del estudiante
- `GET /course/{id}` — Analíticas del curso
- `GET /student/{id}/competency` — Reporte de competencia
- `GET /student/{id}/trend` — Tendencia de rendimiento
- `POST /student/{id}/update` — Actualizar analíticas

#### Recuperación (`/api/recovery`)
- `POST /` — Crear plan de recuperación
- `GET /` — Obtener planes activos
- `PUT /{id}/complete` — Marcar plan como completado
- `PUT /{id}/cancel` — Cancelar plan

#### Alertas (`/api/alerts`)
- `GET /` — Obtener alertas del estudiante
- `PUT /{id}/acknowledge` — Reconocer alerta
- `POST /check` — Verificar y crear alertas
- `GET /all` — Todas las alertas (profesor/admin)
```

- [ ] **Step 2: Verify Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Verify Go build**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs(fase4): update README with assessment system documentation"
```

---

## Summary

Phase 4 adds a comprehensive assessment and evaluation system to matematicarag:

1. **Database Schema:** 7 new tables for assessments, questions, rubrics, grades, analytics, recovery plans, and alerts
2. **Assessment Engine:** Full CRUD with multiple assessment types and modes
3. **Grading Engine:** Manual, rubric-based, and automatic grading with mathematical equivalence validation
4. **Analytics:** Student competency tracking, performance trends, course analytics
5. **Recovery Plans:** Personalized learning recommendations based on assessment performance
6. **Academic Alerts:** Early warning system for at-risk students
7. **Frontend:** Assessment-taking interface with timer, analytics dashboard, recovery plan view

The system integrates with existing Phase 3 components (exercise bank, mastery tracking, adaptive engine) and uses the Math Service for answer validation.
