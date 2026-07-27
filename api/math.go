package api

import (
	"encoding/json"
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

func MathRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	mathClient := NewMathClient(cfg)

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
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
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
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
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
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
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
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
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
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
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
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
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
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})
	}
}
