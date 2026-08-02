from sympy import simplify, factor, expand, collect, cancel, together
from .parser import safe_parse

def _as_expr(expr):
    if isinstance(expr, str):
        return safe_parse(expr)
    return expr

def simplify_expr(expr):
    return simplify(_as_expr(expr))

def factor_expr(expr):
    return factor(_as_expr(expr))

def expand_expr(expr):
    return expand(_as_expr(expr))

def collect_expr(expr, var_str='x'):
    from sympy import Symbol
    return collect(_as_expr(expr), Symbol(var_str))

def cancel_expr(expr):
    return cancel(_as_expr(expr))

def together_expr(expr):
    return together(_as_expr(expr))
