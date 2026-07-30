package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CriticalConcept struct {
	ConceptID       string  `json:"concept_id"`
	Name            string  `json:"name"`
	AverageMastery  float64 `json:"average_mastery"`
	TotalStudents   int     `json:"total_students"`
	StrugglingCount int     `json:"struggling_count"`
	ErrorCount      int     `json:"error_count"`
	ErrorTypes      []string `json:"error_types,omitempty"`
	AttemptCount    int     `json:"attempt_count"`
	SuccessRate     float64 `json:"success_rate"`
	Criticality     float64 `json:"criticality_score"`
	Trend           string  `json:"trend"`
	Priority        string  `json:"priority"`
}

type ConceptDetail struct {
	ConceptID       string                `json:"concept_id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	AverageMastery  float64               `json:"average_mastery"`
	TotalStudents   int                   `json:"total_students"`
	StrugglingCount int                   `json:"struggling_count"`
	StrugglingStudents []StrugglingStudent `json:"struggling_students"`
	ErrorBreakdown  []ErrorBreakdown      `json:"error_breakdown"`
	WeeklyTrend     []ConceptWeekTrend    `json:"weekly_trend"`
	Recommendations []string              `json:"recommendations"`
}

type StrugglingStudent struct {
	StudentID    string  `json:"student_id"`
	Name         string  `json:"name"`
	Mastery      float64 `json:"mastery"`
	ErrorCount   int     `json:"error_count"`
	LastAttempt  string  `json:"last_attempt,omitempty"`
}

type ErrorBreakdown struct {
	ErrorType        string `json:"error_type"`
	Count            int    `json:"count"`
	AffectedStudents int    `json:"affected_students"`
}

type ConceptWeekTrend struct {
	Week           string  `json:"week"`
	MasteryAvg     float64 `json:"mastery_avg"`
	Attempts       int     `json:"attempts"`
	NewStudents    int     `json:"new_students"`
}

func CriticalConceptsRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", getCriticalConcepts(db))
		r.Get("/export", exportCriticalConceptsCSV(db))
		r.Get("/concept/{conceptID}", getConceptDetail(db))
	}
}

func getCriticalConcepts(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseID := r.URL.Query().Get("course_id")
		if courseID == "" {
			courseID = "matematica-1"
		}
		minStudents := 1
		if ms := r.URL.Query().Get("min_students"); ms != "" {
			fmt.Sscanf(ms, "%d", &minStudents)
		}
		threshold := 0.6
		if t := r.URL.Query().Get("threshold"); t != "" {
			fmt.Sscanf(t, "%f", &threshold)
		}

		ctx := r.Context()

		rows, err := db.Query(ctx, `
			SELECT
				c.id, c.name,
				COALESCE(AVG(cm.mastery), 0) as avg_mastery,
				COUNT(DISTINCT cm.student_id) as student_count,
				COUNT(DISTINCT cm.student_id) FILTER (WHERE cm.mastery < 0.3) as struggling,
				COALESCE(SUM(se.count), 0) as total_errors,
				COALESCE(SUM(cm.attempts), 0) as total_attempts,
				COALESCE(
					CASE WHEN SUM(cm.attempts) > 0
					THEN SUM(cm.correct)::float / SUM(cm.attempts)
					ELSE 0 END, 0
				) as success_rate
			FROM concepts c
			LEFT JOIN concept_mastery cm ON c.id = cm.concept_id
			LEFT JOIN student_errors se ON c.id = se.concept_id
			WHERE c.course_id = $1
			GROUP BY c.id, c.name
			HAVING COUNT(DISTINCT cm.student_id) >= $2
			ORDER BY avg_mastery ASC
		`, courseID, minStudents)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		result := make([]CriticalConcept, 0)
		for rows.Next() {
			var cc CriticalConcept
			if err := rows.Scan(&cc.ConceptID, &cc.Name, &cc.AverageMastery,
				&cc.TotalStudents, &cc.StrugglingCount, &cc.ErrorCount,
				&cc.AttemptCount, &cc.SuccessRate); err != nil {
				continue
			}

			errTypes, _ := db.Query(ctx,
				`SELECT error_type FROM student_errors
				 WHERE concept_id = $1 AND course_id = $2
				 GROUP BY error_type ORDER BY SUM(count) DESC LIMIT 3`,
				cc.ConceptID, courseID)
			if errTypes != nil {
				for errTypes.Next() {
					var et string
					errTypes.Scan(&et)
					cc.ErrorTypes = append(cc.ErrorTypes, et)
				}
				errTypes.Close()
			}

			riskFactor := 0.0
			if cc.TotalStudents > 0 {
				strugglingRatio := float64(cc.StrugglingCount) / float64(cc.TotalStudents)
				errorDensity := float64(cc.ErrorCount) / math.Max(float64(cc.AttemptCount), 1)
				masteryGap := 1.0 - cc.AverageMastery
				successGap := 1.0 - cc.SuccessRate
				cc.Criticality = math.Round((strugglingRatio*0.3+masteryGap*0.25+errorDensity*0.25+successGap*0.2)*100) / 100
				riskFactor = strugglingRatio + masteryGap
			}

			switch {
			case riskFactor >= 0.5:
				cc.Priority = "critical"
			case riskFactor >= 0.3:
				cc.Priority = "high"
			case riskFactor >= 0.15:
				cc.Priority = "medium"
			default:
				cc.Priority = "low"
			}

			cc.Trend = computeTrend(ctx, db, cc.ConceptID, courseID)

			if cc.AverageMastery < threshold {
				result = append(result, cc)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func computeTrend(ctx context.Context, db *pgxpool.Pool, conceptID, courseID string) string {
	var recent, previous float64
	db.QueryRow(ctx,
		`SELECT COALESCE(AVG(mastery), 0) FROM concept_mastery
		 WHERE concept_id = $1 AND last_success_at > NOW() - INTERVAL '14 days'`,
		conceptID).Scan(&recent)
	db.QueryRow(ctx,
		`SELECT COALESCE(AVG(mastery), 0) FROM concept_mastery
		 WHERE concept_id = $1 AND last_error_at BETWEEN NOW() - INTERVAL '28 days' AND NOW() - INTERVAL '14 days'`,
		conceptID).Scan(&previous)

	diff := recent - previous
	if diff > 0.05 {
		return "improving"
	}
	if diff < -0.05 {
		return "declining"
	}
	return "stable"
}

func getConceptDetail(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conceptID := chi.URLParam(r, "conceptID")
		courseID := r.URL.Query().Get("course_id")
		if courseID == "" {
			courseID = "matematica-1"
		}

		ctx := r.Context()
		detail := &ConceptDetail{ConceptID: conceptID}

		db.QueryRow(ctx,
			`SELECT name, COALESCE(description, '') FROM concepts WHERE id = $1`,
			conceptID).Scan(&detail.Name, &detail.Description)

		db.QueryRow(ctx,
			`SELECT COALESCE(AVG(mastery), 0),
			        COUNT(DISTINCT student_id),
			        COUNT(DISTINCT student_id) FILTER (WHERE mastery < 0.3)
			 FROM concept_mastery WHERE concept_id = $1`,
			conceptID).Scan(&detail.AverageMastery, &detail.TotalStudents, &detail.StrugglingCount)

		studRows, err := db.Query(ctx,
			`SELECT u.id, u.name || ' ' || COALESCE(u.last_name, ''), cm.mastery,
			        COALESCE((SELECT SUM(count) FROM student_errors se WHERE se.student_id = u.id AND se.concept_id = $1), 0),
			        COALESCE(cm.last_success_at::text, cm.last_error_at::text, '')
			 FROM concept_mastery cm
			 JOIN users u ON cm.student_id = u.id
			 WHERE cm.concept_id = $1 AND cm.mastery < 0.3
			 ORDER BY cm.mastery ASC
			 LIMIT 20`, conceptID)
		if err == nil {
			defer studRows.Close()
			for studRows.Next() {
				var s StrugglingStudent
				if err := studRows.Scan(&s.StudentID, &s.Name, &s.Mastery, &s.ErrorCount, &s.LastAttempt); err != nil {
					continue
				}
				detail.StrugglingStudents = append(detail.StrugglingStudents, s)
			}
		}

		errRows, err := db.Query(ctx,
			`SELECT error_type, SUM(count) as total, COUNT(DISTINCT student_id) as affected
			 FROM student_errors
			 WHERE concept_id = $1 AND course_id = $2
			 GROUP BY error_type
			 ORDER BY total DESC`, conceptID, courseID)
		if err == nil {
			defer errRows.Close()
			for errRows.Next() {
				var eb ErrorBreakdown
				if err := errRows.Scan(&eb.ErrorType, &eb.Count, &eb.AffectedStudents); err != nil {
					continue
				}
				detail.ErrorBreakdown = append(detail.ErrorBreakdown, eb)
			}
		}

		trendRows, err := db.Query(ctx,
			`SELECT DATE_TRUNC('week', le.created_at) as week,
			        AVG(CASE WHEN le.event_type = 'mastery_update' THEN le.score ELSE NULL END) as mastery_avg,
			        COUNT(*) as attempts,
			        COUNT(DISTINCT le.student_id) FILTER (
			          WHERE le.created_at > NOW() - INTERVAL '12 weeks'
			          AND le.student_id NOT IN (
			            SELECT student_id FROM learning_events
			            WHERE concept_id = $1 AND created_at < NOW() - INTERVAL '12 weeks'
			          )
			        ) as new_students
			 FROM learning_events le
			 WHERE le.concept_id = $1 AND le.created_at > NOW() - INTERVAL '12 weeks'
			 GROUP BY DATE_TRUNC('week', le.created_at)
			 ORDER BY week DESC`, conceptID)
		if err == nil {
			defer trendRows.Close()
			for trendRows.Next() {
				var wt ConceptWeekTrend
				if err := trendRows.Scan(&wt.Week, &wt.MasteryAvg, &wt.Attempts, &wt.NewStudents); err != nil {
					continue
				}
				detail.WeeklyTrend = append(detail.WeeklyTrend, wt)
			}
		}

		detail.Recommendations = buildConceptRecommendations(detail)

		if detail.StrugglingStudents == nil {
			detail.StrugglingStudents = []StrugglingStudent{}
		}
		if detail.ErrorBreakdown == nil {
			detail.ErrorBreakdown = []ErrorBreakdown{}
		}
		if detail.WeeklyTrend == nil {
			detail.WeeklyTrend = []ConceptWeekTrend{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}

func buildConceptRecommendations(d *ConceptDetail) []string {
	recs := make([]string, 0)
	if d.AverageMastery < 0.3 {
		recs = append(recs, "Revisar material teórico del concepto y crear ejercicios remedial")
		recs = append(recs, "Asignar plan de recuperación a estudiantes con dominio < 20%")
	}
	if d.AverageMastery < 0.5 {
		recs = append(recs, "Programar sesión de repaso grupal para este concepto")
	}
	if d.StrugglingCount > 0 {
		ratio := float64(d.StrugglingCount) / math.Max(float64(d.TotalStudents), 1)
		if ratio > 0.5 {
			recs = append(recs, "Intervención urgente: más del 50% de estudiantes está por debajo del umbral mínimo")
		}
	}
	if len(d.WeeklyTrend) >= 2 {
		last := d.WeeklyTrend[0]
		prev := d.WeeklyTrend[1]
		if last.MasteryAvg < prev.MasteryAvg {
			recs = append(recs, "La tendencia semanal es negativa. Revisar metodología de enseñanza")
		}
	}
	if len(recs) == 0 {
		recs = append(recs, "Monitoreo regular. No se requiere intervención inmediata")
	}
	return recs
}

func exportCriticalConceptsCSV(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseID := r.URL.Query().Get("course_id")
		if courseID == "" {
			courseID = "matematica-1"
		}
		threshold := 0.6
		if t := r.URL.Query().Get("threshold"); t != "" {
			fmt.Sscanf(t, "%f", &threshold)
		}

		ctx := r.Context()

		rows, err := db.Query(ctx, `
			SELECT c.id, c.name,
			       COALESCE(AVG(cm.mastery), 0),
			       COUNT(DISTINCT cm.student_id),
			       COUNT(DISTINCT cm.student_id) FILTER (WHERE cm.mastery < 0.3),
			       COALESCE(SUM(se.count), 0),
			       COALESCE(SUM(cm.attempts), 0),
			       COALESCE(CASE WHEN SUM(cm.attempts) > 0 THEN SUM(cm.correct)::float / SUM(cm.attempts) ELSE 0 END, 0)
			FROM concepts c
			LEFT JOIN concept_mastery cm ON c.id = cm.concept_id
			LEFT JOIN student_errors se ON c.id = se.concept_id AND se.course_id = $1
			WHERE c.course_id = $1
			GROUP BY c.id, c.name
			HAVING COALESCE(AVG(cm.mastery), 0) < $2
			ORDER BY AVG(cm.mastery) ASC
		`, courseID, threshold)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="critical_concepts_%s.csv"`, courseID))

		writer := csv.NewWriter(w)
		defer writer.Flush()

		writer.Write([]string{"Concepto", "Dominio Promedio", "Estudiantes", "En Dificultad", "Errores", "Intentos", "Tasa Éxito", "Prioridad"})

		for rows.Next() {
			var id, name string
			var avgMastery float64
			var totalStudents, struggling, errorCount, attempts int
			var successRate float64

			if err := rows.Scan(&id, &name, &avgMastery, &totalStudents, &struggling, &errorCount, &attempts, &successRate); err != nil {
				continue
			}

			riskFactor := 0.0
			if totalStudents > 0 {
				strugglingRatio := float64(struggling) / float64(totalStudents)
				masteryGap := 1.0 - avgMastery
				riskFactor = strugglingRatio + masteryGap
			}

			priority := "low"
			switch {
			case riskFactor >= 0.5:
				priority = "CRÍTICA"
			case riskFactor >= 0.3:
				priority = "ALTA"
			case riskFactor >= 0.15:
				priority = "MEDIA"
			}

			writer.Write([]string{
				name,
				fmt.Sprintf("%.1f%%", avgMastery*100),
				fmt.Sprintf("%d", totalStudents),
				fmt.Sprintf("%d", struggling),
				fmt.Sprintf("%d", errorCount),
				fmt.Sprintf("%d", attempts),
				fmt.Sprintf("%.1f%%", successRate*100),
				priority,
			})
		}
	}
}
