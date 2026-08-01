package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionResponse struct {
	SessionID string    `json:"session_id"`
	Mode      string    `json:"mode"`
	Exercise  *Exercise `json:"exercise,omitempty"`
	Message   string    `json:"message,omitempty"`
}

type AnswerRequest struct {
	SessionID  string   `json:"session_id"`
	ExerciseID string   `json:"exercise_id"`
	Answer     string   `json:"answer"`
	Procedure  []string `json:"procedure,omitempty"`
}

type AnswerResponse struct {
	Correct       bool         `json:"correct"`
	Score         float64      `json:"score"`
	Feedback      string       `json:"feedback"`
	FirstErrorStep int         `json:"first_error_step,omitempty"`
	ErrorType     string       `json:"error_type,omitempty"`
	ErrorDetail   string       `json:"error_detail,omitempty"`
	MasteryBefore float64      `json:"mastery_before"`
	MasteryAfter  float64      `json:"mastery_after"`
	MasteryStatus string       `json:"mastery_status"`
	NextExercise  *Exercise    `json:"next_exercise,omitempty"`
	MathVerified  bool         `json:"math_verified"`
	StepAnalysis  []StepResult `json:"step_analysis,omitempty"`
}

type StepResult struct {
	Step   int    `json:"step"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HintResponse struct {
	HintIndex  int    `json:"hint_index"`
	Hint       string `json:"hint"`
	TotalHints int    `json:"total_hints"`
}

func SessionRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/session", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req struct {
				Mode     string `json:"mode"`
				CourseID string `json:"course_id"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.Mode == "" {
				req.Mode = "tutor"
			}
			if req.CourseID == "" {
				req.CourseID = "matematica-1"
			}

			session, err := CreateSession(db, studentID, req.CourseID, req.Mode)
			if err != nil {
				log.Printf("[SESSION] create error: %v", err)
				http.Error(w, `{"error":"failed to create session"}`, http.StatusInternalServerError)
				return
			}

			exercise, _ := NextExerciseForSession(db, cfg, studentID, req.CourseID, session.ID)

			resp := SessionResponse{
				SessionID: session.ID,
				Mode:      req.Mode,
				Exercise:  exercise,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req AnswerRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}

			resp, err := SubmitAnswer(db, cfg, studentID, &req)
			if err != nil {
				log.Printf("[SESSION] answer error: %v", err)
				http.Error(w, `{"error":"failed to process answer"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Post("/hint", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req struct {
				SessionID  string `json:"session_id"`
				ExerciseID string `json:"exercise_id"`
				HintIndex  int    `json:"hint_index"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			resp, err := RequestHint(db, studentID, req.ExerciseID, req.SessionID, req.HintIndex)
			if err != nil {
				http.Error(w, `{"error":"no hints available"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Post("/feedback", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req struct {
				SessionID  string   `json:"session_id"`
				ExerciseID string   `json:"exercise_id"`
				Procedure  []string `json:"procedure"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			results, err := AnalyzeProcedure(db, cfg, studentID, req.ExerciseID, req.Procedure)
			if err != nil {
				http.Error(w, `{"error":"analysis failed"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		})
	}
}

type Session struct {
	ID       string
	Mode     string
	CourseID string
}

func CreateSession(db *pgxpool.Pool, studentID, courseID, mode string) (*Session, error) {
	ctx := context.Background()
	var s Session
	err := db.QueryRow(ctx,
		`INSERT INTO tutor_sessions (student_id, course_id, mode)
		 VALUES ($1, $2, $3) RETURNING id, mode, course_id`,
		studentID, courseID, mode,
	).Scan(&s.ID, &s.Mode, &s.CourseID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func NextExerciseForSession(db *pgxpool.Pool, cfg *config.Config, studentID, courseID, sessionID string) (*Exercise, error) {
	rec, err := RecommendNext(db, studentID, courseID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var ex Exercise
	err = db.QueryRow(ctx,
		`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
		 FROM exercises
		 WHERE concept_id = $1 AND difficulty = $2 AND status = 'validated'
		 ORDER BY RANDOM() LIMIT 1`,
		rec.RecommendedConcept, rec.RecommendedDifficulty,
	).Scan(&ex.ID, &ex.ConceptID, &ex.Difficulty, &ex.Statement, &ex.Latex,
		&ex.ExpectedAnswer, &ex.Solution, &ex.SolutionSteps, &ex.Hints,
		&ex.CommonErrors, &ex.Source, &ex.VerifiedByMath)

	if err != nil {
		exercise, genErr := GenerateExercise(db, cfg, rec.RecommendedConcept, rec.RecommendedDifficulty)
		if genErr != nil {
			return nil, genErr
		}
		return exercise, nil
	}

	if sessionID != "" {
		db.Exec(ctx, `UPDATE tutor_sessions SET exercise_count = exercise_count + 1 WHERE id = $1`, sessionID)
	}
	return &ex, nil
}

func SubmitAnswer(db *pgxpool.Pool, cfg *config.Config, studentID string, req *AnswerRequest) (*AnswerResponse, error) {
	ctx := context.Background()

	var exercise Exercise
	err := db.QueryRow(ctx,
		`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
		 FROM exercises WHERE id = $1`, req.ExerciseID,
	).Scan(&exercise.ID, &exercise.ConceptID, &exercise.Difficulty, &exercise.Statement, &exercise.Latex,
		&exercise.ExpectedAnswer, &exercise.Solution, &exercise.SolutionSteps, &exercise.Hints,
		&exercise.CommonErrors, &exercise.Source, &exercise.VerifiedByMath)
	if err != nil {
		return nil, fmt.Errorf("exercise not found: %w", err)
	}

	mathClient := NewMathClient(cfg)
	correct := false
	score := 0.0
	mathVerified := false

	verifyResult, err := mathClient.Verify(req.Answer, exercise.ExpectedAnswer, "")
	if err == nil && verifyResult != nil {
		correct = verifyResult.Success
		mathVerified = true
		if correct {
			score = 1.0
		} else if verifyResult.Method != "" {
			score = 0.3
		}
	} else {
		correct = strings.EqualFold(strings.TrimSpace(req.Answer), strings.TrimSpace(exercise.ExpectedAnswer))
		if correct {
			score = 0.8
		}
	}

	var stepAnalysis []StepResult
	firstErrorStep := 0
	errorType := ""
	if len(req.Procedure) > 0 && !correct {
		stepAnalysis, firstErrorStep, errorType = analyzeSteps(req.Procedure, exercise)
		score = calculateStepScore(stepAnalysis)
		if score > 0.5 {
			correct = false
		}
	}

	hintsUsed := 0
	db.QueryRow(ctx, `SELECT hints_used FROM tutor_sessions WHERE id = $1`, req.SessionID).Scan(&hintsUsed)

	var masteryBefore float64
	db.QueryRow(ctx,
		`SELECT COALESCE(mastery, 0) FROM concept_mastery WHERE student_id = $1 AND concept_id = $2`,
		studentID, exercise.ConceptID).Scan(&masteryBefore)

	now := time.Now()
	_, insertErr := db.Exec(ctx,
		`INSERT INTO exercise_attempts (session_id, student_id, exercise_id, answer, correct, score, hints_used, first_error_step, time_seconds, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8)`,
		req.SessionID, studentID, req.ExerciseID, req.Answer, correct, score, hintsUsed, firstErrorStep, now)
	if insertErr == nil {
		go func() {
			ctx2 := context.Background()
			points := 5
			switch {
			case exercise.Difficulty >= 4:
				points = 15
			case exercise.Difficulty == 3:
				points = 10
			}
			if correct {
				_ = RecordActivity(ctx2, db, studentID, "exercise_solved", &exercise.ConceptID, points, map[string]any{"difficulty": exercise.Difficulty})
			} else {
				_ = RecordActivity(ctx2, db, studentID, "exercise_attempt", &exercise.ConceptID, 2, map[string]any{"difficulty": exercise.Difficulty})
			}
			_ = TouchStreak(ctx2, db, studentID)
		}()
	}

	if !correct && firstErrorStep > 0 && errorType != "" {
		RecordError(db, studentID, exercise.ConceptID, errorType, "")
	}

	UpdateMastery(db, studentID, exercise.ConceptID, correct, hintsUsed, score)

	var masteryAfter float64
	db.QueryRow(ctx,
		`SELECT COALESCE(mastery, 0) FROM concept_mastery WHERE student_id = $1 AND concept_id = $2`,
		studentID, exercise.ConceptID).Scan(&masteryAfter)

	status := "learning"
	if masteryAfter >= 0.8 {
		status = "mastered"
	} else if masteryAfter >= 0.5 {
		status = "developing"
	} else if masteryAfter > 0 {
		status = "learning"
	}

	feedback := buildFeedback(correct, score, firstErrorStep, errorType, exercise)

	db.Exec(ctx,
		`UPDATE tutor_sessions SET correct_count = correct_count + CASE WHEN $2 THEN 1 ELSE 0 END, total_score = total_score + $3
		 WHERE id = $1`,
		req.SessionID, correct, score)

	var courseID string
	db.QueryRow(ctx, `SELECT course_id FROM tutor_sessions WHERE id = $1`, req.SessionID).Scan(&courseID)

	resp := &AnswerResponse{
		Correct:        correct,
		Score:          score,
		Feedback:       feedback,
		FirstErrorStep: firstErrorStep,
		ErrorType:      errorType,
		MasteryBefore:  masteryBefore,
		MasteryAfter:   masteryAfter,
		MasteryStatus:  status,
		MathVerified:   mathVerified,
		StepAnalysis:   stepAnalysis,
	}

	if correct || score < 0.2 {
		nextEx, _ := NextExerciseForSession(db, cfg, studentID, courseID, req.SessionID)
		resp.NextExercise = nextEx
	}

	return resp, nil
}

func RequestHint(db *pgxpool.Pool, studentID, exerciseID, sessionID string, hintIndex int) (*HintResponse, error) {
	ctx := context.Background()
	var hints json.RawMessage
	err := db.QueryRow(ctx, `SELECT hints FROM exercises WHERE id = $1`, exerciseID).Scan(&hints)
	if err != nil {
		return nil, err
	}

	var hintsList []string
	json.Unmarshal(hints, &hintsList)

	if hintIndex >= len(hintsList) || hintIndex < 0 {
		return nil, fmt.Errorf("no hints available at index %d", hintIndex)
	}

	db.Exec(ctx,
		`UPDATE tutor_sessions SET hints_used = hints_used + 1 WHERE id = $1`, sessionID)

	return &HintResponse{
		HintIndex:  hintIndex,
		Hint:       hintsList[hintIndex],
		TotalHints: len(hintsList),
	}, nil
}

func AnalyzeProcedure(db *pgxpool.Pool, cfg *config.Config, studentID, exerciseID string, procedure []string) ([]StepResult, error) {
	if len(procedure) == 0 {
		return nil, fmt.Errorf("empty procedure")
	}

	var exercise Exercise
	err := db.QueryRow(context.Background(),
		`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
		 FROM exercises WHERE id = $1`, exerciseID,
	).Scan(&exercise.ID, &exercise.ConceptID, &exercise.Difficulty, &exercise.Statement, &exercise.Latex,
		&exercise.ExpectedAnswer, &exercise.Solution, &exercise.SolutionSteps, &exercise.Hints,
		&exercise.CommonErrors, &exercise.Source, &exercise.VerifiedByMath)
	if err != nil {
		return nil, err
	}

	results, _, _ := analyzeSteps(procedure, exercise)
	return results, nil
}

func analyzeSteps(procedure []string, exercise Exercise) ([]StepResult, int, string) {
	results := make([]StepResult, len(procedure))
	firstError := 0
	errorType := ""

	mathClient := NewMathClient(&config.Config{
		MathServiceURL: "",
		MathTimeout:    5,
	})

	for i, step := range procedure {
		results[i] = StepResult{Step: i + 1, Status: "correct"}

		if i == len(procedure)-1 {
			verifyResult, err := mathClient.Verify(step, exercise.ExpectedAnswer, "")
			if err == nil && verifyResult != nil {
				if !verifyResult.Success {
					results[i].Status = "incorrect"
					results[i].Error = "La respuesta final no coincide con el resultado esperado."
					if firstError == 0 {
						firstError = i + 1
						errorType = "arithmetic"
					}
				}
			}
		}
	}

	return results, firstError, errorType
}

func calculateStepScore(steps []StepResult) float64 {
	if len(steps) == 0 {
		return 0
	}
	correct := 0
	for _, s := range steps {
		if s.Status == "correct" {
			correct++
		}
	}
	return float64(correct) / float64(len(steps))
}

func buildFeedback(correct bool, score float64, firstErrorStep int, errorType string, exercise Exercise) string {
	if correct && score >= 0.9 {
		return "¡Correcto! Excelente resolución."
	}
	if correct {
		return "Correcto, pero revisa tu procedimiento para mayor claridad."
	}
	if firstErrorStep > 0 {
		feedback := fmt.Sprintf("El error está en el paso %d.", firstErrorStep)
		switch errorType {
		case "algebraic":
			feedback += " Revisa la manipulación algebraica en ese paso."
		case "arithmetic":
			feedback += " Verifica el cálculo aritmético en ese paso."
		case "sign":
			feedback += " Presta atención a los signos en ese paso."
		case "formula":
			feedback += " Verifica que estás usando la fórmula correcta."
		default:
			feedback += " Analiza ese paso con cuidado."
		}
		return feedback
	}
	return "Tu respuesta no es correcta. Intenta revisar el enunciado y los conceptos involucrados."
}
