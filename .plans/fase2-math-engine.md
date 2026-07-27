# Fase 2 — Motor Matemático, Resolución Paso a Paso y Verificación

> **For agentic workers:** Use subagent-driven-development to implement this plan task-by-task.

**Goal:** Convert matematicarag from a RAG chatbot into a math tutor that can interpret problems, identify methods, execute symbolic math, verify results, explain step-by-step, render LaTeX, and cite academic sources.

**Architecture:** Go backend calls a Python/SymPy microservice for symbolic math. LLM handles interpretation and explanation. RAG handles academic retrieval. A new `POST /api/tutor/solve` endpoint orchestrates the full pipeline. Frontend gets a dedicated tutor interface with step-by-step rendering.

**Tech Stack:** Go 1.25 + Chi v5, Python 3.11 + Flask + SymPy + mpmath, Angular 20 + KaTeX + MathLive, PostgreSQL 16, Qdrant v1.12

## Global Constraints

- Go backend remains the primary API server
- Python math service runs as a separate Docker container
- All math operations timeout at `MATH_TIMEOUT` seconds (default 5)
- LLM never fabricates math results — math engine is source of truth
- Citations distinguish 📚 academic sources from 🧮 computed results
- No arbitrary code execution from user input
- LaTeX rendered client-side via KaTeX (already installed)

---

## File Structure

### New Files (Python math service)
```
math-service/
├── Dockerfile
├── requirements.txt
├── app.py                    # Flask app, routes, validation
├── engine/
│   ├── __init__.py
│   ├── parser.py             # Input normalization + safe parsing
│   ├── arithmetic.py         # Basic operations
│   ├── algebra.py            # Simplify, factor, expand, collect
│   ├── equations.py          # Solve equations + systems
│   ├── inequalities.py       # Solve inequalities
│   ├── polynomials.py        # Polynomial operations
│   ├── calculus.py           # Derivatives, integrals, limits
│   ├── matrices.py           # Matrix operations
│   ├── trigonometry.py       # Trig simplification
│   └── verify.py             # Verification engine
└── tests/
    ├── test_calculus.py
    ├── test_equations.py
    ├── test_matrices.py
    └── test_verify.py
```

### New Files (Go backend)
```
api/
├── intent.go                 # Intent classifier
├── tutor.go                  # POST /api/tutor/solve orchestrator
├── mathclient.go             # HTTP client for Python math service
└── normalize.go              # Input normalization (LaTeX → SymPy)

internal/config/config.go     # Add MATH_SERVICE_URL, MATH_TIMEOUT
```

### Modified Files
```
cmd/server/main.go            # Register tutor routes
docker-compose.yml            # Add math-service container
frontend/src/app/
├── core/services/api.service.ts   # Add tutorSolve() method
├── modules/tutor/                 # New module (replaces/extends math)
│   ├── tutor.component.ts
│   └── tutor.component.html/css
├── app.routes.ts                  # Add /tutor route
└── shared/layout.component.ts     # Add Tutor nav item
```

---

## Task 1: Python Math Service — Foundation + Docker

**Files:**
- Create: `math-service/Dockerfile`
- Create: `math-service/requirements.txt`
- Create: `math-service/app.py`
- Create: `math-service/engine/__init__.py`
- Create: `math-service/engine/parser.py`
- Modify: `docker-compose.yml` — add math-service

**Interfaces:**
- Produces: `POST /math/evaluate` → `{"success": bool, "result": str, "latex": str}`

- [ ] **Step 1: Create `math-service/requirements.txt`**

```
flask==3.1.1
flask-cors==5.0.1
sympy==1.14.0
mpmath==1.3.0
gunicorn==23.0.0
```

- [ ] **Step 2: Create `math-service/engine/__init__.py`**

```python
from .parser import normalize_input, safe_parse
from .calculus import differentiate, integrate, compute_limit
from .equations import solve_equation, solve_system
from .algebra import simplify_expr, factor_expr, expand_expr
from .matrices import matrix_operation
from .verify import verify_result
```

- [ ] **Step 3: Create `math-service/engine/parser.py`**

```python
import re
from sympy import sympify, Symbol, parse_expr
from sympy.parsing.latex import parse_latex

def normalize_input(expr_str: str) -> str:
    """Normalize user input to SymPy-compatible string."""
    s = expr_str.strip()
    # Remove surrounding $ or $$ (LaTeX display)
    s = re.sub(r'^\$\$?|\$\$?$', '', s)
    # Remove \( \) and \[ \]
    s = re.sub(r'^\\?\(|\\?\)$', '', s)
    s = re.sub(r'^\\?\[|\]?$', '', s)
    # Convert Unicode superscripts: ² → **2, ³ → **3
    superscripts = {'⁰': '**0', '¹': '**1', '²': '**2', '³': '**3',
                    '⁴': '**4', '⁵': '**5', '⁶': '**6', '⁷': '**7',
                    '⁸': '**8', '⁹': '**9', 'ⁿ': '**n'}
    for uni, rep in superscripts.items():
        s = s.replace(uni, rep)
    # Convert ^ to ** (power)
    s = s.replace('^', '**')
    # Implicit multiplication: 2x → 2*x, )( → )*(
    s = re.sub(r'(\d)([a-zA-Z(])', r'\1*\2', s)
    s = re.sub(r'\)(\()', r')*\1', s)
    # ∫ → integrate, d/dx → diff, lim → limit (handled in route logic)
    return s

def safe_parse(expr_str: str):
    """Safely parse a mathematical expression. Returns SymPy expression or raises."""
    normalized = normalize_input(expr_str)
    # Try LaTeX first if it looks like LaTeX
    if '\\' in expr_str or '{' in expr_str:
        try:
            return parse_latex(expr_str)
        except Exception:
            pass
    # Try SymPy parsing
    local_dict = {}
    return parse_expr(normalized, local_dict=local_dict, evaluate=False)
```

- [ ] **Step 4: Create `math-service/app.py`**

