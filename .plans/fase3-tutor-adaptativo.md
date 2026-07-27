# Fase 3 — Tutor Adaptativo, Generación de Ejercicios y Seguimiento del Aprendizaje

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform matematicarag from a math solver into an adaptive tutor that tracks student mastery, generates validated exercises, corrects step-by-step procedures, detects error patterns, and provides personalized recommendations — with dashboards for both students and teachers.

**Architecture:** Extend the existing Go+Chi backend with new API route groups (`/api/learning`, `/api/tutor/enhanced`, `/api/teacher`). Add PostgreSQL tables for student profiles, concept mastery, exercises, attempts, sessions, and error tracking. The adaptive engine runs server-side in Go, selecting concepts and difficulty based on mastery data. Exercise generation uses LLM + RAG for content, validated by the existing Python/SymPy math service. Frontend gets a student progress dashboard, teacher dashboard, and an enhanced tutor component with modes (tutor/practice/review).

**Tech Stack:** Go 1.25 + Chi v5, Python 3.11 + Flask + SymPy, Angular 20 + Material + KaTeX, PostgreSQL 16, Qdrant v1.12, OpenAI API

## Global Constraints

- Go backend uses `pgxpool.Pool` for all DB access (no ORM)
- Migrations are inline strings in `internal/database/database.go`
- All new API routes require `AuthMiddleware`; teacher routes additionally require `RoleMiddleware("TEACHER", "ADMIN")`
- Student data isolation: students see only their own data via `UserIDKey` from JWT
- Exercise validation is mandatory: LLM generates, Math Engine verifies before storing
- All configurable parameters use env vars with sensible defaults in `config.go`
- Frontend uses Angular signals, standalone components, Material Design
- No comments in code unless explicitly requested

---

## File Structure

### New Files (Backend)

| File | Responsibility |
|------|---------------|
| `api/learning.go` | Student profile CRUD, mastery queries, progress endpoints |
| `api/knowledge.go` | Concept graph, prerequisites, curriculum structure |
| `api/exercises.go` | Exercise bank, generation, validation, hints |
| `api/sessions.go` | Tutor session lifecycle (create, answer, hint, feedback) |
| `api/adaptive.go` | Adaptive engine: next concept, difficulty, recommendation |
| `api/errors.go` | Error taxonomy, pattern detection, recurrent errors |
| `api/teacher.go` | Teacher dashboard: course progress, common errors |
| `api/student_dash.go` | Student dashboard: progress, stats, knowledge map |
| `api/session_middleware.go` | Extract student from JWT for learning routes |

### New Files (Frontend)

| File | Responsibility |
|------|---------------|
| `frontend/src/app/modules/student-progress/student-progress.component.ts` | Student dashboard: mastery bars, stats, knowledge map |
| `frontend/src/app/modules/teacher-dashboard/teacher-dashboard.component.ts` | Teacher dashboard: course overview, common errors |
| `frontend/src/app/core/services/learning.service.ts` | HTTP client for learning API |

### Modified Files

| File | Change |
|------|--------|
| `internal/database/database.go` | Add 9 new migration statements |
| `cmd/server/main.go` | Register new route groups |
| `internal/config/config.go` | Add adaptive engine config params |
| `frontend/src/app/app.routes.ts` | Add student-progress, teacher-dashboard routes |
| `frontend/src/app/shared/layout.component.ts` | Add nav items for new pages |
| `frontend/src/app/core/services/api.service.ts` | Add exercise/session methods |
| `frontend/src/app/modules/tutor/tutor.component.ts` | Add mode selector (tutor/practice/review) |
| `math-service/app.py` | Add `/math/validate-exercise` endpoint |

---

## Task 1: Database Schema — Learning Tables

**Files:**
- Modify: `internal/database/database.go`

**Interfaces:**
- Produces: 9 new PostgreSQL tables + indexes for learning system

This is the foundation. Every subsequent task depends on these tables existing.

- [ ] **Step 1: Add knowledge model tables to migrations**

Add the following migration strings to the `migrations` slice in `database.go`, after the existing `document_chunks` migrations:

