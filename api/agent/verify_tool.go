package agent

import (
	"context"
	"fmt"
)

func NewVerifyTool(verifyFn func(ctx context.Context, problem, studentAnswer, op string) (map[string]any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        "math_verify",
		Description: "Verify a student's mathematical answer against the expected result",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			problem, _ := input["problem"].(string)
			studentAnswer, _ := input["student_answer"].(string)
			if problem == "" || studentAnswer == "" {
				return nil, fmt.Errorf("problem and student_answer are required")
			}
			op, _ := input["operation"].(string)
			if op == "" {
				op = "derivative"
			}

			result, err := verifyFn(ctx, problem, studentAnswer, op)
			if err != nil {
				return nil, fmt.Errorf("verify: %w", err)
			}

			return map[string]any{
				"correct":    result["verified"],
				"equivalent": result["verified"],
				"expected":   result["expected"],
				"student":    result["actual"],
				"method":     result["method"],
			}, nil
		},
	}
}