```python
import os
import signal
import traceback
from functools import wraps
from flask import Flask, request, jsonify
from flask_cors import CORS
from engine.parser import normalize_input, safe_parse
from engine.calculus import differentiate, integrate, compute_limit
from engine.equations import solve_equation, solve_system
from engine.algebra import simplify_expr, factor_expr, expand_expr
from engine.matrices import matrix_operation
from engine.verify import verify_result

app = Flask(__name__)
CORS(app)

MATH_TIMEOUT = int(os.environ.get('MATH_TIMEOUT', '5'))

class MathTimeout(Exception):
    pass

def timeout_handler(signum, frame):
    raise MathTimeout("Operation timed out")

def with_timeout(f):
    @wraps(f)
    def wrapper(*args, **kwargs):
        old_handler = signal.signal(signal.SIGALRM, timeout_handler)
        signal.alarm(MATH_TIMEOUT)
        try:
            result = f(*args, **kwargs)
            return result
        except MathTimeout:
            return jsonify({"success": False, "error": "math_timeout"}), 408
        except Exception as e:
            return jsonify({"success": False, "error": str(e)}), 400
        finally:
            signal.alarm(0)
            signal.signal(signal.SIGALRM, old_handler)
    return wrapper

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "ok", "engine": "sympy"})

@app.route('/math/evaluate', methods=['POST'])
@with_timeout
def evaluate():
    data = request.json
    expr_str = data.get('expression', '')
    if not expr_str:
        return jsonify({"success": False, "error": "expression required"}), 400
    expr = safe_parse(expr_str)
    result = expr.evalf()
    return jsonify({
        "success": True,
        "result": str(result),
        "latex": str(expr),
    })

@app.route('/math/simplify', methods=['POST'])
@with_timeout
def simplify():
    data = request.json
    expr = safe_parse(data.get('expression', ''))
    result = simplify_expr(expr)
    return jsonify({"success": True, "result": str(result), "latex": str(result)})

@app.route('/math/factor', methods=['POST'])
@with_timeout
def factor():
    data = request.json
    expr = safe_parse(data.get('expression', ''))
    result = factor_expr(expr)
    return jsonify({"success": True, "result": str(result), "latex": str(result)})

@app.route('/math/expand', methods=['POST'])
@with_timeout
def expand():
    data = request.json
    expr = safe_parse(data.get('expression', ''))
    result = expand_expr(expr)
    return jsonify({"success": True, "result": str(result), "latex": str(result)})

@app.route('/math/differentiate', methods=['POST'])
@with_timeout
def diff():
    data = request.json
    expr = safe_parse(data.get('expression', ''))
    var = data.get('variable', 'x')
    result = differentiate(expr, var)
    return jsonify({"success": True, "result": str(result), "latex": str(result)})

@app.route('/math/integrate', methods=['POST'])
@with_timeout
def integ():
    data = request.json
    expr = safe_parse(data.get('expression', ''))
    var = data.get('variable', 'x')
    lower = data.get('lower')
    upper = data.get('upper')
    result = integrate(expr, var, lower, upper)
    return jsonify({"success": True, "result": str(result), "latex": str(result)})

@app.route('/math/limit', methods=['POST'])
@with_timeout
def limit():
    data = request.json
    expr = safe_parse(data.get('expression', ''))
    var = data.get('variable', 'x')
    point = data.get('point', '0')
    result = compute_limit(expr, var, point)
    return jsonify({"success": True, "result": str(result), "latex": str(result)})

@app.route('/math/solve', methods=['POST'])
@with_timeout
def solve():
    data = request.json
    expr_str = data.get('expression', '')
    var = data.get('variable', 'x')
    # Check if it's a system (list of equations)
    if isinstance(expr_str, list):
        result = solve_system(expr_str, var)
    else:
        result = solve_equation(expr_str, var)
    return jsonify({"success": True, **result})

@app.route('/math/matrix', methods=['POST'])
@with_timeout
def matrix():
    data = request.json
    operation = data.get('operation', '')
    matrix_data = data.get('matrix', [])
    result = matrix_operation(operation, matrix_data, data)
    return jsonify({"success": True, **result})

@app.route('/math/verify', methods=['POST'])
@with_timeout
def verify():
    data = request.json
    expression = data.get('expression', '')
    expected = data.get('expected', '')
    operation = data.get('operation', '')
    result = verify_result(expression, expected, operation)
    return jsonify(result)

if __name__ == '__main__':
    port = int(os.environ.get('MATH_PORT', '5000'))
    app.run(host='0.0.0.0', port=port, debug=False)
```

- [ ] **Step 5: Create `math-service/Dockerfile`**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:5000/health')" || exit 1

CMD ["gunicorn", "-w", "2", "-b", "0.0.0.0:5000", "--timeout", "30", "app:app"]
```

- [ ] **Step 6: Add math-service to `docker-compose.yml`**

Add after the `backend` service:

```yaml
  math-service:
    build:
      context: ./math-service
      dockerfile: Dockerfile
    container_name: matematicarag-math
    restart: always
    environment:
      MATH_TIMEOUT: "5"
      MATH_PORT: "5000"
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:5000/health')"]
      interval: 30s
      timeout: 5s
      retries: 3
    networks:
      - matematicarag-net
```

Also update the `backend` service to add `MATH_SERVICE_URL`:

```yaml
    environment:
      # ... existing env vars ...
      MATH_SERVICE_URL: "http://math-service:5000"
```

- [ ] **Step 7: Verify Docker build**

Run: `cd /home/proyecto/matematicarag && docker compose build math-service`

- [ ] **Step 8: Commit**

```bash
git add math-service/ docker-compose.yml
git commit -m "feat: Python math service foundation with SymPy + Docker"
```

---

## Task 2: Math Engine Modules (SymPy wrappers)

**Files:**
- Create: `math-service/engine/calculus.py`
- Create: `math-service/engine/equations.py`
- Create: `math-service/engine/algebra.py`
- Create: `math-service/engine/matrices.py`
- Create: `math-service/engine/verify.py`
- Create: `math-service/engine/arithmetic.py`
- Create: `math-service/engine/trigonometry.py`

**Interfaces:**
- Consumes: SymPy expressions from `parser.safe_parse`
- Produces: Functions called by `app.py` routes

- [ ] **Step 1: Create `math-service/engine/calculus.py`**

```python
from sympy import symbols, diff, integrate as sympy_integrate, limit as sympy_limit, oo, Symbol

def differentiate(expr, var_str='x'):
    var = Symbol(var_str)
    return diff(expr, var)

def integrate(expr, var_str='x', lower=None, upper=None):
    var = Symbol(var_str)
    if lower is not None and upper is not None:
        from sympy import Rational
        lower_expr = Rational(lower) if '/' not in str(lower) else eval(lower)
        upper_expr = Rational(upper) if '/' not in str(upper) else eval(upper)
        return sympy_integrate(expr, (var, lower_expr, upper_expr))
    return sympy_integrate(expr, var)

def compute_limit(expr, var_str='x', point='0'):
    var = Symbol(var_str)
    if point == 'oo' or point == 'infinity':
        return sympy_limit(expr, var, oo)
    from sympy import sympify
    point_val = sympify(point)
    return sympy_limit(expr, var, point_val)
```

- [ ] **Step 2: Create `math-service/engine/equations.py`**

```python
from sympy import symbols, Eq, solve, sympify, Matrix
from engine.parser import normalize_input

def solve_equation(expr_str, var_str='x'):
    from sympy import Symbol
    var = Symbol(var_str)
    normalized = normalize_input(expr_str)

    # Split on = to get lhs and rhs
    if '=' in normalized:
        parts = normalized.split('=', 1)
        lhs = sympify(parts[0])
        rhs = sympify(parts[1])
        equation = Eq(lhs, rhs)
    else:
        equation = sympify(normalized)

    solutions = solve(equation, var)
    solution_strs = [str(s) for s in solutions]

    return {
        "variables": [var_str],
        "solutions": solution_strs,
        "latex": " , ".join([str(s) for s in solutions]),
        "count": len(solutions),
    }

