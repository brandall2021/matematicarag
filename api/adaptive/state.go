package adaptive

import "time"

type AdaptiveConfig struct {
	MasteryOldWeight      float64
	MasteryEvidenceWeight float64
	MasteryHintPenalty    float64
	MasteryErrorPenalty   float64
	MasteryRecencyFactor  float64
	CriticalThreshold     float64
	BeginnerThreshold     float64
	DevelopingThreshold   float64
	CompetentThreshold    float64
	MaxDifficulty         int
}

type LearningEvent struct {
	ID          string                 `json:"id"`
	StudentID   string                 `json:"student_id"`
	CourseID    string                 `json:"course_id"`
	ConceptID   string                 `json:"concept_id"`
	ActivityID  string                 `json:"activity_id,omitempty"`
	EventType   string                 `json:"event_type"`
	Difficulty  int                    `json:"difficulty"`
	Correct     bool                   `json:"correct"`
	Score       float64                `json:"score"`
	TimeSecs    int                    `json:"time_seconds"`
	HintsUsed   int                    `json:"hints_used"`
	ErrorType   string                 `json:"error_type,omitempty"`
	ErrorDetail string                 `json:"error_detail,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type MasteryRecord struct {
	StudentID            string     `json:"student_id"`
	ConceptID            string     `json:"concept_id"`
	CourseID             string     `json:"course_id"`
	Mastery              float64    `json:"mastery"`
	Status               string     `json:"status"`
	Attempts             int        `json:"attempts"`
	Correct              int        `json:"correct"`
	Incorrect            int        `json:"incorrect"`
	HintsUsed            int        `json:"hints_used"`
	IndependentSuccesses int        `json:"independent_successes"`
	AvgTimeSecs          int        `json:"average_time_seconds"`
	Confidence           float64    `json:"confidence"`
	LastAttemptAt        *time.Time `json:"last_attempt_at"`
	LastSuccessAt        *time.Time `json:"last_success_at"`
	LastErrorAt          *time.Time `json:"last_error_at"`
	NextReviewAt         *time.Time `json:"next_review_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type StudentErrorRecord struct {
	ID          string     `json:"id"`
	StudentID   string     `json:"student_id"`
	CourseID    string     `json:"course_id"`
	ConceptID   string     `json:"concept_id"`
	ActivityID  string     `json:"activity_id,omitempty"`
	ErrorType   string     `json:"error_type"`
	ErrorDetail string     `json:"error_detail,omitempty"`
	Severity    string     `json:"severity"`
	Resolved    bool       `json:"resolved"`
	AttemptID   string     `json:"attempt_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type Recommendation struct {
	ConceptID  string  `json:"concept_id"`
	Action     string  `json:"action"`
	Difficulty int     `json:"difficulty"`
	Reason     string  `json:"reason"`
	Score      float64 `json:"score"`
}

type LearnerState struct {
	StudentID            string     `json:"student_id"`
	CourseID             string     `json:"course_id"`
	OverallMastery       float64    `json:"overall_mastery"`
	CurrentTopic         string     `json:"current_topic"`
	CurrentConcept       string     `json:"current_concept"`
	StrongConcepts       []string   `json:"strong_concepts"`
	WeakConcepts         []string   `json:"weak_concepts"`
	RecentErrors         []string   `json:"recent_errors"`
	RecentSuccesses      int        `json:"recent_successes"`
	TotalAttempts        int        `json:"total_attempts"`
	SuccessfulAttempts   int        `json:"successful_attempts"`
	FailedAttempts       int        `json:"failed_attempts"`
	HintUsage            int        `json:"hint_usage"`
	AvgResolutionTime    int        `json:"average_resolution_time"`
	RecommendedAction    string     `json:"recommended_action"`
	RecommendedConcept   string     `json:"recommended_concept"`
	RecommendedDifficulty int       `json:"recommended_difficulty"`
	LastActivityAt       *time.Time `json:"last_activity_at"`
}
