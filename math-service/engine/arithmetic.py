from sympy import sympify

def basic_evaluate(expr_str):
    result = sympify(expr_str)
    return {"result": str(result.evalf()), "latex": str(result)}
