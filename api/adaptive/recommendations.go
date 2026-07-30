package adaptive

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RecommendationEngine struct {
	db     *pgxpool.Pool
	config *AdaptiveConfig
}

func NewRecommendationEngine(db *pgxpool.Pool, cfg *AdaptiveConfig) *RecommendationEngine {
	return &RecommendationEngine{db: db, config: cfg}
}

func (e *RecommendationEngine) GenerateRecommendation(ctx context.Context, studentID, courseID string, state *LearnerState) (*Recommendation, error) {
	weak, err := e.weakConcepts(ctx, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("weak concepts: %w", err)
	}

	if len(weak) > 0 {
		rec := e.buildRemediation(weak[0])
		return rec, nil
	}

	ready, next := e.readyToAdvance(ctx, studentID, courseID, state)
	if ready && next != nil {
		return next, nil
	}

	return e.defaultRecommendation(state), nil
}

func (e *RecommendationEngine) ExplainRecommendation(rec *Recommendation) string {
	var parts []string
	switch rec.Action {
	case "remediate":
		parts = append(parts, fmt.Sprintf("Review %s", rec.ConceptID))
		parts = append(parts, "mastery below threshold")
	case "practice":
		parts = append(parts, fmt.Sprintf("Practice %s at difficulty %d", rec.ConceptID, rec.Difficulty))
		parts = append(parts, "reinforcement recommended")
	case "advance":
		parts = append(parts, fmt.Sprintf("Ready for %s", rec.ConceptID))
		parts = append(parts, "prerequisites met")
	case "review":
		parts = append(parts, fmt.Sprintf("Spaced review of %s", rec.ConceptID))
		parts = append(parts, "retention check due")
	default:
		parts = append(parts, fmt.Sprintf("Continue with %s", rec.ConceptID))
	}
	if rec.Reason != "" {
		parts = append(parts, rec.Reason)
	}
	return strings.Join(parts, ": ") + "."
}