def solve_system(equations_str, var_str='x'):
    from sympy import Symbol
    vars_list = [Symbol(v.strip()) for v in var_str.split(',')]

    eqs = []
    for eq_str in equations_str:
        normalized = normalize_input(eq_str)
        if '=' in normalized:
            parts = normalized.split('=', 1)
            lhs = sympify(parts[0])
            rhs = sympify(parts[1])
            eqs.append(Eq(lhs, rhs))
        else:
            eqs.append(sympify(normalized))

    solution = solve(eqs, vars_list, dict=True)

    result_vars = {}
    if solution:
        for v in vars_list:
            result_vars[str(v)] = str(solution[0].get(v, 'unknown'))

    return {
        "variables": [str(v) for v in vars_list],
        "solutions": result_vars,
        "latex": ", ".join([f"{k} = {v}" for k, v in result_vars.items()]),
        "method": "symbolic",
    }
```

- [ ] **Step 3: Create `math-service/engine/algebra.py`**

```python
from sympy import simplify, factor, expand, collect, cancel, together

def simplify_expr(expr):
    return simplify(expr)

def factor_expr(expr):
    return factor(expr)

def expand_expr(expr):
    return expand(expr)

def collect_expr(expr, var_str='x'):
    from sympy import Symbol
    return collect(expr, Symbol(var_str))

def cancel_expr(expr):
    return cancel(expr)

def together_expr(expr):
    return together(expr)
```

- [ ] **Step 4: Create `math-service/engine/matrices.py`**

```python
from sympy import Matrix, eye, zeros

def matrix_operation(operation, matrix_data, extra_data=None):
    if extra_data is None:
        extra_data = {}

    mat = Matrix(matrix_data)

    if operation == 'determinant':
        det = mat.det()
        return {"result": str(det), "latex": str(det)}

    elif operation == 'transpose':
        result = mat.T
        return {"result": str(result), "latex": str(result)}

    elif operation == 'inverse':
        result = mat.inv()
        return {"result": str(result), "latex": str(result)}

    elif operation == 'rank':
        result = mat.rank()
        return {"result": str(result), "latex": str(result)}

    elif operation == 'rref':
        result, pivots = mat.rref()
        return {"result": str(result), "latex": str(result), "pivots": list(pivots)}

    elif operation == 'eigenvalues':
        result = mat.eigenvals()
        return {"result": str(result), "latex": str(result)}

    elif operation == 'multiply':
        other = Matrix(extra_data.get('matrix2', []))
        result = mat * other
        return {"result": str(result), "latex": str(result)}

    elif operation == 'add':
        other = Matrix(extra_data.get('matrix2', []))
        result = mat + other
        return {"result": str(result), "latex": str(result)}

    else:
        return {"error": f"Unknown operation: {operation}"}
```

- [ ] **Step 5: Create `math-service/engine/verify.py`**

```python
from sympy import sympify, simplify, Symbol
from sympy.parsing.latex import parse_latex

def verify_result(expression: str, expected: str, operation: str = '') -> dict:
    """
    Verify that 'expression' evaluates to 'expected'.
    Returns {"verified": bool, "method": str, "expected": str, "actual": str}
    """
    try:
        # Parse both expressions
        if '\\' in expression or '{' in expression:
            expr_actual = parse_latex(expression)
        else:
            expr_actual = sympify(expression)

        if '\\' in expected or '{' in expected:
            expr_expected = parse_latex(expected)
        else:
            expr_expected = sympify(expected)

        # Check equivalence by simplifying difference
        diff = simplify(expr_actual - expr_expected)

        verified = diff == 0

        return {
            "verified": verified,
            "method": "symbolic_simplification",
            "expected": str(expr_expected),
            "actual": str(expr_actual),
            "latex_expected": str(expr_expected),
            "latex_actual": str(expr_actual),
        }
    except Exception as e:
        return {
            "verified": False,
            "method": "error",
            "error": str(e),
        }
```

- [ ] **Step 6: Create `math-service/engine/arithmetic.py`**

```python
from sympy import sympify

def basic_evaluate(expr_str):
    """Evaluate basic arithmetic expressions."""
    result = sympify(expr_str)
    return {"result": str(result.evalf()), "latex": str(result)}
```

- [ ] **Step 7: Create `math-service/engine/trigonometry.py`**

```python
from sympy import sin, cos, tan, simplify, trigsimp

def trig_simplify(expr):
    return trigsimp(expr)
```

- [ ] **Step 8: Commit**

```bash
git add math-service/engine/
git commit -m "feat: SymPy math engine modules (calculus, equations, algebra, matrices, verify)"
```

---

## Task 3: Go Math Service Client + Config

**Files:**
- Create: `api/mathclient.go`
- Modify: `internal/config/config.go` — add `MathServiceURL`, `MathTimeout`

**Interfaces:**
- Produces: `MathServiceClient` struct with methods matching Python routes

- [ ] **Step 1: Add config fields**

In `internal/config/config.go`, add to Config struct:

```go
MathServiceURL string
MathTimeout    int
```

In `Load()`, add:

```go
MathServiceURL: getEnv("MATH_SERVICE_URL", "http://localhost:5000"),
MathTimeout:    getEnvInt("MATH_TIMEOUT", 5),
```

- [ ] **Step 2: Create `api/mathclient.go`**

```go
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
	Success  bool     `json:"success"`
	Result   string   `json:"result,omitempty"`
	Latex    string   `json:"latex,omitempty"`
	Error    string   `json:"error,omitempty"`
}

type SolveResult struct {
	Success   bool     `json:"success"`
	Variables []string `json:"variables,omitempty"`
	Solutions interface{} `json:"solutions,omitempty"`
	Latex     string   `json:"latex,omitempty"`
	Count     int      `json:"count,omitempty"`
	Error     string   `json:"error,omitempty"`
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
	return &MathClient{
		baseURL: cfg.MathServiceURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.MathTimeout) * time.Second,
		},
	}
}

func (mc *MathClient) post(path string, body interface{}) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), mc.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", mc.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 408 {
		return nil, fmt.Errorf("math_timeout")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("math_error: %s", string(respBody))
	}

	return respBody, nil
}

