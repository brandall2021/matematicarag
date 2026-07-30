package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentDashboard struct {
	Profile          map[string]any `json:"profile"`
	OverallMastery   float64        `json:"overall_mastery"`
	CompetencyLevel  string         `json:"competency_level"`
	CompetencyScore  float64        `json:"competency_score"`
	Concepts         []ConceptSummary `json:"concepts"`
	WeakestConcepts  []string       `json:"weakest_concepts"`
	StrongestConcepts []string      `json:"strongest_concepts"`
	RecentActivity   []ActivityItem  `json:"recent_activity"`
	Recommendations  []string       `json:"recommendations"`
	StudyStreakDays  int            `json:"study_streak_days"`
	WeeklyTrend      []WeekSummary  `json:"weekly_trend"`
	PendingAssessments int         `json:"pending_assessments"`
	LastActiveAt     string         `json:"last_active_at"`
}

type ConceptSummary struct {
	ConceptID   string  `json:"concept_id"`
	Name        string  `json:"name"`
	Mastery     float64 `json:"mastery"`
	Status      string  `json:"status"`
	Attempts    int     `json:"attempts"`
	Correct     int     `json:"correct"`
	ErrorCount  int     `json:"error_count"`
}

type ActivityItem struct {
	Type      string `json:"type"`
	ConceptID string `json:"concept_id"`
	Detail    string `json:"detail"`
	Score     float64 `json:"score,omitempty"`
	Timestamp string `json:"timestamp"`
}

type WeekSummary struct {
	Week            string  `json:"week"`
	AverageScore    float64 `json:"average_score"`
	ActivityCount   int     `json:"activity_count"`
}

type TeacherDashboard struct {
	CourseID          string              `json:"course_id"`
	TotalStudents     int                 `json:"total_students"`
	ActiveThisWeek    int                 `json:"active_this_week"`
	AverageMastery    float64             `json:"average_mastery"`
	AverageScore      float64             `json:"average_score"`
	PassRate          float64             `json:"pass_rate"`
	TopicHeatmap      []TopicHeatmapItem  `json:"topic_heatmap"`
	AtRiskStudents    []AtRiskStudent     `json:"at_risk_students"`
	RecentSubmissions []SubmissionItem    `json:"recent_submissions"`
	CommonErrors      []CommonErrorItem   `json:"common_errors"`
	Interventions     []InterventionItem  `json:"interventions"`
}

type TopicHeatmapItem struct {
	ConceptID       string  `json:"concept_id"`
	Name            string  `json:"name"`
	AverageMastery  float64 `json:"average_mastery"`
	StudentCount    int     `json:"student_count"`
	StrugglingCount int     `json:"struggling_count"`
}

type AtRiskStudent struct {
	StudentID      string  `json:"student_id"`
	Name           string  `json:"name"`
	OverallLevel   float64 `json:"overall_level"`
	ErrorCount     int     `json:"error_count"`
	DaysInactive   int     `json:"days_inactive"`
	WeakConcepts   []string `json:"weak_concepts"`
}

type SubmissionItem struct {
	StudentID   string  `json:"student_id"`
	Name        string  `json:"name"`
	ExerciseID  string  `json:"exercise_id"`
	Score       float64 `json:"score"`
	SubmittedAt string  `json:"submitted_at"`
}

type CommonErrorItem struct {
	ErrorType        string `json:"error_type"`
	Count            int    `json:"count"`
	AffectedStudents int    `json:"affected_students"`
	ConceptID        string `json:"concept_id,omitempty"`
}

type InterventionItem struct {
	StudentID   string   `json:"student_id"`
	Name        string   `json:"name"`
	ConceptID   string   `json:"concept_id"`
	Issue       string   `json:"issue"`
	Severity    string   `json:"severity"`
	Action      string   `json:"action"`
}

func StudentDashboardRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", getStudentDashboard(db))
	}
}

func TeacherDashboardRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", getTeacherDashboard(db))
	}
}

