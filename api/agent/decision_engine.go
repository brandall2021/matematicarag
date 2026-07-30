package agent

import (
	"context"
)

type DecisionEngine struct {
	cfg *AgentConfig
}

func NewDecisionEngine(cfg *AgentConfig) *DecisionEngine {
	return &DecisionEngine{cfg: cfg}
}

func (de *DecisionEngine) SelectTools(ctx context.Context, intent IntentType, studentCtx *StudentContext) []string {
	switch intent {
	case IntentAskTheory, IntentExplainConcept:
		return []string{"rag_search", "student_profile"}
	case IntentSolveExercise:
		return []string{"math_solve", "rag_search"}
	case IntentCheckAnswer, IntentCheckProcedure:
		return []string{"math_evaluate", "student_profile", "rag_search"}
	case IntentGenerateExercise:
		return []string{"student_profile", "exercise_generate"}
	case IntentPractice:
		return []string{"student_profile", "exercise_generate"}
	case IntentGiveHint:
		return []string{"student_profile", "math_verify", "rag_search"}
	case IntentReviewTopic:
		return []string{"rag_search", "student_profile"}
	case IntentShowProgress:
		return []string{"student_profile"}
	case IntentRecommendation:
		return []string{"student_profile", "rag_search"}
	case IntentAskSource:
		return []string{"rag_search"}
	case IntentSummarizeMaterial:
		return []string{"rag_search"}
	case IntentCompareConcepts:
		return []string{"rag_search"}
	case IntentGenerateExample:
		return []string{"rag_search", "exercise_generate"}
	default:
		return []string{"rag_search", "student_profile"}
	}
}

func (de *DecisionEngine) SelectStrategy(ctx context.Context, intent IntentType, studentCtx *StudentContext, frustrationDetected bool) PedagogicalStrategy {
	if frustrationDetected {
		if studentCtx.Mastery < de.cfg.LowMastery {
			return StrategyRemedial
		}
		return StrategyExampleFirst
	}

	m := studentCtx.Mastery
	switch intent {
	case IntentExplainConcept, IntentAskTheory:
		if m < de.cfg.LowMastery {
			return StrategyExampleFirst
		}
		if m < de.cfg.HighMastery {
			return StrategyGuided
		}
		return StrategyDirect
	case IntentSolveExercise, IntentCheckAnswer:
		if m < de.cfg.LowMastery {
			return StrategyGuided
		}
		if m < de.cfg.HighMastery {
			return StrategyStepByStep
		}
		return StrategyDirect
	case IntentPractice:
		if m < de.cfg.LowMastery {
			return StrategyRemedial
		}
		if m >= 0.85 {
			return StrategyChallenge
		}
		return StrategyStepByStep
	case IntentGiveHint:
		return StrategySocratic
	default:
		return StrategyDirect
	}
}
