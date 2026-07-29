package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LLMFunc func(ctx context.Context, prompt string) (string, error)

type IntentClassifier struct {
	db          *pgxpool.Pool
	cfg         *AgentConfig
	callLLM     LLMFunc
	mu          sync.Mutex
}

func NewIntentClassifier(db *pgxpool.Pool, cfg *AgentConfig, callLLM LLMFunc) *IntentClassifier {
	return &IntentClassifier{db: db, cfg: cfg, callLLM: callLLM}
}

func (ic *IntentClassifier) Classify(ctx context.Context, query string, history []Message) (*IntentResult, error) {
	prompt := buildClassificationPrompt(query, history)
	response, err := ic.callLLM(ctx, prompt)
	if err != nil || response == "" {
		return ic.classifyByKeywords(query), nil
	}
	result, err := parseIntentJSON(response)
	if err != nil {
		return ic.classifyByKeywords(query), nil
	}
	if result.Confidence < ic.cfg.IntentThreshold {
		return ic.classifyByKeywords(query), nil
	}
	return result, nil
}

func (ic *IntentClassifier) ClassifyMulti(ctx context.Context, query string, history []Message) (*MultiIntentResult, error) {
	prompt := buildMultiIntentPrompt(query, history)
	response, err := ic.callLLM(ctx, prompt)
	if err != nil || response == "" {
		return ic.fallbackMultiIntent(query), nil
	}
	var result MultiIntentResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return ic.fallbackMultiIntent(query), nil
	}
	if len(result.Intents) == 0 {
		return ic.fallbackMultiIntent(query), nil
	}
	return &result, nil
}

