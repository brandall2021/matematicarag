package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConceptNode struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	ParentID       *string       `json:"parent_id,omitempty"`
	CourseID       string        `json:"course_id"`
	DifficultyBase int           `json:"difficulty_base"`
	Children       []ConceptNode `json:"children,omitempty"`
}

type PrereqInfo struct {
	ConceptID     string   `json:"concept_id"`
	Prerequisites []string `json:"prerequisites"`
}

func KnowledgeRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Get("/courses/{courseID}/concepts", func(w http.ResponseWriter, r *http.Request) {
			courseID := chi.URLParam(r, "courseID")
			tree, err := GetConceptTree(db, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to load concept tree"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tree)
		})

		r.Get("/courses/{courseID}/prerequisites", func(w http.ResponseWriter, r *http.Request) {
			courseID := chi.URLParam(r, "courseID")
			rows, err := db.Query(r.Context(),
				`SELECT c.id, COALESCE(array_agg(cp.prerequisite_id) FILTER (WHERE cp.prerequisite_id IS NOT NULL), '{}') AS prereqs
				 FROM concepts c
				 LEFT JOIN concept_prerequisites cp ON c.id = cp.concept_id
				 WHERE c.course_id = $1
				 GROUP BY c.id`, courseID)
			if err != nil {
				http.Error(w, `{"error":"failed to load prerequisites"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var results []PrereqInfo
			for rows.Next() {
				var p PrereqInfo
				if err := rows.Scan(&p.ConceptID, &p.Prerequisites); err != nil {
					continue
				}
				results = append(results, p)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		})
	}
}

func GetConceptTree(db *pgxpool.Pool, courseID string) ([]ConceptNode, error) {
	rows, err := db.Query(context.Background(),
		`SELECT id, name, description, parent_id, course_id, difficulty_base
		 FROM concepts WHERE course_id = $1 ORDER BY id`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []ConceptNode
	for rows.Next() {
		var n ConceptNode
		if err := rows.Scan(&n.ID, &n.Name, &n.Description, &n.ParentID, &n.CourseID, &n.DifficultyBase); err != nil {
			continue
		}
		all = append(all, n)
	}

	byParent := make(map[string][]ConceptNode)
	var roots []ConceptNode
	for _, n := range all {
		if n.ParentID == nil {
			roots = append(roots, n)
		} else {
			byParent[*n.ParentID] = append(byParent[*n.ParentID], n)
		}
	}

	var buildTree func(nodes []ConceptNode) []ConceptNode
	buildTree = func(nodes []ConceptNode) []ConceptNode {
		for i := range nodes {
			children := byParent[nodes[i].ID]
			if len(children) > 0 {
				nodes[i].Children = buildTree(children)
			}
		}
		return nodes
	}

	return buildTree(roots), nil
}

func GetPrerequisites(db *pgxpool.Pool, conceptID string) ([]string, error) {
	rows, err := db.Query(context.Background(),
		`SELECT prerequisite_id FROM concept_prerequisites WHERE concept_id = $1`, conceptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prereqs []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		prereqs = append(prereqs, p)
	}
	return prereqs, nil
}
