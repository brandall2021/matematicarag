package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type IntentType string

const (
	IntentConceptual       IntentType = "conceptual"
	IntentDefinition       IntentType = "definition"
	IntentFormula          IntentType = "formula"
	IntentExample          IntentType = "example"
	IntentExercise         IntentType = "exercise"
	IntentSolve            IntentType = "solve"
	IntentVerify           IntentType = "verify"
	IntentExplain          IntentType = "explain"
	IntentCompare          IntentType = "compare"
	IntentGenerateExercise IntentType = "generate_exercise"
	IntentSimplify         IntentType = "simplify"
	IntentDifferentiate    IntentType = "differentiate"
	IntentIntegrate        IntentType = "integrate"
	IntentLimit            IntentType = "limit"
	IntentMatrix           IntentType = "matrix"
)

type IntentResult struct {
	Intent        IntentType `json:"intent"`
	Confidence    float64    `json:"confidence"`
	MathOperation string     `json:"math_operation,omitempty"`
	Expression    string     `json:"expression,omitempty"`
	Variable      string     `json:"variable,omitempty"`
	NeedsMath     bool       `json:"needs_math"`
}

var classifySystemPrompt = "Sos un clasificador de intenciones para un tutor de matematicas. Dada una consulta del usuario, clasificala en UNA de las siguientes categorias y extrae los campos relevantes.\n" +
	"\n" +
	"CATEGORIAS:\n" +
	"- conceptual: Preguntas conceptuales sobre matematicas (\"que es un numero primo?\", \"que es el calculo?\")\n" +
	"- definition: Pide una definicion formal (\"define integral\", \"definicion de limite\")\n" +
	"- formula: Pide una formula (\"formula de Bhaskara\", \"como se calcula el area?\")\n" +
	"- example: Pide un ejemplo (\"dame un ejemplo de derivada\", \"mostra un ejemplo\")\n" +
	"- exercise: Quiere resolver un ejercicio planteado (\"resuelve x^2+2x+1=0\")\n" +
	"- solve: Quiere resolver una ecuacion o problema (\"resolver 3x+5=20\", \"cuanto es 2+2\")\n" +
	"- verify: Quiere verificar si una respuesta es correcta (\"es correcto que la integral de x^2 es x^3/3?\")\n" +
	"- explain: Quiere una explicacion paso a paso (\"explica como se resuelve ecuaciones cuadraticas\")\n" +
	"- compare: Compara dos conceptos (\"diferencia entre derivada e integral\")\n" +
	"- generate_exercise: Quiere que se le genere un ejercicio (\"generame un ejercicio de algebra\", \"dame un problema de practica\")\n" +
	"- simplify: Simplificar una expresion (\"simplifica (x+1)(x-1)\", \"reduce esta fraccion\")\n" +
	"- differentiate: Calcular una derivada (\"calcula la derivada de x^2\", \"derivar sen(x)\")\n" +
	"- integrate: Calcular una integral (\"integral de x^2 dx\", \"integrar 1/x\")\n" +
	"- limit: Calcular un limite (\"limite de sin(x)/x cuando x tiende a 0\")\n" +
	"- matrix: Operaciones con matrices (\"multiplicar matrices\", \"determinante de una matriz\")\n" +
	"\n" +
	"Responde UNICAMENTE con un JSON (sin markdown, sin bloques de codigo):\n" +
	"{\n" +
	"  \"intent\": \"categoria\",\n" +
	"  \"confidence\": 0.0-1.0,\n" +
	"  \"math_operation\": \"operacion especifica si aplica\",\n" +
	"  \"expression\": \"expresion matematica si la hay\",\n" +
	"  \"variable\": \"variable principal si la hay\",\n" +
	"  \"needs_math\": true/false\n" +
	"}\n" +
	"\n" +
	"EJEMPLOS:\n" +
	"Entrada: \"que es una integral definida?\"\n" +
	"Salida: {\"intent\":\"definition\",\"confidence\":0.95,\"math_operation\":\"integral\",\"expression\":\"\",\"variable\":\"\",\"needs_math\":false}\n" +
	"\n" +
	"Entrada: \"resolver 2x + 5 = 15\"\n" +
	"Salida: {\"intent\":\"solve\",\"confidence\":0.98,\"math_operation\":\"ecuacion_lineal\",\"expression\":\"2x+5=15\",\"variable\":\"x\",\"needs_math\":true}\n" +
	"\n" +
	"Entrada: \"calcula la derivada de sen(x)\"\n" +
	"Salida: {\"intent\":\"differentiate\",\"confidence\":0.99,\"math_operation\":\"derivada\",\"expression\":\"sen(x)\",\"variable\":\"x\",\"needs_math\":true}\n" +
	"\n" +
	"Entrada: \"integral de x^2 dx\"\n" +
	"Salida: {\"intent\":\"integrate\",\"confidence\":0.99,\"math_operation\":\"integral\",\"expression\":\"x^2\",\"variable\":\"x\",\"needs_math\":true}\n" +
	"\n" +
	"Entrada: \"limite de (sin x)/x cuando x tiende a 0\"\n" +
	"Salida: {\"intent\":\"limit\",\"confidence\":0.97,\"math_operation\":\"limite\",\"expression\":\"sin(x)/x\",\"variable\":\"x\",\"needs_math\":true}\n" +
	"\n" +
	"Entrada: \"dame un ejercicio de ecuaciones cuadraticas\"\n" +
	"Salida: {\"intent\":\"generate_exercise\",\"confidence\":0.92,\"math_operation\":\"ecuaciones_cuadraticas\",\"expression\":\"\",\"variable\":\"\",\"needs_math\":false}\n" +
	"\n" +
	"Entrada: \"simplifica (x+2)^2\"\n" +
	"Salida: {\"intent\":\"simplify\",\"confidence\":0.95,\"math_operation\":\"simplificar\",\"expression\":\"(x+2)^2\",\"variable\":\"x\",\"needs_math\":true}\n" +
	"\n" +
	"Entrada: \"multiplicar matrices A y B\"\n" +
	"Salida: {\"intent\":\"matrix\",\"confidence\":0.93,\"math_operation\":\"multiplicacion_matrices\",\"expression\":\"\",\"variable\":\"\",\"needs_math\":true}"

