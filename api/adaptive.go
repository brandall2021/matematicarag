package api

import (
	"context"
	"math"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdaptiveRecommendation struct {
	RecommendedConcept    string   `json:"recommended_concept"`
	ConceptName           string   `json:"concept_name"`
	RecommendedDifficulty int      `json:"recommended_difficulty"`
	Reason                string   `json:"reason"`
	PrerequisitesMet      bool     `json:"prerequisites_met"`
	MissingPrereqs        []string `json:"missing_prereqs,omitempty"`
}

func RecommendNext(db *pgxpool.Pool, studentID, courseID string) (*AdaptiveRecommendation, error) {
	ctx := context.Background()

	mastery, err := GetMasteryMap(db, studentID, courseID)
	if err != nil {
		return nil, err
	}

	tree, err := GetConceptTree(ctx, db, courseID)
	if err != nil {
		return nil, err
	}

	allConcepts := flattenTree(tree)
	sort.Slice(allConcepts, func(i, j int) bool {
		return allConcepts[i].DifficultyBase < allConcepts[j].DifficultyBase
	})

	for _, concept := range allConcepts {
		cm, exists := mastery[concept.ID]
		if !exists || cm.Status == "not_started" {
			prereqs, _ := GetPrerequisites(db, concept.ID)
			missing := checkPrereqs(mastery, prereqs)
			if len(missing) > 0 {
				continue
			}
			diff := CalculateDifficulty(0, 0, 0, concept.DifficultyBase)
			return &AdaptiveRecommendation{
				RecommendedConcept:    concept.ID,
				ConceptName:           concept.Name,
				RecommendedDifficulty: diff,
				Reason:                "Concepto nuevo recomendado según el plan de estudios.",
				PrerequisitesMet:      true,
			}, nil
		}
	}

	for _, concept := range allConcepts {
		cm, exists := mastery[concept.ID]
		if !exists {
			continue
		}
		if cm.Status == "mastered" && cm.Attempts < 5 {
			continue
		}
		if cm.Mastery < 0.8 {
			diff := CalculateDifficulty(cm.Mastery, cm.ErrorCount, cm.HintsUsed, concept.DifficultyBase)
			reason := buildReason(cm, concept.Name)
			return &AdaptiveRecommendation{
				RecommendedConcept:    concept.ID,
				ConceptName:           concept.Name,
				RecommendedDifficulty: diff,
				Reason:                reason,
				PrerequisitesMet:      true,
			}, nil
		}
	}

	return &AdaptiveRecommendation{
		RecommendedConcept:    allConcepts[0].ID,
		ConceptName:           allConcepts[0].Name,
		RecommendedDifficulty: 1,
		Reason:                "Repaso general — todos los conceptos están dominados.",
		PrerequisitesMet:      true,
	}, nil
}

func CalculateDifficulty(mastery float64, errors, hints int, baseDifficulty int) int {
	diff := float64(baseDifficulty)

	if mastery < 0.4 {
		diff -= 1
	} else if mastery > 0.7 {
		diff += 1
	}

	if errors > 5 {
		diff -= 0.5
	} else if errors > 2 {
		diff -= 0.25
	}

	if hints > 3 {
		diff -= 0.25
	}

	diff = math.Max(1, math.Min(5, diff))
	return int(math.Round(diff))
}

func flattenTree(nodes []ConceptNode) []ConceptNode {
	var result []ConceptNode
	for _, n := range nodes {
		result = append(result, n)
		result = append(result, flattenTree(n.Children)...)
	}
	return result
}

func checkPrereqs(mastery map[string]ConceptMastery, prereqs []string) []string {
	var missing []string
	for _, p := range prereqs {
		cm, exists := mastery[p]
		if !exists || cm.Mastery < 0.3 {
			missing = append(missing, p)
		}
	}
	return missing
}

func buildReason(cm ConceptMastery, name string) string {
	if cm.ErrorCount > 3 {
		return name + ": tienes errores recurrentes. Practiquemos para reforzar."
	}
	if cm.HintsUsed > cm.Attempts {
		return name + ": necesitas apoyo frecuente. Vamos a practicar con ejercicios guiados."
	}
	if cm.Mastery < 0.3 {
		return name + ": estás comenzando. Te recomiendo ejercicios básicos."
	}
	return name + ": estás avanzando. Un poco más de práctica para consolidar."
}