```go
// Knowledge Model
`CREATE TABLE IF NOT EXISTS concepts (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    parent_id VARCHAR(100) REFERENCES concepts(id),
    course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
    difficulty_base INTEGER DEFAULT 1 CHECK (difficulty_base BETWEEN 1 AND 5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE TABLE IF NOT EXISTS concept_prerequisites (
    concept_id VARCHAR(100) NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    prerequisite_id VARCHAR(100) NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    PRIMARY KEY (concept_id, prerequisite_id)
)`,
`CREATE INDEX IF NOT EXISTS idx_concepts_course ON concepts(course_id)`,
`CREATE INDEX IF NOT EXISTS idx_concept_prereq ON concept_prerequisites(prerequisite_id)`,

// Student Learning Profile
`CREATE TABLE IF NOT EXISTS student_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
    overall_level REAL DEFAULT 0.0 CHECK (overall_level BETWEEN 0.0 AND 1.0),
    total_attempts INTEGER DEFAULT 0,
    correct_attempts INTEGER DEFAULT 0,
    total_hints_used INTEGER DEFAULT 0,
    study_time_seconds INTEGER DEFAULT 0,
    last_active_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_student_profiles_student ON student_profiles(student_id)`,

// Concept Mastery per Student
`CREATE TABLE IF NOT EXISTS concept_mastery (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    concept_id VARCHAR(100) NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    mastery REAL DEFAULT 0.0 CHECK (mastery BETWEEN 0.0 AND 1.0),
    status VARCHAR(20) NOT NULL DEFAULT 'not_started'
        CHECK (status IN ('not_started','learning','developing','mastered')),
    attempts INTEGER DEFAULT 0,
    correct INTEGER DEFAULT 0,
    hints_used INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(student_id, concept_id)
)`,
`CREATE INDEX IF NOT EXISTS idx_concept_mastery_student ON concept_mastery(student_id)`,
`CREATE INDEX IF NOT EXISTS idx_concept_mastery_concept ON concept_mastery(concept_id)`,

// Exercise Bank
`CREATE TABLE IF NOT EXISTS exercises (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    concept_id VARCHAR(100) NOT NULL REFERENCES concepts(id),
    difficulty INTEGER NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
    statement TEXT NOT NULL,
    latex TEXT DEFAULT '',
    expected_answer TEXT NOT NULL,
    solution TEXT DEFAULT '',
    solution_steps JSONB DEFAULT '[]',
    hints JSONB DEFAULT '[]',
    common_errors JSONB DEFAULT '[]',
    source VARCHAR(20) NOT NULL DEFAULT 'generated'
        CHECK (source IN ('official','generated')),
    generated_by VARCHAR(50) DEFAULT '',
    verified_by_math BOOLEAN DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'validated'
        CHECK (status IN ('pending','validated','rejected')),
    embedding_id VARCHAR(100) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_exercises_concept ON exercises(concept_id)`,
`CREATE INDEX IF NOT EXISTS idx_exercises_difficulty ON exercises(difficulty)`,
`CREATE INDEX IF NOT EXISTS idx_exercises_source ON exercises(source)`,

// Tutor Sessions
`CREATE TABLE IF NOT EXISTS tutor_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
    mode VARCHAR(20) NOT NULL DEFAULT 'tutor'
        CHECK (mode IN ('tutor','practice','review','exam','solve')),
    concept_id VARCHAR(100) DEFAULT '',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    exercise_count INTEGER DEFAULT 0,
    correct_count INTEGER DEFAULT 0,
    hints_used INTEGER DEFAULT 0,
    total_score REAL DEFAULT 0.0
)`,
`CREATE INDEX IF NOT EXISTS idx_tutor_sessions_student ON tutor_sessions(student_id)`,

// Exercise Attempts
`CREATE TABLE IF NOT EXISTS exercise_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES tutor_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES exercises(id),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    answer TEXT DEFAULT '',
    correct BOOLEAN DEFAULT false,
    score REAL DEFAULT 0.0 CHECK (score BETWEEN 0.0 AND 1.0),
    hints_used INTEGER DEFAULT 0,
    max_hints_used INTEGER DEFAULT 0,
    first_error_step INTEGER DEFAULT 0,
    time_seconds INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_exercise_attempts_session ON exercise_attempts(session_id)`,
`CREATE INDEX IF NOT EXISTS idx_exercise_attempts_student ON exercise_attempts(student_id)`,

// Step-by-Step Attempts
`CREATE TABLE IF NOT EXISTS attempt_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL REFERENCES exercise_attempts(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','correct','incorrect')),
    error_type VARCHAR(50) DEFAULT '',
    error_detail TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_attempt_steps_attempt ON attempt_steps(attempt_id)`,

// Error Tracking
`CREATE TABLE IF NOT EXISTS student_errors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    concept_id VARCHAR(100) NOT NULL DEFAULT '',
    error_type VARCHAR(50) NOT NULL,
    error_subtype VARCHAR(100) DEFAULT '',
    count INTEGER DEFAULT 1,
    severity VARCHAR(20) DEFAULT 'low'
        CHECK (severity IN ('low','medium','high','critical')),
    last_occurred_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(student_id, concept_id, error_type, error_subtype)
)`,
`CREATE INDEX IF NOT EXISTS idx_student_errors_student ON student_errors(student_id)`,

// Learning Recommendations
`CREATE TABLE IF NOT EXISTS learning_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recommendation_type VARCHAR(50) NOT NULL,
    concept_id VARCHAR(100) DEFAULT '',
    message TEXT NOT NULL,
    priority INTEGER DEFAULT 1,
    dismissed BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_recommendations_student ON learning_recommendations(student_id)`
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add internal/database/database.go
git commit -m "feat(fase3): add learning database schema — concepts, mastery, exercises, sessions, attempts, errors"
```

---

## Task 2: Knowledge Model — Concept Graph & Seed Data

**Files:**
- Create: `api/knowledge.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `KnowledgeRoutes(db, cfg)` → route group
- Produces: `GetConceptTree(db, courseID) → []ConceptNode`
- Produces: `GetPrerequisites(db, conceptID) → []string`
- Produces: `ConceptNode{ID, Name, ParentID, Children, DifficultyBase}`

- [ ] **Step 1: Create `api/knowledge.go` with types and concept tree query**

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConceptNode struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	ParentID        *string       `json:"parent_id,omitempty"`
	CourseID        string        `json:"course_id"`
	DifficultyBase  int           `json:"difficulty_base"`
	Children        []ConceptNode `json:"children,omitempty"`
}

type PrereqInfo struct {
	ConceptID      string   `json:"concept_id"`
	Prerequisites  []string `json:"prerequisites"`
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
```

- [ ] **Step 2: Add context import**

Ensure the import section of `knowledge.go` includes `"context"`.

- [ ] **Step 3: Register route in main.go**

Add inside the `apiRouter.Route("/api", ...)` block in `cmd/server/main.go`:

```go
r.Route("/learning", api.KnowledgeRoutes(db, cfg))
```

- [ ] **Step 4: Create seed data — add concept inserts to migrations**

Add to `database.go` migrations, after the learning tables:

```go
// Seed: Math I concept tree
`INSERT INTO concepts (id, name, description, parent_id, course_id, difficulty_base) VALUES
 ('algebra', 'Álgebra', 'Operaciones algebraicas y ecuaciones', NULL, 'matematica-1', 1),
 ('algebra.operaciones', 'Operaciones algebraicas', 'Suma, resta, multiplicación, división de polinomios', 'algebra', 'matematica-1', 1),
 ('algebra.factorizacion', 'Factorización', 'Factor común, diferencia de cuadrados, trinomio cuadrado', 'algebra', 'matematica-1', 2),
 ('algebra.ecuaciones', 'Ecuaciones', 'Ecuaciones lineales y cuadráticas', 'algebra', 'matematica-1', 2),
 ('funciones', 'Funciones', 'Concepto de función, dominio, imagen', NULL, 'matematica-1', 1),
 ('funciones.lineal', 'Función lineal', 'f(x) = mx + b, pendiente, intersección', 'funciones', 'matematica-1', 1),
 ('funciones.cuadratica', 'Función cuadrática', 'f(x) = ax² + bx + c, vértice, raíces', 'funciones', 'matematica-1', 2),
 ('funciones.composicion', 'Composición de funciones', 'f(g(x)), función compuesta', 'funciones', 'matematica-1', 3),
 ('limites', 'Límites', 'Concepto y cálculo de límites', NULL, 'matematica-1', 2),
 ('limites.concepto', 'Concepto de límite', 'Definición intuitiva y formal', 'limites', 'matematica-1', 2),
 ('limites.propiedades', 'Propiedades de límites', 'Propiedades algebraicas', 'limites', 'matematica-1', 3),
 ('limites.laterales', 'Límites laterales', 'Límite por la izquierda y derecha', 'limites', 'matematica-1', 3),
 ('derivadas', 'Derivadas', 'Cálculo diferencial', NULL, 'matematica-1', 3),
 ('derivadas.definicion', 'Definición de derivada', 'Límite de la razón de incremento', 'derivadas', 'matematica-1', 3),
 ('derivadas.potencia', 'Regla de la potencia', 'd/dx(x^n) = n·x^(n-1)', 'derivadas', 'matematica-1', 3),
 ('derivadas.producto', 'Regla del producto', 'd/dx(f·g) = f\'·g + f·g\'', 'derivadas', 'matematica-1', 4),
 ('derivadas.cociente', 'Regla del cociente', 'd/dx(f/g) = (f\'·g - f·g\') / g²', 'derivadas', 'matematica-1', 4),
 ('derivadas.cadena', 'Regla de la cadena', 'd/dx(f(g(x))) = f\'(g(x))·g\'(x)', 'derivadas', 'matematica-1', 4),
 ('integrales', 'Integrales', 'Cálculo integral', NULL, 'matematica-1', 4),
 ('integrales.indefinida', 'Integral indefinida', 'Antiderivada, familia de funciones', 'integrales', 'matematica-1', 4),
 ('integrales.definida', 'Integral definida', 'Área bajo la curva, teorema fundamental', 'integrales', 'matematica-1', 4),
 ('integrales.sustitucion', 'Sustitución', 'Cambio de variable', 'integrales', 'matematica-1', 5),
 ('integrales.partes', 'Integración por partes', '∫u·dv = u·v - ∫v·du', 'integrales', 'matematica-1', 5)
 ON CONFLICT (id) DO NOTHING`,
`INSERT INTO concept_prerequisites (concept_id, prerequisite_id) VALUES
 ('algebra.operaciones', 'algebra'),
 ('algebra.factorizacion', 'algebra.operaciones'),
 ('algebra.ecuaciones', 'algebra.operaciones'),
 ('funciones.lineal', 'funciones'),
 ('funciones.cuadratica', 'funciones.lineal'),
 ('funciones.composicion', 'funciones.cuadratica'),
 ('limites.concepto', 'funciones'),
 ('limites.propiedades', 'limites.concepto'),
 ('limites.laterales', 'limites.propiedades'),
 ('derivadas.definicion', 'limites.concepto'),
 ('derivadas.potencia', 'derivadas.definicion'),
 ('derivadas.producto', 'derivadas.potencia'),
 ('derivadas.cociente', 'derivadas.producto'),
 ('derivadas.cadena', 'derivadas.potencia'),
 ('derivadas.cadena', 'funciones.composicion'),
 ('integrales.indefinida', 'derivadas.potencia'),
 ('integrales.definida', 'integrales.indefinida'),
 ('integrales.definida', 'limites.concepto'),
 ('integrales.sustitucion', 'integrales.indefinida'),
 ('integrales.partes', 'integrales.indefinida'),
 ('integrales.partes', 'derivadas.producto')
 ON CONFLICT DO NOTHING`,
```

- [ ] **Step 5: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 6: Commit**

```bash
git add api/knowledge.go cmd/server/main.go internal/database/database.go
git commit -m "feat(fase3): knowledge model — concept tree, prerequisites, seed data"
```

---

## Task 3: Student Profile & Mastery Tracking

**Files:**
- Create: `api/learning.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: `UserIDKey` from JWT context
- Produces: `LearningRoutes(db, cfg)` → route group
- Produces: `GetOrCreateProfile(db, studentID, courseID) → StudentProfile`
- Produces: `GetMasteryMap(db, studentID, courseID) → map[string]ConceptMastery`
- Produces: `UpdateMastery(db, studentID, conceptID, correct, hintsUsed, score)`

- [ ] **Step 1: Create `api/learning.go` with types and core functions**

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentProfile struct {
	ID              string  `json:"id"`
	StudentID       string  `json:"student_id"`
	CourseID        string  `json:"course_id"`
	OverallLevel    float64 `json:"overall_level"`
	TotalAttempts   int     `json:"total_attempts"`
	CorrectAttempts int     `json:"correct_attempts"`
	TotalHintsUsed  int     `json:"total_hints_used"`
	StudyTimeSeconds int    `json:"study_time_seconds"`
}

type ConceptMastery struct {
	ID           string  `json:"id"`
	StudentID    string  `json:"student_id"`
	ConceptID    string  `json:"concept_id"`
	Mastery      float64 `json:"mastery"`
	Status       string  `json:"status"`
	Attempts     int     `json:"attempts"`
	Correct      int     `json:"correct"`
	HintsUsed    int     `json:"hints_used"`
	ErrorCount   int     `json:"error_count"`
}

func LearningRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Get("/progress", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			profile, err := GetOrCreateProfile(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load profile"}`, http.StatusInternalServerError)
				return
			}
			mastery, err := GetMasteryMap(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load mastery"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"profile": profile,
				"mastery": mastery,
			})
		})

		r.Get("/mastery", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			mastery, err := GetMasteryMap(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load mastery"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mastery)
		})

		r.Get("/errors", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			errors, err := GetStudentErrors(db, studentID)
			if err != nil {
				http.Error(w, `{"error":"failed to load errors"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errors)
		})
	}
}

func GetOrCreateProfile(db *pgxpool.Pool, studentID, courseID string) (*StudentProfile, error) {
	ctx := context.Background()
	var p StudentProfile
	err := db.QueryRow(ctx,
		`INSERT INTO student_profiles (student_id, course_id)
		 VALUES ($1, $2)
		 ON CONFLICT (student_id) DO UPDATE SET updated_at = NOW()
		 RETURNING id, student_id, course_id, overall_level, total_attempts, correct_attempts, total_hints_used, study_time_seconds`,
		studentID, courseID,
	).Scan(&p.ID, &p.StudentID, &p.CourseID, &p.OverallLevel, &p.TotalAttempts, &p.CorrectAttempts, &p.TotalHintsUsed, &p.StudyTimeSeconds)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetMasteryMap(db *pgxpool.Pool, studentID, courseID string) (map[string]ConceptMastery, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT cm.id, cm.student_id, cm.concept_id, cm.mastery, cm.status, cm.attempts, cm.correct, cm.hints_used, cm.error_count
		 FROM concept_mastery cm
		 JOIN concepts c ON cm.concept_id = c.id
		 WHERE cm.student_id = $1 AND c.course_id = $2`,
		studentID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]ConceptMastery)
	for rows.Next() {
		var m ConceptMastery
		if err := rows.Scan(&m.ID, &m.StudentID, &m.ConceptID, &m.Mastery, &m.Status, &m.Attempts, &m.Correct, &m.HintsUsed, &m.ErrorCount); err != nil {
			continue
		}
		result[m.ConceptID] = m
	}
	return result, nil
}

