# Gamificación — Puntos, Rachas y Logros Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Agregar un sistema de gamificación: puntos por actividad, rachas de práctica diaria y logros desbloqueables, con tablero personal visible para el alumno y un endpoint para el docente.

**Architecture:** Tres tablas nuevas (`student_points`, `achievements`, `student_achievements`, `student_streaks`) se agregan al slice de migraciones en `internal/database/database.go:30`. Un nuevo archivo `api/gamification.go` define structs, reglas de puntos/logros y handlers bajo `/api/gamification/*` (montado en `cmd/server/main.go:101-119`, grupo autenticado). La concesión de puntos se dispara desde flujos existentes (resolver ejercicio, aprobar evaluación, sesión diaria) llamando a `RecordActivity`. El frontend agrega un servicio y un componente `gamification` con tarjeta de nivel, racha y grilla de logros.

**Tech Stack:** Go (chi, pgx), Angular 20 (standalone, signals), PostgreSQL.

## Global Constraints

- Regla de copy: español rioplatense con acentos (`Nivel`, `Puntos`, `Racha`, `Logros`, `¡` en exclamaciones).
- No alterar el comportamiento de math/chat/assessment existente: solo se AÑADEN llamadas de registro después de que el flujo principal ya resolvió.
- Los puntos son por alumno (role `STUDENT`); ADMIN/TEACHER no acumulan.
- Dependencia única nueva: ninguna (solo Go stdlib y paquetes ya presentes).
- Verificación backend: `go build ./...` y `go test ./...` (desde `api/`). Frontend: `npm run build` (desde `frontend/`).
- Números de puntos: FÁCIL=5, MEDIO=10, DIFÍCIL=15 (configurable por env con defaults).
- El servicio de gamificación nunca debe hacer fallar el flujo principal: los errores se loguean y se ignoran.

---
### Task 1: Esquema de base de datos — tablas de gamificación

**Files:**
- Modify: `internal/database/database.go:27-40` (slice `migrations`)

**Interfaces:**
- Consumes: nada. Produce:
  - Tabla `student_points(student_id UUID, points INTEGER, source VARCHAR, concept_id UUID NULL, metadata JSONB, created_at)`
  - Tabla `achievements(id UUID, code VARCHAR UNIQUE, title VARCHAR, description VARCHAR, icon VARCHAR, criteria JSONB, points INTEGER, created_at)`
  - Tabla `student_achievements(student_id UUID, achievement_id UUID, created_at, UNIQUE(student_id, achievement_id))`
  - Tabla `student_streaks(student_id UUID UNIQUE, current_streak INTEGER, best_streak INTEGER, last_active_date DATE, updated_at)`
  - Índices: `idx_student_points_student`, `idx_student_achievements_student`.

- [ ] **Step 1: Agregar las tablas al slice de migraciones**

En `internal/database/database.go`, dentro del slice `migrations := []string{ ... }` (después de la tabla `math_step_results`, fin del slice), agregar:

