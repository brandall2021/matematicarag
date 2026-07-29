package adaptive

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LearningPathEngine struct {
	db     *pgxpool.Pool
	config *AdaptiveConfig
}

func NewLearningPathEngine(db *pgxpool.Pool, cfg *AdaptiveConfig) *LearningPathEngine {
	return &LearningPathEngine{db: db, config: cfg}
}

type PathStep struct {
	ConceptID        string  `json:"concept_id"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	Difficulty       int     `json:"difficulty"`
	Mastery          float64 `json:"mastery"`
	Status           string  `json:"status"`
	PrerequisitesMet bool    `json:"prerequisites_met"`
}

type LearningPath struct {
	StudentID string     `json:"student_id"`
	CourseID  string     `json:"course_id"`
	Steps     []PathStep `json:"steps"`
}

func (e *LearningPathEngine) BuildPath(ctx context.Context, studentID, courseID string) (*LearningPath, error) {
	rows, err := e.db.Query(ctx,
		`SELECT c.id, c.code, c.name, c.difficulty_base,
		        COALESCE(cm.mastery, 0), COALESCE(cm.status, 'not_started')
		 FROM concepts c
		 LEFT JOIN concept_mastery cm ON c.id = cm.concept_id AND cm.student_id = $1
		 WHERE c.course_id = $2
		 ORDER BY c.difficulty_base ASC, c.code ASC`,
		studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("build path query: %w", err)
	}
	defer rows.Close()

	var steps []PathStep
	for rows.Next() {
		var s PathStep
		if err := rows.Scan(&s.ConceptID, &s.Code, &s.Name, &s.Difficulty, &s.Mastery, &s.Status); err != nil {
			continue
		}
		s.PrerequisitesMet = e.checkPrerequisitesMet(ctx, studentID, s.ConceptID)
		steps = append(steps, s)
	}
	if steps == nil {
		steps = []PathStep{}
	}

	sort.SliceStable(steps, func(i, j int) bool {
		if steps[i].PrerequisitesMet == steps[j].PrerequisitesMet {
			if steps[i].Difficulty != steps[j].Difficulty {
				return steps[i].Difficulty < steps[j].Difficulty
			}
			return steps[i].Mastery < steps[j].Mastery
		}
		return steps[i].PrerequisitesMet && !steps[j].PrerequisitesMet
	})

	return &LearningPath{
		StudentID: studentID,
		CourseID:  courseID,
		Steps:     steps,
	}, nil
}

func (e *LearningPathEngine) NextStep(ctx context.Context, studentID, courseID string) (*PathStep, error) {
	path, err := e.BuildPath(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}

	for _, s := range path.Steps {
		if s.Mastery < e.config.CompetentThreshold && s.PrerequisitesMet {
			return &s, nil
		}
	}

	if len(path.Steps) > 0 {
		return &path.Steps[len(path.Steps)-1], nil
	}

	return nil, fmt.Errorf("no concepts found for student %s in course %s", studentID, courseID)
}

func (e *LearningPathEngine) checkPrerequisitesMet(ctx context.Context, studentID, conceptID string) bool {
	rows, err := e.db.Query(ctx,
		`SELECT cp.prerequisite_id, COALESCE(cm.mastery, 0)
		 FROM concept_prerequisites cp
		 LEFT JOIN concept_mastery cm ON cp.prerequisite_id = cm.concept_id AND cm.student_id = $1
		 WHERE cp.concept_id = $2`,
		studentID, conceptID)
	if err != nil {
		return true
	}
	defer rows.Close()

	for rows.Next() {
		var prereqID string
		var mastery float64
		if err := rows.Scan(&prereqID, &mastery); err != nil {
			continue
		}
		if mastery < 0.5 {
			return false
		}
	}

	return true
}
