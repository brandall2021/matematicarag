from .parser import safe_parse

def basic_evaluate(expr_str):
    result = safe_parse(expr_str).doit()
    return {"result": str(result.evalf()), "latex": str(result)}
