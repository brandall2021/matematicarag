package adaptive

import (
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
		Errors:        NewErrorAnalyzer(db),
		Recommend:     NewRecommendationEngine(db, cfg),
		LearningPath:  NewLearningPathEngine(db, cfg),
		Events:        NewLearningEventService(db),
		Analytics:     NewProgressAnalyticsService(db),
	}
}

// ——— Stub types (TODO: implement in sub-engine tasks) ———

type LearningPathEngine struct{}
type LearningEventService struct{}
type ProgressAnalyticsService struct{}

func NewLearningPathEngine(db *pgxpool.Pool, cfg *AdaptiveConfig) *LearningPathEngine     { return &LearningPathEngine{} }
func NewLearningEventService(db *pgxpool.Pool) *LearningEventService          { return &LearningEventService{} }
func NewProgressAnalyticsService(db *pgxpool.Pool) *ProgressAnalyticsService  { return &ProgressAnalyticsService{} }


