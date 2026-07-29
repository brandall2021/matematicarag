package agent

import (
	"context"
	"fmt"
)

func NewAssessmentTool() ToolDefinition {
	return ToolDefinition{
		Name:        "assessment",
		Description: "Start, check status, or submit an assessment",
		Permission:  "write",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			action, _ := input["action"].(string)

			switch action {
			case "status":
				return map[string]any{
					"has_active_assessment": false,
					"status":                "none",
				}, nil
			case "start":
				return map[string]any{
					"started": false,
					"message": "Use /api/assessments endpoints for assessment operations",
				}, nil
			default:
				return nil, fmt.Errorf("unknown assessment action: %s", action)
			}
		},
	}
}
