package agent

import (
	"context"
	"fmt"
)

type Planner struct {
	decisionEngine *DecisionEngine
}

func NewPlanner(de *DecisionEngine) *Planner {
	return &Planner{decisionEngine: de}
}

type Plan struct {
	Intent   IntentType          `json:"intent"`
	Strategy PedagogicalStrategy `json:"strategy"`
	Steps    []PlannedStep       `json:"steps"`
}

type PlannedStep struct {
	Tool     string         `json:"tool"`
	Purpose  string         `json:"purpose"`
	Input    map[string]any `json:"input,omitempty"`
}

func (p *Planner) CreatePlan(ctx context.Context, intent IntentType, studentCtx *StudentContext, query string, frustrationDetected bool) *Plan {
	tools := p.decisionEngine.SelectTools(ctx, intent, studentCtx)
	strategy := p.decisionEngine.SelectStrategy(ctx, intent, studentCtx, frustrationDetected)

	steps := make([]PlannedStep, 0, len(tools))
	for _, tool := range tools {
		steps = append(steps, PlannedStep{
			Tool:    tool,
			Purpose: p.toolPurpose(tool),
			Input:   p.buildInput(tool, query, studentCtx),
		})
	}

	return &Plan{
		Intent:   intent,
		Strategy: strategy,
		Steps:    steps,
	}
}

func (p *Planner) toolPurpose(tool string) string {
	purposes := map[string]string{
		"rag_search":        "buscar material académico relevante",
		"math_solve":        "resolver expresión matemática",
		"math_verify":       "verificar respuesta del estudiante",
		"student_profile":   "consultar perfil del estudiante",
		"exercise_generate": "generar ejercicio",
		"grade_answer":      "evaluar respuesta",
		"generate_hint":     "generar pista pedagógica",
		"assessment":        "gestionar evaluación",
	}
	if p, ok := purposes[tool]; ok {
		return p
	}
	return fmt.Sprintf("ejecutar %s", tool)
}

func (p *Planner) buildInput(tool string, query string, studentCtx *StudentContext) map[string]any {
	input := make(map[string]any)
	switch tool {
	case "rag_search":
		input["query"] = query
		if studentCtx != nil {
			input["filters"] = map[string]string{
				"course_id": studentCtx.CourseID,
			}
		}
	case "student_profile":
		if studentCtx != nil {
			input["student_id"] = studentCtx.StudentID
			input["course_id"] = studentCtx.CourseID
		}
	case "exercise_generate":
		if studentCtx != nil {
			input["concept"] = studentCtx.CurrentTopic
			input["student_id"] = studentCtx.StudentID
		}
	case "math_solve":
		input["operation"] = "evaluate"
		input["expression"] = query
	case "math_verify":
		input["operation"] = "derivative"
	}
	return input
}
