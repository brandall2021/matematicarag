package agent

import (
	"context"
)

func NewHintTool() ToolDefinition {
	return ToolDefinition{
		Name:        "generate_hint",
		Description: "Generate a progressive hint based on exercise and student's error",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			exerciseID, _ := input["exercise_id"].(string)
			errorType, _ := input["error_type"].(string)
			hintLevel := 1
			if hl, ok := input["hint_level"].(float64); ok {
				hintLevel = int(hl)
			}

			errorHints := map[string]map[int]string{
				"missing_inner_derivative": {
					1: "Observa qué función está dentro de la otra.",
					2: "Identifica f(g(x)) antes de derivar.",
					3: "Debes multiplicar por la derivada de g(x).",
				},
				"sign_error": {
					1: "Revisa los signos en cada término.",
					2: "Recuerda la regla de los signos en la multiplicación.",
					3: "Verifica el signo del resultado paso a paso.",
				},
				"missing_chain_rule": {
					1: "¿Estás aplicando la regla de la cadena?",
					2: "Recuerda: primero deriva la función exterior, luego multiplica por la derivada de la interior.",
					3: "d/dx f(g(x)) = f'(g(x)) · g'(x)",
				},
				"incorrect_power_rule": {
					1: "Revisa la regla de la potencia: d/dx(x^n) = n·x^(n-1)",
					2: "No olvides multiplicar por el exponente y restar uno al exponente.",
					3: "Ejemplo: d/dx(x^3) = 3x^2",
				},
			}

			hintText := "Revisa el procedimiento paso a paso para encontrar el error."
			if errorHintsMap, ok := errorHints[errorType]; ok {
				if h, ok := errorHintsMap[hintLevel]; ok {
					hintText = h
				} else {
					hintText = errorHintsMap[1]
				}
			}

			return map[string]any{
				"exercise_id": exerciseID,
				"hint_level":  hintLevel,
				"hint":        hintText,
			}, nil
		},
	}
}
