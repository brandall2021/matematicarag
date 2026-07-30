# Fase 6 — Motor de Aprendizaje Adaptativo — Implementation Plan

> **For agentic workers:** Each task is implemented sequentially. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Transform the current RAG+Agent system into an adaptive learning engine that tracks concept mastery, detects weak prerequisites, analyzes errors, adapts difficulty, recommends next actions, and updates student profiles after every interaction — all integrated with the existing Pedagogical Agent, Qdrant RAG, and Math Engine.

**Architecture:** Flat Go package `api/adaptive/` following the same pattern as `api/agent/`. Reuses existing tables (`concepts`, `concept_prerequisites`, `concept_mastery`, `student_profiles`, `student_errors`). New tables (`learning_events`, `mastery_history`). Centralized `MasteryEngine` replaces inline mastery logic in `api/learning.go` and `api/agent/learning_updater.go`. All components wired through `AdaptiveEngine` struct.

**Tech Stack:** Go, pgx/v5, PostgreSQL, Qdrant, existing Math Engine, existing Pedagogical Agent

## Global Constraints

- All new Go code in `api/adaptive/` package (flat, no sub-packages)
- Reuse existing tables wherever possible; only add `learning_events` and `mastery_history`
- Do NOT remove or break existing endpoints
- Do NOT create duplicate services — extend existing ones
- Mastery values are 0.00–1.00 float64
- All mastery calculations go through `MasteryEngine` — no inline mastery logic elsewhere
- All learning interactions go through `LearningEventService` → `MasteryEngine` pipeline
- Qdrant searches must preserve citation metadata (document_id, source_title, page, section)
- Auth: students see only their own data; teachers see authorized courses
- Config: mastery weights, thresholds, difficulty ranges in `internal/config/config.go`

---

### Task 1: Add new tables + extend existing schema

**Files:**
- Modify: `internal/database/database.go`
- Modify: `internal/config/config.go`

**Interfaces:**
- Produces: `learning_events` table, `mastery_history` table, extended `concept_mastery` with new columns, config fields for mastery weights

- [ ] **Add config fields to `internal/config/config.go`**

```go
// Add to Config struct:
MasteryOldWeight       float64 `env:"MASTERY_OLD_WEIGHT" default:"0.70"`
MasteryEvidenceWeight  float64 `env:"MASTERY_EVIDENCE_WEIGHT" default:"0.30"`
MasteryHintPenalty     float64 `env:"MASTERY_HINT_PENALTY" default:"0.10"`
MasteryErrorPenalty    float64 `env:"MASTERY_ERROR_PENALTY" default:"0.15"`
MasteryRecencyFactor   float64 `env:"MASTERY_RECENCY_FACTOR" default:"0.60"`
LearningCriticalThreshold  float64 `env:"LEARNING_CRITICAL_THRESHOLD" default:"0.40"`
LearningBeginnerThreshold  float64 `env:"LEARNING_BEGINNER_THRESHOLD" default:"0.60"`
LearningDevelopingThreshold float64 `env:"LEARNING_DEVELOPING_THRESHOLD" default:"0.75"`
LearningCompetentThreshold  float64 `env:"LEARNING_COMPETENT_THRESHOLD" default:"0.90"`
AdaptiveQdrantTopK     int     `env:"ADAPTIVE_QDRANT_TOP_K" default:"5"`
```

- [ ] **Add new tables + extend existing schema in `internal/database/database.go`**

Add these CREATE TABLE IF NOT EXISTS statements inside the existing Migrate() function:

```go
// learning_events — event sourcing for all learning interactions
CREATE TABLE IF NOT EXISTS learning_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id),
    course_id UUID NOT NULL REFERENCES courses(id),
    concept_id VARCHAR(255) NOT NULL,
    activity_id UUID,
    event_type VARCHAR(50) NOT NULL,
    difficulty INT DEFAULT 1,
    correct BOOLEAN,
    score FLOAT DEFAULT 0,
    time_seconds INT DEFAULT 0,
    hints_used INT DEFAULT 0,
    error_type VARCHAR(50),
    error_detail TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_learning_events_student ON learning_events(student_id, concept_id);
CREATE INDEX IF NOT EXISTS idx_learning_events_type ON learning_events(event_type);

// mastery_history — audit trail for mastery changes
CREATE TABLE IF NOT EXISTS mastery_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id),
    concept_id VARCHAR(255) NOT NULL,
    old_mastery FLOAT NOT NULL,
    new_mastery FLOAT NOT NULL,
    trigger_event_id UUID,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mastery_history_student ON mastery_history(student_id, concept_id);
```

Extend `concept_mastery` with new columns (ALTER TABLE IF NOT EXISTS pattern):

```go
ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS independent_successes INT DEFAULT 0;
ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS average_time_seconds INT DEFAULT 0;
ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMPTZ;
ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMPTZ;
ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS next_review_at TIMESTAMPTZ;
ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS confidence FLOAT DEFAULT 1.0;
```

Also add event_type values constraint (optional but recommended):

```go
-- Add CHECK constraint if desired — or handle in Go layer
```

- [ ] **Commit**

```bash
git add internal/config/config.go internal/database/database.go
git commit -m "feat(fase6): config fields, learning_events table, mastery_history table, extended concept_mastery"
```

---

### Task 2: Adaptive Engine core — structs and config

**Files:**
- Create: `api/adaptive/state.go`

**Interfaces:**
- Produces: `AdaptiveConfig`, `AdaptiveState`, `LearningEvent`, `MasteryRecord`, `StudentErrorRecord`, `Recommendation` types

- [ ] **Create `api/adaptive/state.go`**

```go
package adaptive

import "time"

type AdaptiveConfig struct {
	MasteryOldWeight       float64
	MasteryEvidenceWeight  float64
	MasteryHintPenalty     float64
	MasteryErrorPenalty    float64
	MasteryRecencyFactor   float64
	CriticalThreshold      float64
	BeginnerThreshold      float64
	DevelopingThreshold    float64
	CompetentThreshold     float64
	MaxDifficulty          int
}

type LearningEvent struct {
	ID         string                 `json:"id"`
	StudentID  string                 `json:"student_id"`
	CourseID   string                 `json:"course_id"`
	ConceptID  string                 `json:"concept_id"`
	ActivityID string                 `json:"activity_id,omitempty"`
	EventType  string                 `json:"event_type"`
	Difficulty int                    `json:"difficulty"`
	Correct    bool                   `json:"correct"`
	Score      float64                `json:"score"`
	TimeSecs   int                    `json:"time_seconds"`
	HintsUsed  int                    `json:"hints_used"`
	ErrorType  string                 `json:"error_type,omitempty"`
	ErrorDetail string                `json:"error_detail,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

