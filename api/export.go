package api

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ExportRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/assessment/{assessmentID}/csv", ExportAssessmentCSV(db))
		r.Get("/student/{studentID}/csv", ExportStudentCSV(db))
		r.Get("/course/{courseID}/csv", ExportCourseCSV(db))
	}
}

func ExportAssessmentCSV(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assessmentID := chi.URLParam(r, "assessmentID")

		rows, err := db.Query(context.Background(), `
			SELECT
				u.name,
				u.email,
				sa.total_score,
				sa.max_score,
				sa.percentage,
				sa.passed,
				sa.attempt_number,
				sa.submitted_at
			FROM student_assessments sa
			JOIN users u ON u.id = sa.student_id
			WHERE sa.assessment_id = $1
			ORDER BY sa.submitted_at ASC
		`, assessmentID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to query assessment results: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="assessment_%s.csv"`, assessmentID))

		writer := csv.NewWriter(w)
		defer writer.Flush()

		header := []string{"Nombre", "Email", "Puntuación", "Máximo", "Porcentaje (%)", "Aprobó (Sí/No)", "Intento", "Fecha"}
		if err := writer.Write(header); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to write CSV header: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			var name, email string
			var totalScore, maxScore, percentage float64
			var passed bool
			var attemptNumber int
			var submittedAt *string

			if err := rows.Scan(&name, &email, &totalScore, &maxScore, &percentage, &passed, &attemptNumber, &submittedAt); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"failed to scan row: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}

			aprobó := "No"
			if passed {
				aprobó = "Sí"
			}

			fecha := ""
			if submittedAt != nil {
				fecha = *submittedAt
			}

			record := []string{
				name,
				email,
				fmt.Sprintf("%.2f", totalScore),
				fmt.Sprintf("%.2f", maxScore),
				fmt.Sprintf("%.2f", percentage),
				aprobó,
				fmt.Sprintf("%d", attemptNumber),
				fecha,
			}
			if err := writer.Write(record); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"failed to write CSV row: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
		}

		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"row iteration error: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
	}
}

func ExportStudentCSV(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentID := chi.URLParam(r, "studentID")

		rows, err := db.Query(context.Background(), `
			SELECT
				a.title,
				a.assessment_type,
				sa.total_score,
				sa.max_score,
				sa.percentage,
				sa.passed,
				sa.submitted_at
			FROM student_assessments sa
			JOIN assessments a ON a.id = sa.assessment_id
			WHERE sa.student_id = $1
			ORDER BY sa.submitted_at ASC
		`, studentID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to query student assessments: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="student_%s.csv"`, studentID))

		writer := csv.NewWriter(w)
		defer writer.Flush()

		header := []string{"Evaluación", "Tipo", "Puntuación", "Máximo", "Porcentaje", "Aprobó", "Fecha"}
		if err := writer.Write(header); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to write CSV header: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			var title, assessmentType string
			var totalScore, maxScore, percentage float64
			var passed bool
			var submittedAt *string

			if err := rows.Scan(&title, &assessmentType, &totalScore, &maxScore, &percentage, &passed, &submittedAt); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"failed to scan row: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}

			aprobó := "No"
			if passed {
				aprobó = "Sí"
			}

			fecha := ""
			if submittedAt != nil {
				fecha = *submittedAt
			}

			record := []string{
				title,
				assessmentType,
				fmt.Sprintf("%.2f", totalScore),
				fmt.Sprintf("%.2f", maxScore),
				fmt.Sprintf("%.2f", percentage),
				aprobó,
				fecha,
			}
			if err := writer.Write(record); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"failed to write CSV row: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
		}

		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"row iteration error: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
	}
}

func ExportCourseCSV(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseID := chi.URLParam(r, "courseID")

		rows, err := db.Query(context.Background(), `
			SELECT
				a.title AS assessment_title,
				COUNT(sa.id) AS total_students,
				COALESCE(AVG(sa.total_score), 0) AS avg_score,
				COALESCE(
					SUM(CASE WHEN sa.passed THEN 1 ELSE 0 END)::float / NULLIF(COUNT(sa.id), 0) * 100,
					0
				) AS approval_rate
			FROM assessments a
			LEFT JOIN student_assessments sa ON sa.assessment_id = a.id
			WHERE a.course_id = $1
			GROUP BY a.id, a.title
			ORDER BY a.title ASC
		`, courseID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to query course analytics: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="course_%s.csv"`, courseID))

		writer := csv.NewWriter(w)
		defer writer.Flush()

		header := []string{"Evaluación", "Total Estudiantes", "Promedio", "Tasa Aprobación"}
		if err := writer.Write(header); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to write CSV header: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			var title string
			var totalStudents int
			var avgScore, approvalRate float64

			if err := rows.Scan(&title, &totalStudents, &avgScore, &approvalRate); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"failed to scan row: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}

			record := []string{
				title,
				fmt.Sprintf("%d", totalStudents),
				fmt.Sprintf("%.2f", avgScore),
				fmt.Sprintf("%.2f%%", approvalRate),
			}
			if err := writer.Write(record); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"failed to write CSV row: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
		}

		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"row iteration error: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
	}
}
