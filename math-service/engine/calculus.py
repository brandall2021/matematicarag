from .parser import safe_parse
from sympy import diff, integrate as sym_integrate, limit, Symbol, oo

def differentiate(expr_str: str, var: str = 'x', order: int = 1):
    expr = safe_parse(expr_str)
    x = Symbol(var)
    result = expr
    for _ in range(order):
        result = diff(result, x)
    return str(result)

def integrate(expr_str: str, var: str = 'x', lower=None, upper=None):
    expr = safe_parse(expr_str)
    x = Symbol(var)
    if lower is not None and upper is not None:
        lower_expr = safe_parse(str(lower))
        upper_expr = safe_parse(str(upper))
        result = sym_integrate(expr, (x, lower_expr, upper_expr))
    else:
        result = sym_integrate(expr, x)
    return str(result)

def compute_limit(expr_str: str, var: str = 'x', target: str = '0', direction: str = '+'):
    expr = safe_parse(expr_str)
    x = Symbol(var)
    if target == 'oo' or target == 'inf':
        target_val = oo
    else:
        target_val = safe_parse(target)
    result = limit(expr, x, target_val, direction)
    return str(result)
