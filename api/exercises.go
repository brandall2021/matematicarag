package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Exercise struct {
	ID             string          `json:"id"`
	ConceptID      string          `json:"concept_id"`
	Difficulty     int             `json:"difficulty"`
	Statement      string          `json:"statement"`
	Latex          string          `json:"latex"`
	ExpectedAnswer string          `json:"expected_answer"`
	Solution       string          `json:"solution"`
	SolutionSteps  json.RawMessage `json:"solution_steps"`
	Hints          json.RawMessage `json:"hints"`
	CommonErrors   json.RawMessage `json:"common_errors"`
	Source         string          `json:"source"`
	VerifiedByMath bool            `json:"verified_by_math"`
}

func ExerciseRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/next", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}
			ex, reason, err := GetNextExercise(r.Context(), db, cfg, studentID, courseID)
			if err != nil {
				log.Printf("[EXERCISE] next exercise failed: %v", err)
				http.Error(w, `{"error":"failed to get next exercise"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"exercise": ex,
				"reason":   reason,
			})
		})

		r.Post("/generate", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				ConceptID  string `json:"concept_id"`
				Difficulty int    `json:"difficulty"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Difficulty < 1 || req.Difficulty > 5 {
				req.Difficulty = 2
			}

			exercise, err := GenerateExercise(db, cfg, req.ConceptID, req.Difficulty)
			if err != nil {
				log.Printf("[EXERCISE] generation failed: %v", err)
				http.Error(w, `{"error":"generation failed"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(exercise)
		})

		r.Get("/concept/{conceptID}", func(w http.ResponseWriter, r *http.Request) {
			conceptID := chi.URLParam(r, "conceptID")
			difficulty := r.URL.Query().Get("difficulty")

			var rows pgx.Rows
			var err error
			if difficulty != "" {
				rows, err = db.Query(r.Context(),
					`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
					 FROM exercises WHERE concept_id = $1 AND difficulty = $2 AND status = 'validated'
					 ORDER BY RANDOM() LIMIT 1`, conceptID, difficulty)
			} else {
				rows, err = db.Query(r.Context(),
					`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
					 FROM exercises WHERE concept_id = $1 AND status = 'validated'
					 ORDER BY RANDOM() LIMIT 1`, conceptID)
			}
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			if rows.Next() {
				var ex Exercise
				if err := rows.Scan(&ex.ID, &ex.ConceptID, &ex.Difficulty, &ex.Statement, &ex.Latex, &ex.ExpectedAnswer, &ex.Solution, &ex.SolutionSteps, &ex.Hints, &ex.CommonErrors, &ex.Source, &ex.VerifiedByMath); err != nil {
					http.Error(w, `{"error":"scan failed"}`, http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ex)
			} else {
				http.Error(w, `{"error":"no exercises found"}`, http.StatusNotFound)
			}
		})

		r.Get("/{exerciseID}", func(w http.ResponseWriter, r *http.Request) {
			exerciseID := chi.URLParam(r, "exerciseID")
			var ex Exercise
			err := db.QueryRow(r.Context(),
				`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
				 FROM exercises WHERE id = $1`, exerciseID,
			).Scan(&ex.ID, &ex.ConceptID, &ex.Difficulty, &ex.Statement, &ex.Latex, &ex.ExpectedAnswer, &ex.Solution, &ex.SolutionSteps, &ex.Hints, &ex.CommonErrors, &ex.Source, &ex.VerifiedByMath)
			if err != nil {
				http.Error(w, `{"error":"exercise not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ex)
		})

		r.Post("/{exerciseID}/validate", func(w http.ResponseWriter, r *http.Request) {
			exerciseID := chi.URLParam(r, "exerciseID")
			mathClient := NewMathClient(cfg)
			var ex Exercise
			err := db.QueryRow(r.Context(),
				`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, source, verified_by_math
				 FROM exercises WHERE id = $1`, exerciseID,
			).Scan(&ex.ID, &ex.ConceptID, &ex.Difficulty, &ex.Statement, &ex.Latex, &ex.ExpectedAnswer, &ex.Solution, &ex.Source, &ex.VerifiedByMath)
			if err != nil {
				http.Error(w, `{"error":"exercise not found"}`, http.StatusNotFound)
				return
			}

			verifyResult, err := mathClient.Verify(ex.ExpectedAnswer, ex.ExpectedAnswer, "")
			mathValid := err == nil && verifyResult != nil && verifyResult.Success

			db.Exec(r.Context(),
				`UPDATE exercises SET verified_by_math = $1, status = $2 WHERE id = $3`,
				mathValid, map[bool]string{true: "validated", false: "invalid"}[mathValid], exerciseID,
			)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"exercise_id":  exerciseID,
				"valid":        mathValid,
				"math_checked": true,
			})
		})

		r.Get("/{exerciseID}/hints", func(w http.ResponseWriter, r *http.Request) {
			exerciseID := chi.URLParam(r, "exerciseID")
			var hints json.RawMessage
			err := db.QueryRow(r.Context(),
				`SELECT hints FROM exercises WHERE id = $1`, exerciseID,
			).Scan(&hints)
			if err != nil {
				http.Error(w, `{"error":"exercise not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(hints)
		})
	}
}

func GetNextExercise(ctx context.Context, db *pgxpool.Pool, cfg *config.Config, studentID, courseID string) (*Exercise, string, error) {
	profile, err := GetOrCreateProfile(db, studentID, courseID)
	if err != nil {
		return nil, "", fmt.Errorf("profile: %w", err)
	}

	mastery, err := GetMasteryMap(db, studentID, courseID)
	if err != nil {
		return nil, "", fmt.Errorf("mastery: %w", err)
	}

	targetConcept := ""
	targetDifficulty := 2

	lowestMastery := 1.0
	for cid, cm := range mastery {
		if cm.Mastery < lowestMastery {
			lowestMastery = cm.Mastery
			targetConcept = cid
		}
	}

	if targetConcept == "" {
		var cid string
		err := db.QueryRow(ctx,
			`SELECT id FROM concepts WHERE course_id = $1 ORDER BY difficulty_base ASC LIMIT 1`, courseID,
		).Scan(&cid)
		if err == nil {
			targetConcept = cid
		}
	}

	if targetConcept == "" {
		return nil, "", fmt.Errorf("no concept available")
	}

	m := mastery[targetConcept]
	switch {
	case m.Mastery >= 0.8:
		targetDifficulty = int(math.Min(float64(m.Attempts/3+3), 5))
	case m.Mastery >= 0.5:
		targetDifficulty = int(math.Min(float64(m.Attempts/3+2), 4))
	default:
		targetDifficulty = int(math.Max(2-float64(m.Attempts), 1))
	}
	if targetDifficulty < 1 {
		targetDifficulty = 1
	}
	if targetDifficulty > 5 {
		targetDifficulty = 5
	}

	var ex Exercise
	err = db.QueryRow(ctx,
		`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
		 FROM exercises
		 WHERE concept_id = $1 AND difficulty = $2 AND status = 'validated'
		 ORDER BY RANDOM() LIMIT 1`,
		targetConcept, targetDifficulty,
	).Scan(&ex.ID, &ex.ConceptID, &ex.Difficulty, &ex.Statement, &ex.Latex, &ex.ExpectedAnswer, &ex.Solution, &ex.SolutionSteps, &ex.Hints, &ex.CommonErrors, &ex.Source, &ex.VerifiedByMath)
	if err == nil {
		return &ex, fmt.Sprintf("Ejercicio de %s (dificultad %d)", targetConcept, targetDifficulty), nil
	}

	ex2, err := GenerateExercise(db, cfg, targetConcept, targetDifficulty)
	if err != nil {
		return nil, "", fmt.Errorf("generate: %w", err)
	}

	rec := fmt.Sprintf("Ejercicio generado de %s (dificultad %d). Mastery actual: %.0f%%", targetConcept, targetDifficulty, profile.OverallLevel*100)
	return ex2, rec, nil
}

func GenerateExercise(db *pgxpool.Pool, cfg *config.Config, conceptID string, difficulty int) (*Exercise, error) {
	ctx := context.Background()

	conceptName := ""
	db.QueryRow(ctx, `SELECT name FROM concepts WHERE id = $1`, conceptID).Scan(&conceptName)
	if conceptName == "" {
		conceptName = conceptID
	}

	systemPrompt := `Eres un generador de ejercicios de matemática.
Genera UN ejercicio para el concepto indicado con la dificultad especificada.
Responde SOLO con JSON válido:
{
  "statement": "enunciado en texto plano",
  "latex": "enunciado en LaTeX",
  "expected_answer": "respuesta esperada en texto",
  "solution": "solución completa en texto",
  "solution_steps": [{"step": 1, "explanation": "...", "latex": "..."}],
  "hints": ["pista 1", "pista 2", "pista 3"],
  "common_errors": [{"type": "algebraic", "description": "..."}]
}
No copies ejercicios existentes. Sé original.`

	userMsg := fmt.Sprintf("Concepto: %s (%s)\nDificultad: %d/5\nGenera el ejercicio.", conceptName, conceptID, difficulty)

	messages := []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}

	response, err := callOpenAIWithHistory(db, messages, "")
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	response = strings.TrimSpace(response)
	if idx := strings.Index(response, "{"); idx != -1 {
		response = response[idx:]
	}
	if idx := strings.LastIndex(response, "}"); idx != -1 {
		response = response[:idx+1]
	}

	var parsed struct {
		Statement      string          `json:"statement"`
		Latex          string          `json:"latex"`
		ExpectedAnswer string          `json:"expected_answer"`
		Solution       string          `json:"solution"`
		SolutionSteps  json.RawMessage `json:"solution_steps"`
		Hints          json.RawMessage `json:"hints"`
		CommonErrors   json.RawMessage `json:"common_errors"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	mathClient := NewMathClient(cfg)
	verifyResult, err := mathClient.Verify(parsed.ExpectedAnswer, parsed.ExpectedAnswer, "")
	if err != nil {
		log.Printf("[EXERCISE] math verify failed: %v", err)
	}

	verified := verifyResult != nil && verifyResult.Success

	hintsJSON, _ := json.Marshal(parsed.Hints)
	errorsJSON, _ := json.Marshal(parsed.CommonErrors)
	stepsJSON, _ := json.Marshal(parsed.SolutionSteps)

	exercise := &Exercise{
		ConceptID:      conceptID,
		Difficulty:     difficulty,
		Statement:      parsed.Statement,
		Latex:          parsed.Latex,
		ExpectedAnswer: parsed.ExpectedAnswer,
		Solution:       parsed.Solution,
		SolutionSteps:  stepsJSON,
		Hints:          hintsJSON,
		CommonErrors:   errorsJSON,
		Source:         "generated",
		VerifiedByMath: verified,
	}

	var exID string
	err = db.QueryRow(ctx,
		`INSERT INTO exercises (concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id`,
		exercise.ConceptID, exercise.Difficulty, exercise.Statement, exercise.Latex,
		exercise.ExpectedAnswer, exercise.Solution, exercise.SolutionSteps,
		exercise.Hints, exercise.CommonErrors, exercise.Source, exercise.VerifiedByMath,
		func() string {
			if verified {
				return "validated"
			}
			return "pending"
		}(),
	).Scan(&exID)
	if err != nil {
		return nil, fmt.Errorf("insert exercise: %w", err)
	}
	exercise.ID = exID

	return exercise, nil
}