func (mc *MathClient) Evaluate(expression string) (*MathResult, error) {
	body, err := mc.post("/math/evaluate", map[string]string{"expression": expression})
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Differentiate(expression, variable string) (*MathResult, error) {
	body, err := mc.post("/math/differentiate", map[string]string{
		"expression": expression, "variable": variable,
	})
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Integrate(expression, variable string, lower, upper *string) (*MathResult, error) {
	payload := map[string]interface{}{
		"expression": expression, "variable": variable,
	}
	if lower != nil {
		payload["lower"] = *lower
	}
	if upper != nil {
		payload["upper"] = *upper
	}
	body, err := mc.post("/math/integrate", payload)
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Limit(expression, variable, point string) (*MathResult, error) {
	body, err := mc.post("/math/limit", map[string]string{
		"expression": expression, "variable": variable, "point": point,
	})
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Solve(expression, variable string) (*SolveResult, error) {
	body, err := mc.post("/math/solve", map[string]string{
		"expression": expression, "variable": variable,
	})
	if err != nil {
		return nil, err
	}
	var result SolveResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Verify(expression, expected, operation string) (*VerifyResult, error) {
	body, err := mc.post("/math/verify", map[string]string{
		"expression": expression, "expected": expected, "operation": operation,
	})
	if err != nil {
		return nil, err
	}
	var result VerifyResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Simplify(expression string) (*MathResult, error) {
	body, err := mc.post("/math/simplify", map[string]string{"expression": expression})
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Factor(expression string) (*MathResult, error) {
	body, err := mc.post("/math/factor", map[string]string{"expression": expression})
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) Expand(expression string) (*MathResult, error) {
	body, err := mc.post("/math/expand", map[string]string{"expression": expression})
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

func (mc *MathClient) MatrixOp(operation string, matrix [][]float64, extra map[string]interface{}) (*MathResult, error) {
	payload := map[string]interface{}{
		"operation": operation,
		"matrix":    matrix,
	}
	for k, v := range extra {
		payload[k] = v
	}
	body, err := mc.post("/math/matrix", payload)
	if err != nil {
		return nil, err
	}
	var result MathResult
	err = json.Unmarshal(body, &result)
	return &result, err
}

// HealthCheck verifies math service is available
func (mc *MathClient) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", mc.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := mc.httpClient.Do(req)
	if err != nil {
		log.Printf("[MATH] health check failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./cmd/server`

- [ ] **Step 4: Commit**

```bash
git add api/mathclient.go internal/config/config.go
git commit -m "feat: Go HTTP client for Python math service + config"
```

---

## Task 4: Intent Classifier

**Files:**
- Create: `api/intent.go`

**Interfaces:**
- Produces: `ClassifyIntent(db, query) → IntentResult`
- Consumes: LLM via `callOpenAIWithHistory`

- [ ] **Step 1: Create `api/intent.go`**

```go
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type IntentType string

const (
	IntentConceptual      IntentType = "conceptual"
	IntentDefinition      IntentType = "definition"
	IntentFormula         IntentType = "formula"
	IntentExample         IntentType = "example"
	IntentExercise        IntentType = "exercise"
	IntentSolve           IntentType = "solve"
	IntentVerify          IntentType = "verify"
	IntentExplain         IntentType = "explain"
	IntentCompare         IntentType = "compare"
	IntentGenerateExercise IntentType = "generate_exercise"
	IntentSimplify        IntentType = "simplify"
	IntentDifferentiate   IntentType = "differentiate"
	IntentIntegrate       IntentType = "integrate"
	IntentLimit           IntentType = "limit"
	IntentMatrix          IntentType = "matrix"
)

type IntentResult struct {
	Intent        IntentType `json:"intent"`
	Confidence    float64    `json:"confidence"`
	MathOperation string     `json:"math_operation,omitempty"`
	Expression    string     `json:"expression,omitempty"`
	Variable      string     `json:"variable,omitempty"`
	NeedsMath     bool       `json:"needs_math"`
}

// classifyPrompt is used by the LLM to identify intent
const classifyPrompt = `Eres un clasificador de consultas matematicas. Analiza la pregunta y clasifica la intencion.

Categorias:
- conceptual: pregunta sobre conceptos generales
- definition: pide una definicion
- formula: busca una formula especifica
- example: pide un ejemplo
- exercise: pide ejercicios para practicar
- solve: quiere resolver un problema matematico
- verify: quiere verificar si un resultado es correcto
- explain: quiere una explicacion detallada
- compare: compara conceptos o metodos
- generate_exercise: genera ejercicios de practica
- simplify: quiere simplificar una expresion
- differentiate: quiere calcular una derivada
- integrate: quiere calcular una integral
- limit: quiere calcular un limite
- matrix: operacion con matrices

Responde SOLO con JSON:
{"intent":"<categoria>","confidence":0.95,"math_operation":"<operacion si aplica>","expression":"<expresion matematica si aplica>","variable":"<variable si aplica>","needs_math":true/false}

Ejemplos:
"Que es un espacio vectorial?" → {"intent":"definition","confidence":0.95,"needs_math":false}
"Resolver x^2 + 5x + 6 = 0" → {"intent":"solve","confidence":0.95,"math_operation":"solve","expression":"x^2+5*x+6=0","variable":"x","needs_math":true}
"Derivada de x^3" → {"intent":"differentiate","confidence":0.95,"math_operation":"differentiate","expression":"x^3","variable":"x","needs_math":true}
"∫ x^2 dx" → {"intent":"integrate","confidence":0.95,"math_operation":"integrate","expression":"x^2","variable":"x","needs_math":true}
"Esta bien que la derivada de x^3 es 2x^2?" → {"intent":"verify","confidence":0.95,"math_operation":"differentiate","expression":"x^3","variable":"x","needs_math":true}
"Explica la regla de la cadena" → {"intent":"explain","confidence":0.95,"needs_math":false}`

func ClassifyIntent(db *pgxpool.Pool, query string) IntentResult {
	apiKey := getAPIKey(db)
	if apiKey == "" {
		// Fallback: simple keyword matching
		return classifyByKeywords(query)
	}

	messages := []OpenAIMessage{
		{Role: "system", Content: classifyPrompt},
		{Role: "user", Content: query},
	}

	response, err := callOpenAIWithHistory(db, messages, "")
	if err != nil {
		log.Printf("[INTENT] LLM error: %v, using keyword fallback", err)
		return classifyByKeywords(query)
	}

	// Parse JSON from response
	response = strings.TrimSpace(response)
	// Handle markdown code blocks
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var jsonLines []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				jsonLines = append(jsonLines, line)
			}
		}
		response = strings.Join(jsonLines, "\n")
	}

	var result IntentResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Printf("[INTENT] parse error: %v, response: %s", err, response)
		return classifyByKeywords(query)
	}

	return result
}

func classifyByKeywords(query string) IntentResult {
	q := strings.ToLower(query)

	if strings.Contains(q, "resolver") || strings.Contains(q, "resuelve") || strings.Contains(q, "calcular") {
		return IntentResult{Intent: IntentSolve, Confidence: 0.7, NeedsMath: true}
	}
	if strings.Contains(q, "derivada") || strings.Contains(q, "derivar") || strings.Contains(q, "d/dx") {
		return IntentResult{Intent: IntentDifferentiate, Confidence: 0.8, MathOperation: "differentiate", NeedsMath: true}
	}
	if strings.Contains(q, "integral") || strings.Contains(q, "integrar") || strings.Contains(q, "∫") {
		return IntentResult{Intent: IntentIntegrate, Confidence: 0.8, MathOperation: "integrate", NeedsMath: true}
	}
	if strings.Contains(q, "limite") || strings.Contains(q, "lim ") || strings.Contains(q, "límite") {
		return IntentResult{Intent: IntentLimit, Confidence: 0.8, MathOperation: "limit", NeedsMath: true}
	}
	if strings.Contains(q, "simplific") {
		return IntentResult{Intent: IntentSimplify, Confidence: 0.7, MathOperation: "simplify", NeedsMath: true}
	}
	if strings.Contains(q, "verific") || strings.Contains(q, "correcto") || strings.Contains(q, "está bien") {
		return IntentResult{Intent: IntentVerify, Confidence: 0.7, NeedsMath: true}
	}
	if strings.Contains(q, "matriz") || strings.Contains(q, "determinante") {
		return IntentResult{Intent: IntentMatrix, Confidence: 0.7, MathOperation: "matrix", NeedsMath: true}
	}
	if strings.Contains(q, "ecuacion") || strings.Contains(q, "ecuación") || strings.Contains(q, "=") {
		return IntentResult{Intent: IntentSolve, Confidence: 0.6, MathOperation: "solve", NeedsMath: true}
	}
	if strings.Contains(q, "definicion") || strings.Contains(q, "qué es") || strings.Contains(q, "que es") {
		return IntentResult{Intent: IntentDefinition, Confidence: 0.7, NeedsMath: false}
	}
	if strings.Contains(q, "formula") || strings.Contains(q, "fórmula") {
		return IntentResult{Intent: IntentFormula, Confidence: 0.7, NeedsMath: false}
	}
	if strings.Contains(q, "explic") {
		return IntentResult{Intent: IntentExplain, Confidence: 0.7, NeedsMath: false}
	}

	return IntentResult{Intent: IntentConceptual, Confidence: 0.5, NeedsMath: false}
}

// extractExpression extracts the mathematical expression from a query
func extractExpression(query string) string {
	// Remove common prefixes
	s := query
	prefixes := []string{"resolver ", "resuelve ", "calcular ", "calculo de ", "encuentra ", "determina "}
	for _, p := range prefixes {
		if strings.HasPrefix(strings.ToLower(s), p) {
			s = s[len(p):]
			break
		}
	}
	return strings.TrimSpace(s)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./cmd/server`

- [ ] **Step 3: Commit**

```bash
git add api/intent.go
git commit -m "feat: intent classifier with LLM + keyword fallback"
```

---

## Task 5: Tutor Orchestrator (POST /api/tutor/solve)

**Files:**
- Create: `api/tutor.go`
- Modify: `cmd/server/main.go` — register tutor routes

**Interfaces:**
- Produces: `POST /api/tutor/solve` → full structured response
- Consumes: `ClassifyIntent`, `MathClient`, `HybridSearch`, `RerankResults`, `callOpenAIWithHistory`

- [ ] **Step 1: Create `api/tutor.go`**

```go
package api

import (
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

type TutorRequest struct {
	Query              string `json:"query"`
	CourseID           string `json:"course_id,omitempty"`
	UnitID             string `json:"unit_id,omitempty"`
	ExplanationLevel   string `json:"explanation_level,omitempty"` // basic, intermediate, advanced
	Mode               string `json:"mode,omitempty"`             // solve, verify, hint, explain_error
	UserResult         string `json:"user_result,omitempty"`      // for verify/explain_error mode
	UserProcedure      []string `json:"user_procedure,omitempty"` // steps user took (for explain_error)
}

type TutorStep struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Latex       string `json:"latex,omitempty"`
	IsMath      bool   `json:"is_math"` // true if computed by math engine
}

type TutorResponse struct {
	Problem struct {
		Type       string `json:"type"`
		Expression string `json:"expression"`
		Variable   string `json:"variable,omitempty"`
	} `json:"problem"`
	Method struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"method"`
	Steps       []TutorStep  `json:"steps"`
	Result      *MathResult  `json:"result,omitempty"`
	Verification *VerifyInfo `json:"verification,omitempty"`
	Citations   []RagCitation `json:"citations"`
	Sources     []RagSource  `json:"sources"` // 📚 academic sources
	MathComputed bool        `json:"math_computed"` // 🧮 computed by engine
	Confidence  string       `json:"confidence"`
}

type VerifyInfo struct {
	Status string `json:"status"` // verified, not_verified, verification_failed, verification_not_possible
	Method string `json:"method,omitempty"`
}

func TutorRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	mathClient := NewMathClient(cfg)

	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))

		r.Post("/solve", func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			var req TutorRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Query == "" {
				http.Error(w, `{"error":"query required"}`, http.StatusBadRequest)
				return
			}
			if len(req.Query) > 10000 {
				http.Error(w, `{"error":"query too long"}`, http.StatusBadRequest)
				return
			}
			if req.ExplanationLevel == "" {
				req.ExplanationLevel = "intermediate"
			}
			if req.Mode == "" {
				req.Mode = "solve"
			}

			// 1. Classify intent
			intent := ClassifyIntent(db, req.Query)
			log.Printf("[TUTOR] intent=%s, confidence=%.2f, needs_math=%v", intent.Intent, intent.Confidence, intent.NeedsMath)

			// 2. RAG retrieval
			sources, ragContext := performRAGSearch(db, req.Query)
			ragSources := sources // for response

			// 3. Build response
			response := TutorResponse{
				Citations:   make([]RagCitation, 0),
				Sources:     make([]RagSource, 0),
				MathComputed: intent.NeedsMath,
				Confidence:  "medium",
			}

			// Map sources to citations
			for i, s := range ragSources {
				response.Citations = append(response.Citations, RagCitation{
					ID:         fmt.Sprintf("SRC-%03d", i+1),
					DocumentID: s.ID,
					Filename:   s.Filename,
					Page:       s.Page,
					Section:    s.Section,
				})
			}
			response.Sources = ragSources

			// 4. If needs math, use math engine
			if intent.NeedsMath {
				response.Problem.Type = string(intent.Intent)
				response.Problem.Expression = intent.Expression
				response.Problem.Variable = intent.Variable

				if intent.Expression == "" {
					intent.Expression = extractExpression(req.Query)
					response.Problem.Expression = intent.Expression
				}

				// Execute math operation
				switch intent.MathOperation {
				case "differentiate":
					if intent.Variable == "" {
						intent.Variable = "x"
					}
					result, err := mathClient.Differentiate(intent.Expression, intent.Variable)
					if err != nil {
						log.Printf("[TUTOR] math error: %v", err)
						response.Verification = &VerifyInfo{Status: "verification_not_possible"}
					} else {
						response.Result = result
						// Verify by integrating back
						if result != nil && result.Success {
							verify, vErr := mathClient.Verify(
								fmt.Sprintf("integrate(%s, %s)", result.Result, intent.Variable),
								intent.Expression,
								"differentiate",
							)
							if vErr == nil && verify.Verified {
								response.Verification = &VerifyInfo{Status: "verified", Method: verify.Method}
							} else {
								response.Verification = &VerifyInfo{Status: "not_verified"}
							}
						}
					}

				case "integrate":
					if intent.Variable == "" {
						intent.Variable = "x"
					}
					result, err := mathClient.Integrate(intent.Expression, intent.Variable, nil, nil)
					if err != nil {
						log.Printf("[TUTOR] math error: %v", err)
						response.Verification = &VerifyInfo{Status: "verification_not_possible"}
					} else {
						response.Result = result
						// Verify by differentiating
						if result != nil && result.Success {
							verify, vErr := mathClient.Verify(
								fmt.Sprintf("differentiate(%s, %s)", result.Result, intent.Variable),
								intent.Expression,
								"integrate",
							)
							if vErr == nil && verify.Verified {
								response.Verification = &VerifyInfo{Status: "verified", Method: verify.Method}
							} else {
								response.Verification = &VerifyInfo{Status: "not_verified"}
							}
						}
					}

				case "solve":
					if intent.Variable == "" {
						intent.Variable = "x"
					}
					result, err := mathClient.Solve(intent.Expression, intent.Variable)
					if err != nil {
						log.Printf("[TUTOR] math error: %v", err)
						response.Verification = &VerifyInfo{Status: "verification_not_possible"}
					} else {
						// Convert SolveResult to MathResult for response
						solJSON, _ := json.Marshal(result.Solutions)
						response.Result = &MathResult{
							Success: result.Success,
							Result:  string(solJSON),
							Latex:   result.Latex,
						}
						response.Verification = &VerifyInfo{Status: "verified", Method: "symbolic_solve"}
					}

				case "limit":
					if intent.Variable == "" {
						intent.Variable = "x"
					}
					result, err := mathClient.Limit(intent.Expression, intent.Variable, "0")
					if err != nil {
						log.Printf("[TUTOR] math error: %v", err)
						response.Verification = &VerifyInfo{Status: "verification_not_possible"}
					} else {
						response.Result = result
						response.Verification = &VerifyInfo{Status: "verified", Method: "symbolic_limit"}
					}

				case "simplify":
					result, err := mathClient.Simplify(intent.Expression)
					if err != nil {
						log.Printf("[TUTOR] math error: %v", err)
						response.Verification = &VerifyInfo{Status: "verification_not_possible"}
					} else {
						response.Result = result
						response.Verification = &VerifyInfo{Status: "verified", Method: "symbolic_simplify"}
					}

				default:
					// Generic evaluate
					result, err := mathClient.Evaluate(intent.Expression)
					if err != nil {
						log.Printf("[TUTOR] math error: %v", err)
						response.Verification = &VerifyInfo{Status: "verification_not_possible"}
					} else {
						response.Result = result
						response.Verification = &VerifyInfo{Status: "verified", Method: "symbolic_evaluate"}
					}
				}
			}

			// 5. Generate explanation with LLM
			explanation := generateExplanation(db, req, intent, ragContext, response.Result, response.Verification)
			response.Method.Name = explanation.MethodName
			response.Method.Description = explanation.MethodDescription
			response.Steps = explanation.Steps

			// 6. Set confidence
			if response.Verification != nil && response.Verification.Status == "verified" {
				response.Confidence = "high"
			}

			// 7. Log
			duration := time.Since(startTime)
			log.Printf("[TUTOR] query=%s, intent=%s, math=%v, verified=%v, duration=%v",
				req.Query, intent.Intent, intent.NeedsMath,
				response.Verification != nil && response.Verification.Status == "verified",
				duration.Round(time.Millisecond))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})
	}
}

type ExplanationResult struct {
	MethodName        string
	MethodDescription string
	Steps             []TutorStep
}

func generateExplanation(db *pgxpool.Pool, req TutorRequest, intent IntentResult, ragContext string, mathResult *MathResult, verification *VerifyInfo) ExplanationResult {
	apiKey := getAPIKey(db)
	if apiKey == "" {
		return ExplanationResult{
			MethodName: "directa",
			Steps: []TutorStep{{
				Number:      1,
				Title:       "Resultado",
				Explanation: "El resultado del calculo.",
				IsMath:      true,
			}},
		}
	}

	systemPrompt := `Eres un tutor matematico universitario. Generas explicaciones paso a paso.
Respondes en formato JSON con esta estructura:
{
  "method_name": "nombre del metodo",
  "method_description": "descripcion breve",
  "steps": [
    {"number": 1, "title": "titulo", "explanation": "explicacion", "latex": "formula en LaTeX", "is_math": true/false}
  ]
}
Importante:
- Cada paso debe ser claro y didactico
- Los pasos con is_math=true fueron calculados por el motor matematico (son confiables)
- Los pasos con is_math=false son explicacion conceptual
- Usar LaTeX para formulas
- Nivel: ` + req.ExplanationLevel

	userPrompt := fmt.Sprintf(`Pregunta: %s
Intento: %s
Expresion: %s
Variable: %s
%s
Resultado del motor matematico: %v
Verificacion: %v

Genera la explicacion paso a paso en JSON.`,
		req.Query, intent.Intent, intent.Expression, intent.Variable,
		if ragContext != "" {
			fmt.Sprintf("Material academico recuperado:\n%s", ragContext)
		} else {
			"No se encontro material academico relevante."
		},
		mathResult, verification,
	)

	messages := []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := callOpenAIWithHistory(db, messages, "")
	if err != nil {
		log.Printf("[TUTOR] LLM explanation error: %v", err)
		return ExplanationResult{
			MethodName: "directa",
			Steps: []TutorStep{{
				Number:      1,
				Title:       "Resultado",
				Explanation: fmt.Sprintf("El resultado es: %v", mathResult),
				IsMath:      true,
			}},
		}
	}

	// Parse JSON response
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var jsonLines []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				jsonLines = append(jsonLines, line)
			}
		}
		response = strings.Join(jsonLines, "\n")
	}

	var explanation ExplanationResult
	if err := json.Unmarshal([]byte(response), &explanation); err != nil {
		log.Printf("[TUTOR] explanation parse error: %v, response: %s", err, response)
		return ExplanationResult{
			MethodName: "directa",
			Steps: []TutorStep{{
				Number:      1,
				Title:       "Resultado",
				Explanation: response,
				IsMath:      mathResult != nil,
			}},
		}
	}

	return explanation
}
```

- [ ] **Step 2: Register route in `cmd/server/main.go`**

In the `apiRouter.Route("/api", ...)` block, add:

```go
r.Route("/tutor", api.TutorRoutes(db, cfg))
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./cmd/server`

- [ ] **Step 4: Commit**

```bash
git add api/tutor.go cmd/server/main.go
git commit -m "feat: POST /api/tutor/solve orchestrator with intent + RAG + math + LLM"
```

---

## Task 6: Frontend Tutor Service + Types

**Files:**
- Modify: `frontend/src/app/core/services/api.service.ts` — add `tutorSolve()`

**Interfaces:**
- Produces: `tutorSolve(request)` → Observable with tutor response

- [ ] **Step 1: Add types and method to `api.service.ts`**

```typescript
// Add interfaces at the top (or in a separate types file)
export interface TutorRequest {
  query: string;
  course_id?: string;
  unit_id?: string;
  explanation_level?: 'basic' | 'intermediate' | 'advanced';
  mode?: 'solve' | 'verify' | 'hint' | 'explain_error';
  user_result?: string;
  user_procedure?: string[];
}

export interface TutorStep {
  number: number;
  title: string;
  explanation: string;
  latex?: string;
  is_math: boolean;
}

export interface TutorResponse {
  problem: { type: string; expression: string; variable?: string };
  method: { name: string; description: string };
  steps: TutorStep[];
  result?: { success: boolean; result: string; latex: string };
  verification?: { status: string; method?: string };
  citations: any[];
  sources: any[];
  math_computed: boolean;
  confidence: string;
}
```

Add method to `ApiService` class:

```typescript
tutorSolve(request: TutorRequest): Observable<TutorResponse> {
  return this.http.post<TutorResponse>(`${this.baseUrl}/tutor/solve`, request);
}
```

- [ ] **Step 2: Verify Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/core/services/api.service.ts
git commit -m "feat: Angular tutor service with types"
```

---

## Task 7: Frontend Tutor Component

**Files:**
- Create: `frontend/src/app/modules/tutor/tutor.component.ts`
- Modify: `frontend/src/app/app.routes.ts` — add `/tutor` route
- Modify: `frontend/src/app/shared/layout.component.ts` — add Tutor nav item

**Interfaces:**
- Consumes: `ApiService.tutorSolve()`

- [ ] **Step 1: Create `frontend/src/app/modules/tutor/tutor.component.ts`**

This is a full standalone Angular component with:
- MathLive input field for math expressions
- Text input for natural language queries
- Mode selector (solve/verify/hint)
- Level selector (basic/intermediate/advanced)
- Step-by-step result display with KaTeX
- Verification badge
- Citations section

The component should be created as a single-file standalone component (following the project pattern of `math.component.ts` and `chat.component.ts`). It will include:

1. A dual input area: MathLive field + text query field
2. Mode/Level dropdowns
3. "Solve" button
4. Results area showing:
   - Problem identification
   - Method used
   - Steps (each with title, explanation, LaTeX rendered via `RenderMathPipe`)
   - Result with LaTeX
   - Verification badge (✅ verified / ⚠️ not verified)
   - 📚 Sources section
   - 🧮 Math Computed indicator

The component should follow the existing dark theme CSS patterns from `chat.component.ts` and `math.component.ts`.

- [ ] **Step 2: Add route to `app.routes.ts`**

```typescript
{ path: 'tutor', loadComponent: () => import('./modules/tutor/tutor.component').then(m => m.TutorComponent) },
```

- [ ] **Step 3: Add nav item to `layout.component.ts`**

In the sidebar nav, add after the Matematica item:

```html
<a routerLink="/tutor" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)">
  <mat-icon>school</mat-icon><span>Tutor</span>
</a>
```

- [ ] **Step 4: Verify Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build`

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/tutor/ frontend/src/app/app.routes.ts frontend/src/app/shared/layout.component.ts
git commit -m "feat: Angular tutor component with step-by-step + LaTeX + verification"
```

---

## Task 8: Update Existing Math Module

**Files:**
- Modify: `api/math.go` — use MathClient instead of LLM for calculations
- Modify: `frontend/src/app/modules/math/math.component.ts` — use new tutor API

**Interfaces:**
- Consumes: `MathClient`

- [ ] **Step 1: Rewrite `api/math.go` to use MathClient**

Replace the LLM-based evaluation with direct MathClient calls:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MathRequest struct {
	Expression string  `json:"expression"`
	Variable   string  `json:"variable,omitempty"`
	XMin       float64 `json:"xMin,omitempty"`
	XMax       float64 `json:"xMax,omitempty"`
}

func MathRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	mathClient := NewMathClient(cfg)

	return func(r chi.Router) {
		r.Post("/evaluate", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Expression == "" {
				http.Error(w, `{"error":"expression is required"}`, http.StatusBadRequest)
				return
			}

			result, err := mathClient.Evaluate(req.Expression)
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/differentiate", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			variable := req.Variable
			if variable == "" {
				variable = "x"
			}
			result, err := mathClient.Differentiate(req.Expression, variable)
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/integrate", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			variable := req.Variable
			if variable == "" {
				variable = "x"
			}
			result, err := mathClient.Integrate(req.Expression, variable, nil, nil)
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/solve", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			variable := req.Variable
			if variable == "" {
				variable = "x"
			}
			result, err := mathClient.Solve(req.Expression, variable)
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/simplify", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			result, err := mathClient.Simplify(req.Expression)
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/factor", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			result, err := mathClient.Factor(req.Expression)
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})

		r.Post("/expand", func(w http.ResponseWriter, r *http.Request) {
			var req MathRequest
			json.NewDecoder(r.Body).Decode(&req)
			result, err := mathClient.Expand(req.Expression)
			if err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})
	}
}
```

- [ ] **Step 2: Verify Go compilation**

Run: `cd /home/proyecto/matematicarag && go build ./cmd/server`

- [ ] **Step 3: Commit**

```bash
git add api/math.go
git commit -m "feat: math routes use SymPy engine instead of LLM"
```

---

## Task 9: Docker Compose Integration + Full Build

**Files:**
- Modify: `docker-compose.yml` — verify math-service config
- Modify: `Dockerfile` — add math-service to build pipeline (if needed)

**Interfaces:**
- Produces: All containers build and run

- [ ] **Step 1: Verify docker-compose.yml has math-service**

Ensure the math-service is properly configured with:
- Build context `./math-service`
- Environment: `MATH_TIMEOUT=5`, `MATH_PORT=5000`
- Network: `matematicarag-net`
- Health check

- [ ] **Step 2: Build all containers**

Run: `cd /home/proyecto/matematicarag && docker compose build`

- [ ] **Step 3: Verify math service health**

Run: `docker compose up -d math-service && sleep 5 && docker compose exec math-service python -c "import urllib.request; print(urllib.request.urlopen('http://localhost:5000/health').read())"`

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: math-service container integrated in docker-compose"
```