func (e *RecommendationEngine) weakConcepts(ctx context.Context, studentID, courseID string) ([]MasteryRecord, error) {
	rows, err := e.db.Query(ctx,
		`SELECT cm.concept_id, cm.mastery, cm.status, cm.attempts, cm.correct,
		        COALESCE(cm.incorrect, 0), cm.hints_used,
		        COALESCE(cm.independent_successes, 0), COALESCE(cm.average_time_seconds, 0),
		        COALESCE(cm.confidence, 1.0),
		        cm.last_attempt_at, cm.last_success_at, cm.last_error_at,
		        cm.next_review_at, cm.created_at, cm.updated_at, c.course_id
		 FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE cm.student_id = $1 AND c.course_id = $2 AND cm.mastery < $3
		 ORDER BY cm.mastery ASC`, studentID, courseID, e.config.BeginnerThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MasteryRecord
	for rows.Next() {
		var r MasteryRecord
		if err := rows.Scan(
			&r.ConceptID, &r.Mastery, &r.Status, &r.Attempts, &r.Correct,
			&r.Incorrect, &r.HintsUsed, &r.IndependentSuccesses, &r.AvgTimeSecs,
			&r.Confidence, &r.LastAttemptAt, &r.LastSuccessAt, &r.LastErrorAt,
			&r.NextReviewAt, &r.CreatedAt, &r.UpdatedAt, &r.CourseID,
		); err != nil {
			continue
		}
		records = append(records, r)
	}
	if records == nil {
		return []MasteryRecord{}, nil
	}
	return records, nil
}

func (e *RecommendationEngine) buildRemediation(weak MasteryRecord) *Recommendation {
	m := MasteryEngine{config: e.config}
	d := DifficultyEngine{config: e.config}
	diff := d.SelectDifficulty(weak.Mastery, false)

	action := "remediate"
	if weak.Mastery > 0 {
		action = "practice"
	}

	score := 1.0 - weak.Mastery
	status := m.DetermineStatus(weak.Mastery)

	reason := fmt.Sprintf("Current mastery: %.0f%% (%s)", weak.Mastery*100, status)
	if weak.Attempts > 0 {
		reason += fmt.Sprintf(", %d attempts, %d correct", weak.Attempts, weak.Correct)
	}

	return &Recommendation{
		ConceptID:  weak.ConceptID,
		Action:     action,
		Difficulty: diff,
		Reason:     reason,
		Score:      math.Round(score*100) / 100,
	}
}

func (e *RecommendationEngine) readyToAdvance(ctx context.Context, studentID, courseID string, state *LearnerState) (bool, *Recommendation) {
	if state == nil {
		return false, nil
	}

	if state.OverallMastery >= e.config.CompetentThreshold {
		d := DifficultyEngine{config: e.config}
		next := e.nextConcept(ctx, studentID, courseID)
		if next != "" {
			diff := d.SelectDifficulty(state.OverallMastery, true)
			return true, &Recommendation{
				ConceptID:  next,
				Action:     "advance",
				Difficulty: diff,
				Reason:     fmt.Sprintf("Overall mastery: %.0f%%", state.OverallMastery*100),
				Score:      state.OverallMastery,
			}
		}
		return true, &Recommendation{
			ConceptID:  state.CurrentConcept,
			Action:     "review",
			Difficulty: d.SelectDifficulty(state.OverallMastery, true),
			Reason:     "All concepts at competent or above",
			Score:      state.OverallMastery,
		}
	}

	if state.RecentSuccesses > 3 && len(state.RecentErrors) == 0 {
		d := DifficultyEngine{config: e.config}
		return true, &Recommendation{
			ConceptID:  state.CurrentConcept,
			Action:     "advance",
			Difficulty: d.SelectDifficulty(state.OverallMastery, true),
			Reason:     fmt.Sprintf("%d recent successes, no errors", state.RecentSuccesses),
			Score:      state.OverallMastery,
		}
	}

	return false, nil
}

func (e *RecommendationEngine) nextConcept(ctx context.Context, studentID, courseID string) string {
	row := e.db.QueryRow(ctx,
		`SELECT c.code FROM concepts c
		 WHERE c.course_id = $1
		 AND c.id NOT IN (
		   SELECT cm.concept_id FROM concept_mastery cm
		   WHERE cm.student_id = $2 AND cm.mastery >= $3
		 )
		 ORDER BY c.difficulty_base ASC
		 LIMIT 1`, courseID, studentID, e.config.CompetentThreshold)
	var code string
	if err := row.Scan(&code); err != nil {
		return ""
	}
	return code
}

func (e *RecommendationEngine) defaultRecommendation(state *LearnerState) *Recommendation {
	if state == nil {
		return &Recommendation{Action: "practice", Difficulty: 1, Score: 0}
	}

	d := DifficultyEngine{config: e.config}

	var action string
	switch {
	case state.FailedAttempts > state.SuccessfulAttempts && state.HintUsage > 2:
		action = "remediate"
	case state.OverallMastery < e.config.DevelopingThreshold:
		action = "practice"
	default:
		action = "practice"
	}

	diff := d.SelectDifficulty(state.OverallMastery, state.RecentSuccesses > 2)

	return &Recommendation{
		ConceptID:  state.CurrentConcept,
		Action:     action,
		Difficulty: diff,
		Reason:     fmt.Sprintf("Continue at current level (mastery: %.0f%%)", state.OverallMastery*100),
		Score:      state.OverallMastery,
	}
}

func (e *RecommendationEngine) BatchRecommendations(ctx context.Context, studentID, courseID string, state *LearnerState) ([]Recommendation, error) {
	primary, err := e.GenerateRecommendation(ctx, studentID, courseID, state)
	if err != nil {
		return nil, err
	}

	recs := []Recommendation{*primary}
	seen := map[string]bool{primary.ConceptID: true}

	weak, err := e.weakConcepts(ctx, studentID, courseID)
	if err == nil {
		for _, w := range weak {
			if len(recs) >= 5 {
				break
			}
			if seen[w.ConceptID] {
				continue
			}
			recs = append(recs, *e.buildRemediation(w))
			seen[w.ConceptID] = true
		}
	}

	if state != nil && !seen[state.CurrentConcept] && len(recs) < 5 {
		d := DifficultyEngine{config: e.config}
		recs = append(recs, Recommendation{
			ConceptID:  state.CurrentConcept,
			Action:     "practice",
			Difficulty: d.SelectDifficulty(state.OverallMastery, true),
			Reason:     "Continue building mastery",
			Score:      state.OverallMastery,
		})
	}

	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Score > recs[j].Score
	})

	return recs, nil
}
