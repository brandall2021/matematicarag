package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseProgress struct {
	CourseID       string  `json:"course_id"`
	TotalStudents  int     `json:"total_students"`
	AverageMastery float64 `json:"average_mastery"`
}

type TopicMastery struct {
	ConceptID       string  `json:"concept_id"`
	ConceptName     string  `json:"concept_name"`
	AverageMastery  float64 `json:"average_mastery"`
	StudentCount    int     `json:"student_count"`
	StrugglingCount int     `json:"struggling_count"`
}

type CommonError struct {
	ErrorType        string `json:"error_type"`
	ErrorSubtype     string `json:"error_subtype"`
	Count            int    `json:"count"`
	AffectedStudents int    `json:"affected_students"`
}

type StudentProgress struct {
	StudentID     string  `json:"student_id"`
	StudentName   string  `json:"student_name"`
	Email         string  `json:"email"`
	OverallLevel  float64 `json:"overall_level"`
	TotalAttempts int     `json:"total_attempts"`
	LastActive    string  `json:"last_active"`
}

func TeacherRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {

		r.Get("/course-progress", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			var cp CourseProgress
			cp.CourseID = courseID
			db.QueryRow(r.Context(),
				`SELECT COUNT(DISTINCT sp.student_id), COALESCE(AVG(sp.overall_level), 0)
				 FROM student_profiles sp
				 JOIN users u ON sp.student_id = u.id
				 WHERE sp.course_id = $1 AND u.role = 'STUDENT'`,
				courseID).Scan(&cp.TotalStudents, &cp.AverageMastery)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cp)
		})

		r.Get("/topic-mastery", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			rows, err := db.Query(r.Context(),
				`SELECT c.id, c.name,
				   COALESCE(AVG(cm.mastery), 0) as avg_mastery,
				   COUNT(DISTINCT cm.student_id) as student_count,
				   COUNT(DISTINCT cm.student_id) FILTER (WHERE cm.mastery < 0.3) as struggling
				 FROM concepts c
				 LEFT JOIN concept_mastery cm ON c.id = cm.concept_id
				 WHERE c.course_id = $1
				 GROUP BY c.id, c.name
				 ORDER BY c.id`, courseID)
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var topics []TopicMastery
			for rows.Next() {
				var t TopicMastery
				if err := rows.Scan(&t.ConceptID, &t.ConceptName, &t.AverageMastery, &t.StudentCount, &t.StrugglingCount); err != nil {
					continue
				}
				topics = append(topics, t)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(topics)
		})

		r.Get("/common-errors", func(w http.ResponseWriter, r *http.Request) {
			rows, err := db.Query(r.Context(),
				`SELECT error_type, error_subtype, SUM(count) as total_count,
				   COUNT(DISTINCT student_id) as affected
				 FROM student_errors
				 GROUP BY error_type, error_subtype
				 ORDER BY total_count DESC
				 LIMIT 15`)
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var errs []CommonError
			for rows.Next() {
				var e CommonError
				if err := rows.Scan(&e.ErrorType, &e.ErrorSubtype, &e.Count, &e.AffectedStudents); err != nil {
					continue
				}
				errs = append(errs, e)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errs)
		})

		r.Get("/student-progress", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			rows, err := db.Query(r.Context(),
				`SELECT u.id, u.name || ' ' || COALESCE(u.last_name, ''), u.email,
				   COALESCE(sp.overall_level, 0), COALESCE(sp.total_attempts, 0),
				   COALESCE(sp.last_active_at::text, '')
				 FROM users u
				 LEFT JOIN student_profiles sp ON u.id = sp.student_id AND sp.course_id = $1
				 WHERE u.role = 'STUDENT'
				 ORDER BY sp.overall_level DESC NULLS LAST`, courseID)
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var students []StudentProgress
			for rows.Next() {
				var s StudentProgress
				if err := rows.Scan(&s.StudentID, &s.StudentName, &s.Email, &s.OverallLevel, &s.TotalAttempts, &s.LastActive); err != nil {
					continue
				}
				students = append(students, s)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(students)
		})
	}
}
