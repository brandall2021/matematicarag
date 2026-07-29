package agent

import (
	"context"
	"fmt"
)

func NewExerciseTool(generateFn func(ctx context.Context, concept string, difficulty int, studentID string) (map[string]any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        "exercise_generate",
		Description: "Generate a practice exercise for a given concept and difficulty",
		Permission:  "write",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			concept, _ := input["concept"].(string)
			if concept == "" {
				return nil, fmt.Errorf("concept is required")
			}
			difficulty := 2
			if d, ok := input["difficulty"].(float64); ok {
				difficulty = int(d)
			}
			studentID, _ := input["student_id"].(string)

			result, err := generateFn(ctx, concept, difficulty, studentID)
			if err != nil {
				return nil, fmt.Errorf("generate exercise: %w", err)
			}

			return result, nil
		},
	}
}