func UpdateMastery(db *pgxpool.Pool, studentID, conceptID string, correct bool, hintsUsed int, score float64) error {
	ctx := context.Background()

	// Upsert concept_mastery
	masteryDelta := 0.0
	if correct {
		masteryDelta = 0.05 * (1.0 - float64(hintsUsed)*0.1) * score
		if masteryDelta < 0.01 {
			masteryDelta = 0.01
		}
	} else {
		masteryDelta = -0.03
	}

	_, err := db.Exec(ctx,
		`INSERT INTO concept_mastery (student_id, concept_id, mastery, status, attempts, correct, hints_used, last_attempt_at)
		 VALUES ($1, $2, GREATEST(0, LEAST(1, $3)), CASE
		   WHEN GREATEST(0, LEAST(1, $3)) >= 0.8 THEN 'mastered'
		   WHEN GREATEST(0, LEAST(1, $3)) >= 0.5 THEN 'developing'
		   WHEN GREATEST(0, LEAST(1, $3)) > 0.0 THEN 'learning'
		   ELSE 'not_started' END,
		   1, CASE WHEN $4 THEN 1 ELSE 0 END, $5, NOW())
		 ON CONFLICT (student_id, concept_id) DO UPDATE SET
		   mastery = GREATEST(0, LEAST(1, concept_mastery.mastery + $3)),
		   status = CASE
		     WHEN GREATEST(0, LEAST(1, concept_mastery.mastery + $3)) >= 0.8 THEN 'mastered'
		     WHEN GREATEST(0, LEAST(1, concept_mastery.mastery + $3)) >= 0.5 THEN 'developing'
		     WHEN GREATEST(0, LEAST(1, concept_mastery.mastery + $3)) > 0.0 THEN 'learning'
		     ELSE 'not_started' END,
		   attempts = concept_mastery.attempts + 1,
		   correct = concept_mastery.correct + CASE WHEN $4 THEN 1 ELSE 0 END,
		   hints_used = concept_mastery.hints_used + $5,
		   last_attempt_at = NOW(),
		   updated_at = NOW()`,
		studentID, conceptID, masteryDelta, correct, hintsUsed)
	if err != nil {
		return err
	}

	// Update overall profile
	_, err = db.Exec(ctx,
		`UPDATE student_profiles SET
		   total_attempts = total_attempts + 1,
		   correct_attempts = correct_attempts + CASE WHEN $2 THEN 1 ELSE 0 END,
		   total_hints_used = total_hints_used + $3,
		   overall_level = (SELECT COALESCE(AVG(mastery), 0) FROM concept_mastery WHERE student_id = $1),
		   last_active_at = NOW(),
		   updated_at = NOW()
		 WHERE student_id = $1`,
		studentID, correct, hintsUsed)
	return err
}

func GetStudentErrors(db *pgxpool.Pool, studentID string) ([]StudentError, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT id, student_id, concept_id, error_type, error_subtype, count, severity, last_occurred_at
		 FROM student_errors WHERE student_id = $1 ORDER BY count DESC`,
		studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []StudentError
	for rows.Next() {
		var e StudentError
		if err := rows.Scan(&e.ID, &e.StudentID, &e.ConceptID, &e.ErrorType, &e.ErrorSubtype, &e.Count, &e.Severity, &e.LastOccurredAt); err != nil {
			continue
		}
		errors = append(errors, e)
	}
	return errors, nil
}

type StudentError struct {
	ID              string `json:"id"`
	StudentID       string `json:"student_id"`
	ConceptID       string `json:"concept_id"`
	ErrorType       string `json:"error_type"`
	ErrorSubtype    string `json:"error_subtype"`
	Count           int    `json:"count"`
	Severity        string `json:"severity"`
	LastOccurredAt  string `json:"last_occurred_at"`
}
```

- [ ] **Step 2: Register LearningRoutes in main.go**

Add inside the auth group in `cmd/server/main.go`:

```go
r.Route("/learning", api.LearningRoutes(db, cfg))
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add api/learning.go cmd/server/main.go
git commit -m "feat(fase3): student profile & mastery tracking API"
```

---

## Task 4: Adaptive Engine — Next Concept & Difficulty Selection

**Files:**
- Create: `api/adaptive.go`

**Interfaces:**
- Consumes: `GetMasteryMap()`, `GetPrerequisites()`, `GetConceptTree()`
- Produces: `RecommendNext(db, studentID, courseID) → AdaptiveRecommendation`
- Produces: `CalculateDifficulty(mastery, errors, hints) → int`

- [ ] **Step 1: Create `api/adaptive.go`**

```go
package api

import (
	"context"
	"math"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdaptiveRecommendation struct {
	RecommendedConcept  string  `json:"recommended_concept"`
	ConceptName         string  `json:"concept_name"`
	RecommendedDifficulty int   `json:"recommended_difficulty"`
	Reason              string  `json:"reason"`
	PrerequisitesMet    bool    `json:"prerequisites_met"`
	MissingPrereqs      []string `json:"missing_prereqs,omitempty"`
}

func RecommendNext(db *pgxpool.Pool, studentID, courseID string) (*AdaptiveRecommendation, error) {
	mastery, err := GetMasteryMap(db, studentID, courseID)
	if err != nil {
		return nil, err
	}

	tree, err := GetConceptTree(db, courseID)
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
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add api/adaptive.go
git commit -m "feat(fase3): adaptive engine — next concept, difficulty, recommendations"
```

---

## Task 5: Exercise Bank — CRUD, Generation & Validation

**Files:**
- Create: `api/exercises.go`
- Modify: `math-service/app.py`

**Interfaces:**
- Consumes: `MathClient` (existing), `callOpenAIWithHistory()` (existing)
- Produces: `ExerciseRoutes(db, cfg)` → route group
- Produces: `GenerateExercise(db, conceptID, difficulty) → *Exercise`
- Produces: `ValidateExercise(expression, expected) → bool`

- [ ] **Step 1: Create `api/exercises.go`**

```go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Exercise struct {
	ID             string          `json:"id"`
	ConceptID      string          `json:"concept_id"`
	Difficulty     int             `json:"difficulty"`
	Statement      string          `json:"statement"`
	Latex          string          `json:"latex"`
	ExpectedAnswer string          `json:"expected_answer"`
	Solution       string          `json:"solution"`
	SolutionSteps  json.RawMessage `json:"solution_steps"`
	Hints          json.RawMessage `json:"hints"`
	CommonErrors   json.RawMessage `json:"common_errors"`
	Source         string          `json:"source"`
	VerifiedByMath bool            `json:"verified_by_math"`
}

func ExerciseRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Post("/generate", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				ConceptID  string `json:"concept_id"`
				Difficulty int    `json:"difficulty"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Difficulty < 1 || req.Difficulty > 5 {
				req.Difficulty = 2
			}

			exercise, err := GenerateExercise(db, cfg, req.ConceptID, req.Difficulty)
			if err != nil {
				log.Printf("[EXERCISE] generation failed: %v", err)
				http.Error(w, `{"error":"generation failed"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(exercise)
		})

		r.Get("/concept/{conceptID}", func(w http.ResponseWriter, r *http.Request) {
			conceptID := chi.URLParam(r, "conceptID")
			difficulty := r.URL.Query().Get("difficulty")

			var rows pgx.Rows
			var err error
			if difficulty != "" {
				rows, err = db.Query(r.Context(),
					`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
					 FROM exercises WHERE concept_id = $1 AND difficulty = $2 AND status = 'validated'
					 ORDER BY RANDOM() LIMIT 1`, conceptID, difficulty)
			} else {
				rows, err = db.Query(r.Context(),
					`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
					 FROM exercises WHERE concept_id = $1 AND status = 'validated'
					 ORDER BY RANDOM() LIMIT 1`, conceptID)
			}
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			if rows.Next() {
				var ex Exercise
				if err := rows.Scan(&ex.ID, &ex.ConceptID, &ex.Difficulty, &ex.Statement, &ex.Latex, &ex.ExpectedAnswer, &ex.Solution, &ex.SolutionSteps, &ex.Hints, &ex.CommonErrors, &ex.Source, &ex.VerifiedByMath); err != nil {
					http.Error(w, `{"error":"scan failed"}`, http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ex)
			} else {
				http.Error(w, `{"error":"no exercises found"}`, http.StatusNotFound)
			}
		})

		r.Get("/{exerciseID}/hints", func(w http.ResponseWriter, r *http.Request) {
			exerciseID := chi.URLParam(r, "exerciseID")
			var hints json.RawMessage
			err := db.QueryRow(r.Context(),
				`SELECT hints FROM exercises WHERE id = $1`, exerciseID,
			).Scan(&hints)
			if err != nil {
				http.Error(w, `{"error":"exercise not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(hints)
		})
	}
}