```go
	`CREATE TABLE IF NOT EXISTS student_points (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		points INTEGER NOT NULL,
		source VARCHAR(100) NOT NULL,
		concept_id UUID REFERENCES concepts(id) ON DELETE SET NULL,
		metadata JSONB,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_student_points_student ON student_points(student_id)`,
	`CREATE TABLE IF NOT EXISTS achievements (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		code VARCHAR(100) UNIQUE NOT NULL,
		title VARCHAR(255) NOT NULL,
		description TEXT NOT NULL,
		icon VARCHAR(50) NOT NULL DEFAULT 'emoji_events',
		points INTEGER NOT NULL DEFAULT 0,
		criteria JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS student_achievements (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		achievement_id UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		UNIQUE(student_id, achievement_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_student_achievements_student ON student_achievements(student_id)`,
	`CREATE TABLE IF NOT EXISTS student_streaks (
		student_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		current_streak INTEGER NOT NULL DEFAULT 0,
		best_streak INTEGER NOT NULL DEFAULT 0,
		last_active_date DATE,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`,
```

- [ ] **Step 2: Verificar build**

Run: `go build ./...` (desde `api/`)
Expected: build exitoso.

- [ ] **Step 3: Commit**

```bash
git add internal/database/database.go
git commit -m "feat(gamification): add points, achievements and streaks tables"
```

### Task 2: api/gamification.go — structs, reglas y rutas

**Files:**
- Create: `api/gamification.go`

**Interfaces:**
- Consumes: `db *pgxpool.Pool`, `cfg *config.Config`, middleware de auth existente.
- Produce:
  - `type GamificationSummary { Points int; Level int; LevelName string; NextLevelPoints int; CurrentStreak int; BestStreak int; Achievements []Achievement; RecentActivities []PointActivity }`
  - `type Achievement { ID, Code, Title, Description, Icon string; Points int; Unlocked bool; UnlockedAt *string }`
  - `func GamificationRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router)` — montar en `cmd/server/main.go` grupo autenticado.
  - `func RecordActivity(ctx, db, studentID, source string, conceptID *string, points int, metadata map[string]any) error` — idempotente, no bloqueante.
  - `func TouchStreak(ctx, db, studentID string) error` — incrementa racha diaria.
  - `func CheckAchievements(ctx, db, studentID string) error` — evalúa y desbloquea logros pendientes.

- [ ] **Step 1: Crear el archivo con structs, lógica y handlers**

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PointActivity struct {
	Source    string          `json:"source"`
	Points    int             `json:"points"`
	CreatedAt string          `json:"created_at"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type Achievement struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Icon        string  `json:"icon"`
	Points      int     `json:"points"`
	Unlocked    bool    `json:"unlocked"`
	UnlockedAt  *string `json:"unlocked_at,omitempty"`
}

type GamificationSummary struct {
	Points            int               `json:"points"`
	Level             int               `json:"level"`
	LevelName         string            `json:"level_name"`
	NextLevelPoints   int               `json:"next_level_points"`
	CurrentStreak     int               `json:"current_streak"`
	BestStreak        int               `json:"best_streak"`
	Achievements      []Achievement     `json:"achievements"`
	RecentActivities  []PointActivity   `json:"recent_activities"`
}

func levelName(level int) string {
	switch {
	case level >= 10:
		return "Experto"
	case level >= 7:
		return "Avanzado"
	case level >= 4:
		return "Intermedio"
	case level >= 2:
		return "En camino"
	default:
		return "Iniciante"
	}
}

func levelFromPoints(points int) (int, int) {
	level := points/100 + 1
	next := level * 100
	return level, next
}

func RecordActivity(ctx context.Context, db *pgxpool.Pool, studentID, source string, conceptID *string, points int, metadata map[string]any) error {
	if points <= 0 {
		return nil
	}
	var meta []byte
	var err error
	if metadata != nil {
		meta, err = json.Marshal(metadata)
		if err != nil {
			meta = nil
		}
	}
	_, err = db.Exec(ctx,
		`INSERT INTO student_points (student_id, points, source, concept_id, metadata)
		 VALUES ($1, $2, $3, $4, $5)`,
		studentID, points, source, conceptID, meta)
	return err
}

func TouchStreak(ctx context.Context, db *pgxpool.Pool, studentID string) error {
	today := time.Now().Format("2006-01-02")
	_, err := db.Exec(ctx, `
		INSERT INTO student_streaks (student_id, current_streak, best_streak, last_active_date, updated_at)
		VALUES ($1, 1, 1, $2, NOW())
		ON CONFLICT (student_id) DO UPDATE SET
			current_streak = CASE
				WHEN student_streaks.last_active_date = $2 THEN student_streaks.current_streak
				WHEN student_streaks.last_active_date = ($2::date - INTERVAL '1 day')::date THEN student_streaks.current_streak + 1
				ELSE 1
			END,
			best_streak = GREATEST(student_streaks.best_streak, current_streak),
			last_active_date = $2,
			updated_at = NOW()`,
		studentID, today)
	if err != nil {
		return err
	}
	return CheckAchievements(ctx, db, studentID)
}

func CheckAchievements(ctx context.Context, db *pgxpool.Pool, studentID string) error {
	var streaks studentStreaksRow
	err := db.QueryRow(ctx,
		`SELECT current_streak FROM student_streaks WHERE student_id = $1`, studentID).
		Scan(&streaks.CurrentStreak)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	rows, err := db.Query(ctx, `
		SELECT a.id, a.code, a.title, a.description, a.icon, a.points, a.criteria,
		       sa.created_at IS NOT NULL AS unlocked
		FROM achievements a
		LEFT JOIN student_achievements sa ON sa.achievement_id = a.id AND sa.student_id = $1
		ORDER BY a.points ASC`, studentID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type achievementRow struct {
		ID, Code, Title, Description, Icon string
		Points                             int
		Criteria                           json.RawMessage
		Unlocked                           bool
	}
	var toUnlock []achievementRow
	for rows.Next() {
		var a achievementRow
		var criteria []byte
		if err := rows.Scan(&a.ID, &a.Code, &a.Title, &a.Description, &a.Icon, &a.Points, &criteria, &a.Unlocked); err != nil {
			continue
		}
		if !a.Unlocked {
			toUnlock = append(toUnlock, a)
		}
	}
	rows.Close()

	for _, a := range toUnlock {
		if achievementCriteriaMet(ctx, db, studentID, a.Code, streaks.CurrentStreak, a.Criteria) {
			_, err := db.Exec(ctx,
				`INSERT INTO student_achievements (student_id, achievement_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				studentID, a.ID)
			if err == nil {
				RecordActivity(ctx, db, studentID, "achievement:"+a.Code, nil, a.Points, map[string]any{"achievement_id": a.ID})
			}
		}
	}
	return nil
}

type studentStreaksRow struct {
	CurrentStreak int
}

func achievementCriteriaMet(ctx context.Context, db *pgxpool.Pool, studentID, code string, streak int, criteria json.RawMessage) bool {
	switch code {
	case "first_exercise":
		return firstExerciseDone(ctx, db, studentID)
	case "streak_3":
		return streak >= 3
	case "streak_7":
		return streak >= 7
	case "perfect_assessment":
		return hasPerfectAssessment(ctx, db, studentID)
	case "concept_mastered":
		return hasMasteredConcept(ctx, db, studentID)
	case "ten_exercises":
		return exerciseCount(ctx, db, studentID) >= 10
	default:
		return false
	}
}

func firstExerciseDone(ctx context.Context, db *pgxpool.Pool, studentID string) bool {
	var n int
	db.QueryRow(ctx, `SELECT COUNT(*) FROM exercise_attempts WHERE student_id = $1`, studentID).Scan(&n)
	return n > 0
}

func exerciseCount(ctx context.Context, db *pgxpool.Pool, studentID string) int {
	var n int
	db.QueryRow(ctx, `SELECT COUNT(*) FROM exercise_attempts WHERE student_id = $1`, studentID).Scan(&n)
	return n
}

func hasPerfectAssessment(ctx context.Context, db *pgxpool.Pool, studentID string) bool {
	var n int
	db.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_assessments
		WHERE student_id = $1 AND score >= 100 AND status = 'completed'`, studentID).Scan(&n)
	return n > 0
}

func hasMasteredConcept(ctx context.Context, db *pgxpool.Pool, studentID string) bool {
	var n int
	db.QueryRow(ctx, `
		SELECT COUNT(*) FROM concept_mastery
		WHERE student_id = $1 AND status = 'mastered'`, studentID).Scan(&n)
	return n > 0
}

func getSummary(ctx context.Context, db *pgxpool.Pool, studentID string) (*GamificationSummary, error) {
	var points int
	err := db.QueryRow(ctx,
		`SELECT COALESCE(SUM(points), 0) FROM student_points WHERE student_id = $1`, studentID).Scan(&points)
	if err != nil {
		return nil, err
	}

	var streak, best int
	err = db.QueryRow(ctx,
		`SELECT current_streak, best_streak FROM student_streaks WHERE student_id = $1`, studentID).
		Scan(&streak, &best)
	if err != nil {
		streak, best = 0, 0
	}

	rows, err := db.Query(ctx, `
		SELECT a.id, a.code, a.title, a.description, a.icon, a.points,
		       sa.created_at IS NOT NULL AS unlocked, sa.created_at
		FROM achievements a
		LEFT JOIN student_achievements sa ON sa.achievement_id = a.id AND sa.student_id = $1
		ORDER BY a.points ASC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	achievements := []Achievement{}
	for rows.Next() {
		var a Achievement
		var unlocked bool
		var unlockedAt *time.Time
		if err := rows.Scan(&a.ID, &a.Code, &a.Title, &a.Description, &a.Icon, &a.Points, &unlocked, &unlockedAt); err != nil {
			continue
		}
		a.Unlocked = unlocked
		if unlockedAt != nil {
			ts := unlockedAt.Format(time.RFC3339)
			a.UnlockedAt = &ts
		}
		achievements = append(achievements, a)
	}

	actRows, err := db.Query(ctx, `
		SELECT source, points, created_at, COALESCE(metadata, '{}'::jsonb)
		FROM student_points WHERE student_id = $1
		ORDER BY created_at DESC LIMIT 10`, studentID)
	if err != nil {
		return nil, err
	}
	defer actRows.Close()

	activities := []PointActivity{}
	for actRows.Next() {
		var act PointActivity
		var created time.Time
		if err := actRows.Scan(&act.Source, &act.Points, &created, &act.Metadata); err != nil {
			continue
		}
		act.CreatedAt = created.Format(time.RFC3339)
		activities = append(activities, act)
	}

	level, next := levelFromPoints(points)
	return &GamificationSummary{
		Points:           points,
		Level:            level,
		LevelName:        levelName(level),
		NextLevelPoints:  next,
		CurrentStreak:    streak,
		BestStreak:       best,
		Achievements:     achievements,
		RecentActivities: activities,
	}, nil
}

func GamificationRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			summary, err := getSummary(r.Context(), db, studentID)
			if err != nil {
				http.Error(w, `{"error":"failed to get gamification summary"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(summary)
		})
	}
}
```

- [ ] **Step 2: Agregar imports necesarios y ajustar**

En el archivo agregar al import: `"github.com/go-chi/chi/v5"` y `"github.com/brandall2021/matematicarag/internal/config"`. Verificar que `UserIDKey` se resuelve (definido en `api/middleware.go`).

- [ ] **Step 3: Verificar build**

Run: `go build ./...` (desde `api/`)
Expected: build exitoso.

- [ ] **Step 4: Commit**

```bash
git add api/gamification.go
git commit -m "feat(gamification): add gamification service with summary endpoint"
```

### Task 3: Montar rutas y sembrar logros iniciales

**Files:**
- Modify: `cmd/server/main.go:101-119` (grupo autenticado)
- Create: `api/gamification_seed.go` (o función `SeedAchievements` en `api/gamification.go`)

**Interfaces:**
- Consumes: `GamificationRoutes` (Task 2). Produce: `/api/gamification/me` accesible con token; logros base insertados idempotentemente.

- [ ] **Step 1: Montar las rutas**

En `cmd/server/main.go`, dentro del grupo autenticado (líneas 101-119), después de `r.Route("/alerts", api.AlertRoutes(db, cfg))` agregar:

```go
			r.Route("/gamification", api.GamificationRoutes(db, cfg))
```

- [ ] **Step 2: Agregar seed de logros**

En `api/gamification.go`, agregar:

```go
func SeedAchievements(ctx context.Context, db *pgxpool.Pool) error {
	achievements := []struct {
		Code, Title, Description, Icon string
		Points                         int
	}{
		{"first_exercise", "Primer paso", "Resolviste tu primer ejercicio.", "check_circle", 10},
		{"ten_exercises", "En racha", "Resolviste 10 ejercicios.", "fitness_center", 25},
		{"streak_3", "Constancia", "Practicaste 3 días seguidos.", "local_fire_department", 30},
		{"streak_7", "Semana completa", "Practicaste 7 días seguidos.", "whatshot", 75},
		{"perfect_assessment", "Perfecto", "Aprobaste una evaluación con 100 puntos.", "stars", 50},
		{"concept_mastered", "Dominio", "Llegaste a nivel 'mastered' en un concepto.", "military_tech", 60},
	}
	for _, a := range achievements {
		_, err := db.Exec(ctx, `
			INSERT INTO achievements (code, title, description, icon, points, criteria)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (code) DO NOTHING`,
			a.Code, a.Title, a.Description, a.Icon, a.Points, json.RawMessage(`{"code":"`+a.Code+`"}`))
		if err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 3: Llamar el seed en el arranque**

En `cmd/server/main.go`, después de `database.Migrate(db)` (línea 37-39) agregar:

```go
	if err := api.SeedAchievements(context.Background(), db); err != nil {
		log.Printf("WARN: failed to seed achievements: %v", err)
	}
```

- [ ] **Step 4: Verificar build**

Run: `go build ./...` (desde `api/`)
Expected: build exitoso.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go api/gamification.go
git commit -m "feat(gamification): mount routes and seed initial achievements"
```

### Task 4: Registrar puntos desde los flujos existentes

**Files:**
- Modify: `api/sessions.go` (o el archivo que maneja la resolución de ejercicios — verificar nombre de función)
- Modify: `api/grading.go` (aprobación de evaluación)
- Modify: `api/exercises.go` (intento de ejercicio)

**Interfaces:**
- Consumes: `RecordActivity`, `TouchStreak`, `CheckAchievements` (Task 2). Produce: puntos y rachas acumuladas al resolver ejercicio o aprobar evaluación.

- [ ] **Step 1: Localizar el punto de resolución de ejercicio**

Run: `grep -n 'func.*Exercise\|func.*Attempt\|INSERT INTO exercise_attempts' api/sessions.go api/exercises.go api/math_evaluator.go`
Expected: la función donde se persiste un intento resuelto por un estudiante (p.ej. una función `SaveAttempt` o el handler de `POST /exercises/{id}/attempt`).

En esa función, después de la inserción exitosa del intento, agregar (ajustar `studentID` a la variable real en scope):

```go
	go func() {
		ctx2 := context.Background()
		points := 10
		if difficulty == 5 {
			points = 15
		}
		if isCorrect {
			_ = RecordActivity(ctx2, db, studentID, "exercise_solved", conceptID, points, map[string]any{"difficulty": difficulty})
		} else {
			_ = RecordActivity(ctx2, db, studentID, "exercise_attempt", conceptID, 2, map[string]any{"difficulty": difficulty})
		}
		_ = TouchStreak(ctx2, db, studentID)
	}()
```

> Nota: verificar los nombres reales de las columnas/variables del archivo (p.ej. `conceptID`, `isCorrect`, `difficulty`) y ajustarlos. Si el flujo usa el motor adaptativo (`api/adaptive/engine.go` → `RecordEvent`), también se puede llamar `RecordActivity` allí.

- [ ] **Step 2: Registrar puntos al aprobar una evaluación**

En `api/grading.go`, donde se guarda el resultado calificado de una evaluación (función de calificación automática), después de persistir la nota, agregar:

```go
	if isPassing {
		_ = RecordActivity(ctx, db, studentID, "assessment_passed", nil, 30, map[string]any{"assessment_id": assessmentID})
		_ = CheckAchievements(ctx, db, studentID)
	}
```

> Ajustar `isPassing`, `studentID`, `assessmentID` a los nombres reales del archivo.

- [ ] **Step 3: Verificar build y tests**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: build y tests OK (las nuevas llamadas son no-bloqueantes y loguean silenciosamente).

- [ ] **Step 4: Commit**

```bash
git add api/sessions.go api/exercises.go api/grading.go
git commit -m "feat(gamification): record points and streaks from exercise and assessment flows"
```

### Task 5: Frontend — servicio y componente de gamificación

**Files:**
- Create: `frontend/src/app/core/services/gamification.service.ts`
- Create: `frontend/src/app/modules/gamification/gamification.component.ts`
- Modify: `frontend/src/app/app.routes.ts:6-31`
- Modify: `frontend/src/app/shared/layout.component.ts` (entrada en el menú)

**Interfaces:**
- Consumes: `GET /api/gamification/me` (Task 3). Produce:
  - `GamificationService.getSummary(): Observable<GamificationSummary>`
  - Componente standalone `gamification` con tarjeta de nivel/racha y grilla de logros.

- [ ] **Step 1: Crear el servicio**

```typescript
import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface Achievement {
  id: string;
  code: string;
  title: string;
  description: string;
  icon: string;
  points: number;
  unlocked: boolean;
  unlocked_at?: string;
}

export interface GamificationSummary {
  points: number;
  level: number;
  level_name: string;
  next_level_points: number;
  current_streak: number;
  best_streak: number;
  achievements: Achievement[];
  recent_activities: { source: string; points: number; created_at: string; metadata?: any }[];
}

@Injectable({ providedIn: 'root' })
export class GamificationService {
  private baseUrl = environment.apiUrl + '/api';

  constructor(private http: HttpClient) {}

  getSummary(): Observable<GamificationSummary> {
    return this.http.get<GamificationSummary>(`${this.baseUrl}/gamification/me`);
  }
}
```

- [ ] **Step 2: Crear el componente**

```typescript
import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { GamificationService, GamificationSummary } from '../../core/services/gamification.service';

@Component({
  selector: 'app-gamification',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatProgressBarModule],
  template: `
    <div class="gamification-container">
      <h1>Logros</h1>

      @if (summary(); as s) {
        <div class="stats-grid">
          <div class="stat-card">
            <mat-icon>stars</mat-icon>
            <div class="stat-value">{{ s.points }}</div>
            <div class="stat-label">Puntos</div>
          </div>
          <div class="stat-card">
            <mat-icon>emoji_events</mat-icon>
            <div class="stat-value">{{ s.level }} — {{ s.level_name }}</div>
            <div class="stat-label">Nivel</div>
          </div>
          <div class="stat-card">
            <mat-icon>local_fire_department</mat-icon>
            <div class="stat-value">{{ s.current_streak }}</div>
            <div class="stat-label">Racha actual</div>
          </div>
          <div class="stat-card">
            <mat-icon>trending_up</mat-icon>
            <div class="stat-value">{{ s.best_streak }}</div>
            <div class="stat-label">Mejor racha</div>
          </div>
        </div>

        <div class="level-bar">
          <div class="level-label">Progreso al nivel {{ s.level + 1 }}</div>
          <mat-progress-bar mode="determinate"
            [value]="s.points - ((s.level - 1) * 100)"
            max="100"></mat-progress-bar>
          <div class="level-hint">{{ s.points }} / {{ s.next_level_points }} puntos</div>
        </div>

        <h2>Medallas</h2>
        <div class="achievements-grid">
          @for (a of s.achievements; track a.id) {
            <div class="achievement-card" [class.locked]="!a.unlocked">
              <mat-icon>{{ a.icon }}</mat-icon>
              <div class="ach-title">{{ a.title }}</div>
              <div class="ach-desc">{{ a.description }}</div>
              @if (a.unlocked) {
                <div class="ach-points">+{{ a.points }} pts</div>
              } @else {
                <div class="ach-locked-label">Bloqueado</div>
              }
            </div>
          }
        </div>

        @if (s.recent_activities.length > 0) {
          <h2>Actividad reciente</h2>
          <div class="activity-list">
            @for (act of s.recent_activities; track act.created_at) {
              <div class="activity-item">
                <span>{{ act.source.replace('_', ' ') }}</span>
                <span class="activity-points">+{{ act.points }}</span>
              </div>
            }
          </div>
        }
      } @else {
        <div class="empty">Cargando tus logros...</div>
      }
    </div>
  `,
  styles: [`
    .gamification-container { padding: 1.5rem; max-width: 900px; margin: 0 auto; }
    .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 1rem; margin: 1rem 0; }
    .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 1rem; text-align: center; }
    .stat-card mat-icon { color: var(--accent); font-size: 28px; width: 28px; height: 28px; }
    .stat-value { font-size: 1.4rem; font-weight: 700; margin-top: 0.25rem; }
    .stat-label { color: var(--text-secondary); font-size: 0.8rem; }
    .level-bar { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 1rem; margin-bottom: 1rem; }
    .level-label { font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 0.5rem; }
    .level-hint { font-size: 0.75rem; color: var(--text-tertiary); margin-top: 0.35rem; text-align: right; }
    .achievements-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 1rem; }
    .achievement-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 1rem; text-align: center; }
    .achievement-card mat-icon { font-size: 40px; width: 40px; height: 40px; color: var(--accent); }
    .achievement-card.locked { opacity: 0.45; filter: grayscale(1); }
    .ach-title { font-weight: 600; margin-top: 0.5rem; }
    .ach-desc { font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem; }
    .ach-points { font-size: 0.8rem; color: var(--accent-text); font-weight: 600; margin-top: 0.5rem; }
    .ach-locked-label { font-size: 0.7rem; color: var(--text-tertiary); margin-top: 0.5rem; }
    .activity-list { display: flex; flex-direction: column; gap: 0.4rem; }
    .activity-item { display: flex; justify-content: space-between; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 0.5rem 0.75rem; font-size: 0.85rem; }
    .activity-points { color: var(--accent-text); font-weight: 600; }
    .empty { color: var(--text-secondary); padding: 2rem; text-align: center; }
  `]
})
export class GamificationComponent implements OnInit {
  summary = signal<GamificationSummary | null>(null);

  constructor(private gamificationService: GamificationService) {}

  ngOnInit(): void {
    this.gamificationService.getSummary().subscribe({
      next: s => this.summary.set(s),
      error: () => this.summary.set(null),
    });
  }
}
```

- [ ] **Step 3: Agregar la ruta**

En `frontend/src/app/app.routes.ts`, en los children (después de `aprendizaje`, línea 27):

```typescript
      { path: 'logros', loadComponent: () => import('./modules/gamification/gamification.component').then(m => m.GamificationComponent) },
```

- [ ] **Step 4: Agregar entrada al menú**

En `frontend/src/app/shared/layout.component.ts`, en la sección "Aprendizaje" (después de la línea de `aprendizaje`, línea 47), agregar:

```html
          <a routerLink="/logros" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>emoji_events</mat-icon><span>Logros</span></a>
```

- [ ] **Step 5: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/app/core/services/gamification.service.ts frontend/src/app/modules/gamification frontend/src/app/app.routes.ts frontend/src/app/shared/layout.component.ts
git commit -m "feat(gamification): add gamification service, component, route and nav entry"
```

### Task 6: Verificación final

- [ ] **Step 1: Tests de backend**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: todo OK.

- [ ] **Step 2: Test de integración manual (opcional)**

Con el servidor y la BD levantados: iniciar sesión como alumno, resolver un ejercicio en el Tutor, ir a `/logros`.
Expected: puntos acumulados, nivel calculado, logro "Primer paso" desbloqueado, racha = 1.

- [ ] **Step 3: Build frontend**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.