func (ic *IntentClassifier) DetectFrustration(query string) bool {
	frustrationSignals := []string{
		"no entiendo nada", "ya lo intenté", "no me sale",
		"explicamelo de otra forma", "sigo sin entender",
		"no tiene sentido", "estoy perdido",
	}
	lower := strings.ToLower(query)
	for _, signal := range frustrationSignals {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

func (ic *IntentClassifier) classifyByKeywords(query string) *IntentResult {
	lower := strings.ToLower(query)

	if matchesAny(lower, []string{"qué es", "qué son", "define", "definición", "concepto", "teoría"}) {
		return &IntentResult{Intent: IntentExplainConcept, Confidence: 0.70}
	}
	if matchesAny(lower, []string{"resuelve", "resolver", "calcula", "calcular", "integra", "deriva"}) {
		return &IntentResult{Intent: IntentSolveExercise, Confidence: 0.75}
	}
	if matchesAny(lower, []string{"está bien", "es correcto", "verifica", "revisa"}) {
		return &IntentResult{Intent: IntentCheckAnswer, Confidence: 0.70}
	}
	if matchesAny(lower, []string{"pista", "ayuda", "dame una pista"}) {
		return &IntentResult{Intent: IntentGiveHint, Confidence: 0.80}
	}
	if matchesAny(lower, []string{"práctica", "practicar", "ejercicio", "quiero practicar"}) {
		return &IntentResult{Intent: IntentPractice, Confidence: 0.70}
	}
	if matchesAny(lower, []string{"compara", "diferencia", "semejanza", "versus"}) {
		return &IntentResult{Intent: IntentCompareConcepts, Confidence: 0.70}
	}
	if matchesAny(lower, []string{"ejemplo"}) {
		return &IntentResult{Intent: IntentGenerateExample, Confidence: 0.70}
	}
	if matchesAny(lower, []string{"fuente", "de dónde", "material", "bibliografía"}) {
		return &IntentResult{Intent: IntentAskSource, Confidence: 0.75}
	}
	if matchesAny(lower, []string{"progreso", "avance", "cómo voy", "mi nivel"}) {
		return &IntentResult{Intent: IntentShowProgress, Confidence: 0.75}
	}
	if matchesAny(lower, []string{"recomienda", "sugiere", "qué estudio", "qué seguir"}) {
		return &IntentResult{Intent: IntentRecommendation, Confidence: 0.70}
	}
	if matchesAny(lower, []string{"resume", "resumen", "sintetiza"}) {
		return &IntentResult{Intent: IntentSummarizeMaterial, Confidence: 0.70}
	}
	if matchesAny(lower, []string{"evaluación", "evaluame", "examen", "assessment"}) {
		return &IntentResult{Intent: IntentStartAssessment, Confidence: 0.70}
	}
	return &IntentResult{Intent: IntentAskTheory, Confidence: 0.50}
}

func (ic *IntentClassifier) fallbackMultiIntent(query string) *MultiIntentResult {
	primary := ic.classifyByKeywords(query)
	return &MultiIntentResult{
		Intents: []IntentResult{*primary},
		Plan:    []string{"process_single_intent"},
	}
}

func parseIntentJSON(response string) (*IntentResult, error) {
	cleaned := strings.TrimSpace(response)
	if idx := strings.Index(cleaned, "{"); idx >= 0 {
		cleaned = cleaned[idx:]
	}
	if idx := strings.LastIndex(cleaned, "}"); idx >= 0 {
		cleaned = cleaned[:idx+1]
	}

	var result IntentResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("parse intent json: %w", err)
	}
	if result.Intent == "" {
		return nil, fmt.Errorf("empty intent")
	}
	return &result, nil
}

func matchesAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func buildClassificationPrompt(query string, history []Message) string {
	historyStr := ""
	for _, m := range history {
		historyStr += m.Role + ": " + m.Content + "\n"
	}
	return fmt.Sprintf(`Eres un clasificador de intención para un tutor de matemáticas.
Analiza el mensaje del estudiante y determina su intención.

Intenciones posibles:
- ASK_THEORY: pregunta teórica sobre un concepto
- SOLVE_EXERCISE: pide resolver un ejercicio matemático
- EXPLAIN_CONCEPT: pide explicación de un concepto
- CHECK_ANSWER: pide verificar si su respuesta es correcta
- CHECK_PROCEDURE: pide revisar su procedimiento
- GENERATE_EXERCISE: pide generar un ejercicio
- PRACTICE: quiere practicar un tema
- GIVE_HINT: pide una pista
- REVIEW_TOPIC: quiere repasar un tema
- START_ASSESSMENT: quiere iniciar una evaluación
- CONTINUE_ASSESSMENT: continuar evaluación existente
- SHOW_PROGRESS: ver su progreso
- RECOMMENDATION: recomendación de estudio
- ASK_SOURCE: pregunta sobre la fuente/material
- SUMMARIZE_MATERIAL: resumir material
- COMPARE_CONCEPTS: comparar conceptos
- GENERATE_EXAMPLE: generar ejemplo

Historial de la conversación:
%s
Mensaje del estudiante: %s

Responde SOLO con JSON:
{
  "intent": "EXPLAIN_CONCEPT",
  "confidence": 0.96,
  "concept": "nombre.del.concepto",
  "expression": ""
}`, historyStr, query)
}

func buildMultiIntentPrompt(query string, history []Message) string {
	historyStr := ""
	for _, m := range history {
		historyStr += m.Role + ": " + m.Content + "\n"
	}
	return fmt.Sprintf(`Eres un clasificador de intención múltiple.
Analiza el mensaje y detecta TODAS las intenciones presentes.

Historial:
%s
Mensaje: %s

Responde SOLO con JSON:
{
  "intents": [
    {"intent": "EXPLAIN_CONCEPT", "confidence": 0.95, "concept": "derivatives.chain_rule"},
    {"intent": "GENERATE_EXERCISE", "confidence": 0.85, "concept": "derivatives.chain_rule"}
  ],
  "plan": ["explain_concept", "check_understanding", "generate_exercise", "wait_response", "correct"]
}`, historyStr, query)
}
