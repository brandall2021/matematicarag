import os
import signal
from functools import wraps
from flask import Flask, request, jsonify
from flask_cors import CORS
from engine.calculus import differentiate, integrate, compute_limit
from engine.equations import solve_equation, solve_system
from engine.algebra import simplify_expr, factor_expr, expand_expr
from engine.matrices import matrix_operation
from engine.verify import verify_result
from engine.arithmetic import basic_evaluate as evaluate_expression

app = Flask(__name__)
_allowed_origins = os.environ.get('CORS_ALLOWED_ORIGINS', 'http://localhost:8009').split(',')
CORS(app, resources={r"/*": {"origins": [o.strip() for o in _allowed_origins if o.strip()]}})

MATH_TIMEOUT = int(os.environ.get('MATH_TIMEOUT', 5))
MAX_INPUT_LENGTH = 1000

class MathTimeoutError(Exception):
    pass

def timeout_handler(signum, frame):
    raise MathTimeoutError("Computation timed out")

def with_timeout(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        old_handler = signal.signal(signal.SIGALRM, timeout_handler)
        signal.alarm(MATH_TIMEOUT)
        try:
            result = f(*args, **kwargs)
            signal.alarm(0)
            return result
        except MathTimeoutError:
            return jsonify({'success': False, 'error': 'Computation timed out'}), 504
        except ValueError as e:
            signal.alarm(0)
            return jsonify({'success': False, 'error': str(e)}), 400
        except Exception:
            signal.alarm(0)
            return jsonify({'success': False, 'error': 'Math operation failed'}), 400
        finally:
            signal.signal(signal.SIGALRM, old_handler)
    return decorated_function

def _input(payload, key):
    """Fetch a required string input with a length guard."""
    value = payload.get(key) if isinstance(payload, dict) else None
    if value is None:
        return None
    value = str(value)
    if len(value) > MAX_INPUT_LENGTH:
        raise ValueError('Input too long')
    return value


def route(path, **options):
    """Register a route at path AND at /api/path for backend compatibility."""
    def decorator(f):
        app.add_url_rule(path, endpoint=f.__name__, view_func=f, **options)
        app.add_url_rule('/api' + path, endpoint='api_' + f.__name__, view_func=f, **options)
        return f
    return decorator


@app.route('/health', methods=['GET'])
@app.route('/api/math/health', methods=['GET'])
def health():
    return jsonify({'status': 'healthy', 'service': 'math-service'})


@route('/math/evaluate', methods=['POST'])
@with_timeout
def math_evaluate():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    result = evaluate_expression(expression)
    return jsonify({'success': True, 'result': result['result'], 'latex': result['latex']})


@route('/math/simplify', methods=['POST'])
@with_timeout
def math_simplify():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    result = simplify_expr(expression)
    latex = str(result)
    return jsonify({'success': True, 'result': latex, 'latex': latex})


@route('/math/factor', methods=['POST'])
@with_timeout
def math_factor():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    result = factor_expr(expression)
    latex = str(result)
    return jsonify({'success': True, 'result': latex, 'latex': latex})


@route('/math/expand', methods=['POST'])
@with_timeout
def math_expand():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    result = expand_expr(expression)
    latex = str(result)
    return jsonify({'success': True, 'result': latex, 'latex': latex})


@route('/math/differentiate', methods=['POST'])
@with_timeout
def math_differentiate():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    var = _input(data, 'variable') or 'x'
    order = data.get('order', 1)
    result = differentiate(expression, var, order)
    return jsonify({'success': True, 'result': result, 'latex': result})


@route('/math/integrate', methods=['POST'])
@with_timeout
def math_integrate():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    var = _input(data, 'variable') or 'x'
    lower = data.get('lower')
    upper = data.get('upper')
    result = integrate(expression, var, lower, upper)
    return jsonify({'success': True, 'result': result, 'latex': result})


@route('/math/limit', methods=['POST'])
@with_timeout
def math_limit():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    var = _input(data, 'variable') or 'x'
    target = _input(data, 'target') or '0'
    direction = _input(data, 'direction') or '+'
    result = compute_limit(expression, var, target, direction)
    return jsonify({'success': True, 'result': result, 'latex': result})


@route('/math/solve', methods=['POST'])
@with_timeout
def math_solve():
    data = request.get_json()
    equations = data.get('equations', []) if data else []
    if equations:
        variables = data.get('variables', 'x, y')
        result = solve_system(equations, variables)
    else:
        expression = _input(data, 'expression')
        if not expression:
            return jsonify({'success': False, 'error': 'Expression or equations is required'}), 400
        var = _input(data, 'variable') or 'x'
        result = solve_equation(expression, var)
    result['success'] = True
    return jsonify(result)


@route('/math/matrix', methods=['POST'])
@with_timeout
def math_matrix():
    data = request.get_json()
    if not data or 'operation' not in data:
        return jsonify({'success': False, 'error': 'Operation is required'}), 400
    operation = data['operation']
    result = matrix_operation(operation, data)
    if 'error' in result:
        return jsonify({'success': False, 'error': result['error']}), 400
    result['success'] = True
    return jsonify(result)


@route('/math/verify', methods=['POST'])
@with_timeout
def math_verify():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    expected = _input(data, 'expected')
    result = verify_result(expression, expected)
    result['success'] = True
    return jsonify(result)


@route('/math/validate-exercise', methods=['POST'])
@with_timeout
def math_validate_exercise():
    data = request.get_json()
    expression = _input(data, 'expression')
    if not expression:
        return jsonify({'success': False, 'error': 'Expression is required'}), 400
    expected = _input(data, 'expected') or ''
    result = verify_result(expression, expected)
    result['success'] = True
    return jsonify(result)


if __name__ == '__main__':
    port = int(os.environ.get('MATH_PORT', 5000))
    app.run(host='0.0.0.0', port=port)