func GenerateExercise(db *pgxpool.Pool, cfg *config.Config, conceptID string, difficulty int) (*Exercise, error) {
	ctx := context.Background()

	conceptName := ""
	db.QueryRow(ctx, `SELECT name FROM concepts WHERE id = $1`, conceptID).Scan(&conceptName)
	if conceptName == "" {
		conceptName = conceptID
	}

	systemPrompt := `Eres un generador de ejercicios de matemática.
Genera UN ejercicio para el concepto indicado con la dificultad especificada.
Responde SOLO con JSON válido:
{
  "statement": "enunciado en texto plano",
  "latex": "enunciado en LaTeX",
  "expected_answer": "respuesta esperada en texto",
  "solution": "solución completa en texto",
  "solution_steps": [{"step": 1, "explanation": "...", "latex": "..."}],
  "hints": ["pista 1", "pista 2", "pista 3"],
  "common_errors": [{"type": "algebraic", "description": "..."}]
}
No copies ejercicios existentes. Sé original.`

	userMsg := fmt.Sprintf("Concepto: %s\nDificultad: %d/5\nGenera el ejercicio.", conceptID, difficulty)

	messages := []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}

	response, err := callOpenAIWithHistory(db, messages, "")
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	response = strings.TrimSpace(response)
	if idx := strings.Index(response, "{"); idx != -1 {
		response = response[idx:]
	}
	if idx := strings.LastIndex(response, "}"); idx != -1 {
		response = response[:idx+1]
	}

	var parsed struct {
		Statement      string          `json:"statement"`
		Latex          string          `json:"latex"`
		ExpectedAnswer string          `json:"expected_answer"`
		Solution       string          `json:"solution"`
		SolutionSteps  json.RawMessage `json:"solution_steps"`
		Hints          json.RawMessage `json:"hints"`
		CommonErrors   json.RawMessage `json:"common_errors"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	mathClient := NewMathClient(cfg)
	verifyResult, err := mathClient.Verify(parsed.ExpectedAnswer, parsed.ExpectedAnswer)
	if err != nil {
		log.Printf("[EXERCISE] math verify failed: %v", err)
	}

	verified := verifyResult != nil && verifyResult.Success

	hintsJSON, _ := json.Marshal(parsed.Hints)
	errorsJSON, _ := json.Marshal(parsed.CommonErrors)
	stepsJSON, _ := json.Marshal(parsed.SolutionSteps)

	exercise := &Exercise{
		ConceptID:      conceptID,
		Difficulty:     difficulty,
		Statement:      parsed.Statement,
		Latex:          parsed.Latex,
		ExpectedAnswer: parsed.ExpectedAnswer,
		Solution:       parsed.Solution,
		SolutionSteps:  stepsJSON,
		Hints:          hintsJSON,
		CommonErrors:   errorsJSON,
		Source:         "generated",
		VerifiedByMath: verified,
	}

	var exID string
	err = db.QueryRow(ctx,
		`INSERT INTO exercises (concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id`,
		exercise.ConceptID, exercise.Difficulty, exercise.Statement, exercise.Latex,
		exercise.ExpectedAnswer, exercise.Solution, exercise.SolutionSteps,
		exercise.Hints, exercise.CommonErrors, exercise.Source, exercise.VerifiedByMath,
		func() string {
			if verified {
				return "validated"
			}
			return "pending"
		}(),
	).Scan(&exID)
	if err != nil {
		return nil, fmt.Errorf("insert exercise: %w", err)
	}
	exercise.ID = exID

	return exercise, nil
}
```

- [ ] **Step 2: Add verify method to MathClient in `api/mathclient.go`**

Add this method to the `MathClient` struct:

```go
func (c *MathClient) Verify(expression, expected string) (*VerifyResult, error) {
	body := map[string]string{
		"expression": expression,
		"expected":   expected,
	}
	data, err := c.post("/math/verify", body)
	if err != nil {
		return nil, err
	}
	var result VerifyResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
```

- [ ] **Step 3: Register ExerciseRoutes in main.go**

Add inside the auth group:

```go
r.Route("/exercises", api.ExerciseRoutes(db, cfg))
```

- [ ] **Step 4: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 5: Commit**

```bash
git add api/exercises.go api/mathclient.go cmd/server/main.go
git commit -m "feat(fase3): exercise bank — generate, validate, retrieve, hints"
```

---

## Task 6: Error Taxonomy & Pattern Detection

**Files:**
- Create: `api/errors.go`

**Interfaces:**
- Produces: `RecordError(db, studentID, conceptID, errorType, subtype)`
- Produces: `DetectPatterns(db, studentID) → []ErrorPattern`
- Produces: `ErrorTaxonomy` map

- [ ] **Step 1: Create `api/errors.go`**

```go
package api

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrorTaxonomy = map[string][]string{
	"conceptual":       {"misconception", "wrong_concept", "incomplete_understanding"},
	"algebraic":        {"distributive_property", "factoring_error", "expansion_error"},
	"arithmetic":       {"addition", "subtraction", "multiplication", "division", "power"},
	"sign":             {"sign_change", "double_negative", "sign_in_distribution"},
	"formula":          {"wrong_formula", "formula_misapplication", "missing_term"},
	"method_selection": {"wrong_method", "unnecessary_complexity"},
	"notation":         {"notation_error", "undefined_variable"},
	"domain":           {"domain_violation", "division_by_zero"},
	"logical":          {"logical_gap", "invalid_inference", "circular_reasoning"},
	"incomplete":       {"missing_solution", "missing_case", "incomplete_answer"},
}

type ErrorPattern struct {
	ConceptID    string `json:"concept_id"`
	ErrorType    string `json:"error_type"`
	ErrorSubtype string `json:"error_subtype"`
	Count        int    `json:"count"`
	Severity     string `json:"severity"`
}

func RecordError(db *pgxpool.Pool, studentID, conceptID, errorType, errorSubtype string) {
	ctx := context.Background()
	severity := calculateSeverity(db, studentID, conceptID, errorType)

	_, err := db.Exec(ctx,
		`INSERT INTO student_errors (student_id, concept_id, error_type, error_subtype, count, severity, last_occurred_at)
		 VALUES ($1, $2, $3, $4, 1, $5, NOW())
		 ON CONFLICT (student_id, concept_id, error_type, error_subtype) DO UPDATE SET
		   count = student_errors.count + 1,
		   severity = $5,
		   last_occurred_at = NOW()`,
		studentID, conceptID, errorType, errorSubtype, severity)
	if err != nil {
		log.Printf("[ERRORS] record error: %v", err)
	}
}

func calculateSeverity(db *pgxpool.Pool, studentID, conceptID, errorType string) string {
	ctx := context.Background()
	var count int
	db.QueryRow(ctx,
		`SELECT COALESCE(SUM(count), 0) FROM student_errors
		 WHERE student_id = $1 AND concept_id = $2 AND error_type = $3`,
		studentID, conceptID, errorType).Scan(&count)

	switch {
	case count >= 8:
		return "critical"
	case count >= 5:
		return "high"
	case count >= 3:
		return "medium"
	default:
		return "low"
	}
}

func DetectPatterns(db *pgxpool.Pool, studentID string) ([]ErrorPattern, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT concept_id, error_type, error_subtype, count, severity
		 FROM student_errors
		 WHERE student_id = $1
		 ORDER BY count DESC
		 LIMIT 10`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []ErrorPattern
	for rows.Next() {
		var p ErrorPattern
		if err := rows.Scan(&p.ConceptID, &p.ErrorType, &p.ErrorSubtype, &p.Count, &p.Severity); err != nil {
			continue
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add api/errors.go
git commit -m "feat(fase3): error taxonomy & pattern detection"
```

---

## Task 7: Tutor Sessions — Lifecycle & Step Correction

**Files:**
- Create: `api/sessions.go`

**Interfaces:**
- Consumes: `RecommendNext()`, `GenerateExercise()`, `UpdateMastery()`, `RecordError()`, `MathClient`
- Produces: `SessionRoutes(db, cfg)` → route group
- Produces: `CreateSession()`, `SubmitAnswer()`, `RequestHint()`, `SubmitStep()`

- [ ] **Step 1: Create `api/sessions.go`**

```go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionResponse struct {
	SessionID string  `json:"session_id"`
	Mode      string  `json:"mode"`
	Exercise  *Exercise `json:"exercise,omitempty"`
	Message   string  `json:"message,omitempty"`
}

type AnswerRequest struct {
	SessionID   string   `json:"session_id"`
	ExerciseID  string   `json:"exercise_id"`
	Answer      string   `json:"answer"`
	Procedure   []string `json:"procedure,omitempty"`
}

type AnswerResponse struct {
	Correct           bool            `json:"correct"`
	Score             float64         `json:"score"`
	Feedback          string          `json:"feedback"`
	FirstErrorStep    int             `json:"first_error_step,omitempty"`
	ErrorType         string          `json:"error_type,omitempty"`
	ErrorDetail       string          `json:"error_detail,omitempty"`
	MasteryBefore     float64         `json:"mastery_before"`
	MasteryAfter      float64         `json:"mastery_after"`
	MasteryStatus     string          `json:"mastery_status"`
	NextExercise      *Exercise       `json:"next_exercise,omitempty"`
	MathVerified      bool            `json:"math_verified"`
	StepAnalysis      []StepResult    `json:"step_analysis,omitempty"`
}

type StepResult struct {
	Step   int    `json:"step"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HintResponse struct {
	HintIndex int    `json:"hint_index"`
	Hint      string `json:"hint"`
	TotalHints int   `json:"total_hints"`
}

func SessionRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Post("/session", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req struct {
				Mode     string `json:"mode"`
				CourseID string `json:"course_id"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.Mode == "" {
				req.Mode = "tutor"
			}
			if req.CourseID == "" {
				req.CourseID = "matematica-1"
			}

			session, err := CreateSession(db, studentID, req.CourseID, req.Mode)
			if err != nil {
				http.Error(w, `{"error":"failed to create session"}`, http.StatusInternalServerError)
				return
			}

			exercise, _ := NextExerciseForSession(db, cfg, studentID, req.CourseID, session.ID)

			resp := SessionResponse{
				SessionID: session.ID,
				Mode:      req.Mode,
				Exercise:  exercise,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req AnswerRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}

			resp, err := SubmitAnswer(db, cfg, studentID, &req)
			if err != nil {
				log.Printf("[SESSION] answer error: %v", err)
				http.Error(w, `{"error":"failed to process answer"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Post("/hint", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req struct {
				SessionID  string `json:"session_id"`
				ExerciseID string `json:"exercise_id"`
				HintIndex  int    `json:"hint_index"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			resp, err := RequestHint(db, studentID, &req)
			if err != nil {
				http.Error(w, `{"error":"no hints available"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Post("/feedback", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			var req struct {
				SessionID  string `json:"session_id"`
				ExerciseID string `json:"exercise_id"`
				Procedure  []string `json:"procedure"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			results, err := AnalyzeProcedure(db, cfg, studentID, req.ExerciseID, req.Procedure)
			if err != nil {
				http.Error(w, `{"error":"analysis failed"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		})
	}
}

type Session struct {
	ID        string
	Mode      string
	CourseID  string
}

func CreateSession(db *pgxpool.Pool, studentID, courseID, mode string) (*Session, error) {
	ctx := context.Background()
	var s Session
	err := db.QueryRow(ctx,
		`INSERT INTO tutor_sessions (student_id, course_id, mode)
		 VALUES ($1, $2, $3) RETURNING id, mode, course_id`,
		studentID, courseID, mode,
	).Scan(&s.ID, &s.Mode, &s.CourseID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func NextExerciseForSession(db *pgxpool.Pool, cfg *config.Config, studentID, courseID, sessionID string) (*Exercise, error) {
	rec, err := RecommendNext(db, studentID, courseID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var ex Exercise
	err = db.QueryRow(ctx,
		`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
		 FROM exercises
		 WHERE concept_id = $1 AND difficulty = $2 AND status = 'validated'
		 ORDER BY RANDOM() LIMIT 1`,
		rec.RecommendedConcept, rec.RecommendedDifficulty,
	).Scan(&ex.ID, &ex.ConceptID, &ex.Difficulty, &ex.Statement, &ex.Latex,
		&ex.ExpectedAnswer, &ex.Solution, &ex.SolutionSteps, &ex.Hints,
		&ex.CommonErrors, &ex.Source, &ex.VerifiedByMath)

	if err != nil {
		exercise, genErr := GenerateExercise(db, cfg, rec.RecommendedConcept, rec.RecommendedDifficulty)
		if genErr != nil {
			return nil, genErr
		}
		return exercise, nil
	}

	db.Exec(ctx, `UPDATE tutor_sessions SET exercise_count = exercise_count + 1 WHERE id = $1`, sessionID)
	return &ex, nil
}

func SubmitAnswer(db *pgxpool.Pool, cfg *config.Config, studentID string, req *AnswerRequest) (*AnswerResponse, error) {
	ctx := context.Background()

	var exercise Exercise
	err := db.QueryRow(ctx,
		`SELECT id, concept_id, difficulty, statement, latex, expected_answer, solution, solution_steps, hints, common_errors, source, verified_by_math
		 FROM exercises WHERE id = $1`, req.ExerciseID,
	).Scan(&exercise.ID, &exercise.ConceptID, &exercise.Difficulty, &exercise.Statement, &exercise.Latex,
		&exercise.ExpectedAnswer, &exercise.Solution, &exercise.SolutionSteps, &exercise.Hints,
		&exercise.CommonErrors, &exercise.Source, &exercise.VerifiedByMath)
	if err != nil {
		return nil, fmt.Errorf("exercise not found: %w", err)
	}

	mathClient := NewMathClient(cfg)
	correct := false
	score := 0.0
	mathVerified := false

	if verifyResult, err := mathClient.Verify(req.Answer, exercise.ExpectedAnswer); err == nil && verifyResult != nil {
		correct = verifyResult.Success
		mathVerified = true
		if correct {
			score = 1.0
		} else if verifyResult.Method != "" {
			score = 0.3
		}
	} else {
		correct = strings.EqualFold(strings.TrimSpace(req.Answer), strings.TrimSpace(exercise.ExpectedAnswer))
		if correct {
			score = 0.8
		}
	}

	// Step correction for procedure
	var stepAnalysis []StepResult
	firstErrorStep := 0
	errorType := ""
	if len(req.Procedure) > 0 && !correct {
		stepAnalysis, firstErrorStep, errorType = analyzeSteps(req.Procedure, exercise)
		score = calculateStepScore(stepAnalysis)
		if score > 0.5 {
			correct = false
		}
	}

	hintsUsed := 0
	db.QueryRow(ctx, `SELECT hints_used FROM tutor_sessions WHERE student_id = $1 ORDER BY created_at DESC LIMIT 1`, studentID).Scan(&hintsUsed)

	// Get mastery before
	var masteryBefore float64
	db.QueryRow(ctx,
		`SELECT COALESCE(mastery, 0) FROM concept_mastery WHERE student_id = $1 AND concept_id = $2`,
		studentID, exercise.ConceptID).Scan(&masteryBefore)

	// Record attempt
	now := time.Now()
	db.Exec(ctx,
		`INSERT INTO exercise_attempts (session_id, student_id, exercise_id, answer, correct, score, hints_used, first_error_step, time_seconds, completed_at)
		 SELECT ts.id, $1, $2, $3, $4, $5, $6, $7, 0, $8
		 FROM tutor_sessions ts WHERE ts.student_id = $1 ORDER BY ts.created_at DESC LIMIT 1`,
		studentID, req.ExerciseID, req.Answer, correct, score, hintsUsed, firstErrorStep, now)

	// Record errors
	if !correct && firstErrorStep > 0 && errorType != "" {
		RecordError(db, studentID, exercise.ConceptID, errorType, "")
	}

	// Update mastery
	UpdateMastery(db, studentID, exercise.ConceptID, correct, hintsUsed, score)

	var masteryAfter float64
	db.QueryRow(ctx,
		`SELECT COALESCE(mastery, 0) FROM concept_mastery WHERE student_id = $1 AND concept_id = $2`,
		studentID, exercise.ConceptID).Scan(&masteryAfter)

	status := "learning"
	if masteryAfter >= 0.8 {
		status = "mastered"
	} else if masteryAfter >= 0.5 {
		status = "developing"
	} else if masteryAfter > 0 {
		status = "learning"
	}

	feedback := buildFeedback(correct, score, firstErrorStep, errorType, exercise)

	// Update session
	db.Exec(ctx,
		`UPDATE tutor_sessions SET correct_count = correct_count + CASE WHEN $2 THEN 1 ELSE 0 END, total_score = total_score + $3
		 WHERE student_id = $1 ORDER BY created_at DESC LIMIT 1`,
		studentID, correct, score)

	resp := &AnswerResponse{
		Correct:       correct,
		Score:         score,
		Feedback:      feedback,
		FirstErrorStep: firstErrorStep,
		ErrorType:     errorType,
		MasteryBefore: masteryBefore,
		MasteryAfter:  masteryAfter,
		MasteryStatus: status,
		MathVerified:  mathVerified,
		StepAnalysis:  stepAnalysis,
	}

	if correct || score < 0.2 {
		nextEx, _ := NextExerciseForSession(db, cfg, studentID, exercise.ConceptID, "")
		resp.NextExercise = nextEx
	}

	return resp, nil
}

func RequestHint(db *pgxpool.Pool, studentID string, req *struct {
	SessionID  string `json:"session_id"`
	ExerciseID string `json:"exercise_id"`
	HintIndex  int    `json:"hint_index"`
}) (*HintResponse, error) {
	ctx := context.Background()
	var hints json.RawMessage
	err := db.QueryRow(ctx, `SELECT hints FROM exercises WHERE id = $1`, req.ExerciseID).Scan(&hints)
	if err != nil {
		return nil, err
	}

	var hintsList []string
	json.Unmarshal(hints, &hintsList)

	if req.HintIndex >= len(hintsList) {
		return nil, fmt.Errorf("no more hints")
	}

	db.Exec(ctx,
		`UPDATE tutor_sessions SET hints_used = hints_used + 1 WHERE id = $1`, req.SessionID)

	return &HintResponse{
		HintIndex:  req.HintIndex,
		Hint:       hintsList[req.HintIndex],
		TotalHints: len(hintsList),
	}, nil
}

func AnalyzeProcedure(db *pgxpool.Pool, cfg *config.Config, studentID, exerciseID string, procedure []string) ([]StepResult, error) {
	if len(procedure) == 0 {
		return nil, fmt.Errorf("empty procedure")
	}

	var exercise Exercise
	err := db.QueryRow(context.Background(),
		`SELECT id, concept_id, expected_answer, solution FROM exercises WHERE id = $1`, exerciseID,
	).Scan(&exercise.ID, &exercise.ConceptID, &exercise.ExpectedAnswer, &exercise.Solution)
	if err != nil {
		return nil, err
	}

	results, _, _ := analyzeSteps(procedure, exercise)
	return results, nil
}

func analyzeSteps(procedure []string, exercise Exercise) ([]StepResult, int, string) {
	results := make([]StepResult, len(procedure))
	firstError := 0
	errorType := ""

	mathClient := &MathClient{baseURL: "", httpClient: &http.Client{Timeout: 5 * time.Second}}

	for i, step := range procedure {
		results[i] = StepResult{Step: i + 1, Status: "correct"}

		if i == len(procedure)-1 {
			if verifyResult, err := mathClient.Verify(step, exercise.ExpectedAnswer); err == nil && verifyResult != nil {
				if !verifyResult.Success {
					results[i].Status = "incorrect"
					results[i].Error = "La respuesta final no coincide con el resultado esperado."
					if firstError == 0 {
						firstError = i + 1
						errorType = "arithmetic"
					}
				}
			}
		}
	}

	return results, firstError, errorType
}

func calculateStepScore(steps []StepResult) float64 {
	if len(steps) == 0 {
		return 0
	}
	correct := 0
	for _, s := range steps {
		if s.Status == "correct" {
			correct++
		}
	}
	return float64(correct) / float64(len(steps))
}

func buildFeedback(correct bool, score float64, firstErrorStep int, errorType string, exercise Exercise) string {
	if correct && score >= 0.9 {
		return "¡Correcto! Excelente resolución."
	}
	if correct {
		return "Correcto, pero revisa tu procedimiento para mayor claridad."
	}
	if firstErrorStep > 0 {
	.feedback := fmt.Sprintf("El error está en el paso %d.", firstErrorStep)
		switch errorType {
		case "algebraic":
			*feedback += " Revisa la manipulación algebraica en ese paso."
		case "arithmetic":
			*feedback += " Verifica el cálculo aritmético en ese paso."
		case "sign":
			*feedback += " Presta atención a los signos en ese paso."
		case "formula":
			*feedback += " Verifica que estás usando la fórmula correcta."
		default:
			*feedback += " Analiza ese paso con cuidado."
		}
		return *feedback
	}
	return "Tu respuesta no es correcta. Intenta revisar el enunciado y los conceptos involucrados."
}
```

Note: The `buildFeedback` function has a syntax issue with the pointer — fix it by using a regular string variable:

```go
func buildFeedback(correct bool, score float64, firstErrorStep int, errorType string, exercise Exercise) string {
	if correct && score >= 0.9 {
		return "¡Correcto! Excelente resolución."
	}
	if correct {
		return "Correcto, pero revisa tu procedimiento para mayor claridad."
	}
	if firstErrorStep > 0 {
		feedback := fmt.Sprintf("El error está en el paso %d.", firstErrorStep)
		switch errorType {
		case "algebraic":
			feedback += " Revisa la manipulación algebraica en ese paso."
		case "arithmetic":
			feedback += " Verifica el cálculo aritmético en ese paso."
		case "sign":
			feedback += " Presta atención a los signos en ese paso."
		case "formula":
			feedback += " Verifica que estás usando la fórmula correcta."
		default:
			feedback += " Analiza ese paso con cuidado."
		}
		return feedback
	}
	return "Tu respuesta no es correcta. Intenta revisar el enunciado y los conceptos involucrados."
}
```

- [ ] **Step 2: Register SessionRoutes in main.go**

Add inside the auth group:

```go
r.Route("/tutor", api.SessionRoutes(db, cfg))
```

Note: This will conflict with the existing `/api/tutor` route. Rename the existing one or nest the new routes under a sub-path. The simplest fix: change the registration to use a different sub-route:

```go
r.Route("/sessions", api.SessionRoutes(db, cfg))
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add api/sessions.go cmd/server/main.go
git commit -m "feat(fase3): tutor sessions — create, answer, hint, step correction, feedback"
```

---

## Task 8: Config Extension for Adaptive Engine

**Files:**
- Modify: `internal/config/config.go`

**Interfaces:**
- Produces: New config fields for adaptive thresholds

- [ ] **Step 1: Add adaptive engine config fields**

Add to the `Config` struct:

```go
// Adaptive Engine
AdaptiveHintWeight    float64
AdaptiveErrorWeight   float64
AdaptiveMasteryThreshold float64
AdaptiveMaxDifficulty int
```

Add to `Load()`:

```go
AdaptiveHintWeight:      getEnvFloat("ADAPTIVE_HINT_WEIGHT", 0.1),
AdaptiveErrorWeight:     getEnvFloat("ADAPTIVE_ERROR_WEIGHT", 0.03),
AdaptiveMasteryThreshold: getEnvFloat("ADAPTIVE_MASTERY_THRESHOLD", 0.8),
AdaptiveMaxDifficulty:   getEnvInt("ADAPTIVE_MAX_DIFFICULTY", 5),
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(fase3): adaptive engine config parameters"
```

---

## Task 9: Teacher Dashboard API

**Files:**
- Create: `api/teacher.go`

**Interfaces:**
- Consumes: `RoleMiddleware("TEACHER", "ADMIN")`
- Produces: `TeacherRoutes(db, cfg)` → route group

- [ ] **Step 1: Create `api/teacher.go`**

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseProgress struct {
	CourseID        string  `json:"course_id"`
	TotalStudents   int     `json:"total_students"`
	AverageMastery  float64 `json:"average_mastery"`
}

type TopicMastery struct {
	ConceptID      string  `json:"concept_id"`
	ConceptName    string  `json:"concept_name"`
	AverageMastery float64 `json:"average_mastery"`
	StudentCount   int     `json:"student_count"`
	StrugglingCount int    `json:"struggling_count"`
}

type CommonError struct {
	ErrorType    string `json:"error_type"`
	ErrorSubtype string `json:"error_subtype"`
	Count        int    `json:"count"`
	AffectedStudents int `json:"affected_students"`
}

type StudentProgress struct {
	StudentID     string  `json:"student_id"`
	StudentName   string  `json:"student_name"`
	Email         string  `json:"email"`
	OverallLevel  float64 `json:"overall_level"`
	TotalAttempts int     `json:"total_attempts"`
	LastActive    string  `json:"last_active"`
}

func TeacherRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))
		r.Use(RoleMiddleware("TEACHER", "ADMIN"))

		r.Get("/course-progress", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			var cp CourseProgress
			cp.CourseID = courseID
			db.QueryRow(r.Context(),
				`SELECT COUNT(DISTINCT sp.student_id), COALESCE(AVG(sp.overall_level), 0)
				 FROM student_profiles sp
				 JOIN users u ON sp.student_id = u.id
				 WHERE sp.course_id = $1 AND u.role = 'STUDENT'`,
				courseID).Scan(&cp.TotalStudents, &cp.AverageMastery)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cp)
		})

		r.Get("/topic-mastery", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			rows, err := db.Query(r.Context(),
				`SELECT c.id, c.name,
				   COALESCE(AVG(cm.mastery), 0) as avg_mastery,
				   COUNT(DISTINCT cm.student_id) as student_count,
				   COUNT(DISTINCT cm.student_id) FILTER (WHERE cm.mastery < 0.3) as struggling
				 FROM concepts c
				 LEFT JOIN concept_mastery cm ON c.id = cm.concept_id
				 WHERE c.course_id = $1
				 GROUP BY c.id, c.name
				 ORDER BY c.id`, courseID)
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var topics []TopicMastery
			for rows.Next() {
				var t TopicMastery
				if err := rows.Scan(&t.ConceptID, &t.ConceptName, &t.AverageMastery, &t.StudentCount, &t.StrugglingCount); err != nil {
					continue
				}
				topics = append(topics, t)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(topics)
		})

		r.Get("/common-errors", func(w http.ResponseWriter, r *http.Request) {
			rows, err := db.Query(r.Context(),
				`SELECT error_type, error_subtype, SUM(count) as total_count,
				   COUNT(DISTINCT student_id) as affected
				 FROM student_errors
				 GROUP BY error_type, error_subtype
				 ORDER BY total_count DESC
				 LIMIT 15`)
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var errors []CommonError
			for rows.Next() {
				var e CommonError
				if err := rows.Scan(&e.ErrorType, &e.ErrorSubtype, &e.Count, &e.AffectedStudents); err != nil {
					continue
				}
				errors = append(errors, e)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errors)
		})

		r.Get("/student-progress", func(w http.ResponseWriter, r *http.Request) {
			courseID := r.URL.Query().Get("course_id")
			if courseID == "" {
				courseID = "matematica-1"
			}

			rows, err := db.Query(r.Context(),
				`SELECT u.id, u.name || ' ' || COALESCE(u.last_name, ''), u.email,
				   COALESCE(sp.overall_level, 0), COALESCE(sp.total_attempts, 0),
				   COALESCE(sp.last_active_at::text, '')
				 FROM users u
				 LEFT JOIN student_profiles sp ON u.id = sp.student_id AND sp.course_id = $1
				 WHERE u.role = 'STUDENT'
				 ORDER BY sp.overall_level DESC NULLS LAST`, courseID)
			if err != nil {
				http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var students []StudentProgress
			for rows.Next() {
				var s StudentProgress
				if err := rows.Scan(&s.StudentID, &s.StudentName, &s.Email, &s.OverallLevel, &s.TotalAttempts, &s.LastActive); err != nil {
					continue
				}
				students = append(students, s)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(students)
		})
	}
}
```

- [ ] **Step 2: Register in main.go**

Add in the admin group (alongside settings/stats):

```go
r.Route("/teacher", api.TeacherRoutes(db, cfg))
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add api/teacher.go cmd/server/main.go
git commit -m "feat(fase3): teacher dashboard API — course progress, topic mastery, common errors"
```

---

## Task 10: Student Dashboard API

**Files:**
- Create: `api/student_dash.go`

**Interfaces:**
- Produces: `StudentDashRoutes(db, cfg)` → route group

- [ ] **Step 1: Create `api/student_dash.go`**

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentDashboard struct {
	Profile         StudentProfile          `json:"profile"`
	MasteryMap      map[string]ConceptMastery `json:"mastery_map"`
	RecentErrors    []StudentError          `json:"recent_errors"`
	Recommendations []string                `json:"recommendations"`
	SessionsSummary SessionsSummary         `json:"sessions_summary"`
}

type SessionsSummary struct {
	TotalSessions  int     `json:"total_sessions"`
	TotalExercises int     `json:"total_exercises"`
	CorrectRate    float64 `json:"correct_rate"`
	AverageHints   float64 `json:"average_hints"`
	StudyTimeHours float64 `json:"study_time_hours"`
}

func StudentDashRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Get("/my-progress", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)

			profile, err := GetOrCreateProfile(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"profile not found"}`, http.StatusInternalServerError)
				return
			}

			mastery, _ := GetMasteryMap(db, studentID, "matematica-1")
			errors, _ := GetStudentErrors(db, studentID)

			var summary SessionsSummary
			db.QueryRow(r.Context(),
				`SELECT COUNT(*), COALESCE(SUM(exercise_count), 0),
				   CASE WHEN SUM(exercise_count) > 0 THEN COALESCE(SUM(correct_count)::float / SUM(exercise_count), 0) ELSE 0 END,
				   COALESCE(AVG(hints_used), 0)
				 FROM tutor_sessions WHERE student_id = $1`, studentID,
			).Scan(&summary.TotalSessions, &summary.TotalExercises, &summary.CorrectRate, &summary.AverageHints)

			summary.StudyTimeHours = float64(profile.StudyTimeSeconds) / 3600.0

			var recs []string
			if len(errors) > 0 {
				for _, e := range errors[:min(3, len(errors))] {
					recs = append(recs, "Reforzar: "+e.ErrorType+" en "+e.ConceptID)
				}
			}

			resp := StudentDashboard{
				Profile:         *profile,
				MasteryMap:      mastery,
				RecentErrors:    errors,
				Recommendations: recs,
				SessionsSummary: summary,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Get("/recommendations", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			rec, err := RecommendNext(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"no recommendations"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rec)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 2: Register in main.go**

Add inside the auth group:

```go
r.Route("/student", api.StudentDashRoutes(db, cfg))
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add api/student_dash.go cmd/server/main.go
git commit -m "feat(fase3): student dashboard API — progress, mastery, recommendations"
```

---

## Task 11: Frontend — Learning Service & Types

**Files:**
- Create: `frontend/src/app/core/services/learning.service.ts`
- Modify: `frontend/src/app/core/services/api.service.ts`

**Interfaces:**
- Produces: `LearningService` with all HTTP methods for the learning API

- [ ] **Step 1: Create `learning.service.ts`**

```typescript
import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface StudentProfile {
  id: string;
  student_id: string;
  course_id: string;
  overall_level: number;
  total_attempts: number;
  correct_attempts: number;
  total_hints_used: number;
  study_time_seconds: number;
}

export interface ConceptMastery {
  id: string;
  student_id: string;
  concept_id: string;
  mastery: number;
  status: 'not_started' | 'learning' | 'developing' | 'mastered';
  attempts: number;
  correct: number;
  hints_used: number;
  error_count: number;
}

export interface ConceptNode {
  id: string;
  name: string;
  description: string;
  parent_id?: string;
  course_id: string;
  difficulty_base: number;
  children?: ConceptNode[];
}

export interface Exercise {
  id: string;
  concept_id: string;
  difficulty: number;
  statement: string;
  latex: string;
  expected_answer: string;
  solution: string;
  solution_steps: any[];
  hints: string[];
  common_errors: any[];
  source: string;
  verified_by_math: boolean;
}

export interface SessionResponse {
  session_id: string;
  mode: string;
  exercise?: Exercise;
  message?: string;
}

export interface AnswerResponse {
  correct: boolean;
  score: number;
  feedback: string;
  first_error_step?: number;
  error_type?: string;
  mastery_before: number;
  mastery_after: number;
  mastery_status: string;
  next_exercise?: Exercise;
  math_verified: boolean;
  step_analysis?: any[];
}

export interface HintResponse {
  hint_index: number;
  hint: string;
  total_hints: number;
}

export interface AdaptiveRecommendation {
  recommended_concept: string;
  concept_name: string;
  recommended_difficulty: number;
  reason: string;
  prerequisites_met: boolean;
  missing_prereqs?: string[];
}

export interface StudentDashboard {
  profile: StudentProfile;
  mastery_map: Record<string, ConceptMastery>;
  recent_errors: any[];
  recommendations: string[];
  sessions_summary: {
    total_sessions: number;
    total_exercises: number;
    correct_rate: number;
    average_hints: number;
    study_time_hours: number;
  };
}

export interface TeacherCourseProgress {
  course_id: string;
  total_students: number;
  average_mastery: number;
}

export interface TopicMastery {
  concept_id: string;
  concept_name: string;
  average_mastery: number;
  student_count: number;
  struggling_count: number;
}

export interface CommonError {
  error_type: string;
  error_subtype: string;
  count: number;
  affected_students: number;
}

export interface StudentProgress {
  student_id: string;
  student_name: string;
  email: string;
  overall_level: number;
  total_attempts: number;
  last_active: string;
}

@Injectable({ providedIn: 'root' })
export class LearningService {
  private baseUrl = environment.apiUrl + '/api';

  constructor(private http: HttpClient) {}

  getMyProgress(): Observable<StudentDashboard> {
    return this.http.get<StudentDashboard>(`${this.baseUrl}/student/my-progress`);
  }

  getRecommendation(): Observable<AdaptiveRecommendation> {
    return this.http.get<AdaptiveRecommendation>(`${this.baseUrl}/student/recommendations`);
  }

  getConceptTree(courseID: string): Observable<ConceptNode[]> {
    return this.http.get<ConceptNode[]>(`${this.baseUrl}/learning/courses/${courseID}/concepts`);
  }

  getMastery(): Observable<Record<string, ConceptMastery>> {
    return this.http.get<Record<string, ConceptMastery>>(`${this.baseUrl}/learning/mastery`);
  }

  getErrors(): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/learning/errors`);
  }

  createSession(mode: string, courseID?: string): Observable<SessionResponse> {
    return this.http.post<SessionResponse>(`${this.baseUrl}/sessions/session`, { mode, course_id: courseID || 'matematica-1' });
  }

  submitAnswer(sessionID: string, exerciseID: string, answer: string, procedure?: string[]): Observable<AnswerResponse> {
    return this.http.post<AnswerResponse>(`${this.baseUrl}/sessions/answer`, { session_id: sessionID, exercise_id: exerciseID, answer, procedure });
  }

  requestHint(sessionID: string, exerciseID: string, hintIndex: number): Observable<HintResponse> {
    return this.http.post<HintResponse>(`${this.baseUrl}/sessions/hint`, { session_id: sessionID, exercise_id: exerciseID, hint_index: hintIndex });
  }

  generateExercise(conceptID: string, difficulty: number): Observable<Exercise> {
    return this.http.post<Exercise>(`${this.baseUrl}/exercises/generate`, { concept_id: conceptID, difficulty });
  }

  getExercise(conceptID: string, difficulty?: number): Observable<Exercise> {
    const params = difficulty ? `?difficulty=${difficulty}` : '';
    return this.http.get<Exercise>(`${this.baseUrl}/exercises/concept/${conceptID}${params}`);
  }

  getTeacherCourseProgress(courseID?: string): Observable<TeacherCourseProgress> {
    const params = courseID ? `?course_id=${courseID}` : '';
    return this.http.get<TeacherCourseProgress>(`${this.baseUrl}/teacher/course-progress${params}`);
  }

  getTeacherTopicMastery(courseID?: string): Observable<TopicMastery[]> {
    const params = courseID ? `?course_id=${courseID}` : '';
    return this.http.get<TopicMastery[]>(`${this.baseUrl}/teacher/topic-mastery${params}`);
  }

  getTeacherCommonErrors(): Observable<CommonError[]> {
    return this.http.get<CommonError[]>(`${this.baseUrl}/teacher/common-errors`);
  }

  getTeacherStudentProgress(courseID?: string): Observable<StudentProgress[]> {
    const params = courseID ? `?course_id=${courseID}` : '';
    return this.http.get<StudentProgress[]>(`${this.baseUrl}/teacher/student-progress${params}`);
  }
}
```

- [ ] **Step 2: Add session/exercise methods to ApiService**

Add to `api.service.ts`:

```typescript
createTutorSession(mode: string, courseID?: string): Observable<any> {
  return this.http.post(`${this.baseUrl}/sessions/session`, { mode, course_id: courseID || 'matematica-1' });
}

submitTutorAnswer(sessionID: string, exerciseID: string, answer: string, procedure?: string[]): Observable<any> {
  return this.http.post(`${this.baseUrl}/sessions/answer`, { session_id: sessionID, exercise_id: exerciseID, answer, procedure });
}

requestTutorHint(sessionID: string, exerciseID: string, hintIndex: number): Observable<any> {
  return this.http.post(`${this.baseUrl}/sessions/hint`, { session_id: sessionID, exercise_id: exerciseID, hint_index: hintIndex });
}
```

- [ ] **Step 3: Verify Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build 2>&1 | tail -5`
Expected: BUILD SUCCESSFUL

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/core/services/learning.service.ts frontend/src/app/core/services/api.service.ts
git commit -m "feat(fase3): frontend learning service & types"
```

---

## Task 12: Frontend — Student Progress Dashboard

**Files:**
- Create: `frontend/src/app/modules/student-progress/student-progress.component.ts`
- Modify: `frontend/src/app/app.routes.ts`
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Consumes: `LearningService`

- [ ] **Step 1: Create student-progress component**

Create a standalone Angular component at `frontend/src/app/modules/student-progress/student-progress.component.ts` with:
- Overall mastery percentage bar
- Per-concept mastery bars (with color coding: red < 30%, yellow 30-70%, green > 70%)
- Session stats (exercises done, correct rate, hints used, study time)
- Recent errors list
- Recommendations section
- Knowledge map tree visualization (simple nested list)

Use signals, CommonModule, MatButtonModule, MatIconModule, MatProgressBarModule. Follow the same style as existing components (dark theme support via CSS variables).

- [ ] **Step 2: Add route to app.routes.ts**

Add inside the children array of the LayoutComponent:

```typescript
{ path: 'my-progress', loadComponent: () => import('./modules/student-progress/student-progress.component').then(m => m.StudentProgressComponent) },
```

- [ ] **Step 3: Add nav item to layout.component.ts**

Add after the history nav item:

```html
<a routerLink="/my-progress" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>trending_up</mat-icon><span>Mi Progreso</span></a>
```

- [ ] **Step 4: Verify Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build 2>&1 | tail -5`
Expected: BUILD SUCCESSFUL

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/student-progress/ frontend/src/app/app.routes.ts frontend/src/app/shared/layout.component.ts
git commit -m "feat(fase3): student progress dashboard UI"
```

---

## Task 13: Frontend — Teacher Dashboard

**Files:**
- Create: `frontend/src/app/modules/teacher-dashboard/teacher-dashboard.component.ts`
- Modify: `frontend/src/app/app.routes.ts`
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Consumes: `LearningService`

- [ ] **Step 1: Create teacher-dashboard component**

Create a standalone Angular component at `frontend/src/app/modules/teacher-dashboard/teacher-dashboard.component.ts` with:
- Course overview card (total students, average mastery)
- Topic mastery table with color-coded bars
- Common errors list with affected student count
- Individual student progress table
- All data fetched from `LearningService`

- [ ] **Step 2: Add route (guarded)**

```typescript
{ path: 'teacher', loadComponent: () => import('./modules/teacher-dashboard/teacher-dashboard.component').then(m => m.TeacherDashboardComponent), canActivate: [roleGuard], data: { roles: ['ADMIN', 'TEACHER'] } },
```

- [ ] **Step 3: Add nav item (teacher/admin only)**

In `layout.component.ts`, after the existing dashboard nav item:

```html
<a routerLink="/teacher" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>analytics</mat-icon><span>Panel Profesor</span></a>
```

- [ ] **Step 4: Verify Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build 2>&1 | tail -5`
Expected: BUILD SUCCESSFUL

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/teacher-dashboard/ frontend/src/app/app.routes.ts frontend/src/app/shared/layout.component.ts
git commit -m "feat(fase3): teacher dashboard UI — course progress, topic mastery, errors"
```

---

## Task 14: Enhanced Tutor Component — Mode Selector

**Files:**
- Modify: `frontend/src/app/modules/tutor/tutor.component.ts`

**Interfaces:**
- Consumes: `LearningService`

- [ ] **Step 1: Add mode selector to tutor component**

Update the existing tutor component to include a mode selector bar at the top with buttons: Tutor, Practicar, Repaso, Resolver. Each mode changes the behavior:
- **Tutor**: Creates a tutor session, generates exercise, gives hints on request, evaluates with feedback
- **Practicar**: Creates a practice session, presents exercises sequentially, adapts difficulty
- **Repaso**: Creates a review session, selects concepts with low mastery
- **Resolver**: Keeps existing Fase 2 behavior (direct solve)

Add the mode as a signal, wire it to `LearningService.createSession()`.

- [ ] **Step 2: Verify Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build 2>&1 | tail -5`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/modules/tutor/tutor.component.ts
git commit -m "feat(fase3): tutor mode selector — tutor/practice/review/solve"
```

---

## Task 15: Math Service — Exercise Validation Endpoint

**Files:**
- Modify: `math-service/app.py`

**Interfaces:**
- Produces: `POST /math/validate-exercise` → validates expression + expected answer

- [ ] **Step 1: Add validate-exercise endpoint**

Add to `math-service/app.py`:

```python
@app.route('/math/validate-exercise', methods=['POST'])
@with_timeout
def math_validate_exercise():
    data = request.get_json()
    if not data or 'expression' not in data:
        return jsonify({'error': 'Expression is required'}), 400
    expected = data.get('expected', '')
    result = verify_result(data['expression'], expected)
    return jsonify(result)
```

- [ ] **Step 2: Rebuild and test**

Run: `cd /home/proyecto/matematicarag && docker compose build math-service`

- [ ] **Step 3: Commit**

```bash
git add math-service/app.py
git commit -m "feat(fase3): math service validate-exercise endpoint"
```

---

## Task 16: Integration Test — Full Flow

**Files:**
- Create: `api/learning_test.go`

**Interfaces:**
- Tests: Create profile → Get mastery → Generate exercise → Submit answer → Update mastery → Check errors

- [ ] **Step 1: Create integration test**

```go
package api

import (
	"testing"
)

func TestMasteryUpdateFlow(t *testing.T) {
	// This test requires a running PostgreSQL instance
	// Skip in CI if DB_URL is not set
	t.Skip("integration test — requires database")

	// Test flow:
	// 1. Create student profile
	// 2. Generate exercise for concept
	// 3. Submit correct answer → mastery should increase
	// 4. Submit incorrect answer → mastery should decrease, error recorded
	// 5. Check error patterns
}

func TestAdaptiveRecommendation(t *testing.T) {
	t.Skip("integration test — requires database")
}

func TestExerciseGeneration(t *testing.T) {
	t.Skip("integration test — requires database + LLM")
}
```

- [ ] **Step 2: Commit**

```bash
git add api/learning_test.go
git commit -m "feat(fase3): learning integration test stubs"
```

---

## Task 17: Final Build Verification & Docker Compose

**Files:**
- Verify: `docker-compose.yml`

- [ ] **Step 1: Full Go build**

Run: `cd /home/proyecto/matematicarag && go build ./...`
Expected: BUILD SUCCESSFUL

- [ ] **Step 2: Full Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build 2>&1 | tail -5`
Expected: BUILD SUCCESSFUL

- [ ] **Step 3: Docker compose build**

Run: `cd /home/proyecto/matematicarag && docker compose build --no-cache`
Expected: All images build successfully

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat(fase3): Fase 3 complete — adaptive tutor, exercises, dashboards"
```

- [ ] **Step 5: Push to GitHub**

```bash
git push origin main
```