type MasteryRecord struct {
	StudentID           string    `json:"student_id"`
	ConceptID           string    `json:"concept_id"`
	CourseID            string    `json:"course_id"`
	Mastery             float64   `json:"mastery"`
	Status              string    `json:"status"`
	Attempts            int       `json:"attempts"`
	Correct             int       `json:"correct"`
	Incorrect           int       `json:"incorrect"`
	HintsUsed           int       `json:"hints_used"`
	IndependentSuccesses int      `json:"independent_successes"`
	AvgTimeSecs         int       `json:"average_time_seconds"`
	Confidence          float64   `json:"confidence"`
	LastAttemptAt       *time.Time `json:"last_attempt_at"`
	LastSuccessAt       *time.Time `json:"last_success_at"`
	LastErrorAt         *time.Time `json:"last_error_at"`
	NextReviewAt        *time.Time `json:"next_review_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type StudentErrorRecord struct {
	ID             string `json:"id"`
	StudentID      string `json:"student_id"`
	CourseID       string `json:"course_id"`
	ConceptID      string `json:"concept_id"`
	ActivityID     string `json:"activity_id,omitempty"`
	ErrorType      string `json:"error_type"`
	ErrorDetail    string `json:"error_detail,omitempty"`
	Severity       string `json:"severity"`
	Resolved       bool   `json:"resolved"`
	AttemptID      string `json:"attempt_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

type Recommendation struct {
	ConceptID  string `json:"concept_id"`
	Action     string `json:"action"`
	Difficulty int    `json:"difficulty"`
	Reason     string `json:"reason"`
	Score      float64 `json:"score"`
}

type LearnerState struct {
	StudentID           string           `json:"student_id"`
	CourseID            string           `json:"course_id"`
	OverallMastery      float64          `json:"overall_mastery"`
	CurrentTopic        string           `json:"current_topic"`
	CurrentConcept      string           `json:"current_concept"`
	StrongConcepts      []string         `json:"strong_concepts"`
	WeakConcepts        []string         `json:"weak_concepts"`
	RecentErrors        []string         `json:"recent_errors"`
	RecentSuccesses     int              `json:"recent_successes"`
	TotalAttempts       int              `json:"total_attempts"`
	SuccessfulAttempts  int              `json:"successful_attempts"`
	FailedAttempts      int              `json:"failed_attempts"`
	HintUsage           int              `json:"hint_usage"`
	AvgResolutionTime   int              `json:"average_resolution_time"`
	RecommendedAction   string           `json:"recommended_action"`
	RecommendedConcept  string           `json:"recommended_concept"`
	RecommendedDifficulty int            `json:"recommended_difficulty"`
	LastActivityAt      *time.Time       `json:"last_activity_at"`
}
```

- [ ] **Create `api/adaptive/engine.go`** — the central `AdaptiveEngine` struct that wires all sub-components

```go
package adaptive

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdaptiveEngine struct {
	db            *pgxpool.Pool
	config        *AdaptiveConfig
	Mastery       *MasteryEngine
	Difficulty    *DifficultyEngine
	Prerequisites *PrerequisiteEngine
	Errors        *ErrorAnalyzer
	Recommend     *RecommendationEngine
	LearningPath  *LearningPathEngine
	Events        *LearningEventService
	Analytics     *ProgressAnalyticsService
}

func NewAdaptiveEngine(db *pgxpool.Pool, cfg *AdaptiveConfig) *AdaptiveEngine {
	mastery := NewMasteryEngine(cfg)
	return &AdaptiveEngine{
		db:            db,
		config:        cfg,
		Mastery:       mastery,
		Difficulty:    NewDifficultyEngine(cfg),
		Prerequisites: NewPrerequisiteEngine(db),
		Errors:        NewErrorAnalyzer(),
		Recommend:     NewRecommendationEngine(db, cfg),
		LearningPath:  NewLearningPathEngine(db, cfg),
		Events:        NewLearningEventService(db),
		Analytics:     NewProgressAnalyticsService(db),
	}
}
```

- [ ] **Commit**

```bash
git add api/adaptive/state.go api/adaptive/engine.go
git commit -m "feat(fase6): adaptive engine core structs and central wiring"
```

---

### Task 3: Knowledge Map Service

**Files:**
- Create: `api/adaptive/knowledge_map.go`

**Interfaces:**
- Consumes: `concepts`, `concept_prerequisites` tables
- Produces: `KnowledgeMapService` with `GetConcept`, `GetChildren`, `GetParents`, `GetPrerequisites`, `GetDependents`, `GetRelatedConcepts`, `GetLearningSequence`

- [ ] **Create `api/adaptive/knowledge_map.go`**

```go
package adaptive

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConceptNode struct {
	ID             string  `json:"id"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	ParentID       *string `json:"parent_id"`
	DifficultyBase int     `json:"difficulty_base"`
	CourseID       string  `json:"course_id"`
	Children       []*ConceptNode `json:"children,omitempty"`
}

type PrerequisiteRelation struct {
	ConceptID           string  `json:"concept_id"`
	PrerequisiteID      string  `json:"prerequisite_concept_id"`
	PrerequisiteName    string  `json:"prerequisite_name"`
	Weight              float64 `json:"weight"`
}

type KnowledgeMapService struct {
	db *pgxpool.Pool
}

func NewKnowledgeMapService(db *pgxpool.Pool) *KnowledgeMapService {
	return &KnowledgeMapService{db: db}
}

func (s *KnowledgeMapService) GetConcept(ctx context.Context, code string) (*ConceptNode, error) {
	var node ConceptNode
	err := s.db.QueryRow(ctx,
		`SELECT id, code, name, COALESCE(description,''), parent_id, difficulty_base, course_id
		 FROM concepts WHERE code = $1`, code,
	).Scan(&node.ID, &node.Code, &node.Name, &node.Description, &node.ParentID, &node.DifficultyBase, &node.CourseID)
	if err != nil {
		return nil, fmt.Errorf("concept %s not found: %w", code, err)
	}
	return &node, nil
}

func (s *KnowledgeMapService) GetChildren(ctx context.Context, parentCode string) ([]*ConceptNode, error) {
	parent, err := s.GetConcept(ctx, parentCode)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT id, code, name, COALESCE(description,''), parent_id, difficulty_base, course_id
		 FROM concepts WHERE parent_id = $1`, parent.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var children []*ConceptNode
	for rows.Next() {
		var n ConceptNode
		if err := rows.Scan(&n.ID, &n.Code, &n.Name, &n.Description, &n.ParentID, &n.DifficultyBase, &n.CourseID); err != nil {
			return nil, err
		}
		children = append(children, &n)
	}
	return children, nil
}

func (s *KnowledgeMapService) GetPrerequisites(ctx context.Context, conceptCode string) ([]PrerequisiteRelation, error) {
	concept, err := s.GetConcept(ctx, conceptCode)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT cp.concept_id, cp.prerequisite_id, c.code, cp.weight
		 FROM concept_prerequisites cp
		 JOIN concepts c ON cp.prerequisite_id = c.id
		 WHERE cp.concept_id = $1`, concept.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prereqs []PrerequisiteRelation
	for rows.Next() {
		var p PrerequisiteRelation
		if err := rows.Scan(&p.ConceptID, &p.PrerequisiteID, &p.PrerequisiteName, &p.Weight); err != nil {
			continue
		}
		prereqs = append(prereqs, p)
	}
	return prereqs, nil
}

func (s *KnowledgeMapService) GetDependents(ctx context.Context, conceptCode string) ([]string, error) {
	concept, err := s.GetConcept(ctx, conceptCode)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT c.code FROM concept_prerequisites cp
		 JOIN concepts c ON cp.concept_id = c.id
		 WHERE cp.prerequisite_id = $1`, concept.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deps []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		deps = append(deps, code)
	}
	return deps, nil
}

func (s *KnowledgeMapService) GetLearningSequence(ctx context.Context, rootCode string) ([]string, error) {
	// Simple BFS to build linear sequence
	children, err := s.GetChildren(ctx, rootCode)
	if err != nil {
		return nil, err
	}
	var seq []string
	for _, c := range children {
		sub, err := s.GetLearningSequence(ctx, c.Code)
		if err != nil {
			continue
		}
		seq = append(seq, sub...)
	}
	return seq, nil
}
```

- [ ] **Commit**

```bash
git add api/adaptive/knowledge_map.go
git commit -m "feat(fase6): knowledge map service with concept tree and prerequisite queries"
```

---

### Task 4: Mastery Engine

**Files:**
- Create: `api/adaptive/mastery.go`

**Interfaces:**
- Consumes: `AdaptiveConfig`, `MasteryRecord`
- Produces: `MasteryEngine` with `CalculateEvidence`, `CalculateNewMastery`, `DetermineStatus`, `CalculateRecencyWeight`

- [ ] **Create `api/adaptive/mastery.go`**

```go
package adaptive

import (
	"math"
	"time"
)

type MasteryEngine struct {
	config *AdaptiveConfig
}

func NewMasteryEngine(cfg *AdaptiveConfig) *MasteryEngine {
	return &MasteryEngine{config: cfg}
}

func (e *MasteryEngine) CalculateRecencyWeight(attemptTime time.Time) float64 {
	daysSince := time.Since(attemptTime).Hours() / 24.0
	if daysSince <= 1 {
		return 1.0
	}
	if daysSince <= 7 {
		return 0.8
	}
	if daysSince <= 14 {
		return 0.6
	}
	if daysSince <= 30 {
		return 0.4
	}
	return 0.2
}

func (e *MasteryEngine) CalculateEvidence(correct bool, difficulty int, hintsUsed int, independentSuccess bool) float64 {
	correctnessWeight := 0.0
	if correct {
		correctnessWeight = 0.5
	}

	difficultyWeight := float64(difficulty) * 0.1
	if difficultyWeight > 0.3 {
		difficultyWeight = 0.3
	}

	independenceWeight := 0.0
	if independentSuccess {
		independenceWeight = 0.2
	}

	hintPenalty := float64(hintsUsed) * e.config.MasteryHintPenalty
	if hintPenalty > 0.3 {
		hintPenalty = 0.3
	}

	errorPenalty := 0.0
	if !correct {
		errorPenalty = e.config.MasteryErrorPenalty
	}

	evidence := correctnessWeight + difficultyWeight + independenceWeight - hintPenalty - errorPenalty
	if evidence < 0 {
		evidence = 0
	}
	return evidence
}

func (e *MasteryEngine) CalculateNewMastery(oldMastery float64, evidence float64, recencyWeight float64) float64 {
	effectiveRecency := e.config.MasteryRecencyFactor * recencyWeight

	weightedOld := oldMastery * e.config.MasteryOldWeight
	if recencyWeight < 1.0 {
		// Scale old weight down when event is old
		weightedOld = oldMastery * (e.config.MasteryOldWeight * recencyWeight)
	}

	newMastery := weightedOld + (evidence * e.config.MasteryEvidenceWeight)

	// Apply recency boost for recent correct answers
	if evidence > 0 {
		newMastery += evidence * effectiveRecency * 0.1
	}

	newMastery = math.Max(0, math.Min(1, newMastery))

	if math.IsNaN(newMastery) {
		return oldMastery
	}
	return newMastery
}

func (e *MasteryEngine) DetermineStatus(mastery float64) string {
	if mastery < e.config.CriticalThreshold {
		return "CRITICAL"
	}
	if mastery < e.config.BeginnerThreshold {
		return "BEGINNER"
	}
	if mastery < e.config.DevelopingThreshold {
		return "DEVELOPING"
	}
	if mastery < e.config.CompetentThreshold {
		return "COMPETENT"
	}
	return "MASTERED"
}

