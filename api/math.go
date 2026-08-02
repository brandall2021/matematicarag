package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MathRequest struct {
	Expression string  `json:"expression"`
	Variable   string  `json:"variable,omitempty"`
	XMin       float64 `json:"xMin,omitempty"`
	XMax       float64 `json:"xMax,omitempty"`
}

func mathError(w http.ResponseWriter, err error) {
	log.Printf("[MATH] operation failed: %v", err)
	http.Error(w, `{"error":"math operation failed"}`, http.StatusInternalServerError)
}

func MathRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	mathClient := NewMathClient(cfg)
	evaluator := NewMathEvaluator(mathClient, cfg.MathServiceURL)

	return func(r chi.Router) {
		r.Post("/evaluate", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Expression == "" {
				http.Error(w, `{"error":"expression is required"}`, http.StatusBadRequest)
				return
			}
			result, err := mathClient.Evaluate(req.Expression)
			if err != nil {
				var clientErr *MathClientError
				if errors.As(err, &clientErr) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "La expresión no se pudo evaluar: " + clientErr.Reason})
					return
				}
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/evaluate/steps", func(w http.ResponseWriter, r *http.Request) {
			studentID := ""
			if uid, ok := r.Context().Value(UserIDKey).(string); ok {
				studentID = uid
			}
			var evalReq EvaluateRequest
			if err := json.NewDecoder(r.Body).Decode(&evalReq); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			evalReq.StudentID = studentID
			if evalReq.Expression == "" {
				http.Error(w, `{"error":"expression is required"}`, http.StatusBadRequest)
				return
			}
			result, err := evaluator.Evaluate(r.Context(), &evalReq)
			if err != nil {
				mathError(w, err)
				return
			}
			if studentID != "" {
				attemptID, persistErr := evaluator.PersistAttempt(r.Context(), db, result, &evalReq)
				if persistErr == nil {
					result.AttemptID = attemptID
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/differentiate", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			variable := req.Variable
			if variable == "" {
				variable = "x"
			}
			result, err := mathClient.Differentiate(req.Expression, variable)
			if err != nil {
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/integrate", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			variable := req.Variable
			if variable == "" {
				variable = "x"
			}
			result, err := mathClient.Integrate(req.Expression, variable, nil, nil)
			if err != nil {
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/solve", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			variable := req.Variable
			if variable == "" {
				variable = "x"
			}
			result, err := mathClient.Solve(req.Expression, variable)
			if err != nil {
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/simplify", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			result, err := mathClient.Simplify(req.Expression)
			if err != nil {
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/factor", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			result, err := mathClient.Factor(req.Expression)
			if err != nil {
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/expand", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			result, err := mathClient.Expand(req.Expression)
			if err != nil {
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/plot", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Expression == "" {
				http.Error(w, `{"error":"expression is required"}`, http.StatusBadRequest)
				return
			}
			xmin, xmax := -10.0, 10.0
			if req.XMin != 0 || req.XMax != 0 {
				xmin, xmax = req.XMin, req.XMax
			}
			result, err := mathClient.Plot(req.Expression, xmin, xmax)
			if err != nil {
				var clientErr *MathClientError
				if errors.As(err, &clientErr) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "La expresión no se pudo graficar: " + clientErr.Reason})
					return
				}
				mathError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})
	}
}
