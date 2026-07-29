package adaptive

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LearningEventService struct {
	db     *pgxpool.Pool
	config *AdaptiveConfig
}

func NewLearningEventService(db *pgxpool.Pool) *LearningEventService {
	return &LearningEventService{db: db, config: defaultConfig()}
}

func defaultConfig() *AdaptiveConfig {
	return &AdaptiveConfig{
		MasteryOldWeight:      0.7,
		MasteryEvidenceWeight: 0.3,
		MasteryHintPenalty:    0.1,
		MasteryErrorPenalty:   0.2,
		MasteryRecencyFactor:  0.5,
		CriticalThreshold:     0.2,
		BeginnerThreshold:     0.4,
		DevelopingThreshold:   0.6,
		CompetentThreshold:    0.8,
		MaxDifficulty:         5,
	}
}

func (s *LearningEventService) SetConfig(cfg *AdaptiveConfig) {
	s.config = cfg
}

func (s *LearningEventService) RecordEvent(ctx context.Context, event *LearningEvent) (*LearningEvent, error) {
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.EventType == "" {
		event.EventType = "attempt"
	}

	metadataJSON := "{}"
	if event.Metadata != nil {
		m, err := json.Marshal(event.Metadata)
		if err != nil {
			return nil, fmt.Errorf("metadata marshal: %w", err)
		}
		metadataJSON = string(m)
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO learning_events
		 (id, student_id, course_id, concept_id, activity_id, event_type,
		  difficulty, correct, score, time_seconds, hints_used,
		  error_type, error_detail, metadata, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		event.ID, event.StudentID, event.CourseID, event.ConceptID,
		event.ActivityID, event.EventType,
		event.Difficulty, event.Correct, event.Score, event.TimeSecs, event.HintsUsed,
		event.ErrorType, event.ErrorDetail, metadataJSON, event.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert learning event: %w", err)
	}

	return event, nil
}

func (s *LearningEventService) GetRecentEvents(ctx context.Context, studentID, courseID string, limit int) ([]LearningEvent, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, student_id, course_id, concept_id,
		        COALESCE(activity_id, ''), event_type,
		        difficulty, correct, score, time_seconds, hints_used,
		        COALESCE(error_type, ''), COALESCE(error_detail, ''), created_at
		 FROM learning_events
		 WHERE student_id = $1 AND course_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`,
		studentID, courseID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query recent events: %w", err)
	}
	defer rows.Close()

	var events []LearningEvent
	for rows.Next() {
		var e LearningEvent
		if err := rows.Scan(
			&e.ID, &e.StudentID, &e.CourseID, &e.ConceptID,
			&e.ActivityID, &e.EventType,
			&e.Difficulty, &e.Correct, &e.Score, &e.TimeSecs, &e.HintsUsed,
			&e.ErrorType, &e.ErrorDetail, &e.CreatedAt,
		); err != nil {
			log.Printf("[EVENTS] scan row: %v", err)
			continue
		}
		events = append(events, e)
	}
	if events == nil {
		return []LearningEvent{}, nil
	}
	return events, nil
}

func (s *LearningEventService) ProcessEvent(engine *AdaptiveEngine, event *LearningEvent) error {
	ctx := context.Background()

	recorded, err := s.RecordEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("record event: %w", err)
	}

	row := engine.db.QueryRow(ctx,
		`SELECT COALESCE(mastery, 0), COALESCE(attempts, 0), COALESCE(correct, 0),
		        COALESCE(incorrect, 0), COALESCE(hints_used, 0),
		        COALESCE(independent_successes, 0), COALESCE(average_time_seconds, 0),
		        COALESCE(confidence, 0)
		 FROM concept_mastery
		 WHERE student_id = $1 AND concept_id = $2 AND course_id = $3`,
		recorded.StudentID, recorded.ConceptID, recorded.CourseID,
	)
	var (
		oldMastery        float64
		attempts, correct, incorrect, hintsUsedTotal  int
		independentSuccesses, avgTimeSecs int
		confidence        float64
	)
	row.Scan(&oldMastery, &attempts, &correct, &incorrect, &hintsUsedTotal,
		&independentSuccesses, &avgTimeSecs, &confidence)

	independentSuccess := recorded.HintsUsed == 0 && recorded.Correct
	evidence := engine.Mastery.CalculateEvidence(
		recorded.Correct, recorded.Difficulty, recorded.HintsUsed, independentSuccess,
	)

	recencyWeight := engine.Mastery.CalculateRecencyWeight(time.Now())

	newMastery := engine.Mastery.CalculateNewMastery(oldMastery, evidence, recencyWeight)

	status := engine.Mastery.DetermineStatus(newMastery)

	_, err = engine.db.Exec(ctx,
		`INSERT INTO concept_mastery
		 (student_id, concept_id, course_id, mastery, status, attempts, correct, incorrect,
		  hints_used, independent_successes, average_time_seconds, confidence,
		  last_attempt_at, last_success_at, last_error_at, next_review_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,1,0,0,0,0,0,0,NOW(),NULL,NULL,NOW()+INTERVAL '1 day',NOW(),NOW())
		 ON CONFLICT (student_id, concept_id, course_id) DO UPDATE SET
		   mastery = EXCLUDED.mastery,
		   status = EXCLUDED.status,
		   attempts = concept_mastery.attempts + 1,
		   correct = CASE WHEN $6 THEN concept_mastery.correct + 1 ELSE concept_mastery.correct END,
		   incorrect = CASE WHEN NOT $6 THEN concept_mastery.incorrect + 1 ELSE concept_mastery.incorrect END,
		   hints_used = concept_mastery.hints_used + $7,
		   independent_successes = CASE WHEN $8 THEN concept_mastery.independent_successes + 1 ELSE concept_mastery.independent_successes END,
		   average_time_seconds = CASE WHEN concept_mastery.attempts > 0
		     THEN (concept_mastery.average_time_seconds * concept_mastery.attempts + $9) / (concept_mastery.attempts + 1)
		     ELSE $9 END,
		   last_attempt_at = NOW(),
		   last_success_at = CASE WHEN $6 THEN NOW() ELSE concept_mastery.last_success_at END,
		   last_error_at = CASE WHEN NOT $6 THEN NOW() ELSE concept_mastery.last_error_at END,
		   next_review_at = CASE
		     WHEN $6 AND $4 >= $10 THEN NOW() + INTERVAL '7 day'
		     WHEN $6 THEN NOW() + INTERVAL '3 day'
		     ELSE NOW() + INTERVAL '1 day' END,
		   updated_at = NOW()`,
		recorded.StudentID, recorded.ConceptID, recorded.CourseID,
		newMastery, status,
		recorded.Correct, recorded.HintsUsed, independentSuccess, recorded.TimeSecs,
		s.config.CompetentThreshold,
	)
	if err != nil {
		return fmt.Errorf("upsert concept_mastery: %w", err)
	}

	if recorded.ErrorType != "" {
		engine.Errors.RecordError(
			recorded.StudentID, recorded.CourseID,
			recorded.ConceptID, recorded.ErrorType, recorded.ErrorDetail,
		)
	}

	_, err = engine.db.Exec(ctx,
		`INSERT INTO mastery_history
		 (student_id, concept_id, course_id, mastery, status, evidence, event_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())`,
		recorded.StudentID, recorded.ConceptID, recorded.CourseID,
		newMastery, status, evidence, recorded.ID,
	)
	if err != nil {
		log.Printf("[EVENTS] record mastery history: %v", err)
	}

	updateProfileQuery := `
		INSERT INTO student_profiles
		 (student_id, course_id, total_attempts, correct_attempts, incorrect_attempts,
		  study_time_seconds, last_active_at, created_at, updated_at)
		VALUES ($1,$2,1,0,0,$3,NOW(),NOW(),NOW())
		ON CONFLICT (student_id, course_id) DO UPDATE SET
		  total_attempts = student_profiles.total_attempts + 1,
		  correct_attempts = CASE WHEN $4 THEN student_profiles.correct_attempts + 1 ELSE student_profiles.correct_attempts END,
		  incorrect_attempts = CASE WHEN NOT $4 THEN student_profiles.incorrect_attempts + 1 ELSE student_profiles.incorrect_attempts END,
		  study_time_seconds = student_profiles.study_time_seconds + $3,
		  last_active_at = NOW(),
		  updated_at = NOW()`
	_, err = engine.db.Exec(ctx, updateProfileQuery,
		recorded.StudentID, recorded.CourseID,
		recorded.TimeSecs, recorded.Correct,
	)
	if err != nil {
		log.Printf("[EVENTS] update profile: %v", err)
	}

	currentMastery := s.getOverallMastery(ctx, recorded.StudentID, recorded.CourseID)
	state := &LearnerState{
		StudentID:      recorded.StudentID,
		CourseID:       recorded.CourseID,
		OverallMastery: currentMastery,
		CurrentConcept: recorded.ConceptID,
	}
	rec, recErr := engine.Recommend.GenerateRecommendation(ctx, recorded.StudentID, recorded.CourseID, state)
	if recErr == nil && rec != nil {
		_, _ = engine.db.Exec(ctx,
			`INSERT INTO recommendations
			 (student_id, course_id, concept_id, action, difficulty, reason, score, event_id, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
			recorded.StudentID, recorded.CourseID,
			rec.ConceptID, rec.Action, rec.Difficulty, rec.Reason, rec.Score,
			recorded.ID,
		)
	}

	return nil
}

func (s *LearningEventService) getOverallMastery(ctx context.Context, studentID, courseID string) float64 {
	var avg float64
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(AVG(mastery), 0) FROM concept_mastery
		 WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID,
	).Scan(&avg)
	if err != nil {
		return 0
	}
	return avg
}
