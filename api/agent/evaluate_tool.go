package agent

import (
	"context"
	"fmt"
)

func NewEvaluateTool(evalFn func(ctx context.Context, expression, expected, conceptID, studentID string, steps []map[string]any) (map[string]any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        "math_evaluate",
		Description: "Evaluate a student's mathematical answer with step-by-step analysis, normalized comparison, and error classification.",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			expression, _ := input["expression"].(string)
			if expression == "" {
				return nil, fmt.Errorf("expression (student answer) is required")
			}
			expected, _ := input["expected"].(string)
			if expected == "" {
				return nil, fmt.Errorf("expected (correct answer) is required")
			}
			conceptID, _ := input["concept_id"].(string)
			studentID, _ := input["student_id"].(string)

			var steps []map[string]any
			if stepsRaw, ok := input["steps"].([]any); ok {
				for _, s := range stepsRaw {
					if m, ok := s.(map[string]any); ok {
						steps = append(steps, m)
					}
				}
			}

			result, err := evalFn(ctx, expression, expected, conceptID, studentID, steps)
			if err != nil {
				return nil, fmt.Errorf("evaluate: %w", err)
			}

			return result, nil
		},
	}
}
