package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ContextManager struct {
	db *pgxpool.Pool
}

func NewContextManager(db *pgxpool.Pool) *ContextManager {
	return &ContextManager{db: db}
}

func (cm *ContextManager) LoadStudentContext(ctx context.Context, studentID, courseID string) (*StudentContext, error) {
	sc := &StudentContext{
		StudentID: studentID,
		CourseID:  courseID,
	}

	var overallLevel float64
	var topic string
	err := cm.db.QueryRow(ctx,
		`SELECT overall_level, total_attempts, COALESCE(study_time_seconds, 0)
		 FROM student_profiles WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID).Scan(&overallLevel, &sc.RecentAttempts, &topic)
	if err != nil {
		cm.db.Exec(ctx,
			`INSERT INTO student_profiles (student_id, course_id, overall_level)
			 VALUES ($1, $2, 0) ON CONFLICT DO NOTHING`,
			studentID, courseID)
		sc.Mastery = 0
		sc.CurrentTopic = ""
		return sc, nil
	}
	sc.Mastery = overallLevel
	sc.CurrentTopic = topic

	rows, err := cm.db.Query(ctx,
		`SELECT se.concept_id, se.error_type, se.count
		 FROM student_errors se
		 WHERE se.student_id = $1
		 ORDER BY se.count DESC LIMIT 5`,
		studentID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e StudentErrorInfo
			if err := rows.Scan(&e.Concept, &e.ErrorType, &e.Count); err == nil {
				sc.RecentErrors = append(sc.RecentErrors, e)
			}
		}
	}

	var prefsJSON []byte
	err = cm.db.QueryRow(ctx,
		`SELECT value FROM app_settings WHERE key = 'learning_preferences_' || $1`,
		studentID).Scan(&prefsJSON)
	if err == nil {
		json.Unmarshal(prefsJSON, &sc.LearningPrefs)
	}

	return sc, nil
}

func (cm *ContextManager) LoadSessionContext(ctx context.Context, sessionID string) (*AgentState, error) {
	state := &AgentState{
		SessionID: sessionID,
	}

	var messagesJSON []byte
	err := cm.db.QueryRow(ctx,
		`SELECT messages, student_id, course_id, intent, current_concept, current_exercise_id
		 FROM agent_sessions WHERE session_id = $1`,
		sessionID).Scan(&messagesJSON, &state.StudentID, &state.CourseID,
		&state.Intent, &state.CurrentConcept, &state.CurrentExercise)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	json.Unmarshal(messagesJSON, &state.Messages)
	return state, nil
}

func (cm *ContextManager) SaveSessionState(ctx context.Context, state *AgentState) error {
	messagesJSON, _ := json.Marshal(state.Messages)
	_, err := cm.db.Exec(ctx,
		`INSERT INTO agent_sessions (session_id, student_id, course_id, intent, current_concept, current_exercise_id, messages, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (session_id) DO UPDATE SET
			intent = EXCLUDED.intent,
			current_concept = EXCLUDED.current_concept,
			current_exercise_id = EXCLUDED.current_exercise_id,
			messages = EXCLUDED.messages,
			updated_at = NOW()`,
		state.SessionID, state.StudentID, state.CourseID,
		state.Intent, state.CurrentConcept, state.CurrentExercise, messagesJSON)
	return err
}

func (cm *ContextManager) DetermineStrategy(studentCtx *StudentContext, intent IntentType) PedagogicalStrategy {
	mastery := studentCtx.Mastery

	switch intent {
	case IntentSolveExercise, IntentCheckAnswer, IntentCheckProcedure:
		if mastery < 0.40 {
			return StrategyGuided
		}
		if mastery < 0.70 {
			return StrategyStepByStep
		}
		return StrategyDirect
	case IntentExplainConcept, IntentAskTheory:
		if mastery < 0.40 {
			return StrategyExampleFirst
		}
		if mastery < 0.70 {
			return StrategyGuided
		}
		return StrategyDirect
	case IntentPractice:
		if mastery < 0.40 {
			return StrategyRemedial
		}
		if mastery >= 0.85 {
			return StrategyChallenge
		}
		return StrategyStepByStep
	case IntentGiveHint:
		return StrategySocratic
	default:
		return StrategyDirect
	}
}

func (cm *ContextManager) CalculateDifficulty(studentCtx *StudentContext) int {
	m := studentCtx.Mastery
	switch {
	case m < 0.25:
		return 1
	case m < 0.50:
		return 2
	case m < 0.70:
		return 3
	case m < 0.85:
		return 4
	default:
		return 5
	}
}
