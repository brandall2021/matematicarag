package adaptive

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentProfile struct {
	ID               string
	StudentID        string
	CourseID         string
	OverallLevel     float64
	TotalAttempts    int
	CorrectAttempts  int
	IncorrectAttempts int
	TotalHintsUsed   int
	StudyTimeSeconds int
}

func LoadLearnerState(ctx context.Context, db *pgxpool.Pool, studentID, courseID string) (*LearnerState, error) {
	if courseID == "" {
		courseID = "matematica-1"
	}

	var profile StudentProfile
	err := db.QueryRow(ctx,
		`INSERT INTO student_profiles (student_id, course_id)
		 VALUES ($1, $2)
		 ON CONFLICT (student_id) DO UPDATE SET updated_at = NOW()
		 RETURNING id, student_id, course_id, overall_level, total_attempts, correct_attempts, total_hints_used, study_time_seconds`,
		studentID, courseID,
	).Scan(&profile.ID, &profile.StudentID, &profile.CourseID, &profile.OverallLevel, &profile.TotalAttempts, &profile.CorrectAttempts, &profile.TotalHintsUsed, &profile.StudyTimeSeconds)
	if err != nil {
		return nil, err
	}

	state := &LearnerState{
		StudentID:          studentID,
		CourseID:           courseID,
		OverallMastery:     profile.OverallLevel,
		TotalAttempts:      profile.TotalAttempts,
		SuccessfulAttempts: profile.CorrectAttempts,
		FailedAttempts:     profile.TotalAttempts - profile.CorrectAttempts,
		HintUsage:          profile.TotalHintsUsed,
		AvgResolutionTime:  profile.StudyTimeSeconds,
	}

	rows, err := db.Query(ctx,
		`SELECT cm.concept_id, cm.mastery, cm.status, cm.attempts, cm.last_attempt_at
		 FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE cm.student_id = $1 AND c.course_id = $2`,
		studentID, courseID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var conceptID, status string
			var mastery float64
			var attempts int
			var lastAttempt *time.Time
			if err := rows.Scan(&conceptID, &mastery, &status, &attempts, &lastAttempt); err != nil {
				continue
			}
			if mastery >= 0.75 {
				state.StrongConcepts = append(state.StrongConcepts, conceptID)
			} else if mastery < 0.60 && attempts > 0 {
				state.WeakConcepts = append(state.WeakConcepts, conceptID)
			}
			if lastAttempt != nil && (state.LastActivityAt == nil || lastAttempt.After(*state.LastActivityAt)) {
				state.LastActivityAt = lastAttempt
			}
		}
	}

	errorRows, err := db.Query(ctx,
		`SELECT error_type FROM student_errors
		 WHERE student_id = $1 ORDER BY last_occurred_at DESC LIMIT 10`,
		studentID,
	)
	if err == nil {
		defer errorRows.Close()
		for errorRows.Next() {
			var et string
			if err := errorRows.Scan(&et); err != nil {
				continue
			}
			state.RecentErrors = append(state.RecentErrors, et)
		}
	}

	return state, nil
}

type ConceptMasterySummary struct {
	ConceptID string  `json:"concept_id"`
	Name      string  `json:"name"`
	Mastery   float64 `json:"mastery"`
	Status    string  `json:"status"`
	Attempts  int     `json:"attempts"`
}

type CourseAnalytics struct {
	CourseID         string                 `json:"course_id"`
	TotalStudents    int                    `json:"total_students"`
	AverageMastery   float64                `json:"average_mastery"`
	ConceptBreakdown []ConceptMasterySummary `json:"concept_breakdown"`
}

type ProgressAnalyticsService struct {
	db *pgxpool.Pool
}

func NewProgressAnalyticsService(db *pgxpool.Pool) *ProgressAnalyticsService {
	return &ProgressAnalyticsService{db: db}
}

func (s *ProgressAnalyticsService) GetStudentProgress(ctx context.Context, studentID, courseID string) (*LearnerState, error) {
	return LoadLearnerState(ctx, s.db, studentID, courseID)
}

func (s *ProgressAnalyticsService) GetCourseAnalytics(ctx context.Context, courseID string) (*CourseAnalytics, error) {
	analytics := &CourseAnalytics{CourseID: courseID}

	err := s.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT student_id) FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE c.course_id = $1`, courseID,
	).Scan(&analytics.TotalStudents)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT c.code, c.name, COALESCE(AVG(cm.mastery), 0) as avg_mastery
		 FROM concepts c
		 LEFT JOIN concept_mastery cm ON cm.concept_id = c.id
		 WHERE c.course_id = $1
		 GROUP BY c.code, c.name
		 ORDER BY avg_mastery ASC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalMastery float64
	var count int
	for rows.Next() {
		var s ConceptMasterySummary
		if err := rows.Scan(&s.ConceptID, &s.Name, &s.Mastery); err != nil {
			continue
		}
		analytics.ConceptBreakdown = append(analytics.ConceptBreakdown, s)
		totalMastery += s.Mastery
		count++
	}
	if count > 0 {
		analytics.AverageMastery = totalMastery / float64(count)
	}

	return analytics, nil
}

func (s *ProgressAnalyticsService) GetCommonErrors(ctx context.Context, courseID string) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(ctx,
		`SELECT se.error_type, COUNT(*) as frequency
		 FROM student_errors se
		 JOIN concepts c ON se.concept_id = c.id
		 WHERE c.course_id = $1
		 GROUP BY se.error_type
		 ORDER BY frequency DESC LIMIT 10`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var errorType string
		var frequency int
		if err := rows.Scan(&errorType, &frequency); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"error_type": errorType,
			"frequency":  frequency,
		})
	}
	return results, nil
}