func ClassifyIntent(db *pgxpool.Pool, query string) IntentResult {
	messages := []OpenAIMessage{
		{Role: "system", Content: classifySystemPrompt},
		{Role: "user", Content: query},
	}

	response, err := callOpenAIWithHistory(db, messages, "")
	if err != nil {
		log.Printf("[INTENT] LLM classification failed, using keyword fallback: %v", err)
		return classifyByKeywords(query)
	}

	var result IntentResult
	parsed := parseIntentJSON(response)
	if parsed == nil {
		log.Printf("[INTENT] Failed to parse LLM response, using keyword fallback")
		return classifyByKeywords(query)
	}
	result = *parsed

	if result.Expression == "" {
		result.Expression = extractExpression(query)
	}

	return result
}

func parseIntentJSON(raw string) *IntentResult {
	raw = strings.TrimSpace(raw)

	// Strip markdown code blocks if present
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		var stripped []string
		inBlock := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				if inBlock {
					break
				}
				inBlock = true
				continue
			}
			if inBlock {
				stripped = append(stripped, line)
			}
		}
		if len(stripped) > 0 {
			raw = strings.Join(stripped, "\n")
		}
	}

	// Find JSON object boundaries
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	raw = raw[start : end+1]

	var result IntentResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}

	return &result
}

