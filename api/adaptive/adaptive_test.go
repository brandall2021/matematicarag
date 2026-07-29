package adaptive

import (
	"testing"
	"time"
)

func defaultTestConfig() *AdaptiveConfig {
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

func TestNoHistory(t *testing.T) {
	cfg := defaultTestConfig()
	m := NewMasteryEngine(cfg)

	status := m.DetermineStatus(0)
	if status != "CRITICAL" {
		t.Errorf("DetermineStatus(0) = %q; want CRITICAL", status)
	}

	legacy := m.DetermineLegacyStatus(0)
	if legacy != "not_started" {
		t.Errorf("DetermineLegacyStatus(0) = %q; want not_started", legacy)
	}
}

func TestMasteryEngine_CalculateRecencyWeight(t *testing.T) {
	cfg := defaultTestConfig()
	m := NewMasteryEngine(cfg)

	t.Run("today returns 1.0", func(t *testing.T) {
		weight := m.CalculateRecencyWeight(time.Now())
		if weight != 1.0 {
			t.Errorf("CalculateRecencyWeight(now) = %f; want 1.0", weight)
		}
	})

	t.Run("30 days ago returns less than 0.5", func(t *testing.T) {
		old := time.Now().Add(-25 * 24 * time.Hour)
		weight := m.CalculateRecencyWeight(old)
		if weight >= 0.5 {
			t.Errorf("CalculateRecencyWeight(25d ago) = %f; want < 0.5", weight)
		}
		if weight != 0.4 {
			t.Errorf("CalculateRecencyWeight(25d ago) = %f; want 0.4", weight)
		}
	})
}

func TestMasteryEngine_CalculateEvidence(t *testing.T) {
	cfg := defaultTestConfig()
	m := NewMasteryEngine(cfg)

	t.Run("correct independent answer", func(t *testing.T) {
		e := m.CalculateEvidence(true, 3, 0, true)
		if e != 1.0 {
			t.Errorf("CalculateEvidence(true,3,0,true) = %f; want 1.0", e)
		}
	})

	t.Run("incorrect with hints", func(t *testing.T) {
		e := m.CalculateEvidence(false, 1, 2, false)
		if e != 0.0 {
			t.Errorf("CalculateEvidence(false,1,2,false) = %f; want 0.0", e)
		}
	})

	t.Run("correct with many hints reduces evidence", func(t *testing.T) {
		e := m.CalculateEvidence(true, 4, 4, false)
		if e != 0.5 {
			t.Errorf("CalculateEvidence(true,4,4,false) = %f; want 0.5", e)
		}
	})
}

func TestMasteryEngine_CalculateNewMastery(t *testing.T) {
	cfg := defaultTestConfig()
	m := NewMasteryEngine(cfg)

	t.Run("moderate increase", func(t *testing.T) {
		nm := m.CalculateNewMastery(0.4, 0.5, 1.0)
		diff := nm - 0.455
		if diff < -1e-9 || diff > 1e-9 {
			t.Errorf("CalculateNewMastery(0.4,0.5,1.0) = %g; want ~0.455", nm)
		}
	})

	t.Run("clamp to max", func(t *testing.T) {
		nm := m.CalculateNewMastery(0.95, 1.0, 1.0)
		if nm != 1.0 {
			t.Errorf("CalculateNewMastery(0.95,1.0,1.0) = %f; want 1.0", nm)
		}
	})

	t.Run("zero evidence keeps floor", func(t *testing.T) {
		nm := m.CalculateNewMastery(0.0, 0.0, 0.2)
		if nm != 0.0 {
			t.Errorf("CalculateNewMastery(0,0,0.2) = %f; want 0.0", nm)
		}
	})
}

func TestMasteryEngine_DetermineStatus(t *testing.T) {
	cfg := defaultTestConfig()
	m := NewMasteryEngine(cfg)

	tests := []struct {
		mastery float64
		want    string
	}{
		{0.0, "CRITICAL"},
		{0.1, "CRITICAL"},
		{0.2, "BEGINNER"},
		{0.3, "BEGINNER"},
		{0.4, "DEVELOPING"},
		{0.5, "DEVELOPING"},
		{0.6, "COMPETENT"},
		{0.7, "COMPETENT"},
		{0.8, "MASTERED"},
		{0.9, "MASTERED"},
		{1.0, "MASTERED"},
	}

	for _, tt := range tests {
		got := m.DetermineStatus(tt.mastery)
		if got != tt.want {
			t.Errorf("DetermineStatus(%0.1f) = %q; want %q", tt.mastery, got, tt.want)
		}
	}
}

func TestDifficultyEngine_SelectDifficulty(t *testing.T) {
	cfg := defaultTestConfig()
	d := NewDifficultyEngine(cfg)

	t.Run("low mastery bumps to difficulty 2", func(t *testing.T) {
		diff := d.SelectDifficulty(0.1, true)
		if diff != 2 {
			t.Errorf("SelectDifficulty(0.1,true) = %d; want 2", diff)
		}
	})

	t.Run("high mastery selects difficulty 5", func(t *testing.T) {
		diff := d.SelectDifficulty(0.9, true)
		if diff != 5 {
			t.Errorf("SelectDifficulty(0.9,true) = %d; want 5", diff)
		}
	})

	t.Run("recent errors keep difficulty at base level", func(t *testing.T) {
		diff := d.SelectDifficulty(0.9, false)
		if diff != 4 {
			t.Errorf("SelectDifficulty(0.9,false) = %d; want 4", diff)
		}
	})
}