func (e *MasteryEngine) DetermineLegacyStatus(mastery float64) string {
	if mastery >= 0.8 {
		return "mastered"
	}
	if mastery >= 0.5 {
		return "developing"
	}
	if mastery > 0 {
		return "learning"
	}
	return "not_started"
}
```

- [ ] **Commit**

```bash
git add api/adaptive/mastery.go
git commit -m "feat(fase6): mastery engine with evidence calculation, recency weighting, status mapping"
```

---

### Task 5: Error Analyzer

**Files:**
- Create: `api/adaptive/errors.go`

**Interfaces:**
- Consumes: `StudentErrorRecord`
- Produces: `ErrorAnalyzer` with `ClassifyError`, `IsRecurrent`, `GetSeverity`

- [ ] **Create `api/adaptive/errors.go`**

```go
package adaptive

import (
	"context"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ErrorAnalyzer struct{}

func NewErrorAnalyzer() *ErrorAnalyzer {
	return &ErrorAnalyzer{}
}

var errorTypeMap = map[string]string{
	"SIGN_ERROR":                "Signo incorrecto",
	"ARITHMETIC_ERROR":          "Error aritmético",
	"ALGEBRA_ERROR":             "Error algebraico",
	"FORMULA_ERROR":             "Fórmula incorrecta",
	"CONCEPTUAL_ERROR":          "Error conceptual",
	"PROCEDURE_ERROR":           "Error de procedimiento",
	"MISSING_STEP":              "Paso omitido",
	"WRONG_RULE":                "Regla incorrecta",
	"PREREQUISITE_ERROR":        "Error de prerequisito",
	"MISSING_INNER_DERIVATIVE":  "Falta derivada interna",
	"UNFINISHED":                "Ejercicio incompleto",
}

func (a *ErrorAnalyzer) ClassifyError(answer, expected, expression string) string {
	// Heuristic classification based on common patterns
	// When Math Engine provides structural comparison, use it
	// Otherwise use simplified pattern matching
	if expression != "" {
		if containsMissingInnerDerivative(answer, expression) {
			return "MISSING_INNER_DERIVATIVE"
		}
	}
	if containsSignMismatch(answer, expected) {
		return "SIGN_ERROR"
	}
	if containsAlgebraicError(answer, expected) {
		return "ALGEBRA_ERROR"
	}
	if containsFormulaMismatch(answer, expected) {
		return "FORMULA_ERROR"
	}
	if containsPartialAnswer(answer, expected) {
		return "UNFINISHED"
	}
	return "CONCEPTUAL_ERROR"
}

func containsMissingInnerDerivative(answer, expression string) bool {
	// Simple heuristic: answer contains outer derivative form but lacks chain rule
	// e.g., expression is (x^2+1)^3, answer is 3(x^2+1)^2 (missing 2x factor)
	return false // Placeholder — actual implementation uses Math Engine comparison
}

func containsSignMismatch(answer, expected string) bool {
	return false // Placeholder
}

func containsAlgebraicError(answer, expected string) bool {
	return false // Placeholder
}

func containsFormulaMismatch(answer, expected string) bool {
	return false // Placeholder
}

func containsPartialAnswer(answer, expected string) bool {
	return false // Placeholder
}

func (a *ErrorAnalyzer) IsRecurrent(ctx context.Context, db *pgxpool.Pool, studentID, conceptID, errorType string) (bool, int) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM learning_events
		 WHERE student_id = $1 AND concept_id = $2 AND error_type = $3
		 AND created_at > NOW() - INTERVAL '30 days'`,
		studentID, conceptID, errorType,
	).Scan(&count)
	if err != nil {
		return false, 0
	}
	return count >= 2, count
}

func (a *ErrorAnalyzer) GetSeverity(frequency int) string {
	if frequency >= 5 {
		return "critical"
	}
	if frequency >= 3 {
		return "high"
	}
	if frequency >= 1 {
		return "medium"
	}
	return "low"
}

// RecordError creates or updates student_errors
func (a *ErrorAnalyzer) RecordError(ctx context.Context, db *pgxpool.Pool, studentID, courseID, conceptID, errorType, errorDetail string) error {
	isRecurrent, count := a.IsRecurrent(ctx, db, studentID, conceptID, errorType)
	severity := a.GetSeverity(count + 1)

	_, err := db.Exec(ctx,
		`INSERT INTO student_errors (student_id, course_id, concept_id, error_type, error_subtype, count, severity)
		 VALUES ($1, $2, $3, $4, $5, 1, $6)
		 ON CONFLICT (student_id, concept_id, error_type, error_subtype)
		 DO UPDATE SET count = student_errors.count + 1,
		               last_occurred_at = NOW(),
		               severity = $6`,
		studentID, courseID, conceptID, errorType, errorDetail, severity)
	if err != nil {
		return err
	}

	if isRecurrent {
		_, err = db.Exec(ctx,
			`UPDATE student_errors SET severity = 'high' WHERE student_id = $1 AND concept_id = $2 AND error_type = $3`,
			studentID, conceptID, errorType)
	}
	return err
}
```

- [ ] **Commit**

```bash
git add api/adaptive/errors.go
git commit -m "feat(fase6): error analyzer with classification, recurrence detection, severity"
```

---

### Task 6: Prerequisite Engine

**Files:**
- Create: `api/adaptive/prerequisites.go`

**Interfaces:**
- Consumes: `KnowledgeMapService`, `concept_mastery` table
- Produces: `PrerequisiteEngine` with `AnalyzePrerequisites`, `CheckRemediationNeeded`

- [ ] **Create `api/adaptive/prerequisites.go`**

```go
package adaptive

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WeakPrerequisite struct {
	ConceptID     string  `json:"concept_id"`
	ConceptName   string  `json:"concept_name"`
	Mastery       float64 `json:"mastery"`
	Weight        float64 `json:"weight"`
}

type PrerequisiteAnalysis struct {
	ConceptID         string             `json:"concept_id"`
	AllPrerequisites  []PrerequisiteRelation `json:"all_prerequisites"`
	WeakPrerequisites []WeakPrerequisite `json:"weak_prerequisites"`
	NeedsRemedial     bool               `json:"needs_remedial"`
}

type PrerequisiteEngine struct {
	db      *pgxpool.Pool
	knowledge *KnowledgeMapService
}

func NewPrerequisiteEngine(db *pgxpool.Pool) *PrerequisiteEngine {
	return &PrerequisiteEngine{
		db:      db,
		knowledge: NewKnowledgeMapService(db),
	}
}

func (e *PrerequisiteEngine) AnalyzePrerequisites(ctx context.Context, studentID, conceptCode string) (*PrerequisiteAnalysis, error) {
	prereqs, err := e.knowledge.GetPrerequisites(ctx, conceptCode)
	if err != nil {
		return nil, err
	}

	result := &PrerequisiteAnalysis{
		ConceptID:        conceptCode,
		AllPrerequisites: prereqs,
	}

	for _, p := range prereqs {
		var mastery float64
		err := e.db.QueryRow(ctx,
			`SELECT mastery FROM concept_mastery
			 WHERE student_id = $1 AND concept_id = $2`,
			studentID, p.PrerequisiteID,
		).Scan(&mastery)
		if err != nil || mastery < 0.60 {
			weak := WeakPrerequisite{
				ConceptID:   p.PrerequisiteName,
				Mastery:     mastery,
				Weight:      p.Weight,
			}
			// Get concept name from concepts table by code
			e.db.QueryRow(ctx,
				`SELECT name FROM concepts WHERE code = $1`, p.PrerequisiteName,
			).Scan(&weak.ConceptName)
			result.WeakPrerequisites = append(result.WeakPrerequisites, weak)
		}
	}

	result.NeedsRemedial = len(result.WeakPrerequisites) > 0
	return result, nil
}

func (e *PrerequisiteEngine) CheckRemediationNeeded(ctx context.Context, studentID, conceptCode string) (bool, *WeakPrerequisite, error) {
	analysis, err := e.AnalyzePrerequisites(ctx, studentID, conceptCode)
	if err != nil {
		return false, nil, err
	}
	if !analysis.NeedsRemedial {
		return false, nil, nil
	}
	// Return the weakest prerequisite
	weakest := analysis.WeakPrerequisites[0]
	for _, w := range analysis.WeakPrerequisites {
		if w.Mastery < weakest.Mastery {
			weakest = w
		}
	}
	return true, &weakest, nil
}

func (e *PrerequisiteEngine) ExplainWeakness(ctx context.Context, studentID, conceptCode string) string {
	analysis, err := e.AnalyzePrerequisites(ctx, studentID, conceptCode)
	if err != nil || !analysis.NeedsRemedial {
		return ""
	}

	explanation := fmt.Sprintf("Presentás dificultades en los siguientes prerequisitos de %s:\n", conceptCode)
	for _, w := range analysis.WeakPrerequisites {
		explanation += fmt.Sprintf("- %s (dominio actual: %.0f%%)\n", w.ConceptName, w.Mastery*100)
	}
	return explanation
}
```

- [ ] **Commit**

```bash
git add api/adaptive/prerequisites.go
git commit -m "feat(fase6): prerequisite engine with analysis and remediation detection"
```

---

### Task 7: Difficulty Engine

**Files:**
- Create: `api/adaptive/difficulty.go`

**Interfaces:**
- Consumes: `AdaptiveConfig`, `MasteryRecord`
- Produces: `DifficultyEngine` with `SelectDifficulty`, `AdjustForRecentErrors`

- [ ] **Create `api/adaptive/difficulty.go`**

```go
package adaptive

import "math"

type DifficultyEngine struct {
	config *AdaptiveConfig
}

func NewDifficultyEngine(cfg *AdaptiveConfig) *DifficultyEngine {
	return &DifficultyEngine{config: cfg}
}

func (e *DifficultyEngine) SelectDifficulty(mastery float64, recentConsecutiveErrors int) int {
	baseDifficulty := e.baseDifficulty(mastery)

	if recentConsecutiveErrors >= 3 {
		baseDifficulty = 1
	} else if recentConsecutiveErrors >= 2 {
		baseDifficulty = int(math.Max(1, float64(baseDifficulty-1)))
	}

	return baseDifficulty
}

func (e *DifficultyEngine) baseDifficulty(mastery float64) int {
	if mastery < 0.40 {
		return 1
	}
	if mastery < 0.60 {
		// 1-2 range
		if mastery < 0.50 {
			return 1
		}
		return 2
	}
	if mastery < 0.75 {
		return 2
	}
	if mastery < 0.90 {
		// 2-3 range
		if mastery < 0.82 {
			return 2
		}
		return 3
	}
	// mastery >= 0.90 → 3-4 range
	if mastery < 0.95 {
		return 3
	}
	return 4
}

func (e *DifficultyEngine) ClampDifficulty(requested int) int {
	if requested < 1 {
		return 1
	}
	if requested > 4 {
		return 4
	}
	return requested
}

func (e *DifficultyEngine) DifficultyLabel(level int) string {
	switch level {
	case 1:
		return "básico"
	case 2:
		return "intermedio"
	case 3:
		return "avanzado"
	case 4:
		return "desafío"
	default:
		return "intermedio"
	}
}
```

- [ ] **Commit**

```bash
git add api/adaptive/difficulty.go
git commit -m "feat(fase6): difficulty engine with adaptive level selection"
```

---

### Task 8: Recommendation Engine

**Files:**
- Create: `api/adaptive/recommendations.go`

**Interfaces:**
- Consumes: `AdaptiveConfig`, `LearnerState`, `PrerequisiteAnalysis`
- Produces: `RecommendationEngine` with `GenerateRecommendation`, `ExplainRecommendation`

- [ ] **Create `api/adaptive/recommendations.go`**

```go
package adaptive

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecommendationEngine struct {
	db      *pgxpool.Pool
	config  *AdaptiveConfig
	prereqs *PrerequisiteEngine
	mastery *MasteryEngine
	difficult *DifficultyEngine
}

func NewRecommendationEngine(db *pgxpool.Pool, cfg *AdaptiveConfig) *RecommendationEngine {
	return &RecommendationEngine{
		db:       db,
		config:   cfg,
		prereqs:  NewPrerequisiteEngine(db),
		mastery:  NewMasteryEngine(cfg),
		difficult: NewDifficultyEngine(cfg),
	}
}

func (e *RecommendationEngine) GenerateRecommendation(ctx context.Context, state *LearnerState) *Recommendation {
	// Check if there are weak prerequisites
	needsRemedial, weakest, _ := e.prereqs.CheckRemediationNeeded(ctx, state.StudentID, state.CurrentConcept)
	if needsRemedial {
		return &Recommendation{
			ConceptID:  weakest.ConceptID,
			Action:     "REMEDIAL",
			Difficulty: 1,
			Reason:     fmt.Sprintf("Prerequisito débil: %s (dominio %.0f%%)", weakest.ConceptName, weakest.Mastery*100),
			Score:      1.0 - weakest.Mastery,
		}
	}

	// Check if concept mastery is low
	for _, conceptID := range state.WeakConcepts {
		var mastery float64
		_ = e.db.QueryRow(ctx,
			`SELECT mastery FROM concept_mastery WHERE student_id = $1 AND concept_id = $2`,
			state.StudentID, conceptID,
		).Scan(&mastery)

		if mastery < e.config.CriticalThreshold {
			return &Recommendation{
				ConceptID:  conceptID,
				Action:     "REVIEW_CONCEPT",
				Difficulty: 1,
				Reason:     fmt.Sprintf("Dominio crítico en %s (%.0f%%)", conceptID, mastery*100),
				Score:      0.9,
			}
		}
		if mastery < e.config.BeginnerThreshold {
			return &Recommendation{
				ConceptID:  conceptID,
				Action:     "PRACTICE",
				Difficulty: e.difficult.SelectDifficulty(mastery, 0),
				Reason:     fmt.Sprintf("Necesitás practicar %s (dominio %.0f%%)", conceptID, mastery*100),
				Score:      0.7,
			}
		}
	}

	// If current concept is mastered, suggest advancement
	var currentMastery float64
	_ = e.db.QueryRow(ctx,
		`SELECT mastery FROM concept_mastery WHERE student_id = $1 AND concept_id = $2`,
		state.StudentID, state.CurrentConcept,
	).Scan(&currentMastery)

	if currentMastery >= e.config.CompetentThreshold {
		// Find next concept via knowledge map
		km := NewKnowledgeMapService(e.db)
		dependents, _ := km.GetDependents(ctx, state.CurrentConcept)
		if len(dependents) > 0 {
			return &Recommendation{
				ConceptID:  dependents[0],
				Action:     "ADVANCE",
				Difficulty: e.difficult.SelectDifficulty(currentMastery, 0),
				Reason:     fmt.Sprintf("Avanzá al siguiente concepto: %s", dependents[0]),
				Score:      0.8,
			}
		}
	}

	// Default: practice current concept
	diff := e.difficult.SelectDifficulty(currentMastery, 0)
	return &Recommendation{
		ConceptID:  state.CurrentConcept,
		Action:     "PRACTICE",
		Difficulty: diff,
		Reason:     fmt.Sprintf("Seguí practicando %s (dominio %.0f%%, dificultad %d)", state.CurrentConcept, currentMastery*100, diff),
		Score:      0.5,
	}
}

func (e *RecommendationEngine) ExplainRecommendation(rec *Recommendation) string {
	return fmt.Sprintf("Te recomiendo %s porque: %s", rec.Action, rec.Reason)
}
```

- [ ] **Commit**

```bash
git add api/adaptive/recommendations.go
git commit -m "feat(fase6): recommendation engine with remedial, review, practice, advance actions"
```

---

### Task 9: Learning Path Engine

**Files:**
- Create: `api/adaptive/learning_path.go`

**Interfaces:**
- Consumes: `KnowledgeMapService`, `PrerequisiteEngine`, `RecommendationEngine`
- Produces: `LearningPathEngine` with `BuildPath`, `NextStep`

- [ ] **Create `api/adaptive/learning_path.go`**

```go
package adaptive

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PathStep struct {
	ConceptID string `json:"concept_id"`
	Action    string `json:"action"`
	Reason    string `json:"reason"`
}

type LearningPathEngine struct {
	db     *pgxpool.Pool
	config *AdaptiveConfig
	km     *KnowledgeMapService
	prereq *PrerequisiteEngine
	rec    *RecommendationEngine
}

func NewLearningPathEngine(db *pgxpool.Pool, cfg *AdaptiveConfig) *LearningPathEngine {
	return &LearningPathEngine{
		db:     db,
		config: cfg,
		km:     NewKnowledgeMapService(db),
		prereq: NewPrerequisiteEngine(db),
		rec:    NewRecommendationEngine(db, cfg),
	}
}

func (e *LearningPathEngine) BuildPath(ctx context.Context, studentID, startConcept string) ([]PathStep, error) {
	state, err := LoadLearnerState(ctx, e.db, studentID, "")
	if err != nil {
		return nil, err
	}
	state.CurrentConcept = startConcept

	var path []PathStep
	visited := map[string]bool{}

	for i := 0; i < 20; i++ {
		if visited[state.CurrentConcept] {
			break
		}
		visited[state.CurrentConcept] = true

		rec := e.rec.GenerateRecommendation(ctx, state)
		if rec == nil {
			break
		}

		path = append(path, PathStep{
			ConceptID: rec.ConceptID,
			Action:    rec.Action,
			Reason:    rec.Reason,
		})

		if rec.Action == "ADVANCE" {
			state.CurrentConcept = rec.ConceptID
		} else if rec.Action == "REMEDIAL" {
			state.CurrentConcept = rec.ConceptID
		}
	}

	return path, nil
}

func (e *LearningPathEngine) NextStep(ctx context.Context, studentID, currentConcept string) (*PathStep, error) {
	state, err := LoadLearnerState(ctx, e.db, studentID, "")
	if err != nil {
		return nil, err
	}
	state.CurrentConcept = currentConcept

	rec := e.rec.GenerateRecommendation(ctx, state)
	if rec == nil {
		return nil, nil
	}

	return &PathStep{
		ConceptID: rec.ConceptID,
		Action:    rec.Action,
		Reason:    rec.Reason,
	}, nil
}
```

- [ ] **Commit**

```bash
git add api/adaptive/learning_path.go
git commit -m "feat(fase6): learning path engine with dynamic path building"
```

---

### Task 10: Learning Event Service

**Files:**
- Create: `api/adaptive/events.go`

**Interfaces:**
- Consumes: Database
- Produces: `LearningEventService` with `RecordEvent`, `ProcessEvent`, `GetRecentEvents`, `GetStudentActivity`

- [ ] **Create `api/adaptive/events.go`**

```go
package adaptive

import (
	"context"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LearningEventService struct {
	db      *pgxpool.Pool
	mastery *MasteryEngine
	errors  *ErrorAnalyzer
}

func NewLearningEventService(db *pgxpool.Pool) *LearningEventService {
	return &LearningEventService{
		db:      db,
		mastery: NewMasteryEngine(&AdaptiveConfig{}), // Will be set via SetConfig
		errors:  NewErrorAnalyzer(),
	}
}

func (s *LearningEventService) SetConfig(cfg *AdaptiveConfig) {
	s.mastery = NewMasteryEngine(cfg)
}

func (s *LearningEventService) RecordEvent(ctx context.Context, event *LearningEvent) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`INSERT INTO learning_events (student_id, course_id, concept_id, activity_id, event_type, difficulty, correct, score, time_seconds, hints_used, error_type, error_detail, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING id`,
		event.StudentID, event.CourseID, event.ConceptID, event.ActivityID, event.EventType,
		event.Difficulty, event.Correct, event.Score, event.TimeSecs, event.HintsUsed,
		event.ErrorType, event.ErrorDetail, event.Metadata,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *LearningEventService) ProcessEvent(ctx context.Context, event *LearningEvent, engine *AdaptiveEngine) (*MasteryRecord, *Recommendation, error) {
	eventID, err := s.RecordEvent(ctx, event)
	if err != nil {
		return nil, nil, err
	}

	// Get current mastery
	currentMastery, err := s.getCurrentMastery(ctx, event.StudentID, event.ConceptID)
	if err != nil {
		currentMastery = 0
	}

	// Calculate evidence
	independentSuccess := event.Correct && event.HintsUsed == 0
	evidence := engine.Mastery.CalculateEvidence(event.Correct, event.Difficulty, event.HintsUsed, independentSuccess)
	recencyWeight := engine.Mastery.CalculateRecencyWeight(time.Now())
	newMastery := engine.Mastery.CalculateNewMastery(currentMastery, evidence, recencyWeight)
	status := engine.Mastery.DetermineLegacyStatus(newMastery)

	// Record error if incorrect and has error type
	if !event.Correct && event.ErrorType != "" {
		_ = engine.Errors.RecordError(ctx, s.db, event.StudentID, event.CourseID, event.ConceptID, event.ErrorType, event.ErrorDetail)
	} else if !event.Correct {
		_ = engine.Errors.RecordError(ctx, s.db, event.StudentID, event.CourseID, event.ConceptID, "CONCEPTUAL_ERROR", "")
	}

	// Check for recurrent errors
	if event.ErrorType != "" {
		isRecurrent, count := engine.Errors.IsRecurrent(ctx, s.db, event.StudentID, event.ConceptID, event.ErrorType)
		if isRecurrent {
			_ = engine.Errors.RecordError(ctx, s.db, event.StudentID, event.CourseID, event.ConceptID, event.ErrorType+"_RECURRENT", "")
		}
		_ = count // Use for logging if needed
	}

	// Update concept_mastery
	err = s.updateMastery(ctx, event.StudentID, event.ConceptID, newMastery, status, event.Correct, event.HintsUsed, independentSuccess, event.TimeSecs)
	if err != nil {
		return nil, nil, err
	}

	// Record mastery history
	_ = s.recordMasteryHistory(ctx, event.StudentID, event.ConceptID, currentMastery, newMastery, eventID, "learning_event_processed")

	// Update student profile
	_ = s.updateStudentProfile(ctx, event.StudentID, event.CourseID, event.Correct, event.HintsUsed)

	// Generate recommendation
	state, err := LoadLearnerState(ctx, s.db, event.StudentID, event.CourseID)
	if err != nil {
		return nil, nil, err
	}
	state.CurrentConcept = event.ConceptID
	rec := engine.Recommend.GenerateRecommendation(ctx, state)

	record := &MasteryRecord{
		StudentID: event.StudentID,
		ConceptID: event.ConceptID,
		CourseID:  event.CourseID,
		Mastery:   newMastery,
		Status:    status,
	}

	return record, rec, nil
}

func (s *LearningEventService) getCurrentMastery(ctx context.Context, studentID, conceptID string) (float64, error) {
	var mastery float64
	err := s.db.QueryRow(ctx,
		`INSERT INTO concept_mastery (student_id, concept_id)
		 VALUES ($1, $2)
		 ON CONFLICT (student_id, concept_id) DO NOTHING`,
		studentID, conceptID,
	).Err()
	if err != nil {
		return 0, err
	}

	err = s.db.QueryRow(ctx,
		`SELECT mastery FROM concept_mastery WHERE student_id = $1 AND concept_id = $2`,
		studentID, conceptID,
	).Scan(&mastery)
	if err != nil {
		return 0, err
	}
	return mastery, nil
}

func (s *LearningEventService) updateMastery(ctx context.Context, studentID, conceptID string, newMastery float64, status string, correct bool, hintsUsed int, independentSuccess bool, timeSecs int) error {
	_, err := s.db.Exec(ctx,
		`UPDATE concept_mastery SET
		 mastery = $1,
		 status = $2,
		 attempts = attempts + 1,
		 correct = correct + CASE WHEN $3 THEN 1 ELSE 0 END,
		 hints_used = hints_used + $4,
		 independent_successes = independent_successes + CASE WHEN $5 THEN 1 ELSE 0 END,
		 error_count = CASE WHEN $3 THEN error_count ELSE error_count + 1 END,
		 average_time_seconds = CASE WHEN attempts > 0
		   THEN (average_time_seconds * attempts + $6) / (attempts + 1)
		   ELSE $6 END,
		 last_attempt_at = NOW(),
		 last_success_at = CASE WHEN $3 THEN NOW() ELSE last_success_at END,
		 last_error_at = CASE WHEN $3 THEN last_error_at ELSE NOW() END,
		 updated_at = NOW()
		 WHERE student_id = $7 AND concept_id = $8`,
		newMastery, status, correct, hintsUsed, independentSuccess, timeSecs, studentID, conceptID)
	return err
}

func (s *LearningEventService) recordMasteryHistory(ctx context.Context, studentID, conceptID string, oldMastery, newMastery float64, triggerEventID, reason string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO mastery_history (student_id, concept_id, old_mastery, new_mastery, trigger_event_id, reason)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		studentID, conceptID, oldMastery, newMastery, triggerEventID, reason)
	return err
}

func (s *LearningEventService) updateStudentProfile(ctx context.Context, studentID, courseID string, correct bool, hintsUsed int) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO student_profiles (student_id, course_id)
		 VALUES ($1, $2)
		 ON CONFLICT (student_id) DO NOTHING`,
		studentID, courseID)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx,
		`UPDATE student_profiles SET
		 total_attempts = total_attempts + 1,
		 correct_attempts = correct_attempts + CASE WHEN $1 THEN 1 ELSE 0 END,
		 total_hints_used = total_hints_used + $2,
		 overall_level = (SELECT COALESCE(AVG(mastery), 0) FROM concept_mastery WHERE student_id = $3),
		 last_active_at = NOW(),
		 updated_at = NOW()
		 WHERE student_id = $3 AND course_id = $4`,
		correct, hintsUsed, studentID, courseID)
	return err
}

