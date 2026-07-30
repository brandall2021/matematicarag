package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EvaluationStatus string

const (
	StatusCorrect         EvaluationStatus = "CORRECT"
	StatusPartiallyCorrect EvaluationStatus = "PARTIALLY_CORRECT"
	StatusIncorrect       EvaluationStatus = "INCORRECT"
	StatusUndetermined    EvaluationStatus = "UNDETERMINED"
	StatusInvalidInput    EvaluationStatus = "INVALID_INPUT"
)

type EvaluationResult struct {
	AttemptID         string          `json:"attempt_id"`
	Status            EvaluationStatus `json:"status"`
	Score             float64         `json:"score"`
	FinalAnswerCorrect bool            `json:"final_answer_correct"`
	StepsCorrect      int             `json:"steps_correct"`
	StepsTotal        int             `json:"steps_total"`
	ErrorType         string          `json:"error_type,omitempty"`
	ConceptID         string          `json:"concept_id,omitempty"`
	Confidence        float64         `json:"confidence"`
	Feedback          string          `json:"feedback,omitempty"`
	Evidence          json.RawMessage `json:"evidence,omitempty"`
}

type StepEval struct {
	StepNumber int    `json:"step_number"`
	Expected   string `json:"expected"`
	Actual     string `json:"actual"`
	Status     string `json:"status"`
	ErrorType  string `json:"error_type,omitempty"`
}

type EvaluateRequest struct {
	Expression string    `json:"expression"`
	Expected   string    `json:"expected"`
	ConceptID  string    `json:"concept_id"`
	StudentID  string    `json:"student_id"`
	Steps      []StepEval `json:"steps,omitempty"`
	ActivityID string    `json:"activity_id,omitempty"`
}

type MathEvaluator struct {
	mathClient  *MathClient
	httpClient  *http.Client
	mathSvcURL  string
}

var reSpace = regexp.MustCompile(`\s+`)
var reMultiParen = regexp.MustCompile(`[()]{2,}`)

func NewMathEvaluator(mathClient *MathClient, mathSvcURL string) *MathEvaluator {
	return &MathEvaluator{
		mathClient: mathClient,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		mathSvcURL: mathSvcURL,
	}
}

