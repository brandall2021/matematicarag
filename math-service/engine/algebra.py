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
