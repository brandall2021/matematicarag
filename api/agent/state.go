package agent

import "time"

type IntentType string

const (
	IntentAskTheory          IntentType = "ASK_THEORY"
	IntentSolveExercise      IntentType = "SOLVE_EXERCISE"
	IntentExplainConcept     IntentType = "EXPLAIN_CONCEPT"
	IntentCheckAnswer        IntentType = "CHECK_ANSWER"
	IntentCheckProcedure     IntentType = "CHECK_PROCEDURE"
	IntentGenerateExercise   IntentType = "GENERATE_EXERCISE"
	IntentPractice           IntentType = "PRACTICE"
	IntentGiveHint           IntentType = "GIVE_HINT"
	IntentReviewTopic        IntentType = "REVIEW_TOPIC"
	IntentStartAssessment    IntentType = "START_ASSESSMENT"
	IntentContinueAssessment IntentType = "CONTINUE_ASSESSMENT"
	IntentShowProgress       IntentType = "SHOW_PROGRESS"
	IntentRecommendation     IntentType = "RECOMMENDATION"
	IntentAskSource          IntentType = "ASK_SOURCE"
	IntentSummarizeMaterial  IntentType = "SUMMARIZE_MATERIAL"
	IntentCompareConcepts    IntentType = "COMPARE_CONCEPTS"
	IntentGenerateExample    IntentType = "GENERATE_EXAMPLE"
)

type PedagogicalStrategy string

const (
	StrategyDirect      PedagogicalStrategy = "DIRECT"
	StrategySocratic    PedagogicalStrategy = "SOCRATIC"
	StrategyGuided      PedagogicalStrategy = "GUIDED"
	StrategyExampleFirst PedagogicalStrategy = "EXAMPLE_FIRST"
	StrategyStepByStep   PedagogicalStrategy = "STEP_BY_STEP"
	StrategyRemedial    PedagogicalStrategy = "REMEDIAL"
	StrategyChallenge   PedagogicalStrategy = "CHALLENGE"
)

type IntentResult struct {
	Intent     IntentType `json:"intent"`
	Confidence float64    `json:"confidence"`
	Concept    string     `json:"concept,omitempty"`
	Expression string     `json:"expression,omitempty"`
}

type MultiIntentResult struct {
	Intents []IntentResult `json:"intents"`
	Plan    []string       `json:"plan,omitempty"`
}

type ToolCall struct {
	Tool     string         `json:"tool"`
	Purpose  string         `json:"purpose"`
	Input    map[string]any `json:"input"`
	Result   map[string]any `json:"result,omitempty"`
	Error    string         `json:"error,omitempty"`
	Duration time.Duration  `json:"-"`
}

type ExecutionPlan struct {
	Intent IntentType `json:"intent"`
	Steps  []ToolCall `json:"steps"`
}

type AgentState struct {
	SessionID       string              `json:"session_id"`
	StudentID       string              `json:"student_id"`
	CourseID        string              `json:"course_id"`
	Intent          IntentType          `json:"intent"`
	Mode            string              `json:"mode"`
	Strategy        PedagogicalStrategy `json:"strategy"`
	CurrentConcept  string              `json:"current_concept"`
	CurrentExercise string              `json:"current_exercise_id,omitempty"`
	CurrentStep     int                 `json:"current_step"`
	PendingAction   string              `json:"pending_action,omitempty"`
	ToolCalls       []ToolCall          `json:"tool_calls"`
	Messages        []Message           `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StudentContext struct {
	StudentID      string             `json:"student_id"`
	CourseID       string             `json:"course_id"`
	CurrentTopic   string             `json:"current_topic"`
	Mastery        float64            `json:"mastery"`
	RecentErrors   []StudentErrorInfo `json:"recent_errors"`
	RecentAttempts int                `json:"recent_attempts"`
	CurrentMode    string             `json:"current_mode"`
	LearningPrefs  map[string]any     `json:"learning_preferences,omitempty"`
}

type StudentErrorInfo struct {
	Concept   string `json:"concept"`
	ErrorType string `json:"error_type"`
	Count     int    `json:"count"`
}

type AgentConfig struct {
	MaxToolCalls       int
	MaxRetries         int
	IntentThreshold    float64
	RagTopK            int
	RerankTopK         int
	MathTimeout        int
	QdrantTimeout      int
	LLMTimeout         int
	LowMastery         float64
	HighMastery        float64
	ManualReviewThresh float64
}

func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		MaxToolCalls:       8,
		MaxRetries:         2,
		IntentThreshold:    0.75,
		RagTopK:            20,
		RerankTopK:         5,
		MathTimeout:        5,
		QdrantTimeout:      3,
		LLMTimeout:         30,
		LowMastery:         0.40,
		HighMastery:        0.70,
		ManualReviewThresh: 0.65,
	}
}
