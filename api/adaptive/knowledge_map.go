package adaptive

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConceptNode struct {
	ID             string        `json:"id"`
	Code           string        `json:"code"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	ParentID       *string       `json:"parent_id"`
	DifficultyBase int           `json:"difficulty_base"`
	CourseID       string        `json:"course_id"`
	Children       []*ConceptNode `json:"children,omitempty"`
}

type PrerequisiteRelation struct {
	ConceptID        string  `json:"concept_id"`
	PrerequisiteID   string  `json:"prerequisite_concept_id"`
	PrerequisiteName string  `json:"prerequisite_name"`
	Weight           float64 `json:"weight"`
}

type KnowledgeMapService struct {
	db *pgxpool.Pool
}

func NewKnowledgeMapService(db *pgxpool.Pool) *KnowledgeMapService {
	return &KnowledgeMapService{db: db}
}

func (s *KnowledgeMapService) GetConcept(ctx context.Context, code string) (*ConceptNode, error) {
	var node ConceptNode
	err := s.db.QueryRow(ctx,
		`SELECT id, code, name, COALESCE(description,''), parent_id, difficulty_base, course_id
		 FROM concepts WHERE code = $1`, code,
	).Scan(&node.ID, &node.Code, &node.Name, &node.Description, &node.ParentID, &node.DifficultyBase, &node.CourseID)
	if err != nil {
		return nil, fmt.Errorf("concept %s not found: %w", code, err)
	}
	return &node, nil
}

func (s *KnowledgeMapService) GetChildren(ctx context.Context, parentCode string) ([]*ConceptNode, error) {
	parent, err := s.GetConcept(ctx, parentCode)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT id, code, name, COALESCE(description,''), parent_id, difficulty_base, course_id
		 FROM concepts WHERE parent_id = $1`, parent.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var children []*ConceptNode
	for rows.Next() {
		var n ConceptNode
		if err := rows.Scan(&n.ID, &n.Code, &n.Name, &n.Description, &n.ParentID, &n.DifficultyBase, &n.CourseID); err != nil {
			return nil, err
		}
		children = append(children, &n)
	}
	return children, nil
}

func (s *KnowledgeMapService) GetParents(ctx context.Context, conceptCode string) ([]*ConceptNode, error) {
	concept, err := s.GetConcept(ctx, conceptCode)
	if err != nil {
		return nil, err
	}
	if concept.ParentID == nil {
		return nil, nil
	}
	rows, err := s.db.Query(ctx,
		`WITH RECURSIVE ancestors AS (
			SELECT id, code, name, COALESCE(description,'') AS description, parent_id, difficulty_base, course_id, 1 AS depth
			FROM concepts WHERE id = $1
			UNION ALL
			SELECT c.id, c.code, c.name, COALESCE(c.description,''), c.parent_id, c.difficulty_base, c.course_id, a.depth + 1
			FROM concepts c
			JOIN ancestors a ON c.id = a.parent_id
		)
		SELECT id, code, name, description, parent_id, difficulty_base, course_id
		FROM ancestors WHERE depth > 1 ORDER BY depth DESC`,
		*concept.ParentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var parents []*ConceptNode
	for rows.Next() {
		var n ConceptNode
		if err := rows.Scan(&n.ID, &n.Code, &n.Name, &n.Description, &n.ParentID, &n.DifficultyBase, &n.CourseID); err != nil {
			return nil, err
		}
		parents = append(parents, &n)
	}
	return parents, nil
}

func (s *KnowledgeMapService) GetPrerequisites(ctx context.Context, conceptCode string) ([]PrerequisiteRelation, error) {
	concept, err := s.GetConcept(ctx, conceptCode)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT cp.concept_id, cp.prerequisite_id, c.code, cp.weight
		 FROM concept_prerequisites cp
		 JOIN concepts c ON cp.prerequisite_id = c.id
		 WHERE cp.concept_id = $1`, concept.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prereqs []PrerequisiteRelation
	for rows.Next() {
		var p PrerequisiteRelation
		if err := rows.Scan(&p.ConceptID, &p.PrerequisiteID, &p.PrerequisiteName, &p.Weight); err != nil {
			continue
		}
		prereqs = append(prereqs, p)
	}
	return prereqs, nil
}

func (s *KnowledgeMapService) GetDependents(ctx context.Context, conceptCode string) ([]string, error) {
	concept, err := s.GetConcept(ctx, conceptCode)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT c.code FROM concept_prerequisites cp
		 JOIN concepts c ON cp.concept_id = c.id
		 WHERE cp.prerequisite_id = $1`, concept.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deps []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		deps = append(deps, code)
	}
	return deps, nil
}

func (s *KnowledgeMapService) GetRelatedConcepts(ctx context.Context, conceptCode string) ([]*ConceptNode, error) {
	concept, err := s.GetConcept(ctx, conceptCode)
	if err != nil {
		return nil, err
	}
	var query string
	var args []interface{}
	if concept.ParentID != nil {
		query = `SELECT DISTINCT c.id, c.code, c.name, COALESCE(c.description,''), c.parent_id, c.difficulty_base, c.course_id
			 FROM concepts c
			 WHERE (c.parent_id = $1 OR c.parent_id = $2) AND c.id != $2`
		args = []interface{}{*concept.ParentID, concept.ID}
	} else {
		query = `SELECT DISTINCT c.id, c.code, c.name, COALESCE(c.description,''), c.parent_id, c.difficulty_base, c.course_id
			 FROM concepts c
			 WHERE c.parent_id = $1 AND c.id != $1`
		args = []interface{}{concept.ID}
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var related []*ConceptNode
	for rows.Next() {
		var n ConceptNode
		if err := rows.Scan(&n.ID, &n.Code, &n.Name, &n.Description, &n.ParentID, &n.DifficultyBase, &n.CourseID); err != nil {
			continue
		}
		related = append(related, &n)
	}
	return related, nil
}

func (s *KnowledgeMapService) GetLearningSequence(ctx context.Context, rootCode string) ([]string, error) {
	children, err := s.GetChildren(ctx, rootCode)
	if err != nil {
		return nil, err
	}
	var seq []string
	for _, c := range children {
		sub, err := s.GetLearningSequence(ctx, c.Code)
		if err != nil {
			continue
		}
		seq = append(seq, sub...)
	}
	return seq, nil
}
