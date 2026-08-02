package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMathClientPlot(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/math/plot" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var req plotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Expression != "x^2" || req.XMin != -5 || req.XMax != 5 {
			t.Fatalf("unexpected request: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"expression": "x**2",
			"variable":   "x",
			"xmin":       -5,
			"xmax":       5,
			"points": []interface{}{
				[]interface{}{-5.0, 25.0},
				[]interface{}{0.0, 0.0},
				[]interface{}{1.0, nil},
				[]interface{}{5.0, 25.0},
			},
		})
	}))
	defer ts.Close()

	client := ts.Client()
	client.Timeout = 5 * time.Second
	c := &MathClient{
		baseURL:        ts.URL,
		httpClient:     client,
		circuitBreaker: NewCircuitBreaker("math-client-plot-test", 5, 30*time.Second),
		retryCfg:       RetryConfig{MaxRetries: 0, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond},
	}

	result, err := c.Plot("x^2", -5, 5)
	if err != nil {
		t.Fatalf("Plot: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	if len(result.Points) != 4 {
		t.Fatalf("expected 4 points, got %d", len(result.Points))
	}
	// null y values must decode as nil so the frontend can break the curve
	if result.Points[2][1] != nil {
		t.Fatalf("expected null y to decode as nil, got %v", result.Points[2][1])
	}
}

func TestMathClientPlotPropagatesClientError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"expresión inválida"}`, http.StatusBadRequest)
	}))
	defer ts.Close()

	client := ts.Client()
	client.Timeout = 5 * time.Second
	c := &MathClient{
		baseURL:        ts.URL,
		httpClient:     client,
		circuitBreaker: NewCircuitBreaker("math-client-plot-error-test", 5, 30*time.Second),
		retryCfg:       RetryConfig{MaxRetries: 0, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond},
	}

	if _, err := c.Plot("2+", -10, 10); err == nil {
		t.Fatal("expected error")
	}
}
