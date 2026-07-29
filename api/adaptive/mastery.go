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
		weightedOld = oldMastery * (e.config.MasteryOldWeight * recencyWeight)
	}

	newMastery := weightedOld + (evidence * e.config.MasteryEvidenceWeight)

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
