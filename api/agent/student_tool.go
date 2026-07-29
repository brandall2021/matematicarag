package agent

import (
	"context"
	"fmt"
)

func NewStudentTool(profileFn func(ctx context.Context, studentID, courseID string) (map[string]any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        "student_profile",
		Description: "Get student's academic profile, mastery map, and error patterns",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			studentID, _ := input["student_id"].(string)
			courseID, _ := input["course_id"].(string)
			if studentID == "" {
				return nil, fmt.Errorf("student_id is required")
			}
			if courseID == "" {
				courseID = "matematica-1"
			}

			result, err := profileFn(ctx, studentID, courseID)
			if err != nil {
				return nil, fmt.Errorf("student profile: %w", err)
			}

			return result, nil
		},
	}
}
