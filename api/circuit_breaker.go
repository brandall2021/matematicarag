package api

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed   CircuitState = iota
	StateHalfOpen CircuitState = iota
	StateOpen     CircuitState = iota
)

type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitState
	failureCount    int
	lastFailureTime time.Time
	threshold       int
	cooldown        time.Duration
	halfOpenMax     int
	halfOpenCount   int
	name            string
}

func NewCircuitBreaker(name string, threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:       StateClosed,
		threshold:   threshold,
		cooldown:    cooldown,
		halfOpenMax: 3,
		name:        name,
	}
}

func (cb *CircuitBreaker) wrapHTTPClient(client *http.Client) *http.Client {
	wrapped := *client
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}
	wrapped.Transport = &circuitBreakerRoundTripper{
		original: client.Transport,
		cb:       cb,
		client:   client,
	}
	return &wrapped
}

type circuitBreakerRoundTripper struct {
	original http.RoundTripper
	cb       *CircuitBreaker
	client   *http.Client
}

func (rt *circuitBreakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !rt.cb.Allow() {
		return nil, errors.New("service unavailable: circuit breaker open")
	}

	transport := rt.original
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		rt.cb.RecordFailure()
		return nil, err
	}

	if resp.StatusCode >= 500 {
		rt.cb.RecordFailure()
	} else {
		rt.cb.RecordSuccess()
	}

	return resp, nil
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	if state == StateClosed {
		return true
	}

	if state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.cooldown {
			cb.mu.Lock()
			if cb.state == StateOpen {
				cb.state = StateHalfOpen
				cb.halfOpenCount = 0
				log.Printf("[CIRCUIT] %s: open -> half-open", cb.name)
			}
			cb.mu.Unlock()
			return true
		}
		return false
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.halfOpenCount < cb.halfOpenMax {
		cb.halfOpenCount++
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failureCount = 0
		cb.halfOpenCount = 0
		log.Printf("[CIRCUIT] %s: half-open -> closed (recovered)", cb.name)
	}
	cb.failureCount = 0
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.halfOpenCount = 0
		log.Printf("[CIRCUIT] %s: half-open -> open (probe failed)", cb.name)
		return
	}

	if cb.state == StateClosed && cb.failureCount >= cb.threshold {
		cb.state = StateOpen
		log.Printf("[CIRCUIT] %s: closed -> open (%d failures)", cb.name, cb.failureCount)
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Stats() map[string]any {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	stateName := "closed"
	switch cb.state {
	case StateOpen:
		stateName = "open"
	case StateHalfOpen:
		stateName = "half-open"
	}
	return map[string]any{
		"name":              cb.name,
		"state":             stateName,
		"failure_count":     cb.failureCount,
		"threshold":         cb.threshold,
		"last_failure_ago":  time.Since(cb.lastFailureTime).String(),
	}
}

type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	JitterRatio float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:  3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		JitterRatio: 0.2,
	}
}

func retryWithBackoff(cfg RetryConfig, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := cfg.BaseDelay * (1 << (attempt - 1))
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
			jitter := time.Duration(float64(delay) * cfg.JitterRatio * (rand.Float64()*2 - 1))
			delay += jitter
			if delay < 0 {
				delay = 0
			}
			time.Sleep(delay)
		}

		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}
