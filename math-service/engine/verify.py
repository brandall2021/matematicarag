from sympy import sympify, simplify
from sympy.parsing.latex import parse_latex

def verify_result(expression: str, expected: str, operation: str = '') -> dict:
    try:
        if '\\' in expression or '{' in expression:
            expr_actual = parse_latex(expression)
        else:
            expr_actual = sympify(expression)
        if '\\' in expected or '{' in expected:
            expr_expected = parse_latex(expected)
        else:
            expr_expected = sympify(expected)
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
        return {"verified": False, "method": "error", "error": str(e)}
