from sympy import Symbol

from .parser import safe_parse

MAX_POINTS = 2000


def plot_expression(expr_str, xmin=-10.0, xmax=10.0, n=200):
    """Sample the expression over [xmin, xmax] as (x, y) pairs.

    Points where the expression is not a finite real number (asymptotes,
    complex values, infinities) come back as `None` so the frontend can
    break the curve instead of drawing a misleading vertical line.
    """
    if xmax <= xmin:
        raise ValueError("xmax must be greater than xmin")

    expr = safe_parse(expr_str)

    symbols = sorted(expr.free_symbols, key=lambda s: str(s))
    var = next((s for s in symbols if str(s) == 'x'), symbols[0] if symbols else Symbol('x'))

    n = max(2, min(int(n), MAX_POINTS))
    step = (xmax - xmin) / (n - 1)

    points = []
    for i in range(n):
        xv = xmin + step * i
        try:
            yv = expr.subs({var: xv}).evalf()
        except Exception:
            yv = None
        if yv is None or not getattr(yv, 'is_finite', False):
            points.append([round(xv, 6), None])
            continue
        try:
            fy = float(yv)
        except Exception:
            points.append([round(xv, 6), None])
            continue
        if abs(fy) > 1e12:
            points.append([round(xv, 6), None])
        else:
            points.append([round(xv, 6), fy])

    return {
        'expression': str(expr),
        'variable': str(var),
        'xmin': float(xmin),
        'xmax': float(xmax),
        'points': points,
    }
