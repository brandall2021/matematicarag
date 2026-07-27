package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
)

type MathClient struct {
	baseURL    string
	httpClient *http.Client
}

type MathResult struct {
	Success bool   `json:"success"`
	Result  string `json:"result,omitempty"`
	Latex   string `json:"latex,omitempty"`
	Error   string `json:"error,omitempty"`
}

type SolveResult struct {
	Success   bool        `json:"success"`
	Variables []string    `json:"variables,omitempty"`
	Solutions interface{} `json:"solutions,omitempty"`
	Latex     string      `json:"latex,omitempty"`
	Count     int         `json:"count,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type VerifyResult struct {
	Success       bool   `json:"success"`
	Verified      bool   `json:"verified"`
	Method        string `json:"method,omitempty"`
	Expected      string `json:"expected,omitempty"`
	Actual        string `json:"actual,omitempty"`
	LatexExpected string `json:"latex_expected,omitempty"`
	LatexActual   string `json:"latex_actual,omitempty"`
	Error         string `json:"error,omitempty"`
}

func NewMathClient(cfg *config.Config) *MathClient {
	timeout := time.Duration(cfg.MathTimeout) * time.Second
	return &MathClient{
		baseURL: cfg.MathServiceURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *MathClient) post(path string, body interface{}) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("[MATH] marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("[MATH] create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[MATH] timeout calling %s", path)
			return nil, fmt.Errorf("[MATH] request timeout after %s", c.httpClient.Timeout)
		}
		return nil, fmt.Errorf("[MATH] request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[MATH] read response: %w", err)
	}

	if resp.StatusCode == http.StatusRequestTimeout {
		log.Printf("[MATH] service returned 408 (timeout) for %s", path)
		return nil, fmt.Errorf("[MATH] math service timeout (408)")
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("[MATH] service returned %d for %s: %s", resp.StatusCode, path, string(bodyBytes))
		return nil, fmt.Errorf("[MATH] service error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

type evalRequest struct {
	Expression string `json:"expression"`
}

func (c *MathClient) Evaluate(expression string) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/evaluate", evalRequest{Expression: expression})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type diffRequest struct {
	Expression string `json:"expression"`
	Variable   string `json:"variable"`
}

func (c *MathClient) Differentiate(expression, variable string) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/differentiate", diffRequest{Expression: expression, Variable: variable})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type integrateRequest struct {
	Expression string  `json:"expression"`
	Variable   string  `json:"variable"`
	Lower      *string `json:"lower,omitempty"`
	Upper      *string `json:"upper,omitempty"`
}

func (c *MathClient) Integrate(expression, variable string, lower, upper *string) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/integrate", integrateRequest{
		Expression: expression,
		Variable:   variable,
		Lower:      lower,
		Upper:      upper,
	})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type limitRequest struct {
	Expression string `json:"expression"`
	Variable   string `json:"variable"`
	Point      string `json:"point"`
}

func (c *MathClient) Limit(expression, variable, point string) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/limit", limitRequest{
		Expression: expression,
		Variable:   variable,
		Point:      point,
	})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type solveRequest struct {
	Expression string `json:"expression"`
	Variable   string `json:"variable"`
}

func (c *MathClient) Solve(expression, variable string) (*SolveResult, error) {
	var result SolveResult
	respBody, err := c.post("/api/math/solve", solveRequest{Expression: expression, Variable: variable})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type verifyRequest struct {
	Expression string `json:"expression"`
	Expected   string `json:"expected"`
	Operation  string `json:"operation"`
}

func (c *MathClient) Verify(expression, expected, operation string) (*VerifyResult, error) {
	var result VerifyResult
	respBody, err := c.post("/api/math/verify", verifyRequest{
		Expression: expression,
		Expected:   expected,
		Operation:  operation,
	})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type simplifyRequest struct {
	Expression string `json:"expression"`
}

func (c *MathClient) Simplify(expression string) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/simplify", simplifyRequest{Expression: expression})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type factorRequest struct {
	Expression string `json:"expression"`
}

func (c *MathClient) Factor(expression string) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/factor", factorRequest{Expression: expression})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type expandRequest struct {
	Expression string `json:"expression"`
}

func (c *MathClient) Expand(expression string) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/expand", expandRequest{Expression: expression})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

type matrixRequest struct {
	Operation string                 `json:"operation"`
	Matrix    [][]float64            `json:"matrix"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

func (c *MathClient) MatrixOp(operation string, matrix [][]float64, extra map[string]interface{}) (*MathResult, error) {
	var result MathResult
	respBody, err := c.post("/api/math/matrix", matrixRequest{
		Operation: operation,
		Matrix:    matrix,
		Extra:     extra,
	})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("[MATH] decode response: %w", err)
	}
	return &result, nil
}

func (c *MathClient) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/math/health", nil)
	if err != nil {
		log.Printf("[MATH] health check request failed: %v", err)
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[MATH] health check failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode == http.StatusOK
}
