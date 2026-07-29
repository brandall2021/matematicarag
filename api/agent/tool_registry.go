package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brandall2021/matematicarag/api/adaptive"
)

type ToolFunc func(ctx context.Context, input map[string]any) (map[string]any, error)

type ToolDefinition struct {
	Name        string
	Description string
	Permission  string
	Handler     ToolFunc
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
}

func (tr *ToolRegistry) Register(td ToolDefinition) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.tools[td.Name] = td
}

func (tr *ToolRegistry) Get(name string) (ToolDefinition, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	td, ok := tr.tools[name]
	return td, ok
}

func (tr *ToolRegistry) Execute(ctx context.Context, tc *ToolCall) error {
	td, ok := tr.Get(tc.Tool)
	if !ok {
		return fmt.Errorf("tool %q not found", tc.Tool)
	}

	start := time.Now()
	result, err := td.Handler(ctx, tc.Input)
	tc.Duration = time.Since(start)

	if err != nil {
		tc.Error = err.Error()
		return err
	}
	tc.Result = result
	return nil
}

func RegisterAllTools(tr *ToolRegistry, deps *ToolDependencies) {
	tr.Register(NewRagTool(deps.HybridSearchFn, deps.RerankFn))
	tr.Register(NewMathTool(deps.MathSolveFn))
	tr.Register(NewVerifyTool(deps.VerifyFn))
	tr.Register(NewStudentTool(deps.StudentProfileFn))
	tr.Register(NewExerciseTool(deps.ExerciseGenerateFn))
	tr.Register(NewGradingTool(deps.GradeFn))
	tr.Register(NewHintTool())
	tr.Register(NewAssessmentTool())
}

type ToolDependencies struct {
	HybridSearchFn    func(ctx context.Context, query string, filters map[string]any, topK int, vectorWeight, keywordWeight float64) ([]map[string]any, error)
	RerankFn          func(ctx context.Context, query string, results []map[string]any, topK int) ([]map[string]any, error)
	MathSolveFn       func(ctx context.Context, operation, expression, variable string, lower, upper *float64) (map[string]any, error)
	VerifyFn          func(ctx context.Context, problem, studentAnswer, op string) (map[string]any, error)
	StudentProfileFn   func(ctx context.Context, studentID, courseID string) (map[string]any, error)
	ExerciseGenerateFn func(ctx context.Context, concept string, difficulty int, studentID string) (map[string]any, error)
	GradeFn           func(ctx context.Context, studentAnswer, expectedAnswer string) (map[string]any, error)
	AdaptiveEngine    *adaptive.AdaptiveEngine
}

func (tr *ToolRegistry) ListTools() []ToolDefinition {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	list := make([]ToolDefinition, 0, len(tr.tools))
	for _, td := range tr.tools {
		list = append(list, td)
	}
	return list
}
