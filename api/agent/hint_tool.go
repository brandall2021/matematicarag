package agent

import (
	"context"
)

func NewHintTool() ToolDefinition {
	return ToolDefinition{
		Name:        "generate_hint",
		Description: "Generate a progressive hint based on exercise and student's error. Level 1: conceptual guidance. Level 2: step/rule. Level 3: partial transformation. Level 4: full solution.",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			exerciseID, _ := input["exercise_id"].(string)
			errorType, _ := input["error_type"].(string)
			hintLevel := 1
			if hl, ok := input["hint_level"].(float64); ok {
				hintLevel = int(hl)
			}
			if hintLevel < 1 {
				hintLevel = 1
			}
			if hintLevel > 4 {
				hintLevel = 4
			}

			errorHints := map[string]map[int]string{
				"missing_inner_derivative": {
					1: "Observa la estructura de la función. ¿Reconoces una función compuesta (función dentro de otra)?",
					2: "Identifica claramente f(g(x)). Primero deriva f respecto a g, luego multiplica por la derivada de g respecto a x.",
					3: "Para d/dx de (x²+1)³: u = x²+1, d/du(u³) = 3u², du/dx = 2x → resultado = 3(x²+1)² · 2x",
					4: "La derivada completa es: 3(x²+1)² · 2x = 6x(x²+1)². ¿Coincide con tu resultado?",
				},
				"sign_error": {
					1: "Revisa cada término. Un error de signo cambia completamente el resultado.",
					2: "Recuerda: menos por menos = más. menos por más = menos. Revisa cada multiplicación.",
					3: "Paso a paso: ¿qué signo tiene cada término? Escríbelos uno por uno y verifica.",
					4: "Ejemplo: si tienes -3(x-2) = -3x+6. El signo de cada término importa.",
				},
				"missing_chain_rule": {
					1: "¿Estás aplicando la regla de la cadena? Esta función requiere derivar paso a paso.",
					2: "Regla de la cadena: d/dx f(g(x)) = f'(g(x)) · g'(x). Identifica f y g primero.",
					3: "Paso 1: deriva la función exterior y manten la interior. Paso 2: multiplica por la derivada de la interior.",
					4: "Ejemplo completo: d/dx (3x²+1)⁴ = 4(3x²+1)³ · 6x = 24x(3x²+1)³",
				},
				"incorrect_power_rule": {
					1: "Revisa la regla de la potencia: d/dx(x^n) = n·x^(n-1). ¿Estás aplicando esta fórmula?",
					2: "Multiplica por el exponente original y luego resta uno al exponente.",
					3: "Ejemplo: d/dx(x⁵) = 5·x⁴. El exponente baja multiplicando y se reduce en uno.",
					4: "General: d/dx(x^n) = n·x^(n-1). Para n=3: d/dx(x³) = 3x²",
				},
				"incorrect_answer": {
					1: "Revisa el enunciado del problema. Asegúrate de entender qué operación se pide.",
					2: "Descompón el problema en pasos pequeños y resuelve cada uno por separado.",
					3: "Verifica cada paso intermedio. ¿Hay algún error de álgebra o aritmética?",
					4: "Solución completa: revisa el procedimiento paso a paso para identificar dónde está el error.",
				},
			}

			defaultHints := map[int]string{
				1: "Revisa el concepto fundamental antes de continuar.",
				2: "Identifica la regla o fórmula relevante para este problema.",
				3: "Aplica la regla paso a paso, verificando cada transformación.",
				4: "Solución: revisa el resultado completo y compáralo con tu respuesta.",
			}

			hintText := defaultHints[hintLevel]
			if errorHintsMap, ok := errorHints[errorType]; ok {
				if h, ok := errorHintsMap[hintLevel]; ok {
					hintText = h
				} else {
					hintText = errorHintsMap[1]
				}
			}

			moreAvailable := hintLevel < 4

			return map[string]any{
				"exercise_id":    exerciseID,
				"hint_level":     hintLevel,
				"hint":           hintText,
				"more_available": moreAvailable,
				"is_solution":    hintLevel == 4,
			}, nil
		},
	}
}