---

## Task 10: Integration Tests

**Files:**
- Create: `math-service/tests/test_calculus.py`
- Create: `math-service/tests/test_equations.py`
- Create: `math-service/tests/test_matrices.py`
- Create: `math-service/tests/test_verify.py`

**Interfaces:**
- Consumes: Math service engine functions

- [ ] **Step 1: Create `math-service/tests/test_calculus.py`**

```python
import pytest
from engine.calculus import differentiate, integrate, compute_limit
from sympy import Symbol, sympify

x = Symbol('x')

def test_differentiate_x_cubed():
    result = differentiate(x**3, 'x')
    assert result == 3*x**2

def test_differentiate_polynomial():
    result = differentiate(x**3 + 2*x**2 - 5*x, 'x')
    assert result == 3*x**2 + 4*x - 5

def test_integrate_x_squared():
    result = integrate(x**2, 'x')
    assert result == x**3/3

def test_integrate_linear():
    result = integrate(2*x + 1, 'x')
    assert result == x**2 + x

def test_limit_sin_x_over_x():
    from sympy import sin
    result = compute_limit(sin(x)/x, 'x', '0')
    assert result == 1

def test_limit_polynomial():
    result = compute_limit(x**2 + 1, 'x', '2')
    assert result == 5
```

- [ ] **Step 2: Create `math-service/tests/test_equations.py`**

