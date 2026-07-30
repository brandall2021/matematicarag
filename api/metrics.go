package api

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricsCollector struct {
	mu              sync.RWMutex
	totalRequests   atomic.Int64
	activeRequests  atomic.Int64
	totalErrors     atomic.Int64
	totalDurationNs atomic.Int64

	requestCountByPath map[string]int64
	errorCountByPath   map[string]int64
	durationByPath     map[string]time.Duration
	startedAt          time.Time

	mathRequests       atomic.Int64
	mathErrors         atomic.Int64
	mathDurationMs     atomic.Int64
	ragRequests        atomic.Int64
	ragErrors          atomic.Int64
	openaiRequests     atomic.Int64
	openaiErrors       atomic.Int64
	openaiTotalTokens  atomic.Int64

	circuitBreakers []*CircuitBreaker
}

var globalMetrics = &MetricsCollector{
	requestCountByPath: make(map[string]int64),
	errorCountByPath:   make(map[string]int64),
	durationByPath:     make(map[string]time.Duration),
	startedAt:          time.Now(),
}

func init() {
	go func() {
		for range time.Tick(15 * time.Minute) {
			log.Printf("[METRICS] total=%d active=%d errors=%d math_req=%d math_err=%d",
				globalMetrics.totalRequests.Load(),
				globalMetrics.activeRequests.Load(),
				globalMetrics.totalErrors.Load(),
				globalMetrics.mathRequests.Load(),
				globalMetrics.mathErrors.Load())
		}
	}()
}

func RecordRequest(path string, duration time.Duration, isError bool) {
	m := globalMetrics
	m.totalRequests.Add(1)
	m.totalDurationNs.Add(duration.Nanoseconds())

	m.mu.Lock()
	m.requestCountByPath[path]++
	m.durationByPath[path] += duration
	if isError {
		m.totalErrors.Add(1)
		m.errorCountByPath[path]++
	}
	m.mu.Unlock()
}

func RecordActiveRequest(delta int) {
	if delta > 0 {
		globalMetrics.activeRequests.Add(1)
	} else {
		globalMetrics.activeRequests.Add(-1)
	}
}

func RecordMathRequest(durationMs int64, isError bool) {
	globalMetrics.mathRequests.Add(1)
	globalMetrics.mathDurationMs.Add(durationMs)
	if isError {
		globalMetrics.mathErrors.Add(1)
	}
}

func RecordRAGRequest(isError bool) {
	globalMetrics.ragRequests.Add(1)
	if isError {
		globalMetrics.ragErrors.Add(1)
	}
}

func RecordOpenAIRequest(tokens int, isError bool) {
	globalMetrics.openaiRequests.Add(1)
	globalMetrics.openaiTotalTokens.Add(int64(tokens))
	if isError {
		globalMetrics.openaiErrors.Add(1)
	}
}

func RegisterCircuitBreaker(cb *CircuitBreaker) {
	globalMetrics.mu.Lock()
	globalMetrics.circuitBreakers = append(globalMetrics.circuitBreakers, cb)
	globalMetrics.mu.Unlock()
}

func getMetricsData(db *pgxpool.Pool) map[string]any {
	m := globalMetrics
	m.mu.RLock()
	reqCount := make(map[string]int64)
	errCount := make(map[string]int64)
	avgDuration := make(map[string]string)
	for k, v := range m.requestCountByPath {
		reqCount[k] = v
	}
	for k, v := range m.errorCountByPath {
		errCount[k] = v
	}
	for k, d := range m.durationByPath {
		if reqCount[k] > 0 {
			avgDuration[k] = (d / time.Duration(reqCount[k])).String()
		}
	}
	cbs := make([]map[string]any, len(m.circuitBreakers))
	for i, cb := range m.circuitBreakers {
		cbs[i] = cb.Stats()
	}
	m.mu.RUnlock()

	totalReq := m.totalRequests.Load()
	totalErr := m.totalErrors.Load()
	errorRate := 0.0
	if totalReq > 0 {
		errorRate = float64(totalErr) / float64(totalReq) * 100
	}

	var dbAlive bool
	if db != nil {
		err := db.Ping(nil)
		dbAlive = err == nil
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]any{
		"uptime_seconds": time.Since(m.startedAt).Seconds(),
		"requests": map[string]any{
			"total":          totalReq,
			"active":         m.activeRequests.Load(),
			"errors":         totalErr,
			"error_rate_pct": errorRate,
			"by_path":        reqCount,
			"errors_by_path": errCount,
			"avg_duration":   avgDuration,
		},
		"services": map[string]any{
			"math": map[string]any{
				"requests":    m.mathRequests.Load(),
				"errors":      m.mathErrors.Load(),
				"total_ms":    m.mathDurationMs.Load(),
			},
			"rag": map[string]any{
				"requests": m.ragRequests.Load(),
				"errors":   m.ragErrors.Load(),
			},
			"openai": map[string]any{
				"requests": m.openaiRequests.Load(),
				"errors":   m.openaiErrors.Load(),
				"tokens":   m.openaiTotalTokens.Load(),
			},
		},
		"circuit_breakers": cbs,
		"database": map[string]any{
			"alive": dbAlive,
		},
		"memory": map[string]any{
			"alloc_mb":       memStats.Alloc / 1024 / 1024,
			"total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
			"sys_mb":         memStats.Sys / 1024 / 1024,
			"gc_cycles":      memStats.NumGC,
			"goroutines":     runtime.NumGoroutine(),
		},
	}
}

func MetricsRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			data := getMetricsData(db)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
		})
	}
}

type MetricsMiddleware struct{}

func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{}
}

func (mm *MetricsMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RecordActiveRequest(1)
		defer RecordActiveRequest(-1)

		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		isError := ww.statusCode >= 500
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path = path + "?" + r.URL.RawQuery
		}
		RecordRequest(path, duration, isError)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
