package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentAnalyticsData struct {
	ID                 string          `json:"id"`
	StudentID          string          `json:"student_id"`
	CourseID           string          `json:"course_id"`
	TotalAssessments   int             `json:"total_assessments"`
	PassedAssessments  int             `json:"passed_assessments"`
	AverageScore       float64         `json:"average_score"`
	AverageTimeSeconds int             `json:"average_time_seconds"`
	CompetencyLevel    string          `json:"competency_level"`
	CompetencyScore    float64         `json:"competency_score"`
	WeakestConcepts    json.RawMessage `json:"weakest_concepts"`
	StrongestConcepts  json.RawMessage `json:"strongest_concepts"`
	ImprovementTrend   float64         `json:"improvement_trend"`
	StudyStreakDays    int             `json:"study_streak_days"`
}

type CourseAnalyticsData struct {
	CourseID          string  `json:"course_id"`
	TotalStudents     int     `json:"total_students"`
	TotalAssessments  int     `json:"total_assessments"`
	AverageScore      float64 `json:"average_score"`
	PassRate          float64 `json:"pass_rate"`
	AverageCompetency float64 `json:"average_competency"`
}

func AnalyticsV2Routes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
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

		r.Get("/course/{courseID}/matrix", func(w http.ResponseWriter, r *http.Request) {
			courseID := chi.URLParam(r, "courseID")
			ctx := context.Background()

			rows, err := db.Query(ctx, `
				SELECT
					c.id AS concept_id,
					c.name,
					COUNT(DISTINCT sa.student_id) AS total_students,
					COALESCE(AVG(CASE WHEN sa.is_correct THEN 1.0 ELSE 0.0 END), 0) AS success_rate
				FROM concepts c
				LEFT JOIN question_bank qb ON qb.concept_id = c.id
				LEFT JOIN student_answers sa ON sa.question_id = qb.id
				WHERE c.course_id = $1
				GROUP BY c.id, c.name
				ORDER BY c.name`, courseID)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			type ConceptMatrix struct {
				ConceptID      string  `json:"concept_id"`
				Name           string  `json:"name"`
				TotalStudents  int     `json:"total_students"`
				SuccessRate    float64 `json:"success_rate"`
				NeedsAttention bool    `json:"needs_attention"`
			}

			var matrix []ConceptMatrix
			for rows.Next() {
				var m ConceptMatrix
				if err := rows.Scan(&m.ConceptID, &m.Name, &m.TotalStudents, &m.SuccessRate); err != nil {
					continue
				}
				m.NeedsAttention = m.SuccessRate < 0.6
				matrix = append(matrix, m)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(matrix)
		})

		r.Get("/course/{courseID}/error-patterns", func(w http.ResponseWriter, r *http.Request) {
			courseID := chi.URLParam(r, "courseID")
			ctx := context.Background()

			rows, err := db.Query(ctx, `
				SELECT
					c.id AS concept_id,
					c.name AS concept_name,
					qb.statement AS question,
					COUNT(*) AS error_count,
					COUNT(DISTINCT sa.student_id) AS affected_students
				FROM student_answers sa
				JOIN question_bank qb ON sa.question_id = qb.id
				JOIN concepts c ON qb.concept_id = c.id
				WHERE sa.is_correct = FALSE AND c.course_id = $1
				GROUP BY c.id, c.name, qb.statement
				HAVING COUNT(*) > 2
				ORDER BY error_count DESC
				LIMIT 20`, courseID)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			type ErrorPattern struct {
				ConceptID       string `json:"concept_id"`
				ConceptName     string `json:"concept_name"`
				Question        string `json:"question"`
				ErrorCount      int    `json:"error_count"`
				AffectedStudents int   `json:"affected_students"`
			}

			var patterns []ErrorPattern
			for rows.Next() {
				var p ErrorPattern
				if err := rows.Scan(&p.ConceptID, &p.ConceptName, &p.Question, &p.ErrorCount, &p.AffectedStudents); err != nil {
					continue
				}
				patterns = append(patterns, p)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(patterns)
		})

		r.Get("/student/{studentID}/question-history", func(w http.ResponseWriter, r *http.Request) {
			studentID := chi.URLParam(r, "studentID")
			ctx := context.Background()

			rows, err := db.Query(ctx, `
				SELECT
					sa.question_id,
					qb.statement,
					sa.answer,
					sa.is_correct,
					sa.score,
					sa.created_at
				FROM student_answers sa
				JOIN question_bank qb ON sa.question_id = qb.id
				WHERE sa.student_id = $1
				ORDER BY sa.created_at DESC
				LIMIT 50`, studentID)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			type QuestionHistory struct {
				QuestionID string      `json:"question_id"`
				Statement  string      `json:"statement"`
				Answer     string      `json:"answer"`
				IsCorrect  bool        `json:"is_correct"`
				Score      float64     `json:"score"`
				CreatedAt  interface{} `json:"created_at"`
			}

			var history []QuestionHistory
			for rows.Next() {
				var h QuestionHistory
				if err := rows.Scan(&h.QuestionID, &h.Statement, &h.Answer, &h.IsCorrect, &h.Score, &h.CreatedAt); err != nil {
					continue
				}
				history = append(history, h)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(history)
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
			"week":             week,
			"average_score":    avgScore,
			"assessment_count": count,
		})
	}

	return map[string]interface{}{
		"student_id": studentID,
		"course_id":  courseID,
		"trend":      weeks,
	}, nil
}
