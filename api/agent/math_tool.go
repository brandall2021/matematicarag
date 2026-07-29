package agent

import (
	"context"
	"fmt"
)

func NewMathTool(solveFn func(ctx context.Context, operation, expression, variable string, lower, upper *float64) (map[string]any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        "math_solve",
		Description: "Execute mathematical operations (evaluate, differentiate, integrate, solve, simplify)",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			operation, _ := input["operation"].(string)
			if operation == "" {
				operation = "evaluate"
			}
			expression, _ := input["expression"].(string)
			if expression == "" {
				return nil, fmt.Errorf("expression is required")
			}
			variable, _ := input["variable"].(string)
			if variable == "" {
				variable = "x"
			}

			var lower, upper *float64
			if l, ok := input["lower"].(float64); ok {
				lower = &l
			}
			if u, ok := input["upper"].(float64); ok {
				upper = &u
			}

			result, err := solveFn(ctx, operation, expression, variable, lower, upper)
			if err != nil {
				return nil, fmt.Errorf("math %s: %w", operation, err)
			}

			result["success"] = true
			return result, nil
		},
	}
}
