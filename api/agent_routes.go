package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/brandall2021/matematicarag/api/adaptive"
	"github.com/brandall2021/matematicarag/api/agent"
	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentHandler struct {
	agent *agent.PedagogicalAgent
	db    *pgxpool.Pool
}

func NewAgentHandler(a *agent.PedagogicalAgent, db *pgxpool.Pool) *AgentHandler {
	return &AgentHandler{agent: a, db: db}
}

func (h *AgentHandler) Chat(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(string)

	var req agent.AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	req.StudentID = userID
	if req.CourseID == "" {
		req.CourseID = "matematica-1"
	}
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("agent-%d-%d", time.Now().UnixNano(), rand.Intn(1000))
	}
	if req.Mode == "" {
		req.Mode = "tutor"
	}

	resp, err := h.agent.Process(r.Context(), &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "agent processing failed: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func AgentRoutes(db *pgxpool.Pool, a *agent.PedagogicalAgent) func(r chi.Router) {
	handler := NewAgentHandler(a, db)
	return func(r chi.Router) {
		r.Post("/chat", handler.Chat)
	}
}

func CallLLMForAgent(ctx context.Context, db *pgxpool.Pool, prompt string) (string, error) {
	var provider, model, apiKey string
	err := db.QueryRow(ctx,
		`SELECT COALESCE((SELECT value FROM app_settings WHERE key = 'ai_provider'), 'openai'),
		        COALESCE((SELECT value FROM app_settings WHERE key = 'ai_model'), 'gpt-4o-mini'),
		        COALESCE((SELECT value FROM app_settings WHERE key = 'openai_api_key'), '')`).
		Scan(&provider, &model, &apiKey)
	if err != nil {
		return "", fmt.Errorf("settings: %w", err)
	}

	if apiKey == "" {
		apiKey = getEnvSetting("OPENAI_API_KEY")
	}

	response, err := callOpenAICompatible(provider, apiKey, model, prompt, "")
	if err == nil && response != "" {
		return response, nil
	}

	var anthropicKey string
	db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = 'anthropic_api_key'`).Scan(&anthropicKey)
	if anthropicKey == "" {
		anthropicKey = getEnvSetting("ANTHROPIC_API_KEY")
	}
	if anthropicKey != "" {
		response, err = callAnthropic(anthropicKey, model, prompt, "")
		if err == nil && response != "" {
			return response, nil
		}
	}

	response, err = callOpenAICompatible(provider, apiKey, model, prompt, "")
	if err == nil {
		return response, nil
	}

	return "", fmt.Errorf("LLM call failed: %w", err)
}

func getEnvSetting(key string) string {
	// The config is loaded at startup but not accessible here in handler context
	// Fallback to direct env var
	return os.Getenv(key)
}

func BuildAgentToolDependencies(db *pgxpool.Pool, cfg *config.Config, mathClient *MathClient, adaptEngine *adaptive.AdaptiveEngine) *agent.ToolDependencies {
	return &agent.ToolDependencies{
		HybridSearchFn: func(ctx context.Context, query string, filters map[string]any, topK int, vectorWeight, keywordWeight float64) ([]map[string]any, error) {
			filtersTyped := make(map[string]interface{})
			for k, v := range filters {
				filtersTyped[k] = v
			}
			results, err := HybridSearch(db, query, topK, filtersTyped, vectorWeight, keywordWeight)
			if err != nil {
				return nil, err
			}
			var out []map[string]any
			for _, r := range results {
				out = append(out, map[string]any{
					"chunk_id":      r.ChunkID,
					"document_id":   r.DocID,
					"content":       r.Content,
					"filename":      r.Filename,
					"page":          r.Page,
					"section":       r.Section,
					"score":         r.HybridScore,
					"vector_score":  r.VectorScore,
					"keyword_score": r.KeywordScore,
				})
			}
			return out, nil
		},
		RerankFn: func(ctx context.Context, query string, results []map[string]any, topK int) ([]map[string]any, error) {
			var hybridResults []HybridResult
			for _, r := range results {
				page, _ := r["page"].(int)
				hybridResults = append(hybridResults, HybridResult{
					ChunkID:      toString(r["chunk_id"]),
					DocID:        toString(r["document_id"]),
					Content:      toString(r["content"]),
					Filename:     toString(r["filename"]),
					Page:         page,
					Section:      toString(r["section"]),
					HybridScore:  toFloat64(r["score"]),
				})
			}
			reranked, err := RerankResults(db, query, hybridResults, topK)
			if err != nil {
				return nil, err
			}
			var out []map[string]any
			for _, r := range reranked {
				out = append(out, map[string]any{
					"chunk_id":      r.ChunkID,
					"document_id":   r.DocID,
					"content":       r.Content,
					"filename":      r.Filename,
					"page":          r.Page,
					"section":       r.Section,
					"score":         r.RerankScore,
				})
			}
			return out, nil
		},
		MathSolveFn: func(ctx context.Context, operation, expression, variable string, lower, upper *float64) (map[string]any, error) {
			var mathResult *MathResult
			var err error

			switch operation {
			case "evaluate", "":
				mathResult, err = mathClient.Evaluate(expression)
			case "differentiate", "derivative":
				mathResult, err = mathClient.Differentiate(expression, variable)
			case "integrate":
				var lowerStr, upperStr *string
				if lower != nil {
					s := strconv.FormatFloat(*lower, 'f', -1, 64)
					lowerStr = &s
				}
				if upper != nil {
					s := strconv.FormatFloat(*upper, 'f', -1, 64)
					upperStr = &s
				}
				mathResult, err = mathClient.Integrate(expression, variable, lowerStr, upperStr)
			case "simplify":
				mathResult, err = mathClient.Simplify(expression)
			case "solve":
				solveResult, solveErr := mathClient.Solve(expression, variable)
				if solveErr != nil {
					return nil, solveErr
				}
				return map[string]any{
					"solutions": solveResult.Solutions,
					"latex":     solveResult.Latex,
					"count":     solveResult.Count,
				}, nil
			default:
				return nil, fmt.Errorf("unknown operation: %s", operation)
			}

			if err != nil {
				return nil, err
			}
			return map[string]any{
				"result":  mathResult.Result,
				"latex":   mathResult.Latex,
				"success": mathResult.Success,
			}, nil
		},
		VerifyFn: func(ctx context.Context, problem, studentAnswer, op string) (map[string]any, error) {
			result, err := mathClient.Verify(problem, studentAnswer, op)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"verified": result.Verified,
				"method":   result.Method,
				"expected": result.Expected,
				"actual":   result.Actual,
			}, nil
		},
		StudentProfileFn: func(ctx context.Context, studentID, courseID string) (map[string]any, error) {
			profile, err := GetOrCreateProfile(db, studentID, courseID)
			if err != nil {
				return nil, err
			}

			mastery, err := GetMasteryMap(db, studentID, courseID)
			if err != nil {
				return nil, err
			}

			errors, err := GetStudentErrors(db, studentID)
			if err != nil {
				return nil, err
			}

			weakTopics := make([]string, 0)
			strongTopics := make([]string, 0)
			for concept, cm := range mastery {
				if cm.Mastery < 0.40 {
					weakTopics = append(weakTopics, concept)
				} else if cm.Mastery >= 0.80 {
					strongTopics = append(strongTopics, concept)
				}
			}

			recentErrors := make([]map[string]any, 0)
			for _, e := range errors {
				recentErrors = append(recentErrors, map[string]any{
					"concept":    e.ConceptID,
					"error_type": e.ErrorType,
					"count":      e.Count,
				})
			}

			return map[string]any{
				"overall_mastery": profile.OverallLevel,
				"weak_topics":    weakTopics,
				"strong_topics":  strongTopics,
				"recent_errors":  recentErrors,
			}, nil
		},
		ExerciseGenerateFn: func(ctx context.Context, concept string, difficulty int, studentID string) (map[string]any, error) {
			exercise, err := GenerateExercise(db, cfg, concept, difficulty)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"exercise_id":     exercise.ID,
				"statement":       exercise.Statement,
				"latex":           exercise.Latex,
				"difficulty":      exercise.Difficulty,
				"concept":         concept,
				"expected_answer": exercise.ExpectedAnswer,
			}, nil
		},
		GradeFn: func(ctx context.Context, studentAnswer, expectedAnswer string) (map[string]any, error) {
			return map[string]any{
				"score":         0.0,
				"exact_match":   studentAnswer == expectedAnswer,
				"needs_review":  true,
			}, nil
		},
		AdaptiveEngine: adaptEngine,
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	default:
		return 0
	}
}