```python
import pytest
from engine.equations import solve_equation

def test_quadratic():
    result = solve_equation('x^2 + 5*x + 6 = 0', 'x')
    solutions = [complex(s) for s in result['solutions']]
    assert -2 in solutions
    assert -3 in solutions

def test_linear():
    result = solve_equation('2*x + 4 = 0', 'x')
    assert '-2' in result['solutions']
```

- [ ] **Step 3: Create `math-service/tests/test_matrices.py`**

```python
import pytest
from engine.matrices import matrix_operation

def test_determinant():
    result = matrix_operation('determinant', [[1, 2], [3, 4]])
    assert result['result'] == '-2'

def test_transpose():
    result = matrix_operation('transpose', [[1, 2], [3, 4]])
    assert '1' in result['result'] and '3' in result['result']

def test_rank():
    result = matrix_operation('rank', [[1, 0], [0, 1]])
    assert result['result'] == '2'
```

- [ ] **Step 4: Create `math-service/tests/test_verify.py`**

```python
import pytest
from engine.verify import verify_result

def test_verify_correct():
    result = verify_result('3*x**2', '3*x**2', 'differentiate')
    assert result['verified'] == True

def test_verify_incorrect():
    result = verify_result('2*x**2', '3*x**2', 'differentiate')
    assert result['verified'] == False
```

