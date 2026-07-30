package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/api/adaptive"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentProfile struct {
	ID               string  `json:"id"`
	StudentID        string  `json:"student_id"`
	CourseID         string  `json:"course_id"`
	OverallLevel     float64 `json:"overall_level"`
	TotalAttempts    int     `json:"total_attempts"`
	CorrectAttempts  int     `json:"correct_attempts"`
	TotalHintsUsed   int     `json:"total_hints_used"`
	StudyTimeSeconds int     `json:"study_time_seconds"`
}

type ConceptMastery struct {
	ID         string  `json:"id"`
	StudentID  string  `json:"student_id"`
	ConceptID  string  `json:"concept_id"`
	Mastery    float64 `json:"mastery"`
	Status     string  `json:"status"`
	Attempts   int     `json:"attempts"`
	Correct    int     `json:"correct"`
	HintsUsed  int     `json:"hints_used"`
	ErrorCount int     `json:"error_count"`
}

type StudentError struct {
	ID             string `json:"id"`
	StudentID      string `json:"student_id"`
	ConceptID      string `json:"concept_id"`
	ErrorType      string `json:"error_type"`
	ErrorSubtype   string `json:"error_subtype"`
	Count          int    `json:"count"`
	Severity       string `json:"severity"`
	LastOccurredAt string `json:"last_occurred_at"`
}

func LearningRoutes(db *pgxpool.Pool, adaptEngine *adaptive.AdaptiveEngine) func(r chi.Router) {
	return func(r chi.Router) {

		r.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			profile, err := GetOrCreateProfile(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load profile"}`, http.StatusInternalServerError)
				return
			}
			mastery, err := GetMasteryMap(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load mastery"}`, http.StatusInternalServerError)
				return
			}
			totalExercises := 0
			totalMastery := 0.0
			for _, m := range mastery {
				totalExercises += m.Attempts
				totalMastery += m.Mastery
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"student_id":       profile.StudentID,
				"overall_mastery":  profile.OverallLevel,
				"total_attempts":   profile.TotalAttempts,
				"correct_attempts": profile.CorrectAttempts,
				"total_exercises":  totalExercises,
				"total_hints":      profile.TotalHintsUsed,
				"study_time":       profile.StudyTimeSeconds,
				"mastery":          mastery,
			})
		})

		r.Get("/progress", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			profile, err := GetOrCreateProfile(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load profile"}`, http.StatusInternalServerError)
				return
			}
			mastery, err := GetMasteryMap(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load mastery"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"profile": profile,
				"mastery": mastery,
			})
		})

		r.Get("/mastery", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			mastery, err := GetMasteryMap(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load mastery"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mastery)
		})

		r.Get("/errors", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			errors, err := GetStudentErrors(db, studentID)
			if err != nil {
				http.Error(w, `{"error":"failed to load errors"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errors)
		})

		r.Post("/events", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var event adaptive.LearningEvent
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			event.StudentID = studentID

			if err := adaptEngine.Events.ProcessEvent(adaptEngine, &event); err != nil {
				http.Error(w, `{"error":"failed to process event"}`, http.StatusInternalServerError)
				return
			}

			state, _ := adaptEngine.Analytics.GetStudentProgress(r.Context(), studentID, "matematica-1")
			rec, _ := adaptEngine.Recommend.GenerateRecommendation(r.Context(), studentID, "matematica-1", state)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"mastery":        event.ConceptID,
				"recommendation": rec,
			})
		})

		r.Get("/learner-profile", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			state, err := adaptEngine.Analytics.GetStudentProgress(r.Context(), studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load learner profile"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(state)
		})

		r.Get("/recommendation", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			state, err := adaptEngine.Analytics.GetStudentProgress(r.Context(), studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load learner state"}`, http.StatusInternalServerError)
				return
			}
			rec, err := adaptEngine.Recommend.GenerateRecommendation(r.Context(), studentID, "matematica-1", state)
			if err != nil {
				http.Error(w, `{"error":"failed to generate recommendation"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rec)
		})

		r.Get("/path", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			path, err := adaptEngine.LearningPath.BuildPath(r.Context(), studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to build learning path"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(path)
		})

		r.Post("/suggest", func(w http.ResponseWriter, r *http.Request) {
			var rec adaptive.Recommendation
			if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			explanation := adaptEngine.Recommend.ExplainRecommendation(&rec)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"explanation": explanation,
			})
		})

		r.Get("/course-analytics", func(w http.ResponseWriter, r *http.Request) {
			analytics, err := adaptEngine.Analytics.GetCourseAnalytics(r.Context(), "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load course analytics"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(analytics)
		})

		r.Get("/errors/common", func(w http.ResponseWriter, r *http.Request) {
			errors, err := adaptEngine.Analytics.GetCommonErrors(r.Context(), "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load common errors"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errors)
		})
	}
}

func GetOrCreateProfile(db *pgxpool.Pool, studentID, courseID string) (*StudentProfile, error) {
	ctx := context.Background()
	var p StudentProfile
	err := db.QueryRow(ctx,
		`INSERT INTO student_profiles (student_id, course_id)
		 VALUES ($1, $2)
		 ON CONFLICT (student_id) DO UPDATE SET updated_at = NOW()
		 RETURNING id, student_id, course_id, overall_level, total_attempts, correct_attempts, total_hints_used, study_time_seconds`,
		studentID, courseID,
	).Scan(&p.ID, &p.StudentID, &p.CourseID, &p.OverallLevel, &p.TotalAttempts, &p.CorrectAttempts, &p.TotalHintsUsed, &p.StudyTimeSeconds)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetMasteryMap(db *pgxpool.Pool, studentID, courseID string) (map[string]ConceptMastery, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT cm.id, cm.student_id, cm.concept_id, cm.mastery, cm.status, cm.attempts, cm.correct, cm.hints_used, cm.error_count
		 FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE cm.student_id = $1 AND c.course_id = $2`,
		studentID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]ConceptMastery)
	for rows.Next() {
		var m ConceptMastery
		if err := rows.Scan(&m.ID, &m.StudentID, &m.ConceptID, &m.Mastery, &m.Status, &m.Attempts, &m.Correct, &m.HintsUsed, &m.ErrorCount); err != nil {
			continue
		}
		result[m.ConceptID] = m
	}
	return result, nil
}

