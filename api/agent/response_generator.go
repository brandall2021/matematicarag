package agent

import (
	"context"
	"fmt"
	"strings"
)

type ResponseGenerator struct {
	callLLM LLMFunc
	cfg     *AgentConfig
}

func NewResponseGenerator(callLLM LLMFunc, cfg *AgentConfig) *ResponseGenerator {
	return &ResponseGenerator{callLLM: callLLM, cfg: cfg}
}

func (rg *ResponseGenerator) Generate(ctx context.Context, plan *Plan, toolResults []*ToolCall, query string, citationMgr *CitationManager) (*AgentResponse, error) {
	var toolSummary strings.Builder
	toolSummary.WriteString("Resultados de herramientas:\n")
	for _, tc := range toolResults {
		toolSummary.WriteString(fmt.Sprintf("\n--- %s (%s) ---\n", tc.Tool, tc.Purpose))
		if tc.Error != "" {
			toolSummary.WriteString(fmt.Sprintf("Error: %s\n", tc.Error))
		} else if tc.Result != nil {
			for k, v := range tc.Result {
				toolSummary.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
			}
		}
	}

	response, err := rg.callLLM(ctx, buildResponsePrompt(query, toolSummary.String(), plan))
	if err != nil || response == "" {
		return rg.fallbackResponse(query, toolResults, citationMgr, plan), nil
	}

	resp := &AgentResponse{
		Response: response,
		Intent:   plan.Intent,
		Strategy: plan.Strategy,
		Actions:  []string{string(plan.Intent)},
		Sections: map[string]string{
			"explanation": response,
		},
	}

	if citationMgr.HasCitations() {
		resp.Citations = citationMgr.GetCitationsJSON()
		resp.Sections["academic_source"] = citationMgr.FormatCitations()
	}

	return resp, nil
}

func buildResponsePrompt(query, toolSummary string, plan *Plan) string {
	intentName := string(plan.Intent)
	intentDescriptions := map[IntentType]string{
		IntentAskTheory:      "pregunta teórica",
		IntentExplainConcept: "explicar concepto",
		IntentSolveExercise:  "resolver ejercicio matemático",
		IntentCheckAnswer:    "verificar respuesta",
		IntentCheckProcedure: "revisar procedimiento",
		IntentPractice:       "practicar",
		IntentGiveHint:       "dar pista",
		IntentReviewTopic:    "repasar tema",
	}
	if name, ok := intentDescriptions[plan.Intent]; ok {
		intentName = name
	}

	return fmt.Sprintf(`Eres un tutor de matemáticas universitario. Genera una respuesta pedagógica estructurada.

Contexto:
- Intención: %s
- Estrategia pedagógica: %s
- Consulta del estudiante: %s

%s

Instrucciones:
1. Responde en español, claro y pedagógico.
2. Si hay fuentes académicas disponibles, menciónalas.
3. Si hay verificación matemática, incluye el resultado.
4. Separa claramente: explicación, verificación, recomendación.
5. NO menciones el funcionamiento interno del sistema.
6. NO digas "según los resultados de las herramientas".
7. Si el estudiante cometió un error, explica por qué está mal y cómo corregirlo.
8. Termina con una pregunta o sugerencia para continuar el aprendizaje.`,
		intentName, string(plan.Strategy), query, toolSummary)
}

func (rg *ResponseGenerator) fallbackResponse(query string, toolResults []*ToolCall, citationMgr *CitationManager, plan *Plan) *AgentResponse {
	var sb strings.Builder
	sb.WriteString("Claro, vamos a revisarlo.\n\n")

	for _, tc := range toolResults {
		switch tc.Tool {
		case "rag_search":
			if tc.Error == "" && tc.Result != nil {
				if results, ok := tc.Result["results"].([]any); ok && len(results) > 0 {
					sb.WriteString("He encontrado material relevante sobre el tema.\n\n")
				}
			}
		case "math_solve":
			if tc.Error == "" && tc.Result != nil {
				if latex, ok := tc.Result["latex"].(string); ok && latex != "" {
					sb.WriteString(fmt.Sprintf("Resultado: %s\n\n", latex))
				} else if result, ok := tc.Result["result"].(string); ok {
					sb.WriteString(fmt.Sprintf("Resultado: %s\n\n", result))
				}
			}
		case "math_verify":
			if tc.Error == "" && tc.Result != nil {
				if correct, ok := tc.Result["correct"].(bool); ok {
					if correct {
						sb.WriteString("✓ Tu respuesta es correcta.\n\n")
					} else {
						sb.WriteString("✗ Tu respuesta necesita revisión.\n")
						if expected, ok := tc.Result["expected"].(string); ok {
							sb.WriteString(fmt.Sprintf("Resultado esperado: %s\n", expected))
						}
						sb.WriteString("\n")
					}
				}
			}
		case "student_profile":
			if tc.Error == "" && tc.Result != nil {
				if weak, ok := tc.Result["weak_topics"].([]any); ok && len(weak) > 0 {
					sb.WriteString("Veo que tienes dificultad en: ")
					for i, w := range weak {
						if i > 0 {
							sb.WriteString(", ")
						}
						sb.WriteString(fmt.Sprintf("%v", w))
					}
					sb.WriteString(".\n\n")
				}
			}
		}
	}

	if plan.Strategy == StrategySocratic || plan.Strategy == StrategyGuided {
		sb.WriteString("\n**Pregunta orientadora:** ¿Qué paso crees que deberíamos revisar primero?\n")
	}

	if citationMgr.HasCitations() {
		sb.WriteString(citationMgr.FormatCitations())
	}

	return &AgentResponse{
		Response:  sb.String(),
		Intent:    plan.Intent,
		Strategy:  plan.Strategy,
		Citations: citationMgr.GetCitationsJSON(),
	}
}