func classifyByKeywords(query string) IntentResult {
	lower := strings.ToLower(query)

	// Order matters: check specific operations before general ones
	keywordMap := []struct {
		keywords []string
		intent   IntentType
		op       string
		needs    bool
	}{
		// Calculus operations
		{[]string{"derivada", "derivar", "derivo", "diferenciar"}, IntentDifferentiate, "derivada", true},
		{[]string{"integral", "integrar", "integro", "integrar"}, IntentIntegrate, "integral", true},
		{[]string{"limite", "límite", "limitem", "tiende a"}, IntentLimit, "limite", true},
		{[]string{"matriz", "matrices", "determinante", "determinar matriz"}, IntentMatrix, "matrices", true},
		// Simplify
		{[]string{"simplifica", "simplificar", "simplif", "reducir expresion"}, IntentSimplify, "simplificar", true},
		// Solve
		{[]string{"resolver", "resuelve", "resolv", "calcular", "cuanto es", "cuánto es", "cuanto vale"}, IntentSolve, "resolver", true},
		// Verify
		{[]string{"es correcto", "verificar", "verifica", "está bien", "esta bien", "es verdadero"}, IntentVerify, "verificar", true},
		// Generate exercise
		{[]string{"genera ejercicio", "generame", "crea ejercicio", "dame un ejercicio", "ejercicio de practica", "problema de practica"}, IntentGenerateExercise, "ejercicio", false},
		// Compare
		{[]string{"diferencia entre", "comparar", "compara", "diferencia de", "vs", "versus"}, IntentCompare, "comparacion", false},
		// Explain
		{[]string{"explica", "explicar", "como se", "paso a paso", "como funciona"}, IntentExplain, "explicacion", false},
		// Example
		{[]string{"ejemplo", "ejemplos", "mostra un ejemplo", "dame un ejemplo"}, IntentExample, "ejemplo", false},
		// Formula
		{[]string{"formula", "fórmula", "como se calcula", "ecuacion de", "cuál es la fórmula"}, IntentFormula, "formula", false},
		// Definition
		{[]string{"definicion", "definición", "define ", "que es ", "qué es ", "concepto de"}, IntentDefinition, "definicion", false},
		// Conceptual
		{[]string{"por que", "por qué", "para que sirve", "para qué sirve", "cuando se usa", "cuándo se usa"}, IntentConceptual, "conceptual", false},
	}

	for _, entry := range keywordMap {
		for _, kw := range entry.keywords {
			if strings.Contains(lower, kw) {
				expr := extractExpression(query)
				return IntentResult{
					Intent:        entry.intent,
					Confidence:    0.7,
					MathOperation: entry.op,
					Expression:    expr,
					Variable:      extractVariable(expr),
					NeedsMath:     entry.needs,
				}
			}
		}
	}

	return IntentResult{
		Intent:     IntentConceptual,
		Confidence: 0.5,
		NeedsMath:  false,
	}
}

func extractExpression(query string) string {
	lower := strings.ToLower(query)

	// Strip common Spanish prefixes
	prefixes := []string{
		"resolver ", "resuelve ", "calcular ", "calculo ",
		"calcula ", "computar ", "evaluar ", "evalua ",
		"cuanto es ", "cuánto es ", "cuanto vale ", "cuánto vale ",
		"simplifica ", "simplificar ",
		"derivar ", "deriva ", "derivar la funcion ", "calcula la derivada de ",
		"integrar ", "integra ", "calcula la integral de ",
		"encontrar el limite de ", "encuentra el limite de ",
		"dame un ejemplo de ", "mostra un ejemplo de ",
		"genera un ejercicio de ", "generame un ejercicio de ",
	}

	result := lower
	for _, p := range prefixes {
		if strings.HasPrefix(result, p) {
			result = strings.TrimPrefix(result, p)
			break
		}
	}

	// Try to extract variable declarations like "donde x = 5"
	if idx := strings.Index(result, " donde "); idx != -1 {
		result = result[:idx]
	}

	// Also look for "para x" or "en x"
	for _, sep := range []string{" para ", " en x", " con "} {
		if idx := strings.Index(result, sep); idx != -1 {
			result = result[:idx]
		}
	}

	result = strings.TrimSpace(result)
	if result == lower || len(result) < 2 {
		return ""
	}
	return result
}

func extractVariable(expr string) string {
	if expr == "" {
		return ""
	}

	variables := []string{"x", "y", "z", "t", "n", "θ", "alpha", "beta"}
	for _, v := range variables {
		if strings.Contains(expr, v) {
			return v
		}
	}

	// Check for common function variables
	commonVars := map[string]string{
		"sen": "x", "cos": "x", "tan": "x",
		"ln": "x", "log": "x", "exp": "x",
	}
	for fn := range commonVars {
		if strings.Contains(expr, fn) {
			return commonVars[fn]
		}
	}

	return ""
}

func FormatIntentForPrompt(intent IntentResult) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Intent: %s (confidence: %.0f%%)", intent.Intent, intent.Confidence*100))
	if intent.MathOperation != "" {
		parts = append(parts, fmt.Sprintf("Operation: %s", intent.MathOperation))
	}
	if intent.Expression != "" {
		parts = append(parts, fmt.Sprintf("Expression: %s", intent.Expression))
	}
	if intent.Variable != "" {
		parts = append(parts, fmt.Sprintf("Variable: %s", intent.Variable))
	}
	if intent.NeedsMath {
		parts = append(parts, "Requires math computation: yes")
	}
	return strings.Join(parts, " | ")
}
