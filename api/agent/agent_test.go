package agent

import (
	"context"
	"testing"
)

func TestIntentClassifier_ClassifyByKeywords(t *testing.T) {
	ic := &IntentClassifier{}

	tests := []struct {
		query    string
		expected IntentType
	}{
		{"qué es una derivada", IntentExplainConcept},
		{"resuelve la integral de x^2", IntentSolveExercise},
		{"mi respuesta está bien", IntentCheckAnswer},
		{"dame una pista", IntentGiveHint},
		{"quiero practicar", IntentPractice},
		{"compara derivadas e integrales", IntentCompareConcepts},
		{"dame un ejemplo", IntentGenerateExample},
		{"qué fuente usaste", IntentAskSource},
		{"cómo voy en mi progreso", IntentShowProgress},
		{"qué me recomiendas estudiar", IntentRecommendation},
		{"resume el tema de límites", IntentSummarizeMaterial},
	}

	for _, tt := range tests {
		result := ic.classifyByKeywords(tt.query)
		if result.Intent != tt.expected {
			t.Errorf("classifyByKeywords(%q) = %s, want %s", tt.query, result.Intent, tt.expected)
		}
	}
}

func TestDetectFrustration(t *testing.T) {
	ic := &IntentClassifier{}

	frustrated := []string{
		"no entiendo nada",
		"ya lo intenté muchas veces y no me sale",
		"no entiendo tu explicación",
	}

	normal := []string{
		"¿cómo se resuelve esta integral?",
		"explícame la regla de la cadena",
	}

	for _, q := range frustrated {
		if !ic.DetectFrustration(q) {
			t.Errorf("DetectFrustration(%q) = false, want true", q)
		}
	}
	for _, q := range normal {
		if ic.DetectFrustration(q) {
			t.Errorf("DetectFrustration(%q) = true, want false", q)
		}
	}
}

func TestContextManager_DetermineStrategy(t *testing.T) {
	cm := &ContextManager{}

	tests := []struct {
		mastery  float64
		intent   IntentType
		expected PedagogicalStrategy
	}{
		{0.30, IntentExplainConcept, StrategyExampleFirst},
		{0.55, IntentExplainConcept, StrategyGuided},
		{0.80, IntentExplainConcept, StrategyDirect},
		{0.30, IntentSolveExercise, StrategyGuided},
		{0.55, IntentSolveExercise, StrategyStepByStep},
		{0.80, IntentSolveExercise, StrategyDirect},
		{0.30, IntentPractice, StrategyRemedial},
		{0.90, IntentPractice, StrategyChallenge},
		{0.55, IntentPractice, StrategyStepByStep},
		{0.55, IntentGiveHint, StrategySocratic},
	}

	for _, tt := range tests {
		sc := &StudentContext{Mastery: tt.mastery}
		result := cm.DetermineStrategy(sc, tt.intent)
		if result != tt.expected {
			t.Errorf("DetermineStrategy(mastery=%.2f, intent=%s) = %s, want %s",
				tt.mastery, tt.intent, result, tt.expected)
		}
	}
}

func TestCalculateDifficulty(t *testing.T) {
	cm := &ContextManager{}

	tests := []struct {
		mastery  float64
		expected int
	}{
		{0.10, 1},
		{0.30, 2},
		{0.55, 3},
		{0.75, 4},
		{0.90, 5},
	}

	for _, tt := range tests {
		sc := &StudentContext{Mastery: tt.mastery}
		result := cm.CalculateDifficulty(sc)
		if result != tt.expected {
			t.Errorf("CalculateDifficulty(%.2f) = %d, want %d", tt.mastery, result, tt.expected)
		}
	}
}

func TestPlanner_CreatePlan(t *testing.T) {
	de := NewDecisionEngine(&AgentConfig{
		LowMastery:  0.40,
		HighMastery: 0.70,
	})
	planner := NewPlanner(de)

	tests := []struct {
		intent    IntentType
		mastery   float64
		toolCount int
	}{
		{IntentExplainConcept, 0.50, 2},
		{IntentSolveExercise, 0.50, 2},
		{IntentCheckAnswer, 0.50, 3},
		{IntentPractice, 0.50, 2},
	}

	for _, tt := range tests {
		sc := &StudentContext{Mastery: tt.mastery}
		plan := planner.CreatePlan(context.Background(), tt.intent, sc, "test query", false)
		if len(plan.Steps) != tt.toolCount {
			t.Errorf("CreatePlan(%s) steps = %d, want %d", tt.intent, len(plan.Steps), tt.toolCount)
		}
	}
}

func TestCitationManager_Empty(t *testing.T) {
	cm := NewCitationManager()
	if cm.HasCitations() {
		t.Error("NewCitationManager().HasCitations() = true, want false")
	}
}

func TestCitationManager_FormatCitations(t *testing.T) {
	cm := NewCitationManager()
	cm.AddFromToolResults([]map[string]any{
		{"sources": []map[string]any{
			{"document_title": "Calculo I.pdf", "page": 52, "section": "Regla de la cadena"},
		}},
	})

	if !cm.HasCitations() {
		t.Error("HasCitations() = false after adding sources")
	}

	formatted := cm.FormatCitations()
	if formatted == "" {
		t.Error("FormatCitations() returned empty string")
	}
}

func TestAgentSanity(t *testing.T) {
	intents := []IntentType{
		IntentAskTheory, IntentSolveExercise, IntentExplainConcept,
		IntentCheckAnswer, IntentCheckProcedure, IntentGenerateExercise,
		IntentPractice, IntentGiveHint, IntentReviewTopic,
		IntentStartAssessment, IntentContinueAssessment, IntentShowProgress,
		IntentRecommendation, IntentAskSource, IntentSummarizeMaterial,
		IntentCompareConcepts, IntentGenerateExample,
	}

	for _, intent := range intents {
		if intent == "" {
			t.Error("empty intent constant found")
		}
	}

	strategies := []PedagogicalStrategy{
		StrategyDirect, StrategySocratic, StrategyGuided,
		StrategyExampleFirst, StrategyStepByStep, StrategyRemedial,
		StrategyChallenge,
	}

	for _, s := range strategies {
		if s == "" {
			t.Error("empty strategy constant found")
		}
	}
}

func TestToolRegistry_Basic(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name:        "test_tool",
		Description: "test",
		Permission:  "read",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			return map[string]any{"result": "ok"}, nil
		},
	})

	_, ok := tr.Get("test_tool")
	if !ok {
		t.Error("Get('test_tool') = false, want true")
	}

	_, ok = tr.Get("nonexistent")
	if ok {
		t.Error("Get('nonexistent') = true, want false")
	}
}

func TestToolRegistry_Execute(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name: "echo",
		Handler: func(ctx context.Context, input map[string]any) (map[string]any, error) {
			return input, nil
		},
	})

	tc := &ToolCall{
		Tool:  "echo",
		Input: map[string]any{"key": "value"},
	}

	err := tr.Execute(context.Background(), tc)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if tc.Result["key"] != "value" {
		t.Errorf("Execute() result[key] = %v, want 'value'", tc.Result["key"])
	}
}
