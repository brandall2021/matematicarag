package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AcademicAlert struct {
	ID             string          `json:"id"`
	StudentID      string          `json:"student_id"`
	AlertType      string          `json:"alert_type"`
	Severity       string          `json:"severity"`
	Title          string          `json:"title"`
	Message        string          `json:"message"`
	ConceptID      string          `json:"concept_id"`
	AssessmentID   *string         `json:"assessment_id,omitempty"`
	Acknowledged   bool            `json:"acknowledged"`
	AcknowledgedBy *string         `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *string         `json:"acknowledged_at,omitempty"`
	Metadata       json.RawMessage `json:"metadata"`
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