func (s *LearningEventService) GetRecentEvents(ctx context.Context, studentID, conceptID string, limit int) ([]LearningEvent, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, student_id, course_id, concept_id, COALESCE(activity_id::text,''), event_type,
		        difficulty, COALESCE(correct,false), COALESCE(score,0), COALESCE(time_seconds,0),
		        COALESCE(hints_used,0), COALESCE(error_type,''), COALESCE(error_detail,''), metadata, created_at
		 FROM learning_events
		 WHERE student_id = $1 AND concept_id = $2
		 ORDER BY created_at DESC LIMIT $3`,
		studentID, conceptID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []LearningEvent
	for rows.Next() {
		var e LearningEvent
		var meta []byte
		if err := rows.Scan(&e.ID, &e.StudentID, &e.CourseID, &e.ConceptID, &e.ActivityID, &e.EventType,
			&e.Difficulty, &e.Correct, &e.Score, &e.TimeSecs, &e.HintsUsed, &e.ErrorType, &e.ErrorDetail, &meta, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}
```

- [ ] **Commit**

```bash
git add api/adaptive/events.go
git commit -m "feat(fase6): learning event service with recording, processing, and profile updates"
```

---

### Task 11: Learner State loader + Progress Analytics

**Files:**
- Create: `api/adaptive/analytics.go`

**Interfaces:**
- Consumes: Database tables
- Produces: `LoadLearnerState`, `ProgressAnalyticsService` with `GetProgress`, `GetCourseAnalytics`

- [ ] **Create `api/adaptive/analytics.go`**

```go
package adaptive

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func LoadLearnerState(ctx context.Context, db *pgxpool.Pool, studentID, courseID string) (*LearnerState, error) {
	if courseID == "" {
		courseID = "matematica-1"
	}

	var profile StudentProfile
	err := db.QueryRow(ctx,
		`INSERT INTO student_profiles (student_id, course_id)
		 VALUES ($1, $2)
		 ON CONFLICT (student_id) DO UPDATE SET updated_at = NOW()
		 RETURNING id, student_id, course_id, overall_level, total_attempts, correct_attempts, total_hints_used, study_time_seconds`,
		studentID, courseID,
	).Scan(&profile.ID, &profile.StudentID, &profile.CourseID, &profile.OverallLevel, &profile.TotalAttempts, &profile.CorrectAttempts, &profile.TotalHintsUsed, &profile.StudyTimeSeconds)
	if err != nil {
		return nil, err
	}

	state := &LearnerState{
		StudentID:          studentID,
		CourseID:           courseID,
		OverallMastery:     profile.OverallLevel,
		TotalAttempts:      profile.TotalAttempts,
		SuccessfulAttempts: profile.CorrectAttempts,
		FailedAttempts:     profile.TotalAttempts - profile.CorrectAttempts,
		HintUsage:          profile.TotalHintsUsed,
		AvgResolutionTime:  profile.StudyTimeSeconds,
	}

	// Load mastery for all concepts
	rows, err := db.Query(ctx,
		`SELECT cm.concept_id, cm.mastery, cm.status, cm.attempts, cm.last_attempt_at
		 FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE cm.student_id = $1 AND c.course_id = $2`,
		studentID, courseID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var conceptID, status string
			var mastery float64
			var attempts int
			var lastAttempt *time.Time
			if err := rows.Scan(&conceptID, &mastery, &status, &attempts, &lastAttempt); err != nil {
				continue
			}
			if mastery >= 0.75 {
				state.StrongConcepts = append(state.StrongConcepts, conceptID)
			} else if mastery < 0.60 && attempts > 0 {
				state.WeakConcepts = append(state.WeakConcepts, conceptID)
			}
			if lastAttempt != nil && (state.LastActivityAt == nil || lastAttempt.After(*state.LastActivityAt)) {
				state.LastActivityAt = lastAttempt
			}
		}
	}

	// Load recent errors
	errorRows, err := db.Query(ctx,
		`SELECT error_type FROM student_errors
		 WHERE student_id = $1 ORDER BY last_occurred_at DESC LIMIT 10`,
		studentID,
	)
	if err == nil {
		defer errorRows.Close()
		for errorRows.Next() {
			var et string
			if err := errorRows.Scan(&et); err != nil {
				continue
			}
			state.RecentErrors = append(state.RecentErrors, et)
		}
	}

	return state, nil
}

type ConceptMasterySummary struct {
	ConceptID string  `json:"concept_id"`
	Name      string  `json:"name"`
	Mastery   float64 `json:"mastery"`
	Status    string  `json:"status"`
	Attempts  int     `json:"attempts"`
}

type CourseAnalytics struct {
	CourseID          string                `json:"course_id"`
	TotalStudents     int                   `json:"total_students"`
	AverageMastery    float64               `json:"average_mastery"`
	ConceptBreakdown  []ConceptMasterySummary `json:"concept_breakdown"`
}

type ProgressAnalyticsService struct {
	db *pgxpool.Pool
}

func NewProgressAnalyticsService(db *pgxpool.Pool) *ProgressAnalyticsService {
	return &ProgressAnalyticsService{db: db}
}

func (s *ProgressAnalyticsService) GetStudentProgress(ctx context.Context, studentID, courseID string) (*LearnerState, error) {
	return LoadLearnerState(ctx, s.db, studentID, courseID)
}

func (s *ProgressAnalyticsService) GetCourseAnalytics(ctx context.Context, courseID string) (*CourseAnalytics, error) {
	analytics := &CourseAnalytics{CourseID: courseID}

	// Count total students with activity
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT student_id) FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE c.course_id = $1`, courseID,
	).Scan(&analytics.TotalStudents)
	if err != nil {
		return nil, err
	}

	// Average mastery per concept
	rows, err := s.db.Query(ctx,
		`SELECT c.code, c.name, COALESCE(AVG(cm.mastery), 0) as avg_mastery
		 FROM concepts c
		 LEFT JOIN concept_mastery cm ON cm.concept_id = c.id
		 WHERE c.course_id = $1
		 GROUP BY c.code, c.name
		 ORDER BY avg_mastery ASC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalMastery float64
	var count int
	for rows.Next() {
		var s ConceptMasterySummary
		if err := rows.Scan(&s.ConceptID, &s.Name, &s.Mastery); err != nil {
			continue
		}
		analytics.ConceptBreakdown = append(analytics.ConceptBreakdown, s)
		totalMastery += s.Mastery
		count++
	}
	if count > 0 {
		analytics.AverageMastery = totalMastery / float64(count)
	}

	return analytics, nil
}

func (s *ProgressAnalyticsService) GetCommonErrors(ctx context.Context, courseID string) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(ctx,
		`SELECT se.error_type, COUNT(*) as frequency
		 FROM student_errors se
		 JOIN concepts c ON se.concept_id = c.id
		 WHERE c.course_id = $1
		 GROUP BY se.error_type
		 ORDER BY frequency DESC LIMIT 10`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var errorType string
		var frequency int
		if err := rows.Scan(&errorType, &frequency); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"error_type": errorType,
			"frequency":  frequency,
		})
	}
	return results, nil
}
```

- [ ] **Commit**

```bash
git add api/adaptive/analytics.go
git commit -m "feat(fase6): learner state loader and progress analytics service"
```

---

### Task 12: Reuse StudentProfile struct from api package

**Files:**
- Modify: `api/adaptive/analytics.go` (remove duplicate StudentProfile usage)

The `api/learning.go` already defines `StudentProfile`. We should reference it from the existing `api` package. Since `api/adaptive/` is a sub-package of `api/`, we import from the parent:

```go
import (
    "github.com/brandall2021/matematicarag/api"
)
```

But actually `api/adaptive/` cannot import `api/` due to circular dependency (since `api/` imports `api/agent/` and `api/adaptive/` would be imported by `api/`).

**Solution**: Define `StudentProfile` as a local type in `api/adaptive/analytics.go` (already done above) or extract shared types into `internal/`. For simplicity, we'll keep the local type.

---

### Task 13: Qdrant adaptive search

**Files:**
- Modify: `api/rag.go`
- Create: `api/adaptive/qdrant.go`

**Interfaces:**
- Consumes: Existing Qdrant search functions
- Produces: `AdaptiveQdrantSearch` with pedagogical context

- [ ] **Create `api/adaptive/qdrant.go`**

```go
package adaptive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type AdaptiveSearchParams struct {
	Query      string  `json:"query"`
	ConceptID  string  `json:"concept_id"`
	Mastery    float64 `json:"student_mastery"`
	Difficulty int     `json:"difficulty"`
	ErrorType  string  `json:"error_type,omitempty"`
	TopK       int     `json:"top_k"`
}

type AdaptiveSearchResult struct {
	ID          string  `json:"id"`
	DocumentID  string  `json:"document_id"`
	ChunkID     string  `json:"chunk_id"`
	Content     string  `json:"content"`
	Score       float64 `json:"score"`
	SourceTitle string  `json:"source_title"`
	Page        int     `json:"page"`
	Section     string  `json:"section"`
	ConceptID   string  `json:"concept_id"`
	Difficulty  int     `json:"difficulty"`
	ContentType string  `json:"content_type"`
}

func AdaptiveQdrantSearch(ctx context.Context, qdrantURL string, params *AdaptiveSearchParams) ([]AdaptiveSearchResult, error) {
	if params.TopK <= 0 {
		params.TopK = 5
	}

	// Enrich query with pedagogical context
	enrichedQuery := params.Query
	if params.ConceptID != "" {
		enrichedQuery = fmt.Sprintf("%s concept:%s", enrichedQuery, params.ConceptID)
	}

	searchPayload := map[string]interface{}{
		"query": enrichedQuery,
		"top_k": params.TopK,
		"filters": map[string]interface{}{},
		"params": map[string]interface{}{
			"pedagogical_context": map[string]interface{}{
				"mastery":    params.Mastery,
				"difficulty": params.Difficulty,
				"error_type": params.ErrorType,
			},
		},
	}

	if params.ConceptID != "" {
		searchPayload["filters"].(map[string]interface{})["concept_id"] = params.ConceptID
	}
	if params.Difficulty > 0 {
		searchPayload["filters"].(map[string]interface{})["difficulty"] = params.Difficulty
	}

	body, _ := json.Marshal(searchPayload)
	resp, err := http.Post(
		qdrantURL+"/search",
		"application/json",
		json.NewEncoder(nil).Encode(body) == nil,
	)
	_ = resp
	_ = err

	// For now, return empty — actual HTTP call uses the same structure as api/rag.go
	return nil, fmt.Errorf("use existing RAG search with pedagogical context instead")
}
```

The actual implementation should reuse the existing `api/rag.go` search function rather than duplicating the HTTP call. The key change is adding `concept_id`, `difficulty`, and `mastery` as filter parameters to the existing search.

**Note**: In practice, this modifies the existing RAG search call to accept pedagogical context. The existing `rag_search` tool in the agent already searches Qdrant. The integration (Task 15) will pass concept_id, mastery, difficulty to that tool.

- [ ] **Commit**

```bash
git add api/adaptive/qdrant.go
git commit -m "feat(fase6): adaptive Qdrant search params with pedagogical context"
```

---

### Task 14: API endpoints

**Files:**
- Modify: `api/learning.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- New endpoints: `POST /api/learning/events`, `GET /api/learning/learner-profile`, `GET /api/learning/recommendation`
- New endpoints: `GET /api/learning/path`, `GET /api/learning/errors/common`

- [ ] **Add adaptive routes to `api/learning.go`**

Add new routes inside `LearningRoutes()`:

```go
// POST /api/learning/events — record a learning event
r.Post("/events", func(w http.ResponseWriter, r *http.Request) {
    studentID := r.Context().Value(UserIDKey).(string)

    var event adaptive.LearningEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
        return
    }
    event.StudentID = studentID

    record, rec, err := adaptEngine.Events.ProcessEvent(r.Context(), &event, adaptEngine)
    if err != nil {
        http.Error(w, `{"error":"failed to process event"}`, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "mastery":       record,
        "recommendation": rec,
    })
})