func (e *MathEvaluator) NormalizeAnswer(raw string) string {
	s := raw
	s = strings.TrimSpace(s)
	s = reSpace.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\\ ", "\\")
	s = strings.ReplaceAll(s, "{ }", "{}")
	s = strings.ReplaceAll(s, "\\cdot ", "*")
	s = strings.ReplaceAll(s, "\\times ", "*")
	s = strings.ReplaceAll(s, "\\div ", "/")
	s = strings.ReplaceAll(s, "\\approx ", "=")
	s = strings.ReplaceAll(s, "\\rightarrow ", "=")
	s = strings.ReplaceAll(s, "\\Rightarrow ", "=")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

func (e *MathEvaluator) Evaluate(ctx context.Context, req *EvaluateRequest) (*EvaluationResult, error) {
	normalized := e.NormalizeAnswer(req.Expression)
	expectedNorm := e.NormalizeAnswer(req.Expected)

	evidence := map[string]interface{}{
		"expected":          req.Expected,
		"student":           req.Expression,
		"expected_normalized": expectedNorm,
		"student_normalized":  normalized,
	}

	if normalized == "" && expectedNorm != "" {
		return e.result(req, normalized, StatusInvalidInput, 0, false, "empty_response", evidence, 1.0, "La expresión está vacía."), nil
	}

	if strings.TrimSpace(req.Expression) == strings.TrimSpace(req.Expected) {
		return e.result(req, normalized, StatusCorrect, 1.0, true, "", evidence, 1.0, "Respuesta correcta."), nil
	}

	stepsCorrect := 0
	stepsTotal := len(req.Steps)
	var firstError string

	for _, step := range req.Steps {
		stepNorm := e.NormalizeAnswer(step.Actual)
		stepExpNorm := e.NormalizeAnswer(step.Expected)
		if stepNorm == stepExpNorm || e.symbolicMatch(step.Actual, step.Expected) {
			stepsCorrect++
		} else if firstError == "" {
			firstError = step.ErrorType
			if firstError == "" {
				firstError = fmt.Sprintf("step_%d_error", step.StepNumber)
			}
		}
	}

	stepsMatch := stepsTotal == 0 || stepsCorrect == stepsTotal

	symbolicOK := false
	if normalized != "" && expectedNorm != "" {
		symbolicOK = e.symbolicMatch(req.Expression, req.Expected)
	}

	numOK := false
	if !symbolicOK && normalized != "" && expectedNorm != "" {
		numOK = e.numericMatch(normalized, expectedNorm)
	}

	finalCorrect := symbolicOK || numOK

	if !finalCorrect && symbolicOK {
		finalCorrect = true
	}

	if !finalCorrect && stepsMatch && stepsTotal > 0 {
		finalCorrect = true
	}

	var status EvaluationStatus
	var score float64
	var errorType string
	var feedback string
	var confidence float64

	switch {
	case finalCorrect && stepsMatch:
		status = StatusCorrect
		score = 1.0
		confidence = 0.98
		feedback = "Respuesta correcta."
		errorType = ""
	case finalCorrect && !stepsMatch:
		status = StatusPartiallyCorrect
		score = 0.7
		confidence = 0.85
		errorType = firstError
		feedback = "El resultado final es correcto, pero hay errores en el procedimiento."
	case !finalCorrect && stepsMatch:
		status = StatusPartiallyCorrect
		score = 0.6
		confidence = 0.80
		errorType = firstError
		if firstError == "" {
			errorType = "final_answer_mismatch"
		}
		feedback = "Revisa el resultado final."
	case !finalCorrect && stepsPartiallyCorrect(stepsCorrect, stepsTotal):
		status = StatusPartiallyCorrect
		score = 0.4
		confidence = 0.75
		errorType = firstError
		feedback = "Parte del procedimiento es correcto, pero el resultado final no."
	default:
		status = StatusIncorrect
		score = 0.0
		confidence = 0.90
		errorType = firstError
		if errorType == "" {
			errorType = "incorrect_answer"
		}
		feedback = "Respuesta incorrecta."
	}

	if !symbolicOK && !numOK && normalized != "" && expectedNorm != "" {
		confidence = 0.60
		if status == StatusCorrect {
			status = StatusPartiallyCorrect
			score = 0.5
		}
	}

	if normalized == "" && expectedNorm == "" {
		evJSON, _ := json.Marshal(evidence)
		return &EvaluationResult{
			Status:     StatusUndetermined,
			Score:      0.0,
			Confidence: 0.0,
			Feedback:   "No se pudo evaluar la respuesta.",
			Evidence:   evJSON,
		}, nil
	}

	evJSON, _ := json.Marshal(evidence)

	return &EvaluationResult{
		Status:             status,
		Score:              score,
		FinalAnswerCorrect: finalCorrect,
		StepsCorrect:       stepsCorrect,
		StepsTotal:         stepsTotal,
		ErrorType:          errorType,
		ConceptID:          req.ConceptID,
		Confidence:         confidence,
		Feedback:           feedback,
		Evidence:           evJSON,
	}, nil
}

func (e *MathEvaluator) symbolicMatch(student, expected string) bool {
	if student == "" || expected == "" {
		return false
	}
	payload, _ := json.Marshal(map[string]string{
		"expression": student,
		"expected":   expected,
		"operation":  "verify",
	})
	resp, err := e.httpClient.Post(e.mathSvcURL+"/api/math/verify", "application/json", bytes.NewReader(payload))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Verified bool `json:"verified"`
	}
	if json.Unmarshal(body, &result) != nil {
		return false
	}
	return result.Verified
}

