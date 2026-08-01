from sympy import simplify
from .parser import safe_parse

def verify_result(expression: str, expected: str, operation: str = '') -> dict:
    try:
        expr_actual = safe_parse(expression)
        expr_expected = safe_parse(expected)
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
