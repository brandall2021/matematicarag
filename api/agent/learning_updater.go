package agent

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LearningUpdater struct {
	db *pgxpool.Pool
}

func NewLearningUpdater(db *pgxpool.Pool) *LearningUpdater {
	return &LearningUpdater{db: db}
}

func (lu *LearningUpdater) UpdateAfterInteraction(ctx context.Context, studentID, courseID, conceptID string, correct bool, hintsUsed int, score float64) error {
	if conceptID == "" {
		return nil
	}

	_, err := lu.db.Exec(ctx,
		`INSERT INTO concept_mastery (student_id, concept_id, mastery, attempts, correct, hints_used, status)
		 VALUES ($1, $2, 0, 0, 0, 0, 'learning')
		 ON CONFLICT DO NOTHING`,
		studentID, conceptID)
	if err != nil {
		return err
	}

	var currentMastery float64
	var currentAttempts, currentCorrect, currentHints int
	err = lu.db.QueryRow(ctx,
		`SELECT mastery, attempts, correct, hints_used FROM concept_mastery
		 WHERE student_id = $1 AND concept_id = $2`,
		studentID, conceptID).Scan(&currentMastery, &currentAttempts, &currentCorrect, &currentHints)
	if err != nil {
		return err
	}

	delta := -0.03
	if correct {
		delta = 0.05 * (1.0 - float64(hintsUsed)*0.1) * score
		if delta < 0.01 {
			delta = 0.01
		}
	}

	newMastery := currentMastery + delta
	if newMastery < 0 {
		newMastery = 0
	}
	if newMastery > 1 {
		newMastery = 1
	}

	status := "learning"
	if newMastery >= 0.8 {
		status = "mastered"
	} else if newMastery >= 0.5 {
		status = "developing"
	}

	_, err = lu.db.Exec(ctx,
		`UPDATE concept_mastery
		 SET mastery = $1, attempts = attempts + 1,
		     correct = CASE WHEN $2 THEN correct + 1 ELSE correct END,
		     hints_used = hints_used + $3,
		     status = $4,
		     last_attempt_at = NOW(),
		     updated_at = NOW()
		 WHERE student_id = $5 AND concept_id = $6`,
		newMastery, correct, hintsUsed, status, studentID, conceptID)
	if err != nil {
		return err
	}

	if !correct {
		_, err = lu.db.Exec(ctx,
			`INSERT INTO student_errors (student_id, concept_id, error_type, error_subtype, count, severity)
			 VALUES ($1, $2, 'incorrect_answer', 'general', 1, 'medium')
			 ON CONFLICT (student_id, concept_id, error_type, error_subtype)
			 DO UPDATE SET count = student_errors.count + 1,
			               last_occurred_at = NOW(),
			               severity = CASE
			                 WHEN student_errors.count >= 5 THEN 'critical'
			                 WHEN student_errors.count >= 3 THEN 'high'
			                 ELSE 'medium'
			               END`,
			studentID, conceptID)
		if err != nil {
			return err
		}
	}

	_, err = lu.db.Exec(ctx,
		`UPDATE student_profiles
		 SET total_attempts = total_attempts + 1,
		     correct_attempts = CASE WHEN $1 THEN correct_attempts + 1 ELSE correct_attempts END,
		     total_hints_used = total_hints_used + $2,
		     updated_at = NOW()
		 WHERE student_id = $3 AND course_id = $4`,
		correct, hintsUsed, studentID, courseID)

	return err
}

func (lu *LearningUpdater) RecordRecommendation(ctx context.Context, studentID, conceptID, message string) error {
	_, err := lu.db.Exec(ctx,
		`INSERT INTO learning_recommendations (student_id, recommendation_type, concept_id, message, priority)
		 VALUES ($1, 'agent_recommendation', $2, $3, 'medium')`,
		studentID, conceptID, message)
	return err
}
