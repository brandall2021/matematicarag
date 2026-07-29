package agent

import (
	"context"
	"fmt"
)

func NewGradingTool(gradeFn func(ctx context.Context, studentAnswer, expectedAnswer string) (map[string]any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        "grade_answer",
		Description: "Grade a student's answer against expected answer",
		Permission:  "write",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			studentAnswer, _ := input["student_answer"].(string)
			expectedAnswer, _ := input["expected_answer"].(string)
			if studentAnswer == "" || expectedAnswer == "" {
				return nil, fmt.Errorf("student_answer and expected_answer are required")
			}

			result, err := gradeFn(ctx, studentAnswer, expectedAnswer)
			if err != nil {
				return nil, fmt.Errorf("grade: %w", err)
			}

			return result, nil
		},
	}
}
