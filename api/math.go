package api

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type MathRequest struct {
	Expression string  `json:"expression"`
	Variable   string  `json:"variable,omitempty"`
	Point      string  `json:"point,omitempty"`
	XMin       float64 `json:"xMin,omitempty"`
	XMax       float64 `json:"xMax,omitempty"`
	A          string  `json:"a,omitempty"`
	B          string  `json:"b,omitempty"`
}

type MathResponse struct {
	Success bool   `json:"success"`
	Result  string `json:"result"`
	Error   string `json:"error,omitempty"`
}

type PlotResponse struct {
	XValues         []float64 `json:"xValues"`
	YValues         []float64 `json:"yValues"`
	Expression      string    `json:"expression"`
	LatexExpression string    `json:"latexExpression"`
}

func MathRoutes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/evaluate", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			result := evaluateMath(req.Expression)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(MathResponse{Success: true, Result: result})
		})

		r.Post("/plot", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.XMin == 0 && req.XMax == 0 {
				req.XMin = -10
				req.XMax = 10
			}
			n := 200
			xValues := make([]float64, n+1)
			yValues := make([]float64, n+1)
			step := (req.XMax - req.XMin) / float64(n)
			for i := 0; i <= n; i++ {
				x := req.XMin + float64(i)*step
				xValues[i] = math.Round(x*1000) / 1000
				yValues[i] = 0
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(PlotResponse{
				XValues: xValues, YValues: yValues,
				Expression: req.Expression, LatexExpression: req.Expression,
			})
		})

		r.Post("/derive", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "derive") })
		r.Post("/integrate", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "integrate") })
		r.Post("/solve", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "solve") })
		r.Post("/simplify", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "simplify") })
		r.Post("/factor", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "factor") })
		r.Post("/expand", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "expand") })
		r.Post("/limit", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "limit") })
		r.Post("/sum", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "sum") })
		r.Post("/product", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "product") })
		r.Post("/roots", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "roots") })
		r.Post("/matrix-determinant", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "matrix-determinant") })
		r.Post("/matrix-inverse", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "matrix-inverse") })
		r.Post("/matrix-rank", func(w http.ResponseWriter, r *http.Request) { handleMathOp(w, r, "matrix-rank") })
	}
}

func handleMathOp(w http.ResponseWriter, r *http.Request, operation string) {
	var req MathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MathResponse{Success: true, Result: operation + " placeholder"})
}

func evaluateMath(expr string) string {
	if strings.Contains(expr, "^") {
		parts := strings.Split(expr, "^")
		if len(parts) == 2 {
			base, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			exp, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return strconv.FormatFloat(math.Pow(base, exp), 'f', -1, 64)
			}
		}
	}
	return expr
}