func getStudentDashboard(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentID := r.Context().Value(UserIDKey).(string)
		courseID := r.URL.Query().Get("course_id")
		if courseID == "" {
			courseID = "matematica-1"
		}

		dash := &StudentDashboard{}

		profile, err := GetOrCreateProfile(db, studentID, courseID)
		if err == nil {
			dash.OverallMastery = profile.OverallLevel
			profileMap := map[string]any{
				"overall_level":     profile.OverallLevel,
				"total_attempts":    profile.TotalAttempts,
				"correct_attempts":  profile.CorrectAttempts,
				"total_hints_used":  profile.TotalHintsUsed,
				"study_time_seconds": profile.StudyTimeSeconds,
			}
			dash.Profile = profileMap
		}

		db.QueryRow(r.Context(),
			`SELECT competency_level, competency_score FROM student_analytics WHERE student_id = $1 AND course_id = $2`,
			studentID, courseID).Scan(&dash.CompetencyLevel, &dash.CompetencyScore)

		rows, err := db.Query(r.Context(),
			`SELECT cm.concept_id, c.name, cm.mastery, cm.status,
			        COALESCE(cm.attempts, 0), COALESCE(cm.correct, 0)
			 FROM concept_mastery cm
			 JOIN concepts c ON cm.concept_id = c.id
			 WHERE cm.student_id = $1 AND c.course_id = $2
			 ORDER BY cm.mastery ASC`, studentID, courseID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cs ConceptSummary
				if err := rows.Scan(&cs.ConceptID, &cs.Name, &cs.Mastery, &cs.Status, &cs.Attempts, &cs.Correct); err != nil {
					continue
				}
				cs.ErrorCount = cs.Attempts - cs.Correct
				dash.Concepts = append(dash.Concepts, cs)
			}
			for _, c := range dash.Concepts {
				if c.Mastery < 0.4 {
					dash.WeakestConcepts = append(dash.WeakestConcepts, c.ConceptID)
				} else if c.Mastery >= 0.8 {
					dash.StrongestConcepts = append(dash.StrongestConcepts, c.ConceptID)
				}
			}
		}

		activityRows, err := db.Query(r.Context(),
			`(SELECT 'assessment' as type, a.course_id as concept_id,
			        'Evaluación: ' || a.title as detail, sa.percentage as score, sa.submitted_at::text as ts
			 FROM student_assessments sa
			 JOIN assessments a ON sa.assessment_id = a.id
			 WHERE sa.student_id = $1 AND sa.status = 'graded')
			 UNION ALL
			 (SELECT 'exercise', e.concept_id, 'Ejercicio: ' || e.statement,
			         ea.score, ea.created_at::text
			 FROM exercise_attempts ea
			 JOIN exercises e ON ea.exercise_id = e.id
			 WHERE ea.student_id = $1)
			 UNION ALL
			 (SELECT 'tutor_session', COALESCE(ts.concept_id, ''), 'Sesión de tutoría',
			         ts.total_score, ts.started_at::text
			 FROM tutor_sessions ts
			 WHERE ts.student_id = $1)
			 ORDER BY timestamp DESC LIMIT 10`, studentID)
		if err == nil {
			defer activityRows.Close()
			for activityRows.Next() {
				var ai ActivityItem
				if err := activityRows.Scan(&ai.Type, &ai.ConceptID, &ai.Detail, &ai.Score, &ai.Timestamp); err != nil {
					continue
				}
				dash.RecentActivity = append(dash.RecentActivity, ai)
			}
		}

		db.QueryRow(r.Context(),
			`SELECT COALESCE(study_streak_days, 0) FROM student_analytics WHERE student_id = $1 AND course_id = $2`,
			studentID, courseID).Scan(&dash.StudyStreakDays)

		trendRows, err := db.Query(r.Context(),
			`SELECT DATE_TRUNC('week', ts) as week, AVG(score) as avg_score, COUNT(*) as cnt
			 FROM (
			   SELECT sa.submitted_at as ts, sa.percentage as score FROM student_assessments sa
			   JOIN assessments a ON sa.assessment_id = a.id
			   WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.status = 'graded'
			   UNION ALL
			   SELECT ea.created_at, ea.score FROM exercise_attempts ea WHERE ea.student_id = $1
			 ) acts
			 WHERE ts > NOW() - INTERVAL '12 weeks'
			 GROUP BY DATE_TRUNC('week', ts)
			 ORDER BY week DESC`, studentID, courseID)
		if err == nil {
			defer trendRows.Close()
			for trendRows.Next() {
				var ws WeekSummary
				if err := trendRows.Scan(&ws.Week, &ws.AverageScore, &ws.ActivityCount); err != nil {
					continue
				}
				dash.WeeklyTrend = append(dash.WeeklyTrend, ws)
			}
		}

		db.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM student_assessments sa
			 JOIN assessments a ON sa.assessment_id = a.id
			 WHERE sa.student_id = $1 AND a.course_id = $2 AND sa.status = 'in_progress'`,
			studentID, courseID).Scan(&dash.PendingAssessments)

		if dash.RecentActivity == nil {
			dash.RecentActivity = []ActivityItem{}
		}
		if dash.Concepts == nil {
			dash.Concepts = []ConceptSummary{}
		}
		if dash.WeakestConcepts == nil {
			dash.WeakestConcepts = []string{}
		}
		if dash.StrongestConcepts == nil {
			dash.StrongestConcepts = []string{}
		}
		if dash.WeeklyTrend == nil {
			dash.WeeklyTrend = []WeekSummary{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dash)
	}
}

func getTeacherDashboard(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseID := r.URL.Query().Get("course_id")
		if courseID == "" {
			courseID = "matematica-1"
		}

		ctx := r.Context()
		dash := &TeacherDashboard{CourseID: courseID}

		db.QueryRow(ctx,
			`SELECT COUNT(DISTINCT student_id) FROM student_profiles WHERE course_id = $1`,
			courseID).Scan(&dash.TotalStudents)

		db.QueryRow(ctx,
			`SELECT COUNT(DISTINCT student_id) FROM student_profiles
			 WHERE course_id = $1 AND last_active_at > NOW() - INTERVAL '7 days'`,
			courseID).Scan(&dash.ActiveThisWeek)

		db.QueryRow(ctx,
			`SELECT COALESCE(AVG(overall_level), 0) FROM student_profiles WHERE course_id = $1`,
			courseID).Scan(&dash.AverageMastery)

		db.QueryRow(ctx,
			`SELECT COALESCE(AVG(percentage), 0) FROM student_assessments sa
			 JOIN assessments a ON sa.assessment_id = a.id
			 WHERE a.course_id = $1 AND sa.status = 'graded'`,
			courseID).Scan(&dash.AverageScore)

		db.QueryRow(ctx,
			`SELECT COALESCE(AVG(CASE WHEN passed THEN 1.0 ELSE 0.0 END), 0) FROM student_assessments sa
			 JOIN assessments a ON sa.assessment_id = a.id
			 WHERE a.course_id = $1 AND sa.status = 'graded'`,
			courseID).Scan(&dash.PassRate)

		topicRows, err := db.Query(ctx,
			`SELECT c.id, c.name,
			        COALESCE(AVG(cm.mastery), 0) as avg_mastery,
			        COUNT(DISTINCT cm.student_id) as student_count,
			        COUNT(DISTINCT cm.student_id) FILTER (WHERE cm.mastery < 0.3) as struggling
			 FROM concepts c
			 LEFT JOIN concept_mastery cm ON c.id = cm.concept_id
			 WHERE c.course_id = $1
			 GROUP BY c.id, c.name
			 ORDER BY avg_mastery ASC`, courseID)
		if err == nil {
			defer topicRows.Close()
			for topicRows.Next() {
				var t TopicHeatmapItem
				if err := topicRows.Scan(&t.ConceptID, &t.Name, &t.AverageMastery, &t.StudentCount, &t.StrugglingCount); err != nil {
					continue
				}
				dash.TopicHeatmap = append(dash.TopicHeatmap, t)
			}
		}

		riskRows, err := db.Query(ctx,
			`SELECT u.id, u.name || ' ' || COALESCE(u.last_name, ''),
			        COALESCE(sp.overall_level, 0),
			        COALESCE((SELECT SUM(count) FROM student_errors se WHERE se.student_id = u.id), 0),
			        COALESCE(EXTRACT(DAY FROM NOW() - sp.last_active_at), 999)::int
			 FROM users u
			 JOIN student_profiles sp ON u.id = sp.student_id AND sp.course_id = $1
			 WHERE u.role = 'STUDENT'
			   AND (sp.overall_level < 0.3 OR sp.last_active_at IS NULL OR sp.last_active_at < NOW() - INTERVAL '14 days')
			 ORDER BY sp.overall_level ASC
			 LIMIT 10`, courseID)
		if err == nil {
			defer riskRows.Close()
			for riskRows.Next() {
				var s AtRiskStudent
				if err := riskRows.Scan(&s.StudentID, &s.Name, &s.OverallLevel, &s.ErrorCount, &s.DaysInactive); err != nil {
					continue
				}
				db.QueryRow(ctx,
					`SELECT COALESCE(json_agg(concept_id ORDER BY mastery ASC), '[]'::json)::text
					 FROM concept_mastery
					 WHERE student_id = $1 AND mastery < 0.3
					 LIMIT 3`, s.StudentID).Scan(&s.WeakConcepts)
				dash.AtRiskStudents = append(dash.AtRiskStudents, s)
			}
		}

		subRows, err := db.Query(ctx,
			`SELECT u.id, u.name || ' ' || COALESCE(u.last_name, ''),
			        ea.exercise_id, ea.score, ea.created_at::text
			 FROM exercise_attempts ea
			 JOIN users u ON ea.student_id = u.id
			 ORDER BY ea.created_at DESC
			 LIMIT 20`)
		if err == nil {
			defer subRows.Close()
			for subRows.Next() {
				var s SubmissionItem
				if err := subRows.Scan(&s.StudentID, &s.Name, &s.ExerciseID, &s.Score, &s.SubmittedAt); err != nil {
					continue
				}
				dash.RecentSubmissions = append(dash.RecentSubmissions, s)
			}
		}

		errRows, err := db.Query(ctx,
			`SELECT COALESCE(se.error_type, ''), SUM(se.count) as total,
			        COUNT(DISTINCT se.student_id) as affected,
			        COALESCE(se.concept_id, '')
			 FROM student_errors se
			 JOIN users u ON se.student_id = u.id
			 WHERE se.course_id = $1
			 GROUP BY se.error_type, se.concept_id
			 ORDER BY total DESC
			 LIMIT 10`, courseID)
		if err == nil {
			defer errRows.Close()
			for errRows.Next() {
				var e CommonErrorItem
				if err := errRows.Scan(&e.ErrorType, &e.Count, &e.AffectedStudents, &e.ConceptID); err != nil {
					continue
				}
				dash.CommonErrors = append(dash.CommonErrors, e)
			}
		}

		interventionRows, err := db.Query(ctx,
			`SELECT u.id, u.name || ' ' || COALESCE(u.last_name, ''),
			        cm.concept_id, cm.mastery
			 FROM concept_mastery cm
			 JOIN users u ON cm.student_id = u.id
			 JOIN concepts c ON cm.concept_id = c.id
			 WHERE c.course_id = $1 AND cm.mastery < 0.2
			 ORDER BY cm.mastery ASC
			 LIMIT 10`, courseID)
		if err == nil {
			defer interventionRows.Close()
			for interventionRows.Next() {
				var studentID, name, conceptID string
				var mastery float64
				if err := interventionRows.Scan(&studentID, &name, &conceptID, &mastery); err != nil {
					continue
				}
				dash.Interventions = append(dash.Interventions, InterventionItem{
					StudentID: studentID,
					Name:      name,
					ConceptID: conceptID,
					Issue:     fmt.Sprintf("Mastery muy bajo (%.0f%%) en %s", mastery*100, conceptID),
					Severity:  "alta",
					Action:    "Asignar plan de recuperación y ejercicios remedial",
				})
			}
		}

		if dash.TopicHeatmap == nil {
			dash.TopicHeatmap = []TopicHeatmapItem{}
		}
		if dash.AtRiskStudents == nil {
			dash.AtRiskStudents = []AtRiskStudent{}
		}
		if dash.RecentSubmissions == nil {
			dash.RecentSubmissions = []SubmissionItem{}
		}
		if dash.CommonErrors == nil {
			dash.CommonErrors = []CommonErrorItem{}
		}
		if dash.Interventions == nil {
			dash.Interventions = []InterventionItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dash)
	}
}


