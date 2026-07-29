package adaptive

type DifficultyEngine struct {
	config *AdaptiveConfig
}

func NewDifficultyEngine(cfg *AdaptiveConfig) *DifficultyEngine {
	return &DifficultyEngine{config: cfg}
}

func (de *DifficultyEngine) SelectDifficulty(mastery float64, recentCorrect bool) int {
	cfg := de.config
	var base int
	switch {
	case mastery < cfg.BeginnerThreshold:
		base = 1
	case mastery < cfg.DevelopingThreshold:
		base = 2
	case mastery < cfg.CompetentThreshold:
		base = 3
	default:
		base = 4
	}
	if recentCorrect && base < cfg.MaxDifficulty {
		base++
	}
	return de.ClampDifficulty(base)
}

func (de *DifficultyEngine) ClampDifficulty(difficulty int) int {
	cfg := de.config
	if difficulty < 1 {
		return 1
	}
	if difficulty > cfg.MaxDifficulty {
		return cfg.MaxDifficulty
	}
	return difficulty
}

var difficultyLabels = map[int]string{
	1: "beginner",
	2: "developing",
	3: "competent",
	4: "advanced",
	5: "expert",
}

func (de *DifficultyEngine) DifficultyLabel(difficulty int) string {
	difficulty = de.ClampDifficulty(difficulty)
	if label, ok := difficultyLabels[difficulty]; ok {
		return label
	}
	return "unknown"
}
