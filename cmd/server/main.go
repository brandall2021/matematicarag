package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/brandall2021/matematicarag/api"
	"github.com/brandall2021/matematicarag/api/adaptive"
	"github.com/brandall2021/matematicarag/api/agent"
	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/brandall2021/matematicarag/internal/database"
	"github.com/brandall2021/matematicarag/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

var startTime time.Time

func main() {
	startTime = time.Now()
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	metricsMW := api.NewMetricsMiddleware()

	apiRouter := chi.NewRouter()

	apiRouter.Use(chimw.RequestID)
	apiRouter.Use(chimw.RealIP)
	apiRouter.Use(chimw.Logger)
	apiRouter.Use(chimw.Recoverer)
	apiRouter.Use(chimw.Timeout(60 * time.Second))
	apiRouter.Use(middleware.CORS(cfg.CORSOriginsList()))
	apiRouter.Use(middleware.RateLimit(120))
	apiRouter.Use(metricsMW.Wrap)

	mathClient := api.NewMathClient(cfg)

	agentCfg := agent.DefaultAgentConfig()
	agentCfg.MaxToolCalls = cfg.AgentMaxToolCalls
	agentCfg.MaxRetries = cfg.AgentMaxRetries
	agentCfg.IntentThreshold = cfg.AgentIntentThreshold
	agentCfg.LowMastery = cfg.AgentLowMastery
	agentCfg.HighMastery = cfg.AgentHighMastery

	adaptCfg := &adaptive.AdaptiveConfig{
		MasteryOldWeight:      cfg.MasteryOldWeight,
		MasteryEvidenceWeight: cfg.MasteryEvidenceWeight,
		MasteryHintPenalty:    cfg.MasteryHintPenalty,
		MasteryErrorPenalty:   cfg.MasteryErrorPenalty,
		MasteryRecencyFactor:  cfg.MasteryRecencyFactor,
		CriticalThreshold:     cfg.LearningCriticalThreshold,
		BeginnerThreshold:     cfg.LearningBeginnerThreshold,
		DevelopingThreshold:   cfg.LearningDevelopingThreshold,
		CompetentThreshold:    cfg.LearningCompetentThreshold,
		MaxDifficulty:         cfg.AdaptiveMaxDifficulty,
	}
	adaptEngine := adaptive.NewAdaptiveEngine(db, adaptCfg)

	toolDeps := api.BuildAgentToolDependencies(db, cfg, mathClient, adaptEngine)
	agentRegistry := agent.NewToolRegistry()
	agent.RegisterAllTools(agentRegistry, toolDeps)

	pedagogicalAgent := agent.NewPedagogicalAgent(
		db,
		&agentCfg,
		agentRegistry,
		func(ctx context.Context, prompt string) (string, error) {
			return api.CallLLMForAgent(ctx, db, prompt)
		},
		adaptEngine,
	)

	apiRouter.Route("/api", func(r chi.Router) {
		r.Route("/auth", api.AuthRoutes(db, cfg))
		r.Route("/chat", api.ChatRoutes(db, cfg))
		r.Route("/rag", api.RagRoutes(db, cfg))
		r.Route("/math", api.MathRoutes(db, cfg))
		r.Route("/documents", api.DocumentRoutes(db, cfg))
		r.Route("/analytics", api.AnalyticsRoutes(db))
		r.Route("/indexer", api.IndexerRoutes(db, cfg))
		r.Route("/tutor", api.TutorRoutes(db, cfg))

		r.Group(func(r chi.Router) {
			r.Use(api.AuthMiddleware(cfg.JWTSecret))
			r.Route("/student", api.StudentRoutes(db))
			r.Route("/dashboard/student", api.StudentDashboardRoutes(db))
			r.Route("/history", api.HistoryRoutes(db))
			r.Route("/users", api.UserRoutes(db))
			r.Route("/learning", api.LearningRoutes(db, adaptEngine))
			r.Route("/concepts", api.KnowledgeRoutes(db))
			r.Route("/exercises", api.ExerciseRoutes(db, cfg))
			r.Route("/sessions", api.SessionRoutes(db, cfg))
			r.Route("/assessments", api.AssessmentRoutes(db, cfg))
			r.Route("/grading", api.GradeRoutes(db, cfg))
			r.Route("/questions", api.QuestionRoutes(db, cfg))
			r.Route("/analytics/v2", api.AnalyticsV2Routes(db, cfg))
			r.Route("/recovery", api.RecoveryRoutes(db, cfg))
			r.Route("/alerts", api.AlertRoutes(db, cfg))
			r.Route("/export", api.ExportRoutes(db))
			r.Route("/audit", api.AuditRoutes(db))
		})

		r.Group(func(r chi.Router) {
			r.Use(api.AuthMiddleware(cfg.JWTSecret))
			r.Use(api.RoleMiddleware("ADMIN"))
			r.Route("/settings", api.SettingsRoutes(db))
			r.Route("/stats", api.StatsRoutes(db))
		})

		r.Group(func(r chi.Router) {
			r.Use(api.AuthMiddleware(cfg.JWTSecret))
			r.Use(api.RoleMiddleware("TEACHER", "ADMIN"))
			r.Route("/teacher", api.TeacherRoutes(db))
			r.Route("/teacher/critical-concepts", api.CriticalConceptsRoutes(db))
			r.Route("/dashboard/teacher", api.TeacherDashboardRoutes(db))
		})

		r.Group(func(r chi.Router) {
			r.Use(api.AuthMiddleware(cfg.JWTSecret))
			r.Route("/agent", api.AgentRoutes(db, pedagogicalAgent))
		})

		r.Route("/metrics", api.MetricsRoutes(db))
	})

	apiRouter.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		mathAlive := mathClient.HealthCheck()
		var dbAlive bool
		err := db.Ping(r.Context())
		dbAlive = err == nil

		qdrantBody, qdrantErr := api.QdrantHealthCheck()
		qdrantAlive := qdrantErr == nil && qdrantBody != nil

		status := "ok"
		if !dbAlive {
			status = "degraded"
		} else if !mathAlive {
			status = "degraded"
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"services": map[string]string{
				"database": map[bool]string{true: "up", false: "down"}[dbAlive],
				"math":     map[bool]string{true: "up", false: "down"}[mathAlive],
				"qdrant":   map[bool]string{true: "up", false: "down"}[qdrantAlive],
			},
			"uptime_seconds": time.Since(startTime).Seconds(),
		})
	})

	staticDir := "./static"
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		staticDir = dir
	}

	log.Printf("Serving static files from %s", staticDir)

	handler := spaHandler(staticDir, apiRouter)

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func spaHandler(root string, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)

		if strings.HasPrefix(path, "/api/") || path == "/health" {
			fallback.ServeHTTP(w, r)
			return
		}

		if path == "/" || path == "" {
			path = "/index.html"
		}

		fullPath := filepath.Join(root, path)

		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, fullPath)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	}
}