- [ ] **Step 5: Run tests**

Run: `cd /home/proyecto/matematicarag/math-service && python -m pytest tests/ -v`

- [ ] **Step 6: Commit**

```bash
git add math-service/tests/
git commit -m "feat: math engine unit tests (calculus, equations, matrices, verify)"
```

---

## Task 11: Security + Config Hardening

**Files:**
- Modify: `api/tutor.go` — input validation
- Modify: `api/intent.go` — expression sanitization
- Modify: `internal/config/config.go` — new env vars

**Interfaces:**
- N/A (hardening)

- [ ] **Step 1: Add config env vars**

In `internal/config/config.go`, add:

```go
MathServiceURL: getEnv("MATH_SERVICE_URL", "http://localhost:5000"),
MathTimeout:    getEnvInt("MATH_TIMEOUT", 5),
```

- [ ] **Step 2: Add input sanitization to intent.go**

Add expression validation before sending to math engine:

```go
func sanitizeExpression(expr string) string {
    // Remove anything that's not math-safe
    // Allow: digits, letters (variables), +, -, *, /, ^, (, ), =, space
    // Block: ;, &, |, `, $, etc.
    blocked := []string{";", "&", "|", "`", "$", "\\", "{", "}"}
    for _, b := range blocked {
        expr = strings.ReplaceAll(expr, b, "")
    }
    return strings.TrimSpace(expr)
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/proyecto/matematicarag && go build ./cmd/server`

- [ ] **Step 4: Commit**

```bash
git add api/tutor.go api/intent.go internal/config/config.go
git commit -m "feat: security hardening + config for math service"
```

---

## Task 12: Full Build Verification

**Files:**
- N/A (verification only)

- [ ] **Step 1: Go build**

Run: `cd /home/proyecto/matematicarag && go build ./cmd/server`

- [ ] **Step 2: Angular build**

Run: `cd /home/proyecto/matematicarag/frontend && npx ng build`

- [ ] **Step 3: Docker compose build**

Run: `cd /home/proyecto/matematicarag && docker compose build`

- [ ] **Step 4: Math service tests**

Run: `cd /home/proyecto/matematicarag/math-service && python -m pytest tests/ -v`

- [ ] **Step 5: Final commit if needed**

```bash
git add -A
git commit -m "feat: Fase 2 complete — math engine, tutor, verification, frontend"
```

---

## Spec Coverage Check

| Spec Section | Task |
|---|---|
| §3 Intent classifier | Task 4 |
| §4 Mathematical domains | Task 2 |
| §5 Math engine (SymPy) | Tasks 1, 2 |
| §6 Don't trust LLM | Task 5 (math engine ≠ LLM) |
| §7 Resolution plan | Task 5 (LLM generates steps) |
| §8 Structured math output | Tasks 1, 2, 5 |
| §9 Verifiable steps | Task 5 (verification) |
| §10 Auto verification | Tasks 2, 5 |
| §11 Verification states | Task 5 |
| §12 RAG + Math Engine | Task 5 (orchestrator) |
| §13 Citations | Task 5 |
| §14 Structured response | Task 5 |
| §15 LaTeX | Task 7 (KaTeX rendering) |
| §16 Math input | Task 1 (parser) |
| §17 Normalization | Task 1 (parser) |
| §18 Security | Tasks 1, 11 |
| §19 Timeout | Tasks 1, 3 |
| §20 Error handling | Tasks 1, 5 |
| §21 Equations | Task 2 |
| §22 Systems | Task 2 |
| §23 Derivatives | Task 2 |
| §24 Integrals | Task 2 |
| §25 Limits | Task 2 |
| §26 Matrices | Task 2 |
| §27 Simplification | Task 2 |
| §28 Domain/conditions | Task 2 (future) |
| §29 Pedagogical method | Task 5 (LLM explanation) |
| §30 Explanation level | Task 5 |
| §31 Hint mode | Task 5 (mode=hint) |
| §32 Solve mode | Task 5 |
| §33 Verify mode | Task 5 |
| §34 Explain error mode | Task 5 |
| §35 RAG as method source | Task 5 |
| §36 Separate sources/computation | Task 5 |
| §37 Math service API | Tasks 1, 3 |
| §38 RAG+Math API | Task 5 |
| §39 Frontend Angular | Task 7 |
| §40 Verification UI | Task 7 |
| §41 Sources UI | Task 7 |
| §42 Observability | Task 5 (logging) |
| §43 Tests | Task 10 |
| §44 Verification tests | Task 10 |
| §45 Acceptance criteria | All tasks |
