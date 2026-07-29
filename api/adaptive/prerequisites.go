package adaptive

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PrerequisiteEngine struct {
	db *pgxpool.Pool
}

func NewPrerequisiteEngine(db *pgxpool.Pool) *PrerequisiteEngine {
	return &PrerequisiteEngine{db: db}
}

type PrerequisiteAnalysis struct {
	ConceptID         string  `json:"concept_id"`
	ConceptName       string  `json:"concept_name"`
	PrerequisiteID    string  `json:"prerequisite_concept_id"`
	PrerequisiteName  string  `json:"prerequisite_name"`
	Weight            float64 `json:"weight"`
	StudentMastery    float64 `json:"student_mastery"`
	Status            string  `json:"status"`
	IsWeak            bool    `json:"is_weak"`
}

type PrerequisiteGap struct {
	ConceptID       string  `json:"concept_id"`
	ConceptName     string  `json:"concept_name"`
	CurrentMastery  float64 `json:"current_mastery"`
	RequiredMastery float64 `json:"required_mastery"`
}

func (e *PrerequisiteEngine) AnalyzePrerequisites(ctx context.Context, studentID, conceptID string) ([]PrerequisiteAnalysis, error) {
	rows, err := e.db.Query(ctx,
		`SELECT cp.concept_id, c1.name, cp.prerequisite_id, c2.name, cp.weight
		 FROM concept_prerequisites cp
		 JOIN concepts c1 ON cp.concept_id = c1.id
		 JOIN concepts c2 ON cp.prerequisite_id = c2.id
		 WHERE cp.concept_id = $1`, conceptID)
	if err != nil {
		return nil, fmt.Errorf("prerequisites lookup: %w", err)
	}
	defer rows.Close()

	var results []PrerequisiteAnalysis
	for rows.Next() {
		var a PrerequisiteAnalysis
		if err := rows.Scan(&a.ConceptID, &a.ConceptName, &a.PrerequisiteID, &a.PrerequisiteName, &a.Weight); err != nil {
			continue
		}
		cm := &MasteryEngine{}
		var m float64
		e.db.QueryRow(ctx,
			`SELECT COALESCE(mastery, 0) FROM concept_mastery
			 WHERE student_id = $1 AND concept_id = $2`,
			studentID, a.PrerequisiteID).Scan(&m)
		a.StudentMastery = m
		a.Status = cm.DetermineLegacyStatus(m)
		a.IsWeak = m < 0.5
		results = append(results, a)
	}
	if results == nil {
		results = []PrerequisiteAnalysis{}
	}
	return results, nil
}

func (e *PrerequisiteEngine) CheckRemediationNeeded(ctx context.Context, studentID, conceptID string) (bool, []PrerequisiteGap, error) {
	prereqs, err := e.AnalyzePrerequisites(ctx, studentID, conceptID)
	if err != nil {
		return false, nil, err
	}
	var gaps []PrerequisiteGap
	for _, p := range prereqs {
		if p.IsWeak {
			gaps = append(gaps, PrerequisiteGap{
				ConceptID:       p.PrerequisiteID,
				ConceptName:     p.PrerequisiteName,
				CurrentMastery:  p.StudentMastery,
				RequiredMastery: 0.5,
			})
		}
	}
	if gaps == nil {
		gaps = []PrerequisiteGap{}
	}
	return len(gaps) > 0, gaps, nil
}

func (e *PrerequisiteEngine) ExplainWeakness(ctx context.Context, studentID, conceptID string) (string, error) {
	prereqs, err := e.AnalyzePrerequisites(ctx, studentID, conceptID)
	if err != nil {
		return "", err
	}
	var weak []string
	var total float64
	for _, p := range prereqs {
		total += p.StudentMastery
		if p.IsWeak {
			weak = append(weak, p.PrerequisiteName)
		}
	}
	if len(weak) == 0 {
		return "No prerequisite weaknesses detected.", nil
	}
	avg := 0.0
	if len(prereqs) > 0 {
		avg = total / float64(len(prereqs))
	}
	parts := []string{
		fmt.Sprintf("The student has %d prerequisite gap(s)", len(weak)),
		fmt.Sprintf("average prerequisite mastery is %.0f%%", avg*100),
		fmt.Sprintf("weak areas: %s", strings.Join(weak, ", ")),
	}
	return strings.Join(parts, "; ") + ".", nil
}