func (e *MathEvaluator) numericMatch(student, expected string) bool {
	if !looksNumeric(student) || !looksNumeric(expected) {
		return false
	}
	payload, _ := json.Marshal(map[string]string{
		"expression": student,
	})
	resp, err := e.httpClient.Post(e.mathSvcURL+"/api/math/evaluate", "application/json", bytes.NewReader(payload))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var sResult struct {
		Result string `json:"result"`
	}
	if json.Unmarshal(body, &sResult) != nil || sResult.Result == "" {
		return false
	}

	payload2, _ := json.Marshal(map[string]string{
		"expression": expected,
	})
	resp2, err := e.httpClient.Post(e.mathSvcURL+"/api/math/evaluate", "application/json", bytes.NewReader(payload2))
	if err != nil {
		return false
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	var eResult struct {
		Result string `json:"result"`
	}
	if json.Unmarshal(body2, &eResult) != nil || eResult.Result == "" {
		return false
	}
	return sResult.Result == eResult.Result
}

func (e *MathEvaluator) PersistAttempt(ctx context.Context, db *pgxpool.Pool, result *EvaluationResult, req *EvaluateRequest) (string, error) {
	var attemptID string
	err := db.QueryRow(ctx,
		`INSERT INTO math_attempts (student_id, activity_id, concept_id, answer_raw, answer_normalized, status, score, final_answer_correct, steps_correct, steps_total, error_type, confidence, feedback, evidence)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING id`,
		req.StudentID, req.ActivityID, req.ConceptID, req.Expression,
		e.NormalizeAnswer(req.Expression), string(result.Status), result.Score,
		result.FinalAnswerCorrect, result.StepsCorrect, result.StepsTotal,
		result.ErrorType, result.Confidence, result.Feedback, result.Evidence,
	).Scan(&attemptID)
	if err != nil {
		return "", fmt.Errorf("persist attempt: %w", err)
	}

	for _, step := range req.Steps {
		_, err := db.Exec(ctx,
			`INSERT INTO math_step_results (attempt_id, step_number, expected, actual, status, error_type)
			 VALUES ($1,$2,$3,$4,$5,$6)`,
			attemptID, step.StepNumber, step.Expected, step.Actual,
			step.Status, step.ErrorType,
		)
		if err != nil {
			return attemptID, fmt.Errorf("persist step %d: %w", step.StepNumber, err)
		}
	}

	if result.Status == StatusCorrect || result.Status == StatusPartiallyCorrect {
		e.recordLearningEvent(ctx, db, req.StudentID, req.ConceptID, result)
	}

	return attemptID, nil
}

func (e *MathEvaluator) recordLearningEvent(ctx context.Context, db *pgxpool.Pool, studentID, conceptID string, result *EvaluationResult) {
	isCorrect := result.Status == StatusCorrect
	score := result.Score
	if score < 0.6 {
		score = 0.6
	}
	db.Exec(ctx,
		`INSERT INTO learning_events (student_id, course_id, concept_id, event_type, correct, score, error_type, metadata)
		 VALUES ($1,'matematica-1',$2,$3,$4,$5,$6,
		   jsonb_build_object('attempt_id',$7,'confidence',$8,'feedback',$9))`,
		studentID, conceptID, "math_evaluation", isCorrect, score,
		result.ErrorType, result.AttemptID, result.Confidence, result.Feedback,
	)
}

func (e *MathEvaluator) result(req *EvaluateRequest, normalized string, status EvaluationStatus, score float64, finalCorrect bool, errorType string, evidence map[string]interface{}, confidence float64, feedback string) *EvaluationResult {
	evJSON, _ := json.Marshal(evidence)
	return &EvaluationResult{
		Status:             status,
		Score:              score,
		FinalAnswerCorrect: finalCorrect,
		ErrorType:          errorType,
		ConceptID:          req.ConceptID,
		Confidence:         confidence,
		Feedback:           feedback,
		Evidence:           evJSON,
	}
}

var reNumeric = regexp.MustCompile(`^[\d\s.,+\-*/^()eE]+$`)

func looksNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if unicode.IsLetter(rune(s[0])) && s[0] != 'e' && s[0] != 'E' && s[0] != 'p' && s[0] != 'i' {
		return false
	}
	clean := strings.NewReplacer("pi", "3", "e", "2", "sin", "", "cos", "", "tan", "", "log", "", "ln", "", "sqrt", "", "abs", "").Replace(s)
	return reNumeric.MatchString(clean)
}

func stepsPartiallyCorrect(correct, total int) bool {
	return total > 0 && correct > 0 && correct < total
}
