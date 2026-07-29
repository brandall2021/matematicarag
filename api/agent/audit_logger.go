package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentTrace struct {
	ExecutionID string         `json:"execution_id"`
	StudentID   string         `json:"student_id"`
	SessionID   string         `json:"session_id"`
	Intent      IntentType     `json:"intent"`
	Plan        *Plan          `json:"plan"`
	ToolsUsed   []string       `json:"tools_used"`
	ToolResults []*ToolCall    `json:"tool_results"`
	Response    string         `json:"response"`
	Duration    time.Duration  `json:"duration_ms"`
	Tokens      int            `json:"tokens"`
	Model       string         `json:"model"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type AuditLogger struct {
	db *pgxpool.Pool
}

func NewAuditLogger(db *pgxpool.Pool) *AuditLogger {
	return &AuditLogger{db: db}
}

func (al *AuditLogger) LogExecution(ctx context.Context, trace *AgentTrace) error {
	planJSON, _ := json.Marshal(trace.Plan)
	toolResultsJSON, _ := json.Marshal(trace.ToolResults)

	toolsUsed := make([]string, 0, len(trace.ToolResults))
	for _, tc := range trace.ToolResults {
		toolsUsed = append(toolsUsed, tc.Tool)
	}

	durationMs := int(trace.Duration.Milliseconds())
	status := "completed"
	if trace.Error != "" {
		status = "error"
	}

	_, err := al.db.Exec(ctx,
		`INSERT INTO agent_execution_log
		 (student_id, session_id, intent, plan, tools_used, tool_results, final_response, duration_ms, total_tokens, model, status, error, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		trace.StudentID, trace.SessionID, string(trace.Intent), planJSON,
		toolsUsed, toolResultsJSON, trace.Response, durationMs,
		trace.Tokens, trace.Model, status, trace.Error, time.Now())

	return err
}

func (al *AuditLogger) FormatAgentTrace(trace *AgentTrace) string {
	result := fmt.Sprintf("Intent: %s\n", trace.Intent)
	result += fmt.Sprintf("Duration: %dms\n", trace.Duration.Milliseconds())

	if trace.Plan != nil {
		result += fmt.Sprintf("Strategy: %s\n", trace.Plan.Strategy)
		result += "Plan:\n"
		for i, step := range trace.Plan.Steps {
			result += fmt.Sprintf("  %d. %s (%s)\n", i+1, step.Tool, step.Purpose)
		}
	}

	result += "Tools:\n"
	for _, tc := range trace.ToolResults {
		status := "✓"
		if tc.Error != "" {
			status = "✗"
		}
		result += fmt.Sprintf("  %s %s (%dms)\n", status, tc.Tool, tc.Duration.Milliseconds())
	}

	result += fmt.Sprintf("Response: generated ✓\n")
	return result
}
