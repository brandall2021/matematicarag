package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func MathRoutes(db *pgxpool.Pool) func(r chi.Router) {
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

			// Try local eval first for simple expressions
			if result, ok := localEvaluate(req.Expression); ok {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(MathResponse{Success: true, Result: result})
				return
			}

			// Fall back to OpenAI
			customMathPrompt := getSetting(db, "SYSTEM_PROMPT")
			if customMathPrompt == "" {
				customMathPrompt = `Sos un experto en matematicas. Analizas la expresion o instruccion del usuario y realizas la operacion correspondiente: evaluar, derivar, integrar, resolver ecuaciones, simplificar, factorizar, calcular limites, sumas, productos, raices, determinantes, etc. Respondes con el resultado paso a paso en español. Si la expresion es solo un numero,respondes con ese numero.`
			}
			result, err := callOpenAI(db, customMathPrompt, req.Expression, "")
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(MathResponse{Success: false, Error: err.Error()})
				return
			}
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

		r.Post("/derive", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "derivar", "derivative") })
		r.Post("/integrate", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "integrar", "integral") })
		r.Post("/solve", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "resolver ecuacion", "solve equation") })
		r.Post("/simplify", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "simplificar", "simplify") })
		r.Post("/factor", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "factorizar", "factor") })
		r.Post("/expand", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "expandir", "expand") })
		r.Post("/limit", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "calcular limite", "limit") })
		r.Post("/sum", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "calcular suma", "sum") })
		r.Post("/product", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "calcular producto", "product") })
		r.Post("/roots", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "encontrar raices", "find roots") })
		r.Post("/matrix-determinant", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "determinante de matriz", "matrix determinant") })
		r.Post("/matrix-inverse", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "inversa de matriz", "matrix inverse") })
		r.Post("/matrix-rank", func(w http.ResponseWriter, r *http.Request) { handleMathOpAI(w, r, db, "rango de matriz", "matrix rank") })
	}
}

func handleMathOpAI(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, opName string, opEnglish string) {
	var req MathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.Expression == "" {
		http.Error(w, `{"error":"expression is required"}`, http.StatusBadRequest)
		return
	}

	customPrompt := getSetting(db, "MATH_SYSTEM_PROMPT")
	if customPrompt == "" {
		customPrompt = fmt.Sprintf("Sos un experto en matematicas. %s expresiones matematicas. Respondes con el resultado paso a paso en español.", strings.Title(opName))
	}
	systemPrompt := customPrompt
	userPrompt := fmt.Sprintf("%s: %s", opName, req.Expression)

	result, err := callOpenAI(db, systemPrompt, userPrompt, "")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MathResponse{Success: false, Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MathResponse{Success: true, Result: result})
}

func localEvaluate(expr string) (string, bool) {
	// Try simple power: a^b
	if strings.Contains(expr, "^") {
		parts := strings.Split(expr, "^")
		if len(parts) == 2 {
			base, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			exp, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return strconv.FormatFloat(math.Pow(base, exp), 'f', -1, 64), true
			}
		}
	}
	// Try simple arithmetic
	expr = strings.ReplaceAll(expr, " ", "")
	if isNumeric(expr) {
		return expr, true
	}
	return "", false
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
