package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TutorRequest struct {
	Query            string   `json:"query"`
	CourseID         string   `json:"course_id,omitempty"`
	UnitID           string   `json:"unit_id,omitempty"`
	ExplanationLevel string   `json:"explanation_level,omitempty"`
	Mode             string   `json:"mode,omitempty"`
	UserResult       string   `json:"user_result,omitempty"`
	UserProcedure    []string `json:"user_procedure,omitempty"`
}

type TutorStep struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Latex       string `json:"latex,omitempty"`
	IsMath      bool   `json:"is_math"`
}

type VerifyInfo struct {
	Status string `json:"status"`
	Method string `json:"method,omitempty"`
}

type TutorResponse struct {
	Problem struct {
		Type       string `json:"type"`
		Expression string `json:"expression"`
		Variable   string `json:"variable,omitempty"`
	} `json:"problem"`
	Method struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"method"`
	Steps        []TutorStep   `json:"steps"`
	Result       *MathResult   `json:"result,omitempty"`
	Verification *VerifyInfo   `json:"verification,omitempty"`
	Citations    []RagCitation `json:"citations"`
	Sources      []RagSource   `json:"sources"`
	MathComputed bool          `json:"math_computed"`
	Confidence   string        `json:"confidence"`
}

type explanationJSON struct {
	MethodName        string         `json:"method_name"`
	MethodDescription string         `json:"method_description"`
	Steps             []TutorStep    `json:"steps"`
}

func TutorRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Post("/solve", func(w http.ResponseWriter, r *http.Request) {
			var req TutorRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("[TUTOR] invalid request body: %v", err)
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			if req.Query == "" {
				http.Error(w, `{"error":"query is required"}`, http.StatusBadRequest)
				return
			}
			if len(req.Query) > 10000 {
				http.Error(w, `{"error":"query too long (max 10000 characters)"}`, http.StatusBadRequest)
				return
			}

			log.Printf("[TUTOR] processing query: %s", truncate(req.Query, 100))

			intent := ClassifyIntent(db, req.Query)
			log.Printf("[TUTOR] intent: %s, needs_math: %v, op: %s", intent.Intent, intent.NeedsMath, intent.MathOperation)

			ragSources, ragContext := performRAGSearch(db, req.Query)

			response := TutorResponse{
				Confidence: "low",
				Citations:  make([]RagCitation, 0),
				Sources:    ragSources,
			}

			expression := intent.Expression
			variable := intent.Variable
			if variable == "" {
				variable = "x"
			}

			response.Problem.Type = string(intent.Intent)
			response.Problem.Expression = expression
			response.Problem.Variable = variable

			mathClient := NewMathClient(cfg)

			if intent.NeedsMath && expression != "" {
				response.MathComputed = true

				switch intent.MathOperation {
				case "derivada", "differentiate":
					result, err := mathClient.Differentiate(expression, variable)
					if err != nil {
						log.Printf("[TUTOR] differentiate error: %v", err)
						response.MathComputed = false
					} else {
						response.Result = result
						verify, vErr := mathClient.Integrate(result.Result, variable, nil, nil)
						if vErr != nil {
							log.Printf("[TUTOR] verify (integrate) error: %v", vErr)
							response.Verification = &VerifyInfo{Status: "verification_not_possible"}
						} else {
							verified := verify.Success && strings.Contains(verify.Result, expression)
							response.Verification = &VerifyInfo{
								Status: verifiedStatus(verified),
								Method: "integración inversa",
							}
						}
					}

				case "integral", "integrate":
					result, err := mathClient.Integrate(expression, variable, nil, nil)
					if err != nil {
						log.Printf("[TUTOR] integrate error: %v", err)
						response.MathComputed = false
					} else {
						response.Result = result
						verify, vErr := mathClient.Differentiate(result.Result, variable)
						if vErr != nil {
							log.Printf("[TUTOR] verify (differentiate) error: %v", vErr)
							response.Verification = &VerifyInfo{Status: "verification_not_possible"}
						} else {
							verified := verify.Success && strings.Contains(verify.Result, expression)
							response.Verification = &VerifyInfo{
								Status: verifiedStatus(verified),
								Method: "derivada inversa",
							}
						}
					}

				case "ecuacion", "solve", "ecuaciones":
					solveResult, err := mathClient.Solve(expression, variable)
					if err != nil {
						log.Printf("[TUTOR] solve error: %v", err)
						response.MathComputed = false
					} else {
						solutionsJSON, _ := json.Marshal(solveResult.Solutions)
						response.Result = &MathResult{
							Success: solveResult.Success,
							Result:  string(solutionsJSON),
							Latex:   solveResult.Latex,
							Error:   solveResult.Error,
						}
						response.Verification = &VerifyInfo{
							Status: "verified",
							Method: "resolución directa",
						}
					}

				case "limite", "limit":
					result, err := mathClient.Limit(expression, variable, "0")
					if err != nil {
						log.Printf("[TUTOR] limit error: %v", err)
						response.MathComputed = false
					} else {
						response.Result = result
						response.Verification = &VerifyInfo{
							Status: "verified",
							Method: "cálculo directo",
						}
					}

				case "simplificar", "simplify":
					result, err := mathClient.Simplify(expression)
					if err != nil {
						log.Printf("[TUTOR] simplify error: %v", err)
						response.MathComputed = false
					} else {
						response.Result = result
						response.Verification = &VerifyInfo{
							Status: "verified",
							Method: "simplificación directa",
						}
					}

				default:
					result, err := mathClient.Evaluate(expression)
					if err != nil {
						log.Printf("[TUTOR] evaluate error: %v", err)
						response.MathComputed = false
					} else {
						response.Result = result
						response.Verification = &VerifyInfo{
							Status: "verified",
							Method: "evaluación directa",
						}
					}
				}
			}

			if req.Mode == "verify" && req.UserResult != "" && expression != "" {
				verifyResult, err := mathClient.Verify(expression, req.UserResult, string(intent.Intent))
				if err != nil {
					log.Printf("[TUTOR] verify error: %v", err)
				} else {
					response.Verification = &VerifyInfo{
						Status: verifiedStatus(verifyResult.Verified),
						Method: verifyResult.Method,
					}
					if response.Result == nil {
						response.Result = &MathResult{
							Success: verifyResult.Success,
							Result:  verifyResult.Expected,
							Latex:   verifyResult.LatexExpected,
						}
					}
					response.MathComputed = true
				}
			}

			explanation, err := generateExplanation(db, cfg, req, intent, ragContext, response.Result, response.Verification)
			if err != nil {
				log.Printf("[TUTOR] explanation generation failed: %v", err)
			} else {
				response.Method.Name = explanation.MethodName
				response.Method.Description = explanation.MethodDescription
				response.Steps = explanation.Steps
			}

			if response.Steps == nil {
				response.Steps = []TutorStep{}
			}

			if response.Verification == nil {
				response.Verification = &VerifyInfo{Status: "not_verified"}
			}

			if len(response.Citations) == 0 {
				response.Citations = make([]RagCitation, 0)
			}
			if len(response.Sources) == 0 {
				response.Sources = make([]RagSource, 0)
			}

			if response.Result != nil && response.Result.Success {
				response.Confidence = "high"
			} else if ragSources != nil {
				response.Confidence = "medium"
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})
	}
}

func generateExplanation(db *pgxpool.Pool, cfg *config.Config, req TutorRequest, intent IntentResult, ragContext string, mathResult *MathResult, verification *VerifyInfo) (*explanationJSON, error) {
	systemPrompt := `Eres un tutor experto de matemáticas. Tu tarea es generar una explicación paso a paso para resolver un problema matemático.
Debes responder ÚNICAMENTE con un JSON válido (sin markdown, sin backticks) con esta estructura:
{
  "method_name": "nombre del método o técnica utilizada",
  "method_description": "descripción breve del método",
  "steps": [
    {
      "number": 1,
      "title": "título del paso",
      "explanation": "explicación detallada del paso en texto plano",
      "latex": "expresión LaTeX del paso (si aplica)",
      "is_math": true
    }
  ]
}

Reglas:
- Responde SIEMPRE en español.
- Sé claro y didáctico.
- Cada paso debe ser comprensible por sí mismo.
- Si el nivel es "básico", usa lenguaje simple y explicaciones largas.
- Si el nivel es "intermedio", asume conocimiento base.
- Si el nivel es "avanzado", usa terminología técnica.`

	var userParts []string
	userParts = append(userParts, fmt.Sprintf("Pregunta del usuario: %s", req.Query))
	userParts = append(userParts, FormatIntentForPrompt(intent))

	if req.ExplanationLevel != "" {
		userParts = append(userParts, fmt.Sprintf("Nivel de explicación: %s", req.ExplanationLevel))
	} else {
		userParts = append(userParts, "Nivel de explicación: intermedio")
	}

	if mathResult != nil && mathResult.Success {
		userParts = append(userParts, fmt.Sprintf("Resultado del cálculo: %s", mathResult.Result))
		if mathResult.Latex != "" {
			userParts = append(userParts, fmt.Sprintf("Resultado en LaTeX: %s", mathResult.Latex))
		}
	}

	if verification != nil {
		userParts = append(userParts, fmt.Sprintf("Verificación: %s", verification.Status))
		if verification.Method != "" {
			userParts = append(userParts, fmt.Sprintf("Método de verificación: %s", verification.Method))
		}
	}

	if ragContext != "" {
		if len(ragContext) > 2000 {
			ragContext = ragContext[:2000]
		}
		userParts = append(userParts, fmt.Sprintf("\nContexto de referencia:\n%s", ragContext))
	}

	messages := []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: strings.Join(userParts, "\n")},
	}

	response, err := callOpenAIWithHistory(db, messages, "")
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var stripped []string
		inBlock := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inBlock = !inBlock
				continue
			}
			if !inBlock {
				stripped = append(stripped, line)
			}
		}
		response = strings.Join(stripped, "\n")
	}

	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON found in LLM response")
	}
	response = response[start : end+1]

	var explanation explanationJSON
	if err := json.Unmarshal([]byte(response), &explanation); err != nil {
		return nil, fmt.Errorf("failed to parse explanation JSON: %w", err)
	}

	return &explanation, nil
}

func verifiedStatus(verified bool) string {
	if verified {
		return "verified"
	}
	return "not_verified"
}