func UpdateMastery(db *pgxpool.Pool, studentID, conceptID string, correct bool, hintsUsed int, score float64) error {
	ctx := context.Background()

	masteryDelta := 0.0
	if correct {
		masteryDelta = 0.05 * (1.0 - float64(hintsUsed)*0.1) * score
		if masteryDelta < 0.01 {
			masteryDelta = 0.01
		}
	} else {
		masteryDelta = -0.03
	}

	_, err := db.Exec(ctx,
		`INSERT INTO concept_mastery (student_id, concept_id, mastery, status, attempts, correct, hints_used, last_attempt_at)
		 VALUES ($1, $2, GREATEST(0, LEAST(1, $3)), CASE
		   WHEN GREATEST(0, LEAST(1, $3)) >= 0.8 THEN 'mastered'
		   WHEN GREATEST(0, LEAST(1, $3)) >= 0.5 THEN 'developing'
		   WHEN GREATEST(0, LEAST(1, $3)) > 0.0 THEN 'learning'
		   ELSE 'not_started' END,
		   1, CASE WHEN $4 THEN 1 ELSE 0 END, $5, NOW())
		 ON CONFLICT (student_id, concept_id) DO UPDATE SET
		   mastery = GREATEST(0, LEAST(1, concept_mastery.mastery + $3)),
		   status = CASE
		     WHEN GREATEST(0, LEAST(1, concept_mastery.mastery + $3)) >= 0.8 THEN 'mastered'
		     WHEN GREATEST(0, LEAST(1, concept_mastery.mastery + $3)) >= 0.5 THEN 'developing'
		     WHEN GREATEST(0, LEAST(1, concept_mastery.mastery + $3)) > 0.0 THEN 'learning'
		     ELSE 'not_started' END,
		   attempts = concept_mastery.attempts + 1,
		   correct = concept_mastery.correct + CASE WHEN $4 THEN 1 ELSE 0 END,
		   hints_used = concept_mastery.hints_used + $5,
		   last_attempt_at = NOW(),
		   updated_at = NOW()`,
		studentID, conceptID, masteryDelta, correct, hintsUsed)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx,
		`UPDATE student_profiles SET
		   total_attempts = total_attempts + 1,
		   correct_attempts = correct_attempts + CASE WHEN $2 THEN 1 ELSE 0 END,
		   total_hints_used = total_hints_used + $3,
		   overall_level = (SELECT COALESCE(AVG(mastery), 0) FROM concept_mastery WHERE student_id = $1),
		   last_active_at = NOW(),
		   updated_at = NOW()
		 WHERE student_id = $1`,
		studentID, correct, hintsUsed)
	return err
}

func GetStudentErrors(db *pgxpool.Pool, studentID string) ([]StudentError, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT id, student_id, concept_id, error_type, error_subtype, count, severity, last_occurred_at
		 FROM student_errors WHERE student_id = $1 ORDER BY count DESC`,
		studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []StudentError
	for rows.Next() {
		var e StudentError
		if err := rows.Scan(&e.ID, &e.StudentID, &e.ConceptID, &e.ErrorType, &e.ErrorSubtype, &e.Count, &e.Severity, &e.LastOccurredAt); err != nil {
			continue
		}
		errors = append(errors, e)
	}
	return errors, nil
}