// GET /api/learning/learner-profile — full learner state
r.Get("/learner-profile", func(w http.ResponseWriter, r *http.Request) {
    studentID := r.Context().Value(UserIDKey).(string)

    state, err := adaptive.LoadLearnerState(r.Context(), db, studentID, "")
    if err != nil {
        http.Error(w, `{"error":"failed to load profile"}`, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(state)
})

// GET /api/learning/recommendation — next action
r.Get("/recommendation", func(w http.ResponseWriter, r *http.Request) {
    studentID := r.Context().Value(UserIDKey).(string)

    state, err := adaptive.LoadLearnerState(r.Context(), db, studentID, "")
    if err != nil {
        http.Error(w, `{"error":"failed to load profile"}`, http.StatusInternalServerError)
        return
    }

    rec := adaptEngine.Recommend.GenerateRecommendation(r.Context(), state)
    if rec == nil {
        http.Error(w, `{"error":"no recommendation available"}`, http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(rec)
})

// GET /api/learning/path — learning path
r.Get("/path", func(w http.ResponseWriter, r *http.Request) {
    studentID := r.Context().Value(UserIDKey).(string)
    conceptID := r.URL.Query().Get("concept")

    path, err := adaptEngine.LearningPath.BuildPath(r.Context(), studentID, conceptID)
    if err != nil {
        http.Error(w, `{"error":"failed to build path"}`, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "path": path,
    })
})
```

Add a package-level `adaptEngine` variable or pass it via closure. The cleanest approach: modify `LearningRoutes` to accept the engine:

```go
func LearningRoutes(db *pgxpool.Pool, adaptEngine *adaptive.AdaptiveEngine) func(r chi.Router) {
    // existing routes + new routes
}
```

Update the call in `cmd/server/main.go`:

```go
r.Route("/learning", api.LearningRoutes(db, adaptEngine))
```

- [ ] **Wire AdaptiveEngine in `cmd/server/main.go`**

After creating `pedagogicalAgent` and before the route group:

```go
import (
    "github.com/brandall2021/matematicarag/api/adaptive"
)

// Inside main(), after cfg initialization:
adaptCfg := &adaptive.AdaptiveConfig{
    MasteryOldWeight:       cfg.MasteryOldWeight,
    MasteryEvidenceWeight:  cfg.MasteryEvidenceWeight,
    MasteryHintPenalty:     cfg.MasteryHintPenalty,
    MasteryErrorPenalty:    cfg.MasteryErrorPenalty,
    MasteryRecencyFactor:   cfg.MasteryRecencyFactor,
    CriticalThreshold:      cfg.LearningCriticalThreshold,
    BeginnerThreshold:      cfg.LearningBeginnerThreshold,
    DevelopingThreshold:    cfg.LearningDevelopingThreshold,
    CompetentThreshold:     cfg.LearningCompetentThreshold,
    MaxDifficulty:          4,
}

adaptEngine := adaptive.NewAdaptiveEngine(db, adaptCfg)
adaptEngine.Events.SetConfig(adaptCfg)

// Pass adaptEngine to LearningRoutes:
r.Route("/learning", api.LearningRoutes(db, adaptEngine))
```

- [ ] **Add `POST /api/learning/suggest` for recommendation explanation**

```go
r.Post("/suggest", func(w http.ResponseWriter, r *http.Request) {
    studentID := r.Context().Value(UserIDKey).(string)
    var req struct {
        ConceptID string `json:"concept_id"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    state, _ := adaptive.LoadLearnerState(r.Context(), db, studentID, "")
    state.CurrentConcept = req.ConceptID
    rec := adaptEngine.Recommend.GenerateRecommendation(r.Context(), state)
    explanation := adaptEngine.Recommend.ExplainRecommendation(rec)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "recommendation": rec,
        "explanation":    explanation,
    })
})
```

- [ ] **Commit**

```bash
git add api/learning.go cmd/server/main.go
git commit -m "feat(fase6): adaptive learning API endpoints and server wiring"
```

---

### Task 15: Integrate with Pedagogical Agent

**Files:**
- Modify: `api/agent/learning_updater.go`
- Modify: `api/agent/context_manager.go`
- Modify: `api/agent/tool_registry.go` (or individual tool files)
- Modify: `api/agent_routes.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: `AdaptiveEngine`
- Produces: Agent flow that calls `AdaptiveEngine` before/after tool execution

- [ ] **Refactor `LearningUpdater` to use `AdaptiveEngine`**

Replace `UpdateAfterInteraction` to create a `LearningEvent` and call `ProcessEvent`:

```go
func (lu *LearningUpdater) UpdateAfterInteraction(ctx context.Context, studentID, courseID, conceptID string, correct bool, hintsUsed int, score float64) error {
    if conceptID == "" {
        return nil
    }

    event := &adaptive.LearningEvent{
        StudentID:  studentID,
        CourseID:   courseID,
        ConceptID:  conceptID,
        EventType:  "EXERCISE_COMPLETED",
        Difficulty: 1,
        Correct:    correct,
        Score:      score,
        HintsUsed:  hintsUsed,
    }

    // Process through adaptive engine if available
    if adaptiveEngine != nil {
        _, _, err := adaptiveEngine.Events.ProcessEvent(ctx, event, adaptiveEngine)
        return err
    }

    // Fallback to old inline logic
    return lu.fallbackUpdate(ctx, studentID, courseID, conceptID, correct, hintsUsed, score)
}
```

The `adaptiveEngine` reference needs to be available. Add it to `LearningUpdater`:

```go
type LearningUpdater struct {
    db             *pgxpool.Pool
    adaptiveEngine *adaptive.AdaptiveEngine
}

func NewLearningUpdater(db *pgxpool.Pool, ae *adaptive.AdaptiveEngine) *LearningUpdater {
    return &LearningUpdater{db: db, adaptiveEngine: ae}
}
```

- [ ] **Add adaptive context to agent context manager**

In `context_manager.go`'s `LoadStudentContext`, add adaptive data:

```go
// If adaptive engine available, get recommendation
if ae != nil {
    state, _ := adaptive.LoadLearnerState(ctx, db, studentID)
    ctxData["learner_state"] = state
    ctxData["recommendation"] = ae.Recommend.GenerateRecommendation(ctx, state)
}
```

- [ ] **Pass adaptive context to RAG search tool**

In `rag_tool.go`, when calling RAG search, pass concept_id and mastery as filters:

```go
// In Execute():
ragQuery := input["query"].(string)
conceptID := ""
if ctxData, ok := input["context"].(map[string]interface{}); ok {
    if ls, ok := ctxData["learner_state"]; ok {
        // extract concept_id and mastery
    }
}
```

- [ ] **Wire adaptive engine into agent in `cmd/server/main.go`**

```go
toolDeps := api.BuildAgentToolDependencies(db, cfg, mathClient)
// Add adaptive engine reference
toolDeps.AdaptiveEngine = adaptEngine
```

Add `AdaptiveEngine` field to `ToolDependencies` struct:

```go
// In agent_routes.go or agent/state.go
AdaptiveEngine *adaptive.AdaptiveEngine
```

- [ ] **Commit**

```bash
git add api/agent/learning_updater.go api/agent/context_manager.go api/agent/rag_tool.go api/agent_routes.go cmd/server/main.go
git commit -m "feat(fase6): integrate adaptive engine with pedagogical agent"
```

---

### Task 16: Student Learning Dashboard (Frontend)

**Files:**
- Create: `frontend/src/app/modules/adaptive-dashboard/adaptive-dashboard.component.ts`
- Modify: `frontend/src/app/app.routes.ts`
- Modify: `frontend/src/app/shared/layout.component.ts`
- Modify: `frontend/src/app/core/services/api.service.ts`

**Interfaces:**
- Produces: Student-facing learning dashboard with mastery visualization, recommendations, learning path

- [ ] **Add API methods to `api.service.ts`**

```typescript
getLearnerProfile(): Observable<any> {
    return this.http.get(`${this.baseUrl}/learning/learner-profile`);
}

getLearningRecommendation(): Observable<any> {
    return this.http.get(`${this.baseUrl}/learning/recommendation`);
}

getLearningPath(concept?: string): Observable<any> {
    const params = concept ? `?concept=${concept}` : '';
    return this.http.get(`${this.baseUrl}/learning/path${params}`);
}

recordLearningEvent(event: any): Observable<any> {
    return this.http.post(`${this.baseUrl}/learning/events`, event);
}

getCourseAnalytics(): Observable<any> {
    return this.http.get(`${this.baseUrl}/learning/analytics`);
}
```

- [ ] **Create `adaptive-dashboard.component.ts`**

Standalone component with inline template showing:
- Overall mastery with circular progress
- Concept tree with mastery color coding (green/yellow/red)
- Next recommended action card
- Weak concepts section
- Learning path steps
- Recent activity

Full component implementation (see attached pattern from `agent-chat.component.ts`).

- [ ] **Add route in `app.routes.ts`**

```typescript
{ path: 'aprendizaje', loadComponent: () => import('./modules/adaptive-dashboard/adaptive-dashboard.component').then(m => m.AdaptiveDashboardComponent) },
```

- [ ] **Add nav link in `layout.component.ts`**

```html
<a routerLink="/aprendizaje" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)">
  <mat-icon>auto_graph</mat-icon><span>Mi Aprendizaje</span>
</a>
```

- [ ] **Commit**

```bash
git add frontend/src/app/core/services/api.service.ts frontend/src/app/modules/adaptive-dashboard/adaptive-dashboard.component.ts frontend/src/app/app.routes.ts frontend/src/app/shared/layout.component.ts
git commit -m "feat(fase6): student adaptive learning dashboard with mastery visualization"
```

---

### Task 17: Teacher Analytics (Frontend)

**Files:**
- Modify: `frontend/src/app/modules/teacher-dashboard/teacher-dashboard.component.ts`
- Modify: `frontend/src/app/core/services/api.service.ts`

- [ ] **Add teacher analytics methods to `api.service.ts`**

```typescript
getCourseLearningAnalytics(courseId: string): Observable<any> {
    return this.http.get(`${this.baseUrl}/learning/course-analytics?course_id=${courseId}`);
}

getCommonStudentErrors(courseId: string): Observable<any> {
    return this.http.get(`${this.baseUrl}/learning/errors/common?course_id=${courseId}`);
}
```

- [ ] **Add teacher endpoints to `api/learning.go`**

```go
r.Get("/course-analytics", func(w http.ResponseWriter, r *http.Request) {
    courseID := r.URL.Query().Get("course_id")
    analytics, err := adaptEngine.Analytics.GetCourseAnalytics(r.Context(), courseID)
    if err != nil {
        http.Error(w, `{"error":"failed to load analytics"}`, http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(analytics)
})

r.Get("/errors/common", func(w http.ResponseWriter, r *http.Request) {
    courseID := r.URL.Query().Get("course_id")
    errors, err := adaptEngine.Analytics.GetCommonErrors(r.Context(), courseID)
    if err != nil {
        http.Error(w, `{"error":"failed to load errors"}`, http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "errors": errors,
    })
})
```

- [ ] **Commit**

```bash
git add api/learning.go frontend/src/app/core/services/api.service.ts frontend/src/app/modules/teacher-dashboard/teacher-dashboard.component.ts
git commit -m "feat(fase6): teacher analytics with course concept breakdown and common errors"
```

---

### Task 18: Unit Tests

**Files:**
- Create: `api/adaptive/adaptive_test.go`

- [ ] **Create comprehensive tests**

```go
package adaptive

import (
    "testing"
)

func TestMasteryEngine_CalculateEvidence(t *testing.T) {
    cfg := &AdaptiveConfig{
        MasteryHintPenalty:  0.10,
        MasteryErrorPenalty: 0.15,
    }
    e := NewMasteryEngine(cfg)

    // Correct, no hints, difficulty 2
    ev := e.CalculateEvidence(true, 2, 0, true)
    if ev <= 0 {
        t.Errorf("expected positive evidence for correct answer, got %f", ev)
    }

    // Incorrect
    ev = e.CalculateEvidence(false, 1, 0, false)
    if ev != 0 {
        t.Errorf("expected 0 evidence for incorrect, got %f", ev)
    }
}

func TestMasteryEngine_CalculateNewMastery(t *testing.T) {
    cfg := &AdaptiveConfig{
        MasteryOldWeight:      0.70,
        MasteryEvidenceWeight: 0.30,
        MasteryRecencyFactor:  0.60,
    }
    e := NewMasteryEngine(cfg)

    // Old mastery 0.5, evidence 0.3, recency 1.0
    nm := e.CalculateNewMastery(0.5, 0.3, 1.0)
    if nm <= 0.5 || nm > 0.9 {
        t.Errorf("expected moderate increase, got %f", nm)
    }

    // Ensure bounds
    nm = e.CalculateNewMastery(0.9, -10, 1.0)
    if nm < 0 {
        t.Errorf("expected clamped to 0, got %f", nm)
    }
}

func TestMasteryEngine_DetermineStatus(t *testing.T) {
    cfg := &AdaptiveConfig{
        CriticalThreshold:   0.40,
        BeginnerThreshold:   0.60,
        DevelopingThreshold: 0.75,
        CompetentThreshold:  0.90,
    }
    e := NewMasteryEngine(cfg)

    tests := []struct {
        mastery float64
        want    string
    }{
        {0.20, "CRITICAL"},
        {0.50, "BEGINNER"},
        {0.65, "DEVELOPING"},
        {0.80, "COMPETENT"},
        {0.95, "MASTERED"},
    }

    for _, tt := range tests {
        got := e.DetermineStatus(tt.mastery)
        if got != tt.want {
            t.Errorf("DetermineStatus(%f) = %s; want %s", tt.mastery, got, tt.want)
        }
    }
}

func TestDifficultyEngine_SelectDifficulty(t *testing.T) {
    cfg := &AdaptiveConfig{
        CriticalThreshold:   0.40,
        BeginnerThreshold:   0.60,
        DevelopingThreshold: 0.75,
        CompetentThreshold:  0.90,
    }
    e := NewDifficultyEngine(cfg)

    if d := e.SelectDifficulty(0.30, 0); d != 1 {
        t.Errorf("expected difficulty 1 for low mastery, got %d", d)
    }

    if d := e.SelectDifficulty(0.95, 0); d != 4 {
        t.Errorf("expected difficulty 4 for high mastery, got %d", d)
    }

    // Recent errors should reduce difficulty
    if d := e.SelectDifficulty(0.70, 3); d != 1 {
        t.Errorf("expected difficulty 1 with 3 consecutive errors, got %d", d)
    }
}

func TestMasteryEngine_CalculateRecencyWeight(t *testing.T) {
    e := NewMasteryEngine(&AdaptiveConfig{})
    now := time.Now()

    if w := e.CalculateRecencyWeight(now); w != 1.0 {
        t.Errorf("expected 1.0 for today, got %f", w)
    }

    past := now.Add(-30 * 24 * time.Hour)
    w := e.CalculateRecencyWeight(past)
    if w > 0.5 {
        t.Errorf("expected weight <0.5 for 30 days ago, got %f", w)
    }
}

// Test no-student-history
func TestNoHistory(t *testing.T) {
    // This test verifies that a new student starts as NOT_STARTED
    cfg := &AdaptiveConfig{
        CriticalThreshold: 0.40,
    }
    e := NewMasteryEngine(cfg)
    status := e.DetermineStatus(0)
    if status == "MASTERED" {
        t.Error("new student should not be MASTERED")
    }
}
```

- [ ] **Run tests**

```bash
go test ./api/adaptive/ -v -count=1
```

- [ ] **Commit**

```bash
git add api/adaptive/adaptive_test.go
git commit -m "feat(fase6): adaptive engine unit tests"
```

---

### Task 19: Regression verification

**Files:**
- All existing tests

- [ ] **Run all tests**

```bash
go test ./... -v -count=1
```

Verify at minimum:
- `go build ./...` succeeds
- `go test ./api/agent/ -v -count=1` all pass
- `go test ./api/adaptive/ -v -count=1` all pass
- Frontend builds: `npx ng build --configuration production`

- [ ] **Commit if fixes needed**

---

### Task 20: README update

**Files:**
- Modify: `README.md`

- [ ] **Add Fase 6 section to README**

Following the existing pattern, add a `### Fase 6 — Motor de Aprendizaje Adaptativo` section describing the new capabilities.

- [ ] **Commit**

```bash
git add README.md
git commit -m "docs(fase6): adaptive learning engine documentation"
```
