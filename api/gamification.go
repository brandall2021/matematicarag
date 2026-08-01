package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
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
	Points           int             `json:"points"`
	Level            int             `json:"level"`
	LevelName        string          `json:"level_name"`
	NextLevelPoints  int             `json:"next_level_points"`
	CurrentStreak    int             `json:"current_streak"`
	BestStreak       int             `json:"best_streak"`
	Achievements     []Achievement   `json:"achievements"`
	RecentActivities []PointActivity `json:"recent_activities"`
}

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
			best_streak = GREATEST(student_streaks.best_streak, CASE
				WHEN student_streaks.last_active_date = $2 THEN student_streaks.current_streak
				WHEN student_streaks.last_active_date = ($2::date - INTERVAL '1 day')::date THEN student_streaks.current_streak + 1
				ELSE 1
			END),
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

				if err := CreateNotification(ctx, db, studentID, "achievement_unlocked",
					"¡Logro desbloqueado!",
					"Desbloqueaste el logro: "+a.Title,
					"/logros"); err != nil {
					log.Printf("[NOTIFY] achievement_unlocked failed: %v", err)
				}
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
		WHERE student_id = $1 AND percentage >= 1.0 AND status IN ('graded', 'returned')`, studentID).Scan(&n)
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